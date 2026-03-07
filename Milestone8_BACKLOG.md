# Forge — Milestone 8 Backlog (v0.8.0)

Adaptive background scheduler that transitions `Scheduled → Published` at the
exact `scheduled_at` time with no external cron. Single goroutine, self-resetting
`time.Timer` (60s fallback), fires `AfterPublish` signals, triggers sitemap/feed
debounce, cleans up within the server's 5-second graceful-shutdown window.

**Key decisions:**
- Decision 14 — Content lifecycle (`Scheduled` status, `scheduled_at`, `AfterPublish`)
- Decision 9 — Sitemap strategy (signal-driven regeneration per publish)
- Amendment A23 — `db` struct tags added to `Node.ScheduledAt`, `PublishedAt`, etc.
- Amendment A24 — `NewBackgroundContext(cfg Config) Context` added to `context.go`
- Amendment A25 — `Module[T].processScheduled` implements `schedulableModule`
- Amendment A26 — `App.schedulerModules`, wiring in `Content()` and `Run()`

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | scheduler.go | ✅ Done | 2026-03-07 |
| 2 | integration_full_test.go | 🔲 Not started | — |

---

## Layer 8.A — Core scheduler (no prior M8 dependency)

### Step 1 — `scheduler.go` (new) + amendments A23–A26

**Depends on:** `node.go`, `context.go`, `module.go`, `forge.go`, `storage.go`
**Decisions:** Decision 14, Decision 9, Amendments A23, A24, A25, A26
**Files:** `scheduler.go` (new), `scheduler_test.go` (new) + amendments to
`node.go`, `context.go`, `module.go`, `forge.go`

#### 1.1 — Amendment A23: `db` struct tags on `Node` (node.go)

- [x] Add `db:"published_at"` to `Node.PublishedAt`
- [x] Add `db:"scheduled_at"` to `Node.ScheduledAt`
- [x] Add `db:"created_at"` to `Node.CreatedAt`
- [x] Add `db:"updated_at"` to `Node.UpdatedAt`
- [x] Add godoc note: "Column names follow snake_case from the db tag"
- [x] Verify `dbFields` now resolves `published_at`, `scheduled_at`, `created_at`, `updated_at` columns

#### 1.2 — Amendment A24: `NewBackgroundContext` (context.go)

- [x] Define `NewBackgroundContext(siteName string) Context`:
  - Creates a synthetic `GET /` `*http.Request` (same pattern as `NewTestContext`)
  - Wraps with `context.Background()` (not request context — long-lived)
  - Sets `siteName` from parameter
  - Sets `user: GuestUser`, `locale: "en"`, `requestID: NewID()`
  - Uses `httptest.NewRecorder()` for the ResponseWriter
- [x] Add godoc: "NewBackgroundContext returns a Context for use in background
      goroutines (e.g. the scheduler). It has no HTTP lifecycle and never times out."
- [x] Add to existing context_test.go: `TestNewBackgroundContext` — verify
      `ctx.SiteName()`, `ctx.User()`, `ctx.Request()` non-nil, `ctx.RequestID()` non-empty

#### 1.3 — `schedulableModule` interface (scheduler.go)

- [x] Define unexported `schedulableModule` interface:
  ```go
  type schedulableModule interface {
      processScheduled(ctx Context, now time.Time) (published int, next *time.Time, err error)
  }
  ```
- [x] Add godoc explaining contract: `processScheduled` transitions all items
  with `ScheduledAt <= now` from `Scheduled → Published`, fires `AfterPublish`,
  triggers sitemap; returns count of items published and the soonest remaining
  `ScheduledAt` (nil = none remaining)

#### 1.4 — Amendment A25: `Module[T].processScheduled` (module.go)

- [x] Add unexported reflection helpers to `module.go`:
  - `setNodeTime(item any, field string, t time.Time)` — sets a `time.Time` field
    via `goFieldPath`, calls `reflect.Value.Set`
  - `setNodeTimePtr(item any, field string, t *time.Time)` — sets a `*time.Time` field
