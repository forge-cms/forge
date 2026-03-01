# Forge — Milestone 1 Backlog (v0.1.0)

Implementation plan for the Core milestone. Update status as each step is completed.
Order is dictated by internal dependency rules from ARCHITECTURE.md.

When a step is done: change `🔲` to `✅` and record the date.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | errors.go | ✅ Done | 2026-03-01 |
| 2 | roles.go | ✅ Done | 2026-03-01 |
| 3 | mcp.go | 🔲 Not started | — |
| 4 | node.go | 🔲 Not started | — |
| 5 | context.go | 🔲 Not started | — |
| 6 | signals.go | 🔲 Not started | — |
| 7 | storage.go | 🔲 Not started | — |
| 8 | auth.go | 🔲 Not started | — |
| 9 | middleware.go | 🔲 Not started | — |
| 10 | module.go | 🔲 Not started | — |
| 11 | forge.go | 🔲 Not started | — |
| P1 | forge-pgx (separate module) | 🔲 Not started | — |

---

## Layer 0 — Foundation (no dependencies, can be parallelised)

### Step 1 — errors.go

**Depends on:** nothing
**Decisions:** Decision 16
**Files:** `errors.go`, `errors_test.go`

#### 1.1 — `forge.Error` interface

- [x] Declare `forge.Error` interface embedding `error` with methods `Code() string`, `HTTPStatus() int`, `Public() string`
- [x] godoc comment: all Forge errors implement this; callers use `errors.As` to inspect — never type-assert directly

#### 1.2 — `sentinelError` (unexported concrete type)

- [x] Unexported `sentinelError` struct with fields `code string`, `status int`, `public string`
- [x] `Error() string` returns `Public()`
- [x] Unexported constructor `newSentinel(status int, code, public string) forge.Error`

#### 1.3 — Sentinel vars

- [x] `ErrNotFound` → 404, `"not_found"`, `"Not found"`
- [x] `ErrGone` → 410, `"gone"`, `"This content has been removed"`
- [x] `ErrForbidden` → 403, `"forbidden"`, `"Forbidden"`
- [x] `ErrUnauth` → 401, `"unauthorized"`, `"Unauthorized"`
- [x] `ErrConflict` → 409, `"conflict"`, `"Conflict"`

#### 1.4 — `ValidationError`

- [x] Unexported `fieldError` value type with `Field string` and `Message string`
- [x] Exported `ValidationError` struct implementing `forge.Error`: status 422, code `"validation_failed"`, public `"Validation failed"`; carries `[]fieldError` internally
- [x] `Error()` returns `"validation failed: {field}: {message}"` for single-field; joined for multi-field
- [x] `forge.Err(field, message string) *ValidationError` — creates a single-field ValidationError

#### 1.5 — `forge.Require`

- [x] `forge.Require(errs ...error) error` — skips nils; collects `*ValidationError` values via `errors.As`; returns `nil` if all inputs are nil; returns combined `*ValidationError` with merged `[]fieldError` if any found; returns first unexpected non-nil non-ValidationError as-is

#### 1.6 — `forge.WriteError`

- [x] `forge.WriteError(w http.ResponseWriter, r *http.Request, err error)` with `errors.As` dispatch chain:
  - `*ValidationError` → 422, JSON with populated `fields` array
  - `forge.Error` with `HTTPStatus() < 500` → use its status / code / public directly
  - `forge.Error` with `HTTPStatus() >= 500` → `slog.Error` with `request_id`; respond with generic 500
  - anything else → `slog.Error` with `request_id`; respond with generic 500
- [x] Request ID: read from `w.Header().Get("X-Request-ID")` first, fall back to `r.Header.Get("X-Request-ID")`; if neither present, leave blank (set upstream by `ContextFrom` in normal flow)
- [x] Set `X-Request-ID` on `w` if not already present
- [x] JSON response shape always `Content-Type: application/json`:
  ```json
  {"error": {"code": "...", "message": "...", "request_id": "...", "fields": [{"field": "...", "message": "..."}]}}
  ```
- [x] `fields` key omitted (or empty array) for non-validation errors
- [x] HTML fallback: serve minimal built-in string when `Accept: text/html`; add TODO comment referencing templates.go (Milestone 3)

