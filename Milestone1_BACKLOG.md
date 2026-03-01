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
| 6 | signals.go | ✅ Done | 2026-03-01 |
| 7 | storage.go | ✅ Done | 2026-03-01 |
| 8 | auth.go | ✅ Done | 2026-03-01 |
| 9 | middleware.go | ✅ Done | 2026-03-01 |
| 10 | module.go | ✅ Done | 2026-03-01 |
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
**Decisions:** Amendment P1 (debounce), Amendment S2 (generic On[T])
**Files:** `signals.go`, `signals_test.go`
**Status:** ✅ Done

#### 6.1 — `forge.Signal` type and constants

- [x] `forge.Signal` named `string` type — consistent with `Role` and `Status`
- [x] Ten exported constants: `BeforeCreate`, `AfterCreate`, `BeforeUpdate`,
      `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `AfterPublish`,
      `AfterUnpublish`, `AfterArchive`, `SitemapRegenerate`
- [x] godoc on type and every constant

#### 6.2 — `forge.On[T any]` option

- [x] Unexported `signalHandler` type: `func(Context, any) error` — internal dispatch only
- [x] Unexported `signalOption` struct: `{ signal Signal; handler signalHandler }`;
      implements `Option` via `isOption()` (reuse marker from roles.go — do NOT redeclare)
- [x] `func On[T any](signal Signal, h func(Context, T) error) Option` — wraps
      typed handler in closure: `func(ctx Context, payload any) error { return h(ctx, payload.(T)) }`
- [x] godoc: "On registers a typed signal handler as a module Option. The handler
      receives the content value as its concrete type T — no type assertion required."

#### 6.3 — `dispatchBefore` and `dispatchAfter`

- [x] `dispatchBefore(ctx Context, handlers []signalHandler, payload any) error`
  - Iterates handlers in registration order
  - First non-nil error aborts iteration and is returned to caller
  - Panic: recovered via `recover()`; logged via `log/slog`; returns
    `forge.Error` with HTTP 500 and code `"signal_panic"`
- [x] `dispatchAfter(ctx Context, handlers []signalHandler, payload any)`
  - Launches a single goroutine
  - Iterates all handlers; non-nil errors logged via `log/slog`
  - Panic: recovered and logged; never propagated to caller

#### 6.4 — `debouncer`

- [x] Unexported `debouncer` struct: `{ mu sync.Mutex; timer *time.Timer;
      delay time.Duration; fn func() }`
- [x] `newDebouncer(delay time.Duration, fn func()) *debouncer`
- [x] `(d *debouncer) Trigger()` — stops existing timer and resets to `delay`;
      `fn` fires only after `delay` elapses with no further calls
- [x] Used by `module.go` (Step 10) to coalesce AfterPublish/AfterUnpublish/
      AfterArchive into a single SitemapRegenerate dispatch

#### 6.5 — Tests (`signals_test.go`)

- [x] `TestDispatchBeforeAbortsOnError` — first handler errors; second handler
      never called; error propagated
- [x] `TestDispatchBeforeRunsAllOnSuccess` — all handlers called when none error
- [x] `TestDispatchBeforePanicReturnsError` — panicking handler returns
      forge.Error (HTTP 500), does not crash process
- [x] `TestDispatchAfterIsNonBlocking` — returns before handler finishes
      (handler sleeps; use WaitGroup to verify completion)
- [x] `TestDispatchAfterPanicDoesNotPropagate` — panicking async handler does
      not crash or return error
- [x] `TestDebouncerCoalesces` — 10 rapid Trigger() calls produce exactly 1
      fn invocation after delay elapses
- [x] `TestOnReturnsOption` — On(BeforeCreate, handler) return value satisfies
      Option interface

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestDispatch|TestDebouncer|TestOn" ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment S2 agreed;
      AfterUnpublish gap in ARCHITECTURE.md fixed in same step

---

### Step 7 — storage.go

**Depends on:** node, errors
**Decisions:** Decision 2, Decision 22, Amendment S3 (Repository[T any])
**Unlocks:** forge-pgx (Step P1 can start after this)
**Files:** `storage.go`, `storage_test.go`
**Status:** ✅ Done

#### 7.1 — `forge.DB` interface

- [x] Declare `forge.DB` interface with three methods (verbatim from Decision 22):
  ```go
  QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
  ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
  QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
  ```
- [x] godoc: "DB is satisfied by *sql.DB, *sql.Tx, and forgepgx.Wrap(pool).
      Users never implement this directly."

#### 7.2 — Reflection scan cache

- [x] Unexported `dbField` struct: `{ index int; name string }` — maps a
      column name to the struct field index
- [x] Unexported `dbFieldCache sync.Map` — keyed by `reflect.Type`,
      stores `[]dbField`; populated once per type on first use
- [x] Unexported `dbFields(t reflect.Type) []dbField` — returns cached
      slice; on cache miss: iterate exported fields, map `db` tag name
      (fallback: `strings.ToLower(field.Name)`), store, return

#### 7.3 — `forge.Query[T any]`

- [x] `func Query[T any](ctx context.Context, db DB, query string, args ...any) ([]T, error)`
- [x] Calls `db.QueryContext`; returns wrapped error on failure
- [x] Calls `rows.Columns()` to get ordered column names
- [x] For each row: allocate `T` via `reflect.New`; build scan-target
      slice matched to columns via `dbFields` cache; call `rows.Scan`
- [x] Returns `[]T` — empty slice (not nil) on zero rows
- [x] `T` may be a pointer type (e.g. `*BlogPost`) — handle both `T`
      and `*T` using `reflect.TypeOf((*T)(nil)).Elem()`

#### 7.4 — `forge.QueryOne[T any]`

- [x] `func QueryOne[T any](ctx context.Context, db DB, query string, args ...any) (T, error)`
- [x] Delegates to `Query[T]`
- [x] Returns zero value of `T` + `ErrNotFound` if result slice is empty
- [x] Returns first element otherwise

#### 7.5 — `forge.ListOptions`

- [x] `type ListOptions struct { Page int; PerPage int; OrderBy string; Desc bool }`
- [x] `func (o ListOptions) Offset() int` — `max(0, (Page-1)*PerPage)`;
      Page ≤ 0 treated as page 1
- [x] godoc on struct and method

#### 7.6 — `forge.Repository[T any]` interface

- [x] Declare `Repository[T any]` interface with five methods using
      stdlib `context.Context` (not `forge.Context` — dependency rule):
  ```go
  FindByID(ctx context.Context, id string) (T, error)
  FindBySlug(ctx context.Context, slug string) (T, error)
  FindAll(ctx context.Context, opts ListOptions) ([]T, error)
  Save(ctx context.Context, node T) error
  Delete(ctx context.Context, id string) error
  ```
- [x] godoc: "Repository is the storage interface for a content type.
      Implement it to provide a custom backend. Use MemoryRepo for tests."

#### 7.7 — `forge.MemoryRepo[T any]` and `forge.NewMemoryRepo[T any]`

- [x] `type MemoryRepo[T any] struct` — unexported fields:
      `mu sync.RWMutex`, `items map[string]T`, `order []string`
- [x] `func NewMemoryRepo[T any]() *MemoryRepo[T]` — initialises map
- [x] `FindByID` — read-lock; return copy + `ErrNotFound` if absent
- [x] `FindBySlug` — read-lock; iterate items; match via reflection on
      `Slug` field (string); return `ErrNotFound` if no match
- [x] `FindAll` — read-lock; collect all items in insertion order;
      apply `ListOptions`: OrderBy (reflect string field, case-insensitive;
      fallback: insertion order), Desc flag, Page+PerPage slice
- [x] `Save` — write-lock; read `ID` field via reflection; upsert into
      map; append to `order` only on insert
- [x] `Delete` — write-lock; return `ErrNotFound` if absent; delete from
      map and remove from `order` slice
- [x] All reflection field access uses `dbFields` cache (same pattern
      as Query[T]) — no duplicate reflection logic

#### 7.8 — Tests (`storage_test.go`)

- [x] Fake `database/sql` driver: minimal implementation of
      `driver.Driver`, `driver.Conn`, `driver.Stmt`, `driver.Rows`
      inline in test file — zero imports beyond stdlib
- [x] `TestQueryScansRows` — Query[T] returns correctly scanned slice
- [x] `TestQueryReturnsEmptySliceNotNil` — zero rows → `[]T{}`, not nil
- [x] `TestQueryOneNotFound` — zero rows → `ErrNotFound`
- [x] `TestQueryOneReturnsFirst` — multiple rows → first row returned
- [x] `TestListOptionsOffset` — table-driven: page 1 → 0, page 2/PerPage
      10 → 10, page 0 → 0
- [x] `TestMemoryRepoSaveAndFindByID` — Save then FindByID round-trip
- [x] `TestMemoryRepoFindBySlug` — Save then FindBySlug round-trip
- [x] `TestMemoryRepoFindAll` — pagination via ListOptions
- [x] `TestMemoryRepoDelete` — Delete removes item; second Delete →
      `ErrNotFound`
- [x] `TestMemoryRepoDeleteNotFound` — Delete on empty repo → `ErrNotFound`
- [x] `BenchmarkQueryScanCached` — first call vs. subsequent; confirm
      cache prevents repeated reflection

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestQuery|TestMemoryRepo|TestListOptions" ./...` — all green
- [x] `go test -bench BenchmarkQuery ./...` — runs without error
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment S3 drafted:
      Repository[T any] and MemoryRepo[T any] use unconstrained type
      parameter (not [T forge.Node]) — Go generics do not support
      struct constraints; consistent with Query[T any] and On[T any]

