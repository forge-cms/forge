# Forge Milestone 10 — Test Strategy

**Status:** Draft — pre-implementation review  
**Date:** 2026-03-16  
**Scope:** forge core (Amendment A49) + `forge-mcp` module (Steps 1–5)

---

## 0. Cross-cutting test infrastructure

### Canonical test content type

Used in all steps:

```go
type testMCPPost struct {
    forge.Node
    Title  string `forge:"required,min=3"`
    Body   string `forge:"required,min=10"`
    Rating int
    Tags   string `json:"tags"`
}
```

This type exercises: required fields, min constraints, a numeric field, a
`json:` tag override, and the embedded `Node`. It is the canonical fixture for
all M10 tests.

### Context construction rule

All `MCPModule` methods receive a `forge.Context` and call
`ctx.Request().Context()` for repo operations. A plain `context.Context` cannot
be passed. The correct test patterns are:

```go
// For operations that need authentication:
authorCtx := forge.NewTestContext(forge.User{ID: "u1", Roles: []forge.Role{forge.Author}})
editorCtx := forge.NewTestContext(forge.User{ID: "e1", Roles: []forge.Role{forge.Editor}})
guestCtx  := forge.NewTestContext(forge.GuestUser)

// For background/scheduler-style operations:
bgCtx := forge.NewBackgroundContext("example.com")
```

### Field lookup helper (schema tests)

Schema field ordering is not specified. All schema assertions must use a lookup
helper rather than positional indexing:

```go
func findField(schema []forge.MCPField, name string) (forge.MCPField, bool) {
    for _, f := range schema {
        if f.Name == name {
            return f, true
        }
    }
    return forge.MCPField{}, false
}
```

---

## Step 1 — forge-mcp scaffold + Amendment A49

### 1.1 Observable contracts

| Observable | Contract |
|---|---|
| `MCP(MCPRead)` stored | `m.MCPMeta().Operations` contains exactly `MCPRead` |
| `MCP(MCPRead, MCPWrite)` stored | `Operations` contains both, in order |
| No `MCP(...)` option | `MCPMeta()` returns zero value (`Operations` nil or empty) |
| `App.MCPModules()` | Returns only modules registered with `MCP(...)`; a module without `MCP(...)` is absent |
| `App.MCPModules()` ordering | Returned in `Content()` registration order |
| `MCPSchema()` — required field | `MCPField.Required == true` for a field tagged `forge:"required"` |
| `MCPSchema()` — min constraint | `MCPField.MinLength == 3` for `forge:"required,min=3"` |
| `MCPSchema()` — json: tag | `MCPField.JSONName == "tags"` for `json:"tags"` |
| `MCPSchema()` — numeric type | `MCPField.Type == "number"` for `int` field |
| `MCPSchema()` — Node ID excluded | No `MCPField` with `Name == "ID"` |
| `MCPSchema()` — Slug included | `MCPField` with `Name == "Slug"` present |
| `inputSchema` output | Produces a JSON Schema object with `"required"` array for required fields |

### 1.2 Edge cases

- `MCP()` called with no args: `Operations` is empty (not nil), or nil — the
  test must specify which; schema derivation and tool generation must handle both
- Content type with only `forge.Node` embedded and no other fields:
  `MCPSchema()` returns only Node fields (Slug, Status, PublishedAt, ScheduledAt)
- Two modules registered, only one with `MCP(...)`: `App.MCPModules()` has length 1
- Field with both `json:"-"` and `forge:"required"`: excluded from schema
  (consistent with JSON marshalling — backlog is currently silent on this)

### 1.3 Test location

| Location | What it covers |
|---|---|
| `integration_full_test.go` G22 | `MCPMeta()`, `MCPSchema()`, `App.MCPModules()` from within the forge package |
| `forge-mcp/mcp_test.go` | `inputSchema()` JSON Schema correctness; `moduleByPrefix()` lookup |

### 1.4 Test doubles and fixtures

