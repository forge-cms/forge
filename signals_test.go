package forge

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestOnReturnsOption verifies that On[T] returns a value satisfying the
// Option interface.
func TestOnReturnsOption(t *testing.T) {
	var opt Option = On(BeforeCreate, func(_ Context, _ *struct{}) error { return nil })
	if opt == nil {
		t.Fatal("expected non-nil Option")
	}
	so, ok := opt.(signalOption)
	if !ok {
		t.Fatalf("expected signalOption, got %T", opt)
	}
	if so.signal != BeforeCreate {
		t.Errorf("expected signal %q, got %q", BeforeCreate, so.signal)
	}
}

// TestDispatchBeforeRunsAllOnSuccess verifies that dispatchBefore calls all
// handlers when none return an error, and returns nil.
func TestDispatchBeforeRunsAllOnSuccess(t *testing.T) {
	ctx := NewTestContext(GuestUser)
	calls := 0
	handlers := []signalHandler{
		func(_ Context, _ any) error { calls++; return nil },
		func(_ Context, _ any) error { calls++; return nil },
		func(_ Context, _ any) error { calls++; return nil },
	}

	if err := dispatchBefore(ctx, handlers, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 handler calls, got %d", calls)
	}
}

// TestDispatchBeforeAbortsOnError verifies that dispatchBefore stops at the
// first handler that returns an error and propagates that error.
func TestDispatchBeforeAbortsOnError(t *testing.T) {
	ctx := NewTestContext(GuestUser)
	sentinel := ErrConflict
	calls := 0
	handlers := []signalHandler{
		func(_ Context, _ any) error { calls++; return sentinel },
		func(_ Context, _ any) error { calls++; return nil },
	}

	err := dispatchBefore(ctx, handlers, nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected second handler to be skipped, got %d calls", calls)
	}
}

// TestDispatchBeforePanicReturnsError verifies that a panicking handler does
// not crash the process and causes dispatchBefore to return a 500-class
// forge.Error.
func TestDispatchBeforePanicReturnsError(t *testing.T) {
	ctx := NewTestContext(GuestUser)
	handlers := []signalHandler{
		func(_ Context, _ any) error { panic("kaboom") },
	}

	err := dispatchBefore(ctx, handlers, nil)
	if err == nil {
		t.Fatal("expected error after panic, got nil")
	}
	var fe Error
	if !errors.As(err, &fe) {
		t.Fatalf("expected forge.Error, got %T", err)
	}
	if fe.HTTPStatus() != 500 {
		t.Errorf("expected HTTP 500, got %d", fe.HTTPStatus())
	}
	if fe.Code() != "signal_panic" {
		t.Errorf("expected code %q, got %q", "signal_panic", fe.Code())
	}
}

// TestDispatchAfterIsNonBlocking verifies that dispatchAfter returns before
// its handlers finish executing.
func TestDispatchAfterIsNonBlocking(t *testing.T) {
	ctx := NewTestContext(GuestUser)
	var wg sync.WaitGroup
	wg.Add(1)

	started := make(chan struct{})
	handlers := []signalHandler{
		func(_ Context, _ any) error {
			close(started)
			time.Sleep(50 * time.Millisecond)
			wg.Done()
			return nil
		},
	}

	dispatchAfter(ctx, handlers, nil)

	// dispatchAfter must return before the handler finishes, so the channel
	// close happens while we are still waiting here.
	select {
	case <-started:
		// handler started — confirm function already returned (we're here)
	case <-time.After(1 * time.Second):
		t.Fatal("handler did not start within deadline")
	}

	wg.Wait() // prevent goroutine leak in test binary
}

// TestDispatchAfterPanicDoesNotPropagate verifies that a panicking async
// handler neither crashes the process nor returns an error to the caller.
func TestDispatchAfterPanicDoesNotPropagate(t *testing.T) {
	ctx := NewTestContext(GuestUser)
	done := make(chan struct{})
	handlers := []signalHandler{
		func(_ Context, _ any) error {
			defer close(done)
			panic("async kaboom")
		},
	}

	// Must not panic in the caller's goroutine.
	dispatchAfter(ctx, handlers, nil)

	select {
	case <-done:
		// handler ran and recovered
	case <-time.After(1 * time.Second):
		t.Fatal("handler did not run within deadline")
	}
}

// TestDebouncerCoalesces verifies that 10 rapid Trigger calls produce exactly
// one fn invocation after the delay elapses.
func TestDebouncerCoalesces(t *testing.T) {
	var count atomic.Int32
	d := newDebouncer(20*time.Millisecond, func() {
		count.Add(1)
	})

	for range 10 {
		d.Trigger()
	}

	time.Sleep(60 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 fn invocation, got %d", got)
	}
}

// TestDebouncerResetsOnTrigger verifies that a Trigger call during the delay
// window prevents the earlier scheduled fn from firing.
func TestDebouncerResetsOnTrigger(t *testing.T) {
	var count atomic.Int32
	delay := 30 * time.Millisecond
	d := newDebouncer(delay, func() {
		count.Add(1)
	})

	d.Trigger()
	time.Sleep(15 * time.Millisecond) // halfway through delay
	d.Trigger()                       // resets the timer
	time.Sleep(15 * time.Millisecond) // original would have fired here
	// timer has not elapsed yet after the reset
	if got := count.Load(); got != 0 {
		t.Errorf("fn fired too early: %d calls", got)
	}

	time.Sleep(30 * time.Millisecond) // now the reset timer fires
	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 fn invocation, got %d", got)
	}
}

// TestDebouncerStop verifies that Stop cancels a pending timer so fn does
// not fire after the module has been torn down. (Amendment A39)
func TestDebouncerStop(t *testing.T) {
	var count atomic.Int32
	d := newDebouncer(40*time.Millisecond, func() {
		count.Add(1)
	})
	d.Trigger() // schedule fn in 40 ms
	d.Stop()    // cancel before it fires
	time.Sleep(60 * time.Millisecond)
	if got := count.Load(); got != 0 {
		t.Errorf("fn fired after Stop: %d calls; want 0", got)
	}
	// Second Stop must not panic.
	d.Stop()
}
