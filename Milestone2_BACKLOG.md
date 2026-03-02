# Forge — Milestone 2 Backlog (v0.2.0)

A developer can write `forge.New(cfg)`, wire up modules with `app.Content`, and run a
production-ready HTTP server with graceful shutdown.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | forge.go | ✅ Done | 2026-03-02 |
| P1 | forge-pgx/pgx.go | ✅ Done | 2026-03-02 |

---

## Layer 1 — App Bootstrap (depends on all M1 files)

### Step 1 — forge.go

**Deferred from:** Milestone 1, Step 11
**Depends on:** errors.go, roles.go, context.go, middleware.go, module.go, storage.go, auth.go
**Decisions:** Decision 20 (configuration model), Decision 21 (Context interface), Decision 22 (DB interface and Config.DB)
**Files:** `forge.go`, `forge_test.go`

#### 1.1 — Config type and defaults

- [x] Define `Config` struct with fields:
  - `BaseURL string` — required; e.g. `"https://example.com"` (no trailing slash)
  - `Secret []byte` — required; min 16 bytes; used for HMAC tokens and cookies
  - `DB DB` — optional; database connection satisfying `forge.DB`
  - `HTTPS bool` — optional; when true, adds HTTP→HTTPS redirect middleware and marks cookies Secure
  - `ReadTimeout time.Duration` — optional; defaults to 5 s
  - `WriteTimeout time.Duration` — optional; defaults to 10 s
  - `IdleTimeout time.Duration` — optional; defaults to 120 s
- [x] Godoc comment on `Config` clearly describing each field, zero values, and defaults
- [x] Define unexported package-level constants `defaultReadTimeout = 5*time.Second`, `defaultWriteTimeout = 10*time.Second`, `defaultIdleTimeout = 120*time.Second`
- [x] Define `MustConfig(cfg Config) Config` — validates Config, panics with a descriptive `forge.Error` message if invalid:
  - `BaseURL` empty → panic `"forge: Config.BaseURL is required (e.g. \"https://example.com\")"`
  - `BaseURL` not parseable as an absolute URL → panic with the parse error
  - `len(Secret) < 16` → panic `"forge: Config.Secret must be at least 16 bytes"`
  - Returns the validated Config unchanged (allows `forge.New(forge.MustConfig(cfg))`)
- [x] Godoc comment on `MustConfig` explaining the panic behaviour and the typical usage pattern

#### 1.2 — App type and constructor

- [x] Define unexported `App` struct with fields:
  - `cfg Config` — stored Config (with defaults applied)
  - `mux *http.ServeMux` — freshly allocated mux
  - `middleware []func(http.Handler) http.Handler` — global middleware stack
- [x] Define `New(cfg Config) *App`:
  - Applies default timeouts: if `cfg.ReadTimeout == 0` set `defaultReadTimeout`; same for Write and Idle
  - Allocates a new `http.ServeMux`
  - Returns `&App{cfg: cfg, mux: ...}`
  - Does NOT call `MustConfig` — validation is opt-in via `MustConfig`
- [x] Godoc comment on `App` and `New`

#### 1.3 — App.Use and App.Handle

- [x] Define `(a *App) Use(mws ...func(http.Handler) http.Handler)`:
  - Appends `mws` to `a.middleware` in order
  - Safe to call multiple times; all calls are additive
- [x] Define `(a *App) Handle(pattern string, handler http.Handler)`:
  - Delegates directly to `a.mux.Handle(pattern, handler)`
  - Allows raw handler registration alongside `Content` modules
- [x] Godoc comments on both methods

#### 1.4 — App.Content

- [x] Define `(a *App) Content(v any, opts ...Option)` with `Registrator` interface:
  - If `v` implements `Registrator` (`*Module[T]` does), call `v.(Registrator).Register(a.mux)` directly — type-safe path
  - Otherwise call `NewModule[any](v, opts...)` and register — untyped fallback
  - `Registrator` interface exported so users can implement it for custom modules
- [x] Godoc comment on `Content` and `Registrator` explaining the typed vs untyped paths