- `forge.NewMemoryRepo[*testMCPPost]()`
- `forge.NewTestContext(authorUser)`
- No HTTP server required for any Step 1 assertion

### 1.5 Flags

**Flag A — `forge.Authenticator` does not exist. [BLOCKER]**  
The backlog (§1.4) defines `Server.auth forge.Authenticator`. Forge core
exposes `forge.AuthFunc` (an interface), not `forge.Authenticator`. Resolution:
change `Server.auth` field type to `forge.AuthFunc`. No new interface needed.

**Flag B — `MCPSchema` field ordering unspecified. [Low]**  
Tests must use `findField()` helper (§0), not positional indexing.

**Flag C — `json:"-"` field interaction with `MCPSchema` unspecified. [Low]**  
Should be excluded from schema (consistent with JSON marshalling). The backlog
must state this explicitly. A test `TestMCPSchema_excludesJSONDashField` should
be added.

---

## Step 2 — `forge-mcp/resource.go` (read path)

### 2.1 Observable contracts

| Observable | Contract |
|---|---|
| `resources/list` — lifecycle filter | Only Published items appear; Draft, Scheduled, Archived items are absent |
| `resources/list` — multi-module | Items from all MCPRead modules aggregated; MCPWrite-only modules absent |
| `resources/list` — URI format | Each resource URI is `"forge://{prefix}/{slug}"` |
| `resources/list` — empty module | Module with zero Published items contributes nothing to the list |
| `resources/read` — published | Returns `{"contents": [{uri, mimeType, text: "<json>"}]}` |
| `resources/read` — draft | Returns MCP error -32001 |
| `resources/read` — bad URI | Returns -32001 (unknown prefix or malformed `forge://` scheme) |
| `resources/templates/list` | Returns exactly one template per MCPRead module, including modules with no items |

### 2.2 Edge cases

- Module with both `MCPRead` and `MCPWrite`: appears in both `resources/list`
  and `tools/list`
- Slug containing hyphens (`hello-world`): URI round-trips correctly
  `forge://posts/hello-world` → parse → `prefix=/posts slug=hello-world`
- URI with extra path segments (`forge://posts/a/b`): backlog only handles two
  path components — behaviour for extra segments is unspecified (should return
  -32001)
- Two modules with the same prefix (misconfiguration): `moduleByPrefix` returns
  the first match — backlog should document this assumption

### 2.3 Test location

`forge-mcp/mcp_test.go` — all; cannot use forge internal types directly;
items are seeded via `forge.MemoryRepo` and status is set at construction time
(not via `MCPPublish`, to avoid a Step 3 dependency).

### 2.4 Test doubles and fixtures

```go
repo := forge.NewMemoryRepo[*testMCPPost]()
mod  := forge.NewModule((*testMCPPost)(nil),
    forge.Repo(repo), forge.At("/posts"), forge.MCP(forge.MCPRead))
app  := forge.New(forge.MustConfig(forge.Config{
    BaseURL: "https://example.com",
    Secret:  []byte("16bytessecretkey"),
}))
app.Content(mod)
srv := forgemcp.New(app)
ctx := forge.NewTestContext(forge.GuestUser)
```

Seed published and draft items directly into the repo by setting
`forge.Node.Status` at construction time.

### 2.5 Flags

**Flag D — `MCPList` returns `[]any`; item inspection in `forge-mcp` requires
JSON round-trip. [Low]**  
Tests in `forge-mcp` cannot import `testMCPPost`. To assert field values, items
must be marshalled to JSON and fields inspected via `map[string]any`. This is
workable but should be the documented testing pattern.

**Flag E — `MCPGet` has no lifecycle filter. [Low]**  
`MCPGet(ctx, slug)` returns any item regardless of status. The Published check
sits in `handleResourcesRead`, not in `MCPGet`. The `MCPModule` interface godoc
must explicitly state that `MCPGet` has no lifecycle filter and callers are
responsible for enforcement.

---

## Step 3 — `forge-mcp/tool.go` (write path)

### 3.1 Observable contracts