- [x] Implement `func (m *Module[T]) processScheduled(ctx Context, now time.Time) (int, *time.Time, error)`:
  1. `items, err := m.repo.FindAll(ctx.Request().Context(), ListOptions{Status: []Status{Scheduled}})`
  2. `published := 0; var next *time.Time`
  3. For each item:
     - Read `ScheduledAt` via `goFieldPath` on the pointer's elem type
     - If `ScheduledAt == nil || ScheduledAt.After(now)`: update `next` if sooner, continue
     - `setNodeTime(item, "PublishedAt", now)` — sets `PublishedAt`  
     - `setNodeTimePtr(item, "ScheduledAt", nil)` — clears `ScheduledAt`
     - Set `Status = Published` via `goFieldPath`
     - `m.repo.Save(ctx.Request().Context(), item)`
     - `dispatchAfter(ctx, m.signals[AfterPublish], item)`
     - `published++`
     - After loop: if `published > 0`, call `m.invalidateCache()` + `m.triggerSitemap(ctx)`
  4. Return `published, next, nil`
- [x] Satisfies `schedulableModule` interface (compile-time check via type assertion in forge.go)

#### 1.5 — `Scheduler` struct and `newScheduler` (scheduler.go)

- [x] Define `type Scheduler struct`:
  - `modules []schedulableModule`
  - `bgCtx   Context`
  - `done    chan struct{}`
- [x] Implement `func newScheduler(modules []schedulableModule, bgCtx Context) *Scheduler`
- [x] Implement `func (s *Scheduler) Start(ctx context.Context)`:
  - Spawns goroutine running `s.run(ctx)`
- [x] Implement `func (s *Scheduler) Wait()`:
  - Blocks until `s.done` is closed (set at end of `run`)
- [x] Implement `func (s *Scheduler) tick(ctx context.Context) *time.Time`:
  - Calls `processScheduled(s.bgCtx, time.Now().UTC())` on each module
  - Aggregates minimum `next` across all modules
  - Returns overall minimum `next` (nil = no scheduled items remain)
- [x] Implement `func (s *Scheduler) run(ctx context.Context)`:
  - Defer: `close(s.done)`
  - Call `s.tick()` once immediately (process overdue items from before boot)
  - Compute `dur` from tick result: if `next == nil`, use 60s; else `max(1ms, time.Until(*next))`
  - `timer := time.NewTimer(dur)`
  - Loop: `select { case <-ctx.Done(): timer.Stop(); return; case <-timer.C: next = s.tick(); timer.Reset(nextDur(next)) }`
  - Unexported `nextDur(next *time.Time) time.Duration` helper:
    if `next == nil`, return 60s; else `max(time.Millisecond, time.Until(*next))`

#### 1.6 — Amendment A26: forge.go wiring

- [x] Add `schedulerModules []schedulableModule` field to `App` struct
- [x] In `App.Content()`: after registering the module, append `r` to `a.schedulerModules`
  (always — every `*Module[T]` satisfies `schedulableModule` after A25)

  ```go
  if sm, ok := r.(schedulableModule); ok {
      a.schedulerModules = append(a.schedulerModules, sm)
  }
  ```
- [x] In `App.Run()`: before `srv.ListenAndServe`, if `len(a.schedulerModules) > 0`:
  1. Parse `a.cfg.BaseURL` to extract hostname for background context
  2. Create `bgCtx := NewBackgroundContext(hostname)`
  3. Create `sched := newScheduler(a.schedulerModules, bgCtx)`
  4. Create `schedCtx, schedCancel := context.WithCancel(context.Background())`
  5. `sched.Start(schedCtx)`
  6. In the shutdown block (after `srv.Shutdown`): `schedCancel(); sched.Wait()`
- [x] Add godoc to the scheduler startup block: "The scheduler goroutine is
      started before the HTTP server and stopped after graceful HTTP shutdown
      to allow any in-progress tick to complete (bounded to the 5-second shutdown window)"

#### 1.7 — `scheduler_test.go` (new)