#### 1.7 — Tests (`errors_test.go`)

- [x] All 5 sentinels: correct `HTTPStatus()`, `Code()`, `Public()`, `Error()`
- [x] `forge.Err("title", "required")`: correct field, message, 422 status
- [x] `forge.Require(nil, forge.Err("x","y"), nil, forge.Err("a","b"))`: collects both, ignores nils
- [x] `forge.Require(nil, nil)`: returns nil
- [x] `forge.WriteError` with sentinel → correct HTTP status in response recorder
- [x] `forge.WriteError` with `*ValidationError` → 422, `fields` array in JSON body
- [x] `forge.WriteError` with `fmt.Errorf("internal")` → 500, no internal detail in body
- [x] `forge.WriteError` echoes `X-Request-ID` when present on request
- [x] All test cases table-driven with `t.Run`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestError ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 2 — roles.go

**Depends on:** nothing
**Decisions:** Decision 15
**Files:** `roles.go`, `roles_test.go`

#### 2.1 — `forge.Role` type and built-in constants

- [x] Declare `forge.Role` as a named `string` type with godoc
- [x] Declare unexported `roleLevels` as `map[Role]int` — the single source of
      truth for level lookups; populated at init time for built-in roles
- [x] Declare the four built-in constants with **spaced levels** (10/20/30/40)
      so custom roles can always be inserted between built-ins:
  - `Guest  Role = "guest"`  → level 10
  - `Author Role = "author"` → level 20
  - `Editor Role = "editor"` → level 30
  - `Admin  Role = "admin"`  → level 40
- [x] Unexported `levelOf(r Role) int` — single map lookup; returns 0 for unknown roles
- [x] godoc on every exported symbol

#### 2.2 — Custom role registration (fluent builder)

- [x] Declare unexported `roleBuilder` struct: `name string`, `level int`
- [x] `forge.NewRole(name string) roleBuilder` — exported constructor; level starts at 0
- [x] `(rb roleBuilder) Above(r Role) roleBuilder` — returns new builder with
      `level = levelOf(r) + 1`; does not mutate existing roles
- [x] `(rb roleBuilder) Below(r Role) roleBuilder` — returns new builder with
      `level = levelOf(r) - 1`; minimum level enforced at 1
- [x] `(rb roleBuilder) Register() (Role, error)` — writes name+level into
      `roleLevels`; idempotent if same name+level already registered; returns
      a `*ValidationError` via `forge.Err` if same name registered with
      different level; returns `forge.Role(rb.name)`
- [x] godoc on `NewRole`, `Above`, `Below`, `Register`

#### 2.3 — Role comparison (free functions)

- [x] `forge.HasRole(userRoles []Role, required Role) bool` — returns true if
      any role in `userRoles` has `levelOf(role) >= levelOf(required)`;
      unknown roles (level 0) never satisfy any requirement; no allocations
- [x] `forge.IsRole(userRoles []Role, required Role) bool` — returns true if
      any role in `userRoles` exactly matches `required`; no allocations
- [x] godoc on both functions; note that `HasRole` is hierarchical and
      `IsRole` is an exact match

#### 2.4 — `forge.Option` stub and module permission options

- [x] Declare `forge.Option` as an exported interface with unexported marker
      method `isOption()` — this is the canonical definition; module.go (Step 10)
      will use it directly without redeclaring
- [x] Declare unexported `roleOption` struct: `signal string`, `role Role`;
      implements `Option` via `isOption()`
- [x] `forge.Read(r Role) Option`   → `roleOption{signal: "read",   role: r}`
- [x] `forge.Write(r Role) Option`  → `roleOption{signal: "write",  role: r}`
- [x] `forge.Delete(r Role) Option` → `roleOption{signal: "delete", role: r}`
- [x] godoc on `Option`, `Read`, `Write`, `Delete`

#### 2.5 — Tests (`roles_test.go`)

- [x] `TestRoleLevel` — all four built-in roles return correct level from `levelOf`
- [x] `TestHasRole` — table-driven:
  - Admin satisfies Admin, Editor, Author, Guest
  - Editor satisfies Editor, Author, Guest; not Admin
  - Author satisfies Author, Guest; not Editor, not Admin
  - Guest satisfies Guest only
  - Unknown role satisfies nothing
