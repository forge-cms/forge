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
| 3 | mcp.go | ✅ Done | 2026-03-01 |
| 4 | node.go | ✅ Done | 2026-03-01 |
| 5 | context.go | ✅ Done | 2026-03-01 |
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

**Depends on:** roles (for `forge.Option`)
**Decisions:** Decision 19
**Files:** `mcp.go`, `mcp_test.go`

#### 3.1 — `MCPOperation` type and constants

- [x] Declare unexported `MCPOperation` type as `string` — keeps constants typesafe
- [x] `MCPRead  MCPOperation = "read"` — signals read-only MCP resource exposure
- [x] `MCPWrite MCPOperation = "write"` — signals read+write MCP resource exposure
- [x] godoc on both: "reserved for v2 MCP support; no-op in v1"

#### 3.2 — `mcpOption` concrete type

- [x] Declare unexported `mcpOption` struct (empty in v1 — no data needed)
- [x] Implement `forge.Option` via `isOption()` marker method
- [x] Do NOT re-declare `forge.Option` — use the canonical definition from `roles.go`

#### 3.3 — `forge.MCP` function

- [x] `func MCP(ops ...MCPOperation) Option` — accepts typed `MCPOperation` varargs
- [x] Returns `mcpOption{}` — no-op in v1
- [x] godoc: "MCP is reserved for v2 Model Context Protocol support. In v1, this
      option compiles but has no effect at runtime. See Decision 19."

#### 3.4 — Tests (`mcp_test.go`)

- [x] `TestMCP` — table-driven:
  - `MCP(MCPRead)` returns non-nil `Option`; type-asserts to `mcpOption`
  - `MCP(MCPWrite)` returns non-nil `Option`; type-asserts to `mcpOption`
  - `MCP(MCPRead, MCPWrite)` returns non-nil `Option`
  - `MCP()` (zero args) returns valid no-op `Option`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestMCP ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — no new decisions required;
      MCPOperation exported correctly, Option used from roles.go as intended.

---

## Layer 1 — Depends on Layer 0

### Step 4 — node.go

**Depends on:** errors
**Decisions:** Decision 1, 10, 14; Amendment S1
**Files:** `node.go`, `node_test.go`

#### 4.1 — `forge.Status` type and constants

- [x] Declare `forge.Status` as a named `string` type with godoc
- [x] Constants: `Draft`, `Published`, `Scheduled`, `Archived` (string values match Decision 14)
- [x] godoc on each constant explaining its meaning and visibility rules

#### 4.2 — `forge.Node` struct

- [x] `forge.Node` struct with fields in order:
  - `ID string` — UUID v7, primary key, immutable after creation
  - `Slug string` — URL-safe, unique within a module
  - `Status Status` — lifecycle state, enforced by public endpoints
  - `PublishedAt time.Time` — zero until first publish
  - `ScheduledAt *time.Time` — nil unless `Status == Scheduled`
  - `CreatedAt time.Time` — set on insert, never updated
  - `UpdatedAt time.Time` — updated on every Save
- [x] godoc on struct (embed in content types; carries lifecycle) and each field

#### 4.3 — `forge.NewID()` — UUID v7

- [x] Implement UUID v7 spec: 48-bit millisecond timestamp (big-endian in bytes 0–5),
      version nibble `7` in high 4 bits of byte 6, variant `10` in high 2 bits of byte 8,
      remaining bits filled with `crypto/rand`
- [x] Output format: `xxxxxxxx-xxxx-7xxx-xxxx-xxxxxxxxxxxx` (36 chars)
- [x] Panic on `crypto/rand` failure (unrecoverable platform error)
- [x] No third-party UUID library — stdlib `crypto/rand` + `crypto/rand.Reader` only
- [x] godoc: references Amendment S1; explains time-ordering benefit

#### 4.4 — `forge.GenerateSlug` and `forge.UniqueSlug`