| Observable | Contract |
|---|---|
| `tools/list` — 6 tools per MCPWrite module | `create_`, `update_`, `publish_`, `schedule_`, `archive_`, `delete_` all present |
| `tools/list` — MCPRead-only module | Contributes nothing to `tools/list` |
| `create_*` — valid fields | Item saved with `Status == Draft`; `ID` and `Slug` non-empty |
| `create_*` — missing required field | MCP error -32602; no item saved |
| `update_*` — partial update | Supplied fields overwrite; unsupplied fields retain original values |
| `publish_*` | Draft → Published; `PublishedAt` non-zero and >= time before the call |
| `schedule_*` | Status == Scheduled; `ScheduledAt` matches the supplied RFC3339 time |
| `archive_*` | Status == Archived |
| `delete_*` | Item removed from repo; subsequent `MCPGet` returns not-found |
| Role enforcement — Guest | `handleToolsCall` returns an error before calling `MCPCreate` |
| Unknown tool name | -32602 |
| Unknown type in tool name | -32602 |

### 3.2 Edge cases

- `create_*` with a caller-supplied `ID` in the args map: should be ignored
  (backlog says "set if not already set" — which means a supplied ID would be
  honoured via JSON unmarshal). The backlog is ambiguous; the safe choice is to
  always generate a fresh ID and document this explicitly.
- `update_*` with no `slug` key in args: dispatcher must return -32602, not panic
- `schedule_*` with a past time: no validation error — the scheduler fires on
  next tick. This is correct but should be verified not to error.
- `publish_*` on an already-Published item: behaviour unspecified (see Flag H)
- `AfterCreate` / `AfterPublish` signal fire: verified by registering
  `forge.On[*testMCPPost]` on the module before constructing the Server, then
  checking an atomic counter in the signal handler

### 3.3 Test location

`forge-mcp/mcp_test.go` — all. Role enforcement tests require constructing
contexts with different users.

### 3.4 Test doubles and fixtures

Same as Step 2 pattern, with write-capable context:

```go
authorCtx := forge.NewTestContext(forge.User{ID: "u1", Roles: []forge.Role{forge.Author}})
guestCtx  := forge.NewTestContext(forge.GuestUser)
```

Signal fire verification:

```go
var published int64
mod := forge.NewModule((*testMCPPost)(nil),
    forge.Repo(repo),
    forge.At("/posts"),
    forge.MCP(forge.MCPWrite),
    forge.On[*testMCPPost](forge.AfterPublish, func(_ forge.Context, _ *testMCPPost) error {
        atomic.AddInt64(&published, 1)
        return nil
    }),
)
```

### 3.5 Flags

**Flag F — `MCPDelete` success response via `tools/call` unspecified. [Medium]**  
`MCPDelete` returns `error` only. The dispatcher must return something on
success. Recommended: `map[string]any{"deleted": true, "slug": slug}`. The
backlog must specify this before implementation.

**Flag G — `MCPUpdate` cannot clear a field to zero. [Medium]**  
"Copy non-zero fields" means a caller cannot set a string field to `""`. There
is no escape hatch specified. This is a known limitation that must be documented
in `MCPModule` godoc. A test `TestMCPToolsCall_update_cannot_clear_field` should
verify and document the behaviour rather than leaving it implicit.

**Flag H — `MCPPublish` on already-Published item unspecified. [Low]**  
The plan says "check `item.Status != Published`" without specifying the error
path. Recommended: silently succeed (idempotent publish). Add
`TestMCPToolsCall_publish_already_published` to the backlog.

**Flag I — Acronym handling in camelCase → snake_case conversion. [HIGH]**  
`testMCPPost.TypeName == "testMCPPost"`. The algorithm for converting `MCP` in
a type name to snake_case is unspecified: `mcp_post` (words) or `m_c_p_post`
(letters)? This directly affects all tool names and the `tools/call` dispatcher
lookup. Both places must use the same algorithm.