---

### Step 8 — auth.go

**Depends on:** errors, roles, context
**Decisions:** Decision 15, Amendment S6 (CSRF), Amendment S7 (BasicAuth warning), Amendment S8 (AuthFunc interface)
**Files:** `auth.go`, `auth_test.go`
**Status:** ✅ Done

#### 8.1 — `forge.AuthFunc` interface and capability interfaces

- [x] Declare `forge.AuthFunc` interface with one unexported method:
  ```go
  type AuthFunc interface{ authenticate(*http.Request) (User, bool) }
  ```
- [x] Declare unexported capability interface `productionWarner`:
  ```go
  type productionWarner interface{ warnIfProduction(w io.Writer) }
  ```
  Used by Step 11 (`app.Run`) to detect `BasicAuth` in non-development environments.
- [x] Declare unexported capability interface `csrfAware`:
  ```go
  type csrfAware interface{ csrfEnabled() bool }
  ```
  Used by Step 9 (auth middleware) to decide whether to validate CSRF tokens.
- [x] godoc on `AuthFunc`: "AuthFunc authenticates an incoming HTTP request and
  returns the identified User and whether authentication succeeded. Use
  BearerHMAC, CookieSession, BasicAuth, or AnyAuth to obtain an AuthFunc.
  Implement this interface to provide a custom authentication scheme."

