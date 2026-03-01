package forge

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// — statusRecorder ————————————————————————————————————————————————————————

// statusRecorder wraps an http.ResponseWriter and captures the HTTP status
// code written by the handler. Used by [RequestLogger].
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// — RequestLogger —————————————————————————————————————————————————————————

// RequestLogger returns middleware that logs each request using structured
// [log/slog] output. Fields: method, path, status, duration_ms, request_id.
//
// RequestLogger calls [ContextFrom] before the next handler, which ensures
// X-Request-ID is set on the response prior to any downstream code running.
// It should be the outermost middleware in [app.Use].
func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := ContextFrom(w, r)
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			slog.InfoContext(r.Context(), "forge: request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", float64(time.Since(start).Microseconds())/1000.0,
				"request_id", ctx.RequestID(),
			)
		})
	}
}

// — Recoverer —————————————————————————————————————————————————————————————

// Recoverer returns middleware that recovers from panics in downstream
// handlers. On panic it returns a 500 response via [WriteError] and logs
// the stack trace. The process is never crashed.
func Recoverer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					slog.ErrorContext(r.Context(), "forge: panic recovered",
						"panic", fmt.Sprintf("%v", rec),
						"stack", string(buf[:n]),
					)
					WriteError(w, r, newSentinel(
						http.StatusInternalServerError,
						"panic",
						"Internal server error",
					))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// — CORS ——————————————————————————————————————————————————————————————————

// CORS returns middleware that sets cross-origin resource sharing headers
// allowing requests from origin. On OPTIONS preflight requests it responds
// with 204 No Content without calling the next handler.
func CORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// — MaxBodySize ———————————————————————————————————————————————————————————

// MaxBodySize returns middleware that limits the size of request bodies to n
// bytes. Requests exceeding the limit receive a 413 error response.
func MaxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

// — SecurityHeaders ———————————————————————————————————————————————————————

// SecurityHeaders returns middleware that sets a standard set of security
// response headers on every response:
//
//   - Strict-Transport-Security (2-year max-age, includeSubDomains)
//   - X-Frame-Options: DENY
//   - X-Content-Type-Options: nosniff
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Content-Security-Policy: default-src 'self'
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'self'")
			next.ServeHTTP(w, r)
		})
	}
}

// — RateLimit (token bucket per IP) ———————————————————————————————————————

// ipBucket is a token bucket for a single remote IP address.
type ipBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastSeen time.Time
}

// rateLimiter holds the per-IP buckets and shared rate parameters.
type rateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*ipBucket
	rate    float64 // tokens per second
	max     float64 // burst capacity
}

func (rl *rateLimiter) allow(ip string) (bool, time.Duration) {
	rl.mu.RLock()
	b := rl.buckets[ip]
	rl.mu.RUnlock()

	if b == nil {
		b = &ipBucket{tokens: rl.max, lastSeen: time.Now()}
		rl.mu.Lock()
		if existing := rl.buckets[ip]; existing != nil {
			b = existing
		} else {
			rl.buckets[ip] = b
		}
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.max {
		b.tokens = rl.max
	}
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	wait := time.Duration((1-b.tokens)/rl.rate*1e9) * time.Nanosecond
	return false, wait
}

func (rl *rateLimiter) sweep(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	for ip, b := range rl.buckets {
		b.mu.Lock()
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, ip)
		}
		b.mu.Unlock()
	}
}

// RateLimit returns middleware that enforces a per-IP token bucket rate limit
// of n requests per duration d. Requests exceeding the limit receive a 429
// Too Many Requests response with a Retry-After header.
//
// A background goroutine sweeps stale IP buckets every d to bound memory usage.
func RateLimit(n int, d time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		buckets: make(map[string]*ipBucket),
		rate:    float64(n) / d.Seconds(),
		max:     float64(n),
	}
	// Background sweep: remove buckets idle for more than 2×d.
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for range ticker.C {
			rl.sweep(2 * d)
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			ok, wait := rl.allow(ip)
			if !ok {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(wait.Seconds())+1))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// — InMemoryCache (LRU) ———————————————————————————————————————————————————

// cacheMaxEntriesOption implements [Option] and carries a custom LRU size for
// [InMemoryCache]. Use [CacheMaxEntries] to create one.
type cacheMaxEntriesOption struct{ n int }

func (cacheMaxEntriesOption) isOption() {}

// CacheMaxEntries returns an [Option] that configures [InMemoryCache] to hold
// at most n entries, evicting the least-recently-used entry when full.
// The default is 1000 entries.
func CacheMaxEntries(n int) Option { return cacheMaxEntriesOption{n: n} }