- [x] `forge.GenerateSlug(input string) string`:
  - Lowercase the input (Unicode-aware via `strings.ToLower`)
  - Byte-level loop: spaces → `-`; `[a-z0-9-]` kept; all others dropped
  - Collapse consecutive hyphens to one
  - Trim leading/trailing hyphens
  - Truncate to max 200 bytes
  - Return `"untitled"` if result is empty
  - No `regexp` — byte loop for zero extra allocations
- [x] `forge.UniqueSlug(base string, exists func(string) bool) string`:
  - Returns `base` if `exists(base)` is false
  - Otherwise tries `base-2`, `base-3`, … until `exists` returns false
  - No upper bound required (callers ensure slugs eventually become available)
- [x] godoc on both functions

#### 4.5 — Reflection-cached tag validation engine

- [x] `fieldConstraint` unexported type: stores field index, field kind, and a slice
      of checker functions `func(reflect.Value) *fieldError`
- [x] `typeCache sync.Map` — keyed by `reflect.Type`; value `[]fieldConstraint`
- [x] `parseConstraints(t reflect.Type) []fieldConstraint` — unexported; iterates
      struct fields; parses `forge:"..."` tag; builds checker slice; panics on
      unrecognised tag key with a clear message (fail-fast at startup)
- [x] Supported constraints (comma-separated in one tag):
  - `required` — `reflect.Value.IsZero()` fails
  - `min=N` — `len(string)` or numeric value `< N` fails
  - `max=N` — `len(string)` or numeric value `> N` fails
  - `email` — must contain exactly one `@` with non-empty local and domain parts
  - `url` — parsed by `url.Parse`; must have scheme and host
  - `slug` — only `[a-z0-9-]`, non-empty
  - `oneof=a|b|c` — string value must be one of the listed options (pipe-separated; see Amendment R2)
- [x] All errors for a single struct are collected (no short-circuit); each produces
      a `*fieldError` with `Field` = struct field name

#### 4.6 — Public validation API

- [x] `forge.ValidateStruct(v any) error`:
  - Dereferences pointer; panics if `v` is not a struct
  - Loads constraints from `typeCache` (Store on first call via `LoadOrStore`)
  - Runs all constraints; collects `*fieldError` values; returns `*ValidationError` or nil
- [x] `forge.Validatable` interface: `Validate() error` — godoc: implement on content
      types for business-rule validation; called after tag validation
- [x] `forge.RunValidation(v any) error`:
  - Calls `ValidateStruct(v)`; if non-nil, returns immediately (no Validate() called)
  - If nil and `v` implements `Validatable`, calls `v.Validate()`
  - If `Validate()` returns a `*ValidationError`, merge its fields into a new
    `*ValidationError` and return
  - If `Validate()` returns any other non-nil error, return it as-is
- [x] godoc on all three

#### 4.7 — Tests (`node_test.go`)

- [x] `TestNewID` — generate 1000 IDs; assert:
  - length = 36; hyphens at positions 8, 13, 18, 23
  - byte 14 (version nibble) = `'7'`
  - byte 19 (variant) is `'8'`, `'9'`, `'a'`, or `'b'`
  - no two IDs are equal (uniqueness)
- [x] `TestGenerateSlug` — table-driven:
  - `"Hello World"` → `"hello-world"`
  - `"Go 1.22!"` → `"go-122"`
  - `"  --leading"` → `"leading"`
  - `"a/b/c"` → `"abc"`
  - string of 250 `'a'` → length ≤ 200
  - `""` → `"untitled"`
  - `"café"` → `"caf"` (non-ASCII dropped)
- [x] `TestUniqueSlug` — no collision, one collision, five collisions
- [x] `TestValidateStruct` — table-driven for every constraint; multi-constraint
      field; unknown forge tag panics; nested struct not traversed (flat only)