- [x] `TestIsRole` — table-driven: exact match only; Admin does not satisfy Editor
- [x] `TestNewRole` — `NewRole("publisher").Above(Author).Below(Editor)` gives
      level between Author (20) and Editor (30); verified via `levelOf` after `Register()`
- [x] `TestRegisterIdempotent` — same name + same level → no error
- [x] `TestRegisterConflict` — same name + different level → returns `forge.Error`
- [x] `TestModuleOptionStubs` — `Read`, `Write`, `Delete` return non-nil `Option`;
      type-assert to `roleOption` and check `signal` and `role` fields

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestRole|TestHasRole|TestIsRole|TestNewRole|TestRegister|TestModuleOption" ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment R1 drafted:
      built-in role levels use spacing of 10 (10/20/30/40) to allow custom roles
      between adjacent built-ins. Decision 15 updated to reflect new values.

---

### Step 3 — mcp.go

**Depends on:** nothing
**Decisions:** Decision 19

- [ ] `forge.MCPRead` and `forge.MCPWrite` exported constants
- [ ] `forge.MCP(options ...any) Option` — returns no-op `Option`; godoc comment states reserved for v2
- [ ] Test: `forge.MCP(forge.MCPRead)` compiles and returns a valid (no-op) Option

---

## Layer 1 — Depends on Layer 0

### Step 4 — node.go

**Depends on:** errors
**Decisions:** Decision 1, 10, 14; Amendment S1

- [ ] `forge.Status` type + constants: `Draft`, `Published`, `Scheduled`, `Archived`
- [ ] `forge.Node` struct:
  - `ID string`
  - `Slug string`
  - `Status Status`
  - `PublishedAt time.Time`
  - `ScheduledAt *time.Time`
  - `CreatedAt time.Time`
  - `UpdatedAt time.Time`
- [ ] `forge.NewID() string` — UUID v7 via `crypto/rand`
- [ ] `forge.GenerateSlug(input string) string` — lowercase, whitelist `[a-z0-9-]`, max 200 chars, trim leading/trailing hyphens
- [ ] `forge.UniqueSlug(base string, exists func(string) bool) string` — appends `-2`, `-3` etc. on collision
- [ ] Struct tag validation via reflection (cached in `sync.Map`):
  - `forge:"required"` — field must not be zero value
  - `forge:"min=N"` — string min length / number min value
  - `forge:"max=N"` — string max length / number max value
  - `forge:"email"` — valid email address
  - `forge:"url"` — valid URL
  - `forge:"slug"` — valid slug `[a-z0-9-]`
  - `forge:"oneof=a,b,c"` — value must be one of the listed options
- [ ] `forge.ValidateStruct(v any) error` — runs tag validation; returns `*ValidationError` or nil
- [ ] `forge.Validatable` interface: `Validate() error`
- [ ] `forge.RunValidation(v any) error` — runs `ValidateStruct` first; if OK and `v` implements `Validatable`, calls `v.Validate()`
- [ ] Reflection cache: `sync.Map` keyed by `reflect.Type`; populated on first use per type
- [ ] Table-driven tests: UUID format, slug generation, all tag constraints, collision suffix, RunValidation chain
- [ ] Benchmark: `ValidateStruct` per type (first call vs. cached)

---

### Step 5 — context.go

**Depends on:** roles
**Decisions:** Decision 6, 21; Amendment S1 (RequestID)

- [ ] `forge.Context` interface:
  ```go
  context.Context
  User() forge.User
  Locale() string
  SiteName() string
  RequestID() string
  Request() *http.Request
  Response() http.ResponseWriter
  ```
- [ ] Unexported `contextImpl` struct implementing `forge.Context`
- [ ] `forge.ContextFrom(r *http.Request) forge.Context` — builds context from request; generates UUID v7 RequestID; sets `X-Request-ID` response header
- [ ] `forge.NewTestContext(user forge.User) forge.Context` — for unit tests; `Request()` returns a synthetic `*http.Request`; `Response()` returns an `httptest.ResponseRecorder`-compatible writer
- [ ] `Locale()` always returns `"en"` in v1
- [ ] `forge.Context` is always non-nil — Forge guarantees this before user code is called
- [ ] Tests: `NewTestContext` with and without user; `ContextFrom` sets RequestID; Locale returns "en"