**Resolution required before Step 3:** Specify the algorithm (e.g. consecutive
uppercase letters are treated as one word: `MCPPost` → `mcp_post`) and add a
dedicated unit test for `snakeCase()` in `forge-mcp/mcp_test.go`.

---

## Step 4 — `forge-mcp/transport.go`

### 4.1 Observable contracts

| Observable | Contract |
|---|---|
| stdio `initialize` | Returns `{"jsonrpc":"2.0","id":1,"result":{...}}` with correct `protocolVersion` |
| stdio unknown method | Returns `{"error":{"code":-32601,...}}` |
| stdio malformed JSON | Returns `{"error":{"code":-32700,...}}` (parse error, not panic) |
| SSE `POST /mcp/message` — no token | Returns JSON-RPC error (or 401 HTTP status) |
| SSE `POST /mcp/message` — valid token | 200 + correct result |
| SSE `GET /mcp` — connection | `Content-Type: text/event-stream`; sends `event: open` |
| `jsonrpc: "2.0"` on all responses | Every response (including errors) carries this field |
| `id` echo | Response `id` matches request `id`; `null` for notifications |

### 4.2 Edge cases

- Stdio: empty line (`\n`): must not panic; return parse error
- Stdio: concurrent requests without waiting for responses: synchronous
  processing is correct for stdio but must be documented
- SSE: token signed with a different secret: authentication failure
- SSE: valid token but insufficient role for the requested tool: MCP error,
  **not** 401 HTTP status — authentication succeeded; authorisation failed
- SSE: request body larger than a reasonable limit: server must not OOM; the
  backlog is silent on body size limits

### 4.3 Test location

`forge-mcp/mcp_test.go` — all transport tests.  
SSE tests use `httptest.NewRequest` + `httptest.NewRecorder`.  
SSE auth tests use `forge.SignToken` to create valid Bearer tokens.

### 4.4 Test doubles and fixtures

**Critical — `ServeStdio` must accept injectable I/O.**

The backlog says `ServeStdio` reads from `os.Stdin`. The backlog test
`TestMCPServeStdio_roundtrip` says to "pipe stdin/stdout through `io.Pipe`".
These are contradictory: `io.Pipe` cannot replace `os.Stdin`.

**The `ServeStdio` signature must be changed to:**

```go
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error
```

A thin wrapper using `os.Stdin`/`os.Stdout` for production CLI use is the
correct pattern (same as `gopls`, `terraform`, and all other MCP servers).

Test pattern:

```go
func TestMCPServeStdio_roundtrip(t *testing.T) {
    pr, pw := io.Pipe()
    var buf bytes.Buffer

    srv := forgemcp.New(app)
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    done := make(chan error, 1)
    go func() { done <- srv.ServeStdio(ctx, pr, &buf) }()

    req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
    io.WriteString(pw, req)
    pw.Close()
    <-done

    // assert buf contains a valid initialize response
}
```

SSE auth test:

```go
secret := []byte("16bytessecretkey")
token, _ := forge.SignToken(forge.User{ID: "u1", Roles: []forge.Role{forge.Author}}, secret, 0)
r := httptest.NewRequest("POST", "/mcp/message", body)
r.Header.Set("Authorization", "Bearer "+token)
```

### 4.5 Flags

**Flag J — `ServeStdio` hardcodes `os.Stdin`/`os.Stdout`. [BLOCKER]**  
Described in §4.4. Tests `TestMCPServeStdio_*` cannot be implemented as written.
The signature must be changed. This must be resolved in the backlog before
Step 4 begins.

**Flag K — Stdio Admin-role assumption should be explicit. [Low]**  
The background context for stdio uses `forge.Admin` role (the process is
locally trusted). A test `TestMCPServeStdio_contextHasAdminRole` should verify
this. The documentation must note that any process able to write to stdin has
full Admin access — appropriate for local tools only.

**Flag L — SSE auth type depends on Flag A. [BLOCKER]**  
The SSE transport authenticates via a `forge.AuthFunc` (not `forge.Authenticator`).
This is blocked by Flag A. Both must be resolved together in A49.