- [x] `TestRunValidation` — three cases:
  - Tags fail → `Validate()` NOT called (use a spy)
  - Tags pass, `Validate()` returns error → returned as-is
  - Tags pass, `Validate()` returns nil → nil
- [x] `BenchmarkValidateStructCached` — first-call vs subsequent; subsequent must
      not use reflection

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestNewID|TestGenerateSlug|TestUniqueSlug|TestValidate|TestRunValidation" ./...` — all green
- [x] `go test -bench BenchmarkValidateStructCached ./...` — ~100 ns/op cached
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment R2 drafted: `oneof=`
      tag uses `|` as value separator to avoid conflict with the tag constraint
      comma separator. Decision 10 example updated.

---

### Step 5 — context.go

**Depends on:** roles
**Decisions:** Decision 6, 21; Amendment S1 (RequestID)
**Files:** `context.go`, `context_test.go`

#### 5.1 — `forge.User` struct

- [x] `forge.User` value struct: `ID string`, `Name string`, `Roles []Role`
- [x] `GuestUser` package-level var — zero-value User with no roles; represents an
      unauthenticated request
- [x] godoc: "User represents an authenticated identity. The zero value is equivalent
      to GuestUser (unauthenticated)."
- [x] Defined in context.go because it only depends on Role (Layer 0); auth.go (Step 8)
      adds authentication machinery on top

#### 5.2 — `forge.Context` interface

- [x] `forge.Context` interface embedding `context.Context` with methods:
  - `User() User`
  - `Locale() string`
  - `SiteName() string`
  - `RequestID() string`
  - `Request() *http.Request`
  - `Response() http.ResponseWriter`
- [x] godoc on interface and every method

#### 5.3 — `contextImpl` unexported struct

- [x] Unexported `contextImpl` implementing all `forge.Context` methods
- [x] Embeds `context.Context` (stores the request's context for deadline/cancel propagation)
- [x] Fields: `user User`, `locale string`, `siteName string`, `requestID string`,
      `req *http.Request`, `w http.ResponseWriter`
- [x] All methods are simple field accessors — no allocation on the hot path

#### 5.4 — `forge.ContextFrom`

- [x] `func ContextFrom(w http.ResponseWriter, r *http.Request) Context`
- [x] RequestID: read from `w.Header().Get("X-Request-ID")` first; then
      `r.Header.Get("X-Request-ID")`; generate new `NewID()` if both empty
- [x] Write RequestID to `w.Header().Set("X-Request-ID", ...)` unconditionally
- [x] User: read from request context via unexported `contextKey` type; zero value
      (GuestUser) if not present
- [x] `Locale()` returns `"en"` (i18n deferred to v2 per Decision 11)
- [x] `SiteName()` returns `""` in v1 (wired in forge.go, Step 11)

#### 5.5 — `forge.NewTestContext`

- [x] `func NewTestContext(user User) Context` — no HTTP overhead
- [x] `Request()` returns `httptest.NewRequest("GET", "/", nil)`
- [x] `Response()` returns `*httptest.ResponseRecorder`
- [x] Locale `"en"`, SiteName `""`, RequestID generated via `NewID()`

#### 5.6 — Tests (`context_test.go`)

- [x] `TestContextFrom` — Locale returns `"en"`; Response is non-nil; Request is non-nil
- [x] `TestContextFromGeneratesRequestID` — no incoming ID → RequestID non-empty;
      response header `X-Request-ID` set
- [x] `TestContextFromPreservesRequestID` — incoming `X-Request-ID` on request →
      same ID echoed on response header and returned by `RequestID()`
- [x] `TestNewTestContext` — returns non-nil; correct User; Locale `"en"`;
      Request non-nil; Response non-nil
- [x] `TestNewTestContextGuest` — `NewTestContext(User{})` → User equals GuestUser;
      no roles

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestContext" ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment R3 drafted: forge.User
      defined in context.go (not auth.go) to resolve layer dependency ordering.

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