#### 1.5 — App.Handler and App.Run

- [x] Define `(a *App) Handler() http.Handler`:
  - Returns `Chain(a.mux, a.middleware...)` (uses existing `Chain` from middleware.go)
  - When `cfg.HTTPS` is true, prepends `httpsRedirect()` before all user-supplied middleware
- [x] Define unexported `httpsRedirect() func(http.Handler) http.Handler` — the redirect middleware, extracted for testability
- [x] Define `(a *App) Run(addr string) error`:
  - Constructs `http.Server` with `Handler: a.Handler()`, `Addr: addr`, and the three timeouts from `a.cfg`
  - Launches `srv.ListenAndServe()` in a goroutine
  - Blocks on `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`
  - On signal: calls `srv.Shutdown` with a 5-second deadline context
  - Returns the shutdown error (nil on clean shutdown; `http.ErrServerClosed` is suppressed)
  - Imports: `os/signal`, `syscall`, `os` — all stdlib, no new third-party deps
- [x] Godoc comments on `Handler`, `Run`, and `httpsRedirect`

#### 1.6 — Tests (forge_test.go)

- [x] `TestMustConfig_valid` — passes a valid Config, confirms it is returned unchanged
- [x] `TestMustConfig_emptyBaseURL` — confirms panic with relevant message
- [x] `TestMustConfig_invalidBaseURL` — passes a non-URL string, confirms panic
- [x] `TestMustConfig_relativeURL` — passes a relative path, confirms panic (added)
- [x] `TestMustConfig_shortSecret` — len < 16, confirms panic
- [x] `TestNew_defaults` — zero duration fields are set to defaults after New
- [x] `TestNew_preservesTimeouts` — non-zero fields are not overwritten
- [x] `TestApp_Use_order` — two middlewares applied in order (first-in, first-applied)
- [x] `TestApp_Handle` — registers a handler, makes a request via `httptest`, confirms response
- [x] `TestApp_Content_list` — registers a content type, GET list endpoint returns 200 JSON
- [x] `TestApp_Content_create` — POST to content endpoint stores item, GET confirms it
- [x] `TestApp_Handler_middlewareChain` — Use adds a header, Handler() applies it
- [x] `TestApp_Handler_httpsRedirect` — with cfg.HTTPS=true, HTTP request receives 301 to https://
- [x] `TestApp_Handler_httpsRedirect_xForwardedProto` — reverse-proxy HTTPS passthrough via header
- [x] `TestApp_Handler_httpsRedirect_disabled` — cfg.HTTPS=false, plain HTTP passes through
- [x] `TestApp_Run_gracefulShutdown` — starts server, polls until ready, sends os.Interrupt; skips cleanly on Windows (signal not supported)

#### 1.7 — Benchmark (forge_test.go)