- [x] `TestScheduler_processesOverdue` — `MemoryRepo[*testSchedPost]` with 1
  past-due Scheduled item; call `processScheduled(bgCtx, now)`; verify
  `Status == Published`, `PublishedAt` non-zero, `ScheduledAt == nil`
- [x] `TestScheduler_skipsNotYetDue` — `ScheduledAt = now.Add(1h)`; call `processScheduled`;
  item unchanged, `next` returned points to the future time
- [x] `TestScheduler_firesAfterPublish` — register `On[T](AfterPublish, ...)` handler;
  tick; verify handler fired exactly once with the published item
- [x] `TestScheduler_multipleModules_aggregatesNext` — two modules;
  one past-due, one future; tick aggregates minimum `next` correctly
- [x] `TestScheduler_noModules` — empty `newScheduler(nil, bgCtx)`;
  `Start + cancel context + Wait` completes without panic
- [x] `TestNewBackgroundContext` — `ctx.SiteName() == "example.com"`,
  `ctx.User() == GuestUser`, `ctx.Request() != nil`, `ctx.RequestID() != ""`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestScheduler|TestNewBackground ./...` — all green
- [x] `go test ./...` — full suite green
- [x] `BACKLOG.md` — step 1 row and summary checkbox updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 8.B — Integration + README (depends on Layer 8.A)

### Step 2 — `integration_full_test.go` G19–G20 + README badge

**Depends on:** `scheduler.go`, `module.go` (A25), `forge.go` (A26)
**Decisions:** Decision 14
**Files:** `integration_full_test.go` (append G19–G20), `README.md`

#### 2.1 — G19: Scheduler end-to-end with MemoryRepo (M8, Decision 14)

- [ ] Seed 2 items in a `MemoryRepo[*testSchedPost]`: one past-due Scheduled,
  one future Scheduled
- [ ] Create `Module[*testSchedPost]` and call `processScheduled(bgCtx, now)` directly
- [ ] Assert past-due: `Status == Published`, `PublishedAt` non-zero, `ScheduledAt == nil`
- [ ] Assert future: `Status == Scheduled`, unchanged
- [ ] Assert returned `next` ≈ the future item's `ScheduledAt`
- [ ] Cross-milestone M1: register `On[*testSchedPost](AfterPublish, ...)` and assert fired once

#### 2.2 — G20: Scheduler integration with App, signals, sitemap (M8 + M3 + M2)

- [ ] Build a full `App` with a module using `SitemapConfig`
- [ ] Seed a past-due Scheduled item
- [ ] Call `app.Handler()` to wire routes + start conditions
- [ ] Manually call `tick()` on the scheduler (same package — accessible)
- [ ] Assert the item is now Published (via the module's repo)
- [ ] Wait for debounce to settle (small sleep or direct trigger)
- [ ] Assert sitemap fragment has been updated with the newly published item

#### 2.3 — README badge update

- [ ] Update README.md line 273: `🔲 **Coming in Milestone 8**` →
  `✅ **Available** — the adaptive ticker and automatic Scheduled → Published
  transition are implemented as of Milestone 8.`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestFull_scheduler ./...` — all green
- [ ] `go test ./...` — full suite green
- [ ] `BACKLOG.md` — step 2 row and summary checkbox updated; M8 milestone row marked ✅
- [ ] `README.md` — Scheduled publishing badge updated to ✅ Available
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 8

- [ ] `Node` has correct `db:"published_at"` / `db:"scheduled_at"` tags (A23)
- [ ] `NewBackgroundContext(siteName)` exists and is used by the scheduler (A24)
- [ ] `Module[T].processScheduled` — transitions Scheduled → Published, fires
      AfterPublish, triggers sitemap/feed debounce (A25)
- [ ] `Scheduler` — adaptive timer with 60s fallback; starts in `App.Run()`;
      shuts down cleanly after graceful HTTP shutdown (A26)
- [ ] Integration tests G19–G20 appended and passing
- [ ] README Scheduled publishing badge updated to ✅ Available
- [ ] `go test ./...` green; `go vet ./...` clean; `gofmt -l .` empty