// lruEntry is a node in the doubly-linked LRU list and the cached response.
type lruEntry struct {
	key     string
	body    []byte
	header  http.Header
	status  int
	expires time.Time
	prev    *lruEntry
	next    *lruEntry
}

// lruCache is a thread-safe LRU cache of HTTP responses.
type lruCache struct {
	mu      sync.Mutex
	entries map[string]*lruEntry
	head    *lruEntry // most recently used
	tail    *lruEntry // least recently used
	max     int
	count   int
	ttl     time.Duration
}

func newLRUCache(max int, ttl time.Duration) *lruCache {
	return &lruCache{entries: make(map[string]*lruEntry), max: max, ttl: ttl}
}

// get retrieves an entry. Returns nil if absent or expired (lazy eviction).
func (c *lruCache) get(key string) *lruEntry {
	e := c.entries[key]
	if e == nil {
		return nil
	}
	if time.Now().After(e.expires) {
		c.remove(e)
		return nil
	}
	c.moveToFront(e)
	return e
}

// set inserts or replaces an entry, evicting the LRU tail if at capacity.
func (c *lruCache) set(e *lruEntry) {
	if old, ok := c.entries[e.key]; ok {
		c.remove(old)
	}
	c.entries[e.key] = e
	c.count++
	c.pushFront(e)
	if c.count > c.max {
		c.evict()
	}
}

func (c *lruCache) remove(e *lruEntry) {
	delete(c.entries, e.key)
	c.count--
	c.unlink(e)
}

func (c *lruCache) evict() {
	if c.tail != nil {
		c.remove(c.tail)
	}
}

func (c *lruCache) pushFront(e *lruEntry) {
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *lruCache) moveToFront(e *lruEntry) {
	if c.head == e {
		return
	}
	c.unlink(e)
	c.pushFront(e)
}

func (c *lruCache) unlink(e *lruEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
	e.prev = nil
	e.next = nil
}

// sweep removes all expired entries.
func (c *lruCache) sweep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for _, e := range c.entries {
		if now.After(e.expires) {
			c.remove(e)
		}
	}
}

// cacheRecorder captures the response written by a handler for caching.
type cacheRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
	header http.Header
}

func newCacheRecorder(w http.ResponseWriter) *cacheRecorder {
	// Copy headers already set on the delegate writer.
	h := make(http.Header)
	for k, v := range w.Header() {
		h[k] = v
	}
	return &cacheRecorder{ResponseWriter: w, header: h, status: http.StatusOK}
}

func (cr *cacheRecorder) WriteHeader(code int) {
	cr.status = code
	cr.ResponseWriter.WriteHeader(code)
}

func (cr *cacheRecorder) Write(b []byte) (int, error) {
	cr.body = append(cr.body, b...)
	return cr.ResponseWriter.Write(b)
}

// InMemoryCache returns middleware that caches successful GET responses in an
// LRU cache. Responses are keyed by method + full URL (including query
// parameters) + Accept header. Every response receives an X-Cache header
// (HIT or MISS).
//
// Default capacity is 1000 entries. Use [CacheMaxEntries] to override.
// A background goroutine sweeps expired entries every 60 seconds.
func InMemoryCache(ttl time.Duration, opts ...Option) func(http.Handler) http.Handler {
	max := 1000
	for _, o := range opts {
		if m, ok := o.(cacheMaxEntriesOption); ok {
			max = m.n
		}
	}
	cache := newLRUCache(max, ttl)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cache.sweep()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Method + " " + r.URL.RequestURI() + " " + r.Header.Get("Accept")

			cache.mu.Lock()
			entry := cache.get(key)
			cache.mu.Unlock()

			if entry != nil {
				for k, vals := range entry.header {
					for _, v := range vals {
						w.Header().Add(k, v)
					}
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(entry.status)
				w.Write(entry.body) //nolint:errcheck
				return
			}

			w.Header().Set("X-Cache", "MISS")
			rec := newCacheRecorder(w)
			next.ServeHTTP(rec, r)

			if rec.status == http.StatusOK {
				// Snapshot headers at response time (after handler set them).
				h := make(http.Header)
				for k, v := range rec.ResponseWriter.Header() {
					h[k] = v
				}
				e := &lruEntry{
					key:     key,
					body:    rec.body,
					header:  h,
					status:  rec.status,
					expires: time.Now().Add(ttl),
				}
				cache.mu.Lock()
				cache.set(e)
				cache.mu.Unlock()
			}
		})
	}
}

// — Chain —————————————————————————————————————————————————————————————————

// Chain applies a list of middleware to an http.Handler. The first middleware
// in the slice becomes the outermost wrapper (executed first on each request).
//
//	Chain(myHandler, RequestLogger(), Recoverer(), SecurityHeaders())
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