---

## Step 5 — `forge-mcp/README.md` + G22 + badges

### 5.1 Observable contracts (forge core — G22)

G22 lives in `integration_full_test.go` and tests the forge core side of A49
without any `forge-mcp` transport. `forge` core cannot import `forge-mcp`.

| Test | Contract |
|---|---|
| `TestFull_G22_MCPModuleInterface` | `App.MCPModules()` returns the correct modules; `MCPMeta()` returns correct Prefix and Operations; `MCPSchema()` returns at least one field with `Required: true` matching a `forge:"required"` tag |
| `TestFull_G22_MCPCreatePublishLifecycle` | `MCPCreate` → item in repo with Draft status; `MCPPublish` → item has Published status and non-zero `PublishedAt`; `MCPList(ctx, Published)` returns the item; `MCPList(ctx, Draft)` returns empty |

### 5.2 G22 edge cases

- Verify `AfterPublish` signal fires after `MCPPublish` by registering a counter
  handler before calling `MCPPublish`
- `MCPList` with no status args returns all items regardless of status (default
  `ListOptions{}` behaviour)

### 5.3 Test location

| Location | What it covers |
|---|---|
| `integration_full_test.go` G22 | forge core MCPModule interface (no transport) |
| `forge-mcp/mcp_test.go` | `ExampleNew` compile test |

### 5.4 Test doubles and fixtures

G22: `NewTestContext(forge.User{Roles: []forge.Role{forge.Author}})`,
`MemoryRepo[*testMCPPost]`, `NewModule` with `MCP(forge.MCPRead, forge.MCPWrite)`.

### 5.5 Flags

**Flag M — G22 `MCPCreate` map keys must use JSONNames. [Low]**  
`MCPCreate` receives a `map[string]any` and JSON-round-trips it into a `T`. The
map keys must match the struct's effective JSON names (`"title"` not `"Title"`,
`"tags"` not `"Tags"`). The G22 test fixture must be written to match the
JSONName convention to avoid a first-run failure that looks like a bug but is
just a test error.

---

## Summary of flags

| # | Severity | Description | Blocks |
|---|---|---|---|
| A | **BLOCKER** | `forge.Authenticator` does not exist; use `forge.AuthFunc` | Steps 1, 4 |
| B | Low | `MCPSchema` field ordering unspecified; tests must use name lookup | Step 1 |
| C | Low | `json:"-"` field interaction with `MCPSchema` unspecified | Step 1 |
| D | Low | `MCPList` returns `[]any`; inspection in `forge-mcp` requires JSON round-trip | Step 2 |
| E | Low | `MCPGet` has no lifecycle filter; callers responsible; must be documented | Step 2 |
| F | Medium | `MCPDelete` success response via `tools/call` unspecified | Step 3 |
| G | Medium | `MCPUpdate` cannot clear a field to zero value; must be documented | Step 3 |
| H | Low | `MCPPublish` on already-Published item — behaviour unspecified | Step 3 |
| I | **High** | Acronym handling in camelCase→snake_case must be specified before implementation | Steps 3, 4 |
| J | **BLOCKER** | `ServeStdio` hardcodes `os.Stdin`/`os.Stdout`; change signature to `(ctx, io.Reader, io.Writer)` | Step 4 |
| K | Low | Stdio Admin-role assumption should be explicit and tested | Step 4 |
| L | **BLOCKER** | Depends on Flag A; SSE auth field type unresolvable | Step 4 |
| M | Low | G22 `MCPCreate` test must use JSONName keys; easy first-run failure | Step 5 |

### Required backlog amendments before implementation begins

1. **Flags A + L:** Change `Server.auth` to `forge.AuthFunc` in §1.4 and §4.4
2. **Flag I:** Specify the camelCase→snake_case algorithm (consecutive uppercase
   = one word: `MCPPost` → `mcp_post`); add `snakeCase()` unit test to backlog
3. **Flag J:** Change `ServeStdio` signature to
   `ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error`