- [x] `BenchmarkApp_Handler` — allocates App, wires one Content module, benchmarks a GET list request via `httptest.NewRecorder()`
- [x] Confirms allocations-per-request are bounded (b.ReportAllocs()) — 56 allocs/op measured

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestMustConfig ./...` — all MustConfig tests green
- [x] `go test -v -run TestNew ./...` — all New tests green
- [x] `go test -v -run TestApp ./...` — all App tests green (graceful-shutdown test skips cleanly on Windows; signal path exercised on Unix)
- [x] `go test -bench BenchmarkApp -benchmem ./...` — 20022 ns/op, 5660 B/op, 56 allocs/op on i5-9300HF
- [x] `BACKLOG.md` — Step 1 row and summary checkbox updated
- [x] `README.md` — status callout updated: forge.New and app.Content now available; v0.2.0 note added
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

## Layer 2 — Native pgx Adapter (depends on Step 1)

### Step P1 — forge-pgx/pgx.go

**Deferred from:** Milestone 1, Step P1
**Depends on:** forge.go (Step 1 of this milestone — Config and DB interface must be stable)
**Decisions:** Decision 22 (DB interface and forge-pgx adapter)
**Module:** `github.com/forge-cms/forge-pgx` (scaffolded as `./forge-pgx/` subdirectory module; extracted to its own repository before v1.0)
**Files:** `forge-pgx/pgx.go`, `forge-pgx/pgx_test.go`

#### P1.1 — Module scaffold

- [x] Create `forge-pgx/` directory in the workspace root
- [x] Create `forge-pgx/go.mod`: `module github.com/forge-cms/forge-pgx`, `go 1.22` / toolchain resolved by tidy, requires `github.com/forge-cms/forge` (replace directive pointing to `../`) and `github.com/jackc/pgx/v5 v5.8.0`
- [x] Create `forge-pgx/go.sum` by running `go mod tidy` in the subdirectory
- [x] Confirm `forge-pgx/go.mod` does NOT appear in the root `go.mod` — it is a sibling module, not a sub-package

#### P1.2 — poolAdapter type

- [x] Define `package forgepgx` in `forge-pgx/pgx.go`
- [x] Define unexported `poolAdapter struct{ db *sql.DB }` — stores `*sql.DB` opened once via `stdlib.OpenDBFromPool(p)` at `Wrap` time
- [x] Define `Wrap(p *pgxpool.Pool) forge.DB` — the public constructor; returns `&poolAdapter{db: stdlib.OpenDBFromPool(p)}`; godoc explains this is the entry point
- [x] Implement `(a *poolAdapter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)`: delegates to `a.db.QueryContext`
- [x] Implement `(a *poolAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)`: delegates to `a.db.ExecContext`
- [x] Implement `(a *poolAdapter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row`: delegates to `a.db.QueryRowContext`
- [x] Godoc comment on `Wrap` and `poolAdapter` methods
- [x] Confirm `poolAdapter` satisfies `forge.DB` via compile-time assertion: `var _ forge.DB = (*poolAdapter)(nil)`

#### P1.3 — Tests

- [x] `TestWrap_compilesAsForgeDB` in `pgx_test.go` — no build tag; documents the compile-time guarantee; runs without a database
- [x] `TestWrap_integration` in `pgx_integration_test.go` tagged `//go:build integration`:
  - Reads `DATABASE_URL` from environment; skips with `t.Skip` if absent
  - Creates `pgxpool.Pool`, calls `forgepgx.Wrap(pool)`, exercises `ExecContext`/`QueryContext`/`QueryRowContext` with a `CREATE TEMP TABLE`, `INSERT`, `SELECT`
  - Confirms `forge.Query[T]` works end-to-end with the wrapped pool
- [x] `go test -v ./forge-pgx/...` (without `//go:build integration`) — compile-check test green, no database required

#### Verification

- [x] `go build ./forge-pgx/...` — no errors
- [x] `go vet ./forge-pgx/...` — clean
- [x] `gofmt -l ./forge-pgx/` — returns nothing
- [x] `go test -v ./forge-pgx/...` — compile-check test green (no database required)
- [x] `BACKLOG.md` — Step P1 row and summary checkbox updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required

---

## Completion criteria for Milestone 2

- [x] `go build ./...` — no errors, no warnings
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test ./...` — all non-integration tests green (forge and forge-pgx)
- [x] All exported symbols have godoc comments
- [x] `forge.New` / `app.Content` / `app.Use` / `app.Handle` / `app.Run` / `app.Handler` all work end-to-end in tests using `httptest`
- [x] `MustConfig` panics cleanly with descriptive messages on bad input
- [x] Graceful shutdown confirmed in test: server started, interrupt sent, clean exit within 2 s
- [x] `forge-pgx`: `Wrap(pool)` satisfies `forge.DB` at compile time; integration test tagged separately
- [x] `ARCHITECTURE.md` updated: forge.go symbols added, App bootstrap lifecycle documented
- [x] `README.md` status callout updated: forge.New and app.Content available in v0.2.0
- [x] Post-milestone DRY/performance/security review completed and findings resolved
- [x] Any deferred steps documented in target milestone with reason
- [x] Retrospective completed before milestone gate commit