---

## Layer 2 — Depends on Layer 0+1 (can be parallelised within layer)

### Step 6 — signals.go

**Depends on:** context, errors
**Decisions:** Amendment P1 (debounce)

- [ ] `forge.Signal` type (string constant)
- [ ] Exported signal constants:
  - `BeforeCreate`, `AfterCreate`
  - `BeforeUpdate`, `AfterUpdate`
  - `BeforeDelete`, `AfterDelete`
  - `AfterPublish`, `AfterUnpublish`, `AfterArchive`
  - `SitemapRegenerate`
- [ ] `forge.SignalHandler` type: `func(ctx forge.Context, payload any) error`
- [ ] `forge.On(signal Signal, handler SignalHandler) Option` — module option
- [ ] Internal `dispatchBefore(ctx, signal, payload)` — synchronous; error → aborts operation; panic → recovered, logged, returns 500 error
- [ ] Internal `dispatchAfter(ctx, signal, payload)` — spawns goroutine; errors logged; panics recovered and logged
- [ ] `SitemapRegenerate` debounce: 2-second timer; reset on each new AfterPublish/AfterUnpublish/AfterArchive; fires only once after a burst
- [ ] Tests: BeforeX can abort operation; AfterX is non-blocking; debounce coalesces 10 signals into 1 rebuild

---

### Step 7 — storage.go

**Depends on:** node, errors
**Decisions:** Decision 2, 22
**Unlocks:** forge-pgx (Step P1 can start after this)

- [ ] `forge.DB` interface:
  ```go
  QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
  ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
  QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
  ```
- [ ] `forge.Query[T any](ctx context.Context, db forge.DB, query string, args ...any) ([]T, error)` — struct scanning with reflection cache
- [ ] `forge.QueryOne[T any](ctx context.Context, db forge.DB, query string, args ...any) (T, error)` — returns `ErrNotFound` if no rows
- [ ] Field mapping: `db` tag first, then field name lowercased
- [ ] Reflection cache: `sync.Map` keyed by `reflect.Type`; scan struct fields once per type
- [ ] `forge.Repository[T forge.Node]` interface: `FindByID`, `FindBySlug`, `FindAll`, `Save`, `Delete`
- [ ] `forge.MemoryRepo[T forge.Node]` struct + `forge.NewMemoryRepo[T]() *MemoryRepo[T]`
  - Thread-safe via `sync.RWMutex`
  - `FindByID`, `FindBySlug`, `FindAll` (respects `ListOptions`)
  - `Save` — upsert
  - `Delete` — returns `ErrNotFound` if not found
- [ ] `forge.ListOptions` struct: `Page int`, `PerPage int`, `OrderBy string`, `Desc bool`; `Offset() int` method
- [ ] Tests: `Query[T]` scanning, `QueryOne[T]` not-found, `MemoryRepo` full CRUD + `ListOptions`
- [ ] Benchmark: `Query[T]` scanning (first call vs. cached reflection)

---

### Step 8 — auth.go

**Depends on:** errors, roles, context
**Decisions:** Decision 15; Amendment S6 (CSRF), S7 (BasicAuth warning)

- [ ] `forge.User` struct: `ID string`, `Name string`, `Roles []Role`
- [ ] `user.HasRole(role forge.Role) bool` — hierarchical (Admin includes Editor includes Author)
- [ ] `user.Is(role forge.Role) bool` — exact match only
- [ ] `forge.AuthFunc` type: `func(r *http.Request) (forge.User, bool)`
- [ ] `forge.BearerHMAC(secret string) forge.AuthFunc` — HMAC-SHA256; Bearer prefix in Authorization header
- [ ] `forge.SignToken(user forge.User, secret string) (string, error)` — generates HMAC-signed token
- [ ] `forge.CookieSession(name, secret string, opts ...Option) forge.AuthFunc`
  - Cookie-based auth
  - Automatic CSRF: token in `forge_csrf` Necessary cookie; client echoes via `X-CSRF-Token` header or `_csrf` form field; rotates on new auth
  - `forge.WithoutCSRF` opt-out option