#### 8.2 — `User.HasRole` and `User.Is` methods

- [x] `func (u User) HasRole(role Role) bool` — declared in `auth.go`;
      delegates to `HasRole(u.Roles, role)` from `roles.go`
- [x] `func (u User) Is(role Role) bool` — delegates to `IsRole(u.Roles, role)`
      from `roles.go`
- [x] godoc on both methods including examples:
  `user.HasRole(forge.Editor)` — true for Editor and Admin
  `user.Is(forge.Author)` — true only for exactly Author

#### 8.3 — `SignToken` and token helpers

- [x] Token format: `base64url(json(User)) + "." + base64url(hmac-sha256(secret, payload))`
      — pure stdlib (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`)
- [x] Unexported `encodeToken(user User, secret string) (string, error)` —
      JSON-marshal user; compute HMAC-SHA256 over payload bytes; return `payload.sig`
- [x] Unexported `decodeToken(token, secret string) (User, error)` — split on `.`;
      verify HMAC constant-time; JSON-unmarshal payload; return User or `ErrUnauth`
- [x] `func SignToken(user User, secret string) (string, error)` — exported thin
      wrapper over `encodeToken`
- [x] godoc: "SignToken produces a signed token encoding the given User. Pass the
      token to the client; validate it later with BearerHMAC or CookieSession."

#### 8.4 — `BearerHMAC`

- [x] Unexported struct `bearerAuthFn{ secret string }` implementing `AuthFunc`:
  - `authenticate(r)`: extract `Authorization: Bearer <token>` header; call
    `decodeToken`; return `(GuestUser, false)` on any failure
- [x] `func BearerHMAC(secret string) AuthFunc` — returns `&bearerAuthFn{secret}`
- [x] godoc: "BearerHMAC returns an AuthFunc that validates HMAC-signed bearer tokens
      in the Authorization header. Generate tokens with SignToken."

#### 8.5 — `CSRFCookieName`, `WithoutCSRF`, and `CookieSession`

- [x] `const CSRFCookieName = "forge_csrf"` — exported; used by client-side AJAX
      code to read the CSRF cookie and populate `X-CSRF-Token`
- [x] Unexported struct `withoutCSRFOption{}` implementing `forge.Option`
      (marker `isOption()` from `roles.go`)
- [x] `var WithoutCSRF Option = withoutCSRFOption{}` — exported opt-out flag
- [x] Unexported struct `cookieAuthFn{ name, secret string; csrf bool }`
      implementing `AuthFunc` and `csrfAware`:
  - `authenticate(r)`: read named cookie; call `decodeToken`; return
    `(GuestUser, false)` on missing/invalid cookie
  - `csrfEnabled() bool`: return `c.csrf`
- [x] `func CookieSession(name, secret string, opts ...Option) AuthFunc` —
      inspect opts for `withoutCSRFOption`; default `csrf = true`
- [x] godoc on `CookieSession`, `WithoutCSRF`, `CSRFCookieName`

#### 8.6 — `BasicAuth`

- [x] Unexported struct `basicAuthFn{ username, password string }` implementing
      `AuthFunc` and `productionWarner`:
  - `authenticate(r)`: parse Basic credentials from `Authorization` header;
    use `subtle.ConstantTimeCompare` for both username and password;
    on success return `User{ID: username, Name: username, Roles: []Role{Guest}}`
  - `warnIfProduction(w io.Writer)`: write Amendment S7 warning text to `w`
- [x] `func BasicAuth(username, password string) AuthFunc`
- [x] godoc warning note: "BasicAuth should not be used in production.
      Consider BearerHMAC or CookieSession."

#### 8.7 — `AnyAuth`

- [x] Unexported struct `anyAuthFn{ fns []AuthFunc }` implementing `AuthFunc`,
      `productionWarner`, and `csrfAware`:
  - `authenticate(r)`: iterate `fns`; return first `(user, true)` result;
    return `(GuestUser, false)` if none match
  - `warnIfProduction(w io.Writer)`: forward call to any child implementing `productionWarner`
  - `csrfEnabled() bool`: return true if any child implements `csrfAware` and returns true
- [x] `func AnyAuth(fns ...AuthFunc) AuthFunc` — returns `&anyAuthFn{fns: fns}`
- [x] godoc: "AnyAuth returns an AuthFunc that tries each provided AuthFunc in order
      and returns the first successful result."

#### 8.8 — Tests (`auth_test.go`)

- [x] `TestUserHasRole` — hierarchical delegation to `HasRole` free function
- [x] `TestUserIs` — exact match delegation to `IsRole` free function
- [x] `TestSignTokenRoundTrip` — `SignToken` then `decodeToken` → same User
- [x] `TestSignTokenTampered` — altered payload → `ErrUnauth`
- [x] `TestBearerHMACValid` — correct token → `(user, true)`
- [x] `TestBearerHMACInvalid` — wrong secret → `(GuestUser, false)`
- [x] `TestBearerHMACMissingHeader` — no Authorization → `(GuestUser, false)`
- [x] `TestCookieSessionValid` — signed cookie → `(user, true)`
- [x] `TestCookieSessionInvalid` — bad cookie value → `(GuestUser, false)`
- [x] `TestCookieSessionNoCookie` — missing cookie → `(GuestUser, false)`
- [x] `TestCookieSessionCSRFEnabled` — default `csrfEnabled() == true`
- [x] `TestCookieSessionWithoutCSRF` — `WithoutCSRF` opt → `csrfEnabled() == false`
- [x] `TestBasicAuthValid` — matching credentials → `(user, true)`
- [x] `TestBasicAuthInvalid` — wrong password → `(GuestUser, false)`
- [x] `TestBasicAuthProductionWarn` — `warnIfProduction` writes expected string
- [x] `TestAnyAuthFirstWins` — first matching func wins; second not called
- [x] `TestAnyAuthNoneMatch` — all fail → `(GuestUser, false)`
- [x] `TestAnyAuthForwardsWarn` — `productionWarner` forwarded through `AnyAuth`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestUser|TestSign|TestBearer|TestCookie|TestBasic|TestAnyAuth" ./...` — all green
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendment S8 agreed:
      AuthFunc is an interface with unexported `authenticate` method; consistent
      with Option and Signal; enables capability detection in Steps 9 and 11

---

### Step 9 — middleware.go

**Depends on:** errors, context
**Decisions:** Amendment P2 (LRU cache)
**Files:** `middleware.go`, `middleware_test.go`
**Status:** ✅ Done

#### 9.1 — `statusRecorder` (unexported helper)

- [x] Unexported struct `statusRecorder{ http.ResponseWriter; status int }` that
      captures the HTTP status code written by handlers
- [x] Implement `WriteHeader(code int)` — stores code, delegates to embedded writer
- [x] Default status 200 if `WriteHeader` is never called (set on first `Write`)

#### 9.2 — `RequestLogger`

- [x] `func RequestLogger() func(http.Handler) http.Handler`
- [x] Before `next.ServeHTTP`: record `start := time.Now()`, call
      `ctx := ContextFrom(w, r)` (sets `X-Request-ID` on response), wrap `w`
      with `statusRecorder`
- [x] After `next.ServeHTTP`: log via `slog.InfoContext` with fields:
      `method`, `path`, `status`, `duration_ms` (float64 milliseconds),
      `request_id` from `ctx.RequestID()`

#### 9.3 — `Recoverer`

- [x] `func Recoverer() func(http.Handler) http.Handler`
- [x] Defers a recovery closure around `next.ServeHTTP`
- [x] On panic: call `WriteError(w, r, err)` with a 500-class `forge.Error`;
      log stack trace via `slog.ErrorContext` using `runtime.Stack`

#### 9.4 — `CORS`

- [x] `func CORS(origin string) func(http.Handler) http.Handler`
- [x] Sets on every request:
  - `Access-Control-Allow-Origin: <origin>`
  - `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
  - `Access-Control-Allow-Headers: Content-Type, Authorization, X-CSRF-Token`
- [x] On `OPTIONS` preflight: respond `204 No Content`, do not call `next`

#### 9.5 — `MaxBodySize`

- [x] `func MaxBodySize(n int64) func(http.Handler) http.Handler`
- [x] Wraps `r.Body` with `http.MaxBytesReader(w, r.Body, n)` before calling `next`

#### 9.6 — `SecurityHeaders`

- [x] `func SecurityHeaders() func(http.Handler) http.Handler`
- [x] Sets on every response (before calling `next`):
  - `Strict-Transport-Security: max-age=63072000; includeSubDomains`
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Content-Security-Policy: default-src 'self'`

#### 9.7 — `RateLimit` (token bucket per IP)

- [x] Unexported `ipBucket{ tokens float64; lastSeen time.Time; mu sync.Mutex }`
- [x] Unexported `rateLimiter{ buckets map[string]*ipBucket; mu sync.RWMutex;
      rate float64; max float64 }`
- [x] `func RateLimit(n int, d time.Duration) func(http.Handler) http.Handler`
- [x] Per-request: extract IP via `net.SplitHostPort`; replenish tokens since
      `lastSeen`; cap at `n`; if tokens ≥ 1 decrement and proceed; else 429 with
      `Retry-After` header (seconds until 1 token available)
- [x] Background goroutine (spawned once at middleware creation): `time.NewTicker(d)`
      sweeps map, deletes buckets with `lastSeen > 2×d` ago

#### 9.8 — `InMemoryCache` + `CacheMaxEntries` option

- [x] Unexported `cacheMaxEntriesOption{ n int }` implementing `forge.Option`
      (`isOption()` marker from `roles.go`)
- [x] `func CacheMaxEntries(n int) Option` — returns `cacheMaxEntriesOption{n}`
- [x] Unexported LRU types (~40 lines):
  - `lruEntry{ key string; body []byte; header http.Header; status int;
    expires time.Time; prev, next *lruEntry }`
  - `lruCache{ mu sync.Mutex; entries map[string]*lruEntry; head, tail *lruEntry;
    max, count int; ttl time.Duration }`
  - `get(key) (*lruEntry, bool)` — lazy TTL check; move-to-front
  - `set(key string, e *lruEntry)` — add to front; evict LRU tail if `count > max`
  - `sweep()` — remove all expired entries; called by background goroutine
- [x] `func InMemoryCache(ttl time.Duration, opts ...Option) func(http.Handler) http.Handler`
- [x] Cache key: `r.Method + " " + r.URL.RequestURI() + " " + r.Header.Get("Accept")`
- [x] Cache only `GET` requests; always set `X-Cache: HIT` or `X-Cache: MISS`
- [x] On HIT: write cached headers + status + body; skip `next`
- [x] On MISS: wrap `w` with `cacheRecorder`; after `next`, store if status 200
- [x] Background goroutine: `time.NewTicker(60 * time.Second)` → `cache.sweep()`

#### 9.9 — `Chain`

- [x] `func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler`
- [x] Apply middlewares in reverse order so slice index 0 is the outermost wrapper
- [x] ~5 lines

#### 9.10 — Tests (`middleware_test.go`)

- [x] `TestRecovererCatchesPanic` — panicking handler → 500 response, no crash
- [x] `TestRecovererPassesThrough` — non-panicking handler → 200, unaffected
- [x] `TestRequestLoggerSetsRequestID` — `X-Request-ID` present on response
- [x] `TestCORSHeaders` — CORS headers present
- [x] `TestCORSPreflight` — OPTIONS → 204, `next` not called
- [x] `TestMaxBodySizeRejects` — oversized body → 413
- [x] `TestSecurityHeadersPresent` — all five security headers set
- [x] `TestRateLimitAllows` — under limit → 200
- [x] `TestRateLimitRejects` — over limit → 429 with `Retry-After`
- [x] `TestInMemoryCacheMISS` — first request → `X-Cache: MISS`
- [x] `TestInMemoryCacheHIT` — second identical request → `X-Cache: HIT`, body matches
- [x] `TestInMemoryCacheEviction` — `CacheMaxEntries(1)` + two keys → LRU entry evicted
- [x] `TestInMemoryCacheTTLExpiry` — entry past TTL → MISS on next request
- [x] `TestChain` — middlewares applied in correct order
- [x] `BenchmarkInMemoryCacheHIT` — hot-path benchmark for cached GET response

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestRecoverer|TestRequest|TestCORS|TestMax|TestSecurity|TestRate|TestInMemory|TestChain" ./...` — all green
- [x] `go test -bench BenchmarkInMemoryCache ./...` — runs without error
- [x] Review ARCHITECTURE.md and DECISIONS.md — no new decisions required
      (Amendment P2 covers LRU; no auth middleware in this step)

---

## Layer 3 — Depends on all of Layer 0+1+2

### Step 10 — module.go

**Depends on:** node, context, auth, signals, storage, errors, middleware
**Decisions:** Decision 4 (content negotiation), 14 (lifecycle), 19 (MCP no-op); Amendments M1, M2, M3, P2
**Files:** `module.go`, `module_test.go`
**Status:** ✅ Done

#### 10.1 — Amendments (M1, M2, M3, M4)

- [x] Document Amendment M1 (`forge.Repo[T any]` storage injection) in `DECISIONS.md`
- [x] Document Amendment M2 (export `CacheStore` from `middleware.go`) in `DECISIONS.md`
- [x] Document Amendment M3 (`Module[T any]` not `[T forge.Node]`) in `DECISIONS.md`
- [x] Document Amendment M4 (`stringField` embedded struct fix in `storage.go`) in `DECISIONS.md`
- [x] Update `middleware.go`: rename `lruCache` → `CacheStore` (exported), add `NewCacheStore`,
      add `Flush()`, update `lruEntry` → `cacheEntry` (exported), update `InMemoryCache` to use
      `*CacheStore`; keep all existing behaviour unchanged
- [x] Update `storage.go`: fix `stringField` to handle embedded struct fields via `goFieldPath`
- [x] Update `middleware_test.go` if any internal type references need updating (none needed)

#### 10.2 — Option types (`module.go`)

- [x] `atOption{ prefix string }` implementing `isOption()` + `forge.At(prefix string) Option`
- [x] `cacheOption{ ttl time.Duration }` + `forge.Cache(ttl time.Duration) Option`
- [x] `middlewareOption{ mws []func(http.Handler) http.Handler }` +
      `forge.Middleware(mws ...func(http.Handler) http.Handler) Option`
- [x] `authOption{ opts []Option }` + `forge.Auth(opts ...Option) Option`
      (wraps `roleOption` values: `Read`, `Write`, `Delete` from `roles.go`)
- [x] `repoOption[T any]{ repo Repository[T] }` (generic struct implementing `isOption()`) +
      `forge.Repo[T any](r Repository[T]) Option`
- [x] Default role constants used when no `forge.Auth(...)` given:
      `Read(Guest)`, `Write(Author)`, `Delete(Editor)`

#### 10.3 — `Markdownable` interface + `contentNegotiator`

- [x] `type Markdownable interface{ Markdown() string }` — exported; declared in `module.go`
- [x] Unexported `contentNegotiator` struct built once at module construction:
  ```go
  type contentNegotiator struct {
      json  bool // always true
      html  bool // true only if forge.Templates option given (Milestone 3)
      md    bool // true if prototype implements Markdownable
      plain bool // always true
  }
  ```
- [x] `negotiate(r *http.Request) string` method — returns canonical content-type string;
      uses `strings.Contains` on `Accept` header; order: json → html → md → plain → json (fallback)
- [x] `Vary: Accept` set on every negotiated response

#### 10.4 — `Module[T any]` struct + `NewModule` + reflection helpers

- [x] Unexported reflection helpers (cached in `sync.Map` keyed by `reflect.Type`):
  - `nodeStatus(v any) Status`  — reads `Status` field
  - `nodeSlug(v any) string`    — reads `Slug` field
  - `nodeID(v any) string`      — reads `ID` field
  - `setNodeID(v any, id string)` — sets `ID` field
  - `setNodeSlug(v any, slug string)` — sets `Slug` field
- [x] `Module[T any]` struct:
  ```go
  type Module[T any] struct {
      prefix      string
      repo        Repository[T]
      readRole    Role
      writeRole   Role
      deleteRole  Role
      signals     map[Signal][]signalHandler
      cache       *CacheStore          // nil if no forge.Cache option
      middlewares []func(http.Handler) http.Handler
      neg         contentNegotiator
      debounce    *debouncer
      proto       reflect.Type         // reflect.TypeOf(T)
  }
  ```
- [x] `NewModule[T any](proto T, opts ...Option) *Module[T]`:
  - Parses all options via type switch
  - Panics if no `repoOption[T]` found: `"forge: Module[T] requires a Repository; use forge.Repo(...)"`
  - Defaults: `readRole = Guest`, `writeRole = Author`, `deleteRole = Editor`
  - Detects `Markdownable` by interface assertion on zero-value `proto`
  - Creates `*CacheStore` via `NewCacheStore(ttl, 1000)` if `cacheOption` present
  - Wires `signalOption` values from `On[T]` calls into `signals` map
  - Creates `debouncer` (2s) for `SitemapRegenerate` signal

#### 10.5 — `Register(mux *http.ServeMux)` and middleware wrapping

- [x] `Register(mux *http.ServeMux)` registers five routes:
  ```
  GET  /{prefix}         → listHandler
  GET  /{prefix}/{slug}  → showHandler
  POST /{prefix}         → createHandler
  PUT  /{prefix}/{slug}  → updateHandler
  DELETE /{prefix}/{slug}→ deleteHandler
  ```
- [x] Each route handler is wrapped with `Chain(handler, m.middlewares...)` if middlewares present
- [x] `ContextFrom(w, r)` called at the start of every handler (before any other logic)

#### 10.6 — `listHandler`

- [x] Lifecycle filter: build `ListOptions` with appropriate status filter based on `ctx.User()` roles:
  - Guest → only `Published`
  - Author → all statuses
  - Editor+ → all statuses
- [x] Call `m.repo.List(ctx, opts)` — return 200 with negotiated content type
- [x] On cache hit (`GET` only): write cached response + `X-Cache: HIT`; return early
- [x] On cache miss: serve response, store in cache if status 200, set `X-Cache: MISS`

#### 10.7 — `showHandler`

- [x] Extract `slug` via `r.PathValue("slug")`
- [x] `m.repo.Get(ctx, slug)` — on `ErrNotFound`: `WriteError(w, r, ErrNotFound)`
- [x] Lifecycle enforcement:
  - If `nodeStatus(item) != Published && !ctx.User().HasRole(Author)` → `WriteError(w, r, ErrNotFound)`
  - Author (no Editor) sees all non-published content in this module (Amendment M3 simplified rule)
  - Editor+ sees everything
- [x] Content negotiation:
  - `application/json` → `json.NewEncoder(w).Encode(item)`
  - `text/markdown` → `item.(Markdownable).Markdown()` if `neg.md`, else 406
  - `text/plain` → naive markdown-strip (remove `#`, `*`, `_`, links); stdlib only
  - `text/html` → 406 `"HTML templates not registered"` until Milestone 3
- [x] Cache: same HIT/MISS pattern as listHandler

#### 10.8 — `createHandler`

- [x] Role check: `ctx.User().HasRole(m.writeRole)` → else `WriteError(w, r, ErrForbidden)`
- [x] Decode: `json.NewDecoder(r.Body).Decode(&item)` → on error: 400
- [x] Set `ID = NewID()`, auto-generate `Slug` from first required string field if empty
- [x] `RunValidation(&item)` → on error: `WriteError(w, r, err)` (aborts; 422)
- [x] `dispatchBefore(ctx, m.signals[BeforeCreate], item)` → on error: `WriteError`
- [x] `m.repo.Save(ctx, item)`
- [x] `dispatchAfter(ctx, m.signals[AfterCreate], item)`
- [x] Signal status-based hooks: `AfterPublish` if `nodeStatus == Published`
- [x] Invalidate cache: `if m.cache != nil { m.cache.Flush() }`
- [x] Debounce `SitemapRegenerate` signal
- [x] Respond 201 with JSON-encoded item

#### 10.9 — `updateHandler`

- [x] Role check: `ctx.User().HasRole(m.writeRole)` → else `WriteError(w, r, ErrForbidden)`
- [x] Fetch existing: `m.repo.Get(ctx, slug)` → ErrNotFound → WriteError
- [x] Decode request body into `item`
- [x] `RunValidation(&item)` → on error: WriteError (422)
- [x] `dispatchBefore(ctx, m.signals[BeforeUpdate], item)` → on error: WriteError
- [x] `m.repo.Save(ctx, item)`
- [x] `dispatchAfter(ctx, m.signals[AfterUpdate], item)`
- [x] Status-based hooks: `AfterPublish` / `AfterUnpublish` / `AfterArchive`
- [x] Cache flush + sitemap debounce
- [x] Respond 200 with JSON-encoded item

#### 10.10 — `deleteHandler`

- [x] Role check: `ctx.User().HasRole(m.deleteRole)` → else `WriteError(w, r, ErrForbidden)`
- [x] Fetch existing: `m.repo.Get(ctx, slug)` → ErrNotFound → WriteError
- [x] `dispatchBefore(ctx, m.signals[BeforeDelete], item)` → on error: WriteError
- [x] `m.repo.Delete(ctx, slug)` (assumes `Repository[T]` has `Delete(ctx, slug string) error`;
      add `Delete` to `Repository[T]` interface in storage.go if not present)
- [x] `dispatchAfter(ctx, m.signals[AfterDelete], item)`
- [x] Cache flush + sitemap debounce
- [x] Respond 204 No Content

#### 10.11 — Tests (`module_test.go`)

- [x] `TestModuleListGuestPublishedOnly` — Guest GET list; only published items returned
- [x] `TestModuleListAuthorSeesAll` — Author GET list; all statuses returned
- [x] `TestModuleShowPublishedGuest` — Guest GET show; published item → 200
- [x] `TestModuleShowDraftGuest` — Guest GET show; draft item → 404
- [x] `TestModuleShowDraftAuthor` — Author GET show; draft item → 200
- [x] `TestModuleCreateValidation` — POST with invalid body → 422, repo unchanged
- [x] `TestModuleCreateSuccess` — POST valid body → 201, ID and Slug set
- [x] `TestModuleUpdateForbiddenGuest` — Guest PUT → 403
- [x] `TestModuleDeleteForbiddenAuthor` — Author DELETE → 403
- [x] `TestModuleContentNegotiationJSON` — GET with `Accept: application/json` → JSON body
- [x] `TestModuleContentNegotiationMarkdown` — GET with `Accept: text/markdown` + Markdownable → markdown body
- [x] `TestModuleContentNegotiationMarkdownUnsupported` — GET `Accept: text/markdown` + non-Markdownable → 406
- [x] `TestModuleContentNegotiationHTML` — GET `Accept: text/html` → 406 (no templates)
- [x] `TestModuleCacheMISS` — first GET → `X-Cache: MISS`
- [x] `TestModuleCacheHIT` — second identical GET → `X-Cache: HIT`
- [x] `TestModuleCacheInvalidatedOnCreate` — POST → subsequent GET is MISS
- [x] `TestModuleSignalBeforeCreateAborts` — BeforeCreate returns error → 500, not saved
- [x] `TestModuleSignalAfterCreateFires` — AfterCreate goroutine fires
- [x] `BenchmarkModuleRequest` — in-memory repo, JSON GET show, warm cache

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run "TestModule" ./...` — all green
- [x] `go test -bench BenchmarkModuleRequest ./...` — runs without error
- [x] `go test ./...` — full suite green
- [x] Review ARCHITECTURE.md and DECISIONS.md — Amendments M1, M2, M3, M4 drafted and documented

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