- [ ] `forge.BasicAuth(username, password string) forge.AuthFunc`
  - Standard HTTP Basic Auth
  - Logs structured `WARN` at startup if `Env != forge.Development` (once in `app.Run`, not per request)
- [ ] `forge.AnyAuth(fns ...forge.AuthFunc) forge.AuthFunc` — first match wins
- [ ] Tests: BearerHMAC valid/invalid token, CookieSession CSRF rotation, BasicAuth warning trigger, AnyAuth fallback chain

---

### Step 9 — middleware.go

**Depends on:** errors, context
**Decisions:** Amendment P2 (LRU cache)

- [ ] `forge.RequestLogger() func(http.Handler) http.Handler` — structured slog; fields: `method`, `path`, `status`, `duration`, `request_id`
- [ ] `forge.Recoverer() func(http.Handler) http.Handler` — panic → 500 via `forge.WriteError`; logs stack trace
- [ ] `forge.CORS(origin string) func(http.Handler) http.Handler` — sets `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`
- [ ] `forge.MaxBodySize(n int64) func(http.Handler) http.Handler` — wraps `http.MaxBytesReader`
- [ ] `forge.RateLimit(n int, d time.Duration) func(http.Handler) http.Handler` — per-IP token bucket; returns 429 on exceeded
- [ ] `forge.SecurityHeaders() func(http.Handler) http.Handler` — HSTS, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, strict CSP default
- [ ] `forge.InMemoryCache(ttl time.Duration, opts ...Option) func(http.Handler) http.Handler`
  - LRU: doubly-linked list + map (~40 lines, stdlib only)
  - Default max 1000 entries; `forge.CacheMaxEntries(n int)` option
  - Cache key: method + full URL including query params + Accept header
  - `X-Cache: HIT` / `X-Cache: MISS` always set
  - Background sweep every 60s; lazy expiry on read
- [ ] `forge.Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler` — composition helper
- [ ] Tests: Recoverer catches panic, RateLimit returns 429, LRU MISS→HIT→eviction, SecurityHeaders present
- [ ] Benchmark: LRU cache HIT throughput

---

## Layer 3 — Depends on all of Layer 0+1+2

### Step 10 — module.go

**Depends on:** node, context, auth, signals, storage, errors
**Decisions:** Decision 4 (content negotiation), 14 (lifecycle), 19 (MCP no-op); Amendment P2 (cache)

- [ ] `forge.Option` type (consistent with Steps 2 and 3)
- [ ] Internal `forge.Module[T forge.Node]` struct
- [ ] `app.Content(prototype T, opts ...Option)` — registers module; derives prefix from type name as default
- [ ] `forge.At(prefix string) Option` — overrides URL prefix
- [ ] `forge.Cache(ttl time.Duration) Option` — enables per-module LRU; max 1000 entries; cache key: `"{method}:{fullURL}:{Accept}"`; `X-Cache: HIT/MISS`; invalidated on AfterCreate/Update/Delete
- [ ] `forge.Middleware(mws ...func(http.Handler) http.Handler) Option` — per-module middleware
- [ ] Auto-routing via Go 1.22 `net/http` ServeMux:
  - `GET /{prefix}` → list
  - `GET /{prefix}/{slug}` → show
  - `POST /{prefix}` → create
  - `PUT /{prefix}/{slug}` → update
  - `DELETE /{prefix}/{slug}` → delete
- [ ] Lifecycle enforcement on all public GET:
  - Draft / Scheduled / Archived → 404 for Guest (never leaks existence)
  - Editor+ → sees all statuses
  - Author → sees own Draft/Scheduled/Archived
- [ ] Content negotiation (pre-compiled Accept matching per module, not per request):
  - `application/json` → always available
  - `text/html` → requires `forge.Templates(...)` registered
  - `text/markdown` → requires T implements `forge.Markdownable`; else 406
  - `text/plain` → always available, derived from content
  - `*/*` or missing Accept → JSON
  - `Vary: Accept` set automatically
- [ ] Struct tag validation + `Validate()` run automatically before Save (via `forge.RunValidation`)
- [ ] `forge.MCP(options ...any) Option` delegates to mcp.go no-op
- [ ] Tests: lifecycle enforcement (Guest 404, Editor 200, Author own), content negotiation (all types), cache HIT/MISS/invalidation, validation aborts create/update
- [ ] Benchmark: full request lifecycle (in-memory repo, JSON response)

---

## Layer 4 — Depends on everything

### Step 11 — forge.go

**Depends on:** all other files
**Decisions:** Decision 20 (configuration), Decision 22 (DB in Config)

- [ ] `forge.Env` type + constants: `Development`, `Production`, `Test`
- [ ] `forge.Config` struct:
  - `BaseURL string` — required in production; fallback: `FORGE_BASE_URL`, then `http://localhost:{PORT}`
  - `Secret string` — fallback: `FORGE_SECRET`
  - `Env Env` — fallback: `FORGE_ENV`; default: `Development`
  - `Logger *slog.Logger` — default: `slog.Default()`
  - `LogLevel slog.Level` — fallback: `FORGE_LOG_LEVEL`; default: `slog.LevelInfo`
  - `DB forge.DB` — optional (not all apps use a database)
- [ ] `forge.MustConfig(cfg Config) Config` — startup validation:
  - FATAL `"forge: Config.BaseURL is required in production"` if `Env == Production && BaseURL == ""`
  - WARN `"forge: FORGE_SECRET is not set"` if `Secret == ""`
  - WARN `"forge: FORGE_SECRET is under 32 bytes"` if `len(Secret) < 32`
  - WARN on BasicAuth in non-development (logged once at startup)
  - Fills missing fields from env vars
- [ ] `forge.New(cfg Config) *App` — calls `MustConfig` internally; creates ServeMux
- [ ] `App.Use(middleware func(http.Handler) http.Handler)` — global middleware (applied in order)
- [ ] `App.Content(prototype any, opts ...Option)` — delegates to module.go
- [ ] `App.Roles(roles ...Role)` — registers custom roles
- [ ] `App.Handle(pattern string, handler http.Handler)`
- [ ] `App.HandleFunc(pattern string, fn http.HandlerFunc)`
- [ ] `App.Handler() http.Handler` — returns assembled `http.Handler` without starting server (for tests)
- [ ] `App.Run(addr string)` — `addr == ""` → use `PORT` env var → fallback `:8080`; graceful shutdown on SIGINT/SIGTERM with 30s timeout
- [ ] Global middleware chain order: RequestLogger → Recoverer → SecurityHeaders → CORS → MaxBodySize → RateLimit
- [ ] Tests: `MustConfig` validation (all FATAL/WARN scenarios), `App.Handler()` + `httptest`, graceful shutdown signal

---

## Parallel track — forge-pgx

### Step P1 — github.com/forge-cms/forge-pgx

**Can start:** after Step 7 (forge.DB is defined)
**Separate Go module** — new repository under forge-cms org

- [ ] New repo created: `github.com/forge-cms/forge-pgx`
- [ ] `go.mod` with `module github.com/forge-cms/forge-pgx` and `go 1.22`
- [ ] Dependencies: `github.com/forge-cms/forge` + `github.com/jackc/pgx/v5`
- [ ] `forgepgx.Wrap(pool *pgxpool.Pool) forge.DB` — ~25 lines; thin translation layer, no business logic
- [ ] Tests against a real PostgreSQL instance
- [ ] README with throughput table:
  - `database/sql` + `lib/pq` → 1× (baseline)
  - `pgx/v5/stdlib` shim → ~1.8×
  - `forgepgx` native pool → ~2.5×

---

## Completion criteria for Milestone 1

Milestone is complete when all of the following are satisfied:

- [ ] `go build ./...` — no errors, no warnings
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test ./...` — all tests green
- [ ] All exported symbols have godoc comments
- [ ] Benchmarks implemented for: UUID generation, struct tag validation (cached vs. uncached), `Query[T]` scanning, LRU cache HIT/MISS, full request lifecycle
- [ ] `forge.NewTestContext` + `forge.NewMemoryRepo[T]` used in tests — no database required for unit tests
- [ ] forge-pgx: `forgepgx.Wrap(pool)` tested against real PostgreSQL
