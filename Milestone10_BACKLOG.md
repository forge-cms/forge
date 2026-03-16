# Forge — Milestone 10 Backlog (v2.0.0)

Introduce `forge-mcp` — a separate Go module that wraps a `forge.App` and
exposes MCP-registered content modules as MCP resources and tools, enabling
AI assistants to create, read, update, publish, and delete content through
the Model Context Protocol. Schema derivation is automatic from existing Go
struct tags. All role, lifecycle, and validation rules from the HTTP layer
apply without modification.

**Key decisions:**
- Decision 19 — MCP syntax reserved in v1 (`forge.MCP` option, `mcp.go`)
- Amendment A49 — `MCPModule` interface in forge core (`mcp.go`, `module.go`,
  `forge.go`); makes `mcpOption` carry its operations; lets `Module[T]`
  implement the interface; adds `App.MCPModules() []MCPModule`

**Constraints:**
- `forge-mcp` is a separate module (`github.com/forge-cms/forge-mcp`) with its
  own `go.mod`. It imports `forge` core; `forge` core must **never** import it.
- `forge` core defines the `MCPModule` interface so `Module[T]` can implement
  it and `forge-mcp` can consume it — no circular import, no reflection into
  forge internals from outside the package.
- Zero external dependencies in either `forge` core or `forge-mcp`. All MCP
  transport and protocol handling is implemented with stdlib only.
- All auth, validation, and lifecycle rules are enforced through the same code
  paths as the HTTP layer — no special MCP bypass.
- `forge` core version stays on the v1.x line. `forge-mcp` is a new module
  starting at v1.0.0.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | forge-mcp/mcp.go | ✅ Complete | 2026-03-16 |
| 2 | forge-mcp/resource.go | ✅ Complete | 2026-03-16 |
| 3 | forge-mcp/tool.go | 🔲 Not started | — |
| 4 | forge-mcp/transport.go | 🔲 Not started | — |
| 5 | forge-mcp/README.md | 🔲 Not started | — |

---

## Layer 10.A — forge core amendments + module scaffold (no prior M10 dependency)

### Step 1 — `forge-mcp/mcp.go` (new) + Amendment A49

**Depends on:** `mcp.go`, `module.go`, `forge.go`, `storage.go`
**Decisions:** Decision 19, Amendment A49
**Files:** `forge-mcp/mcp.go` (new), `forge-mcp/mcp_test.go` (new),
`forge-mcp/go.mod` (new); amendments to `mcp.go`, `module.go`, `forge.go`
in forge core; `go.work` updated

#### 1.1 — Amendment A49: `mcp.go` (forge core)

**Goal:** give `mcpOption` real data; define the contract types that
`Module[T]` will implement and `forge-mcp` will consume.

- [x] Change `mcpOption` to carry its operations:
  ```go
  type mcpOption struct{ ops []MCPOperation }
  ```
  Update `MCP(ops ...MCPOperation) Option` to store them:
  ```go
  func MCP(ops ...MCPOperation) Option { return mcpOption{ops: ops} }
  ```
- [x] Add `MCPMeta` struct (exported):
  ```go
  // MCPMeta describes the MCP registration of a content module.
  type MCPMeta struct {
      Prefix     string         // URL prefix, e.g. "/posts"
      TypeName   string         // content type name, e.g. "BlogPost"
      Operations []MCPOperation // MCPRead and/or MCPWrite
  }
  ```
- [x] Add `MCPField` struct (exported):
  ```go
  // MCPField describes a single field in a content type's MCP schema,
  // derived automatically from Go struct type and forge: struct tags.
  type MCPField struct {
      Name      string   // Go field name
      JSONName  string   // lowercase snake_case name used in MCP messages
      Type      string   // "string" | "number" | "boolean" | "datetime"
      Required  bool
      MinLength int      // 0 = no constraint
      MaxLength int      // 0 = no constraint
      Enum      []string // nil = no constraint
  }
  ```
- [x] Add `MCPModule` interface (exported):
  ```go
  // MCPModule is implemented by any Module[T] that has been registered with
  // forge.MCP(...). forge-mcp reads this interface to build MCP resources
  // and tools without accessing Module internals directly.
  //
  // All methods receive a forge.Context carrying the authenticated user.
  // Callers must construct the Context with appropriate Role before calling
  // any mutating method — the MCPModule implementation enforces roles and
  // validation identically to the HTTP layer.
  type MCPModule interface {
      MCPMeta() MCPMeta
      MCPSchema() []MCPField
      MCPList(ctx Context, status ...Status) ([]any, error)
      MCPGet(ctx Context, slug string) (any, error)
      MCPCreate(ctx Context, fields map[string]any) (any, error)
      MCPUpdate(ctx Context, slug string, fields map[string]any) (any, error)
      MCPPublish(ctx Context, slug string) error
      MCPSchedule(ctx Context, slug string, at time.Time) error
      MCPArchive(ctx Context, slug string) error
      MCPDelete(ctx Context, slug string) error
  }
  ```
- [x] Update godoc on `MCPRead`, `MCPWrite`, and `MCP()` to reference
  the v2 implementation (remove "no-op in v1" language, add "see MCPModule").
- [x] Add `"time"` import to `mcp.go` (required for `MCPSchedule`).

#### 1.2 — Amendment A49: `module.go` (forge core)

**Goal:** implement `MCPModule` on `Module[T]`.

- [x] `Module[T].MCPMeta() MCPMeta`:
  - Walk `m.options` looking for `mcpOption`; if not found, return zero value
  - Return `MCPMeta{Prefix: m.prefix, TypeName: typeName(reflect.TypeOf((*T)(nil)).Elem()), Operations: opt.ops}`
  - `typeName` helper: `reflect.Type.Name()`, stripping any package prefix
- [x] `Module[T].MCPSchema() []MCPField`:
  - `t := reflect.TypeOf((*T)(nil)).Elem()`
  - Walk all fields; skip embedded `forge.Node` struct itself (include its
    exported non-ID fields: Slug, Status, PublishedAt, ScheduledAt)
  - For each remaining field: derive JSONName (lowercase snake_case from field
    name; honour existing `json:` tag if present; consecutive uppercase letters
    are treated as a single word — `MCPPost → mcp_post`, not `m_c_p_post`),
    Type (map Go type →
    "string" | "number" | "boolean" | "datetime"), parse `forge:` tag using
    the same tag-parsing logic as `runValidation` — reuse unexported helpers
    already in `node.go` (`parseForgeTag` or equivalent); extract Required,
    MinLength, MaxLength, Enum
  - Return `[]MCPField`
- [x] `Module[T].MCPList(ctx Context, status ...Status) ([]any, error)`:
  - `opts := ListOptions{}`; if `len(status) > 0`, set `opts.Status = status`
  - Call `m.repo.FindAll(ctx.Request().Context(), opts)`
  - Convert `[]T` to `[]any` and return
- [x] `Module[T].MCPGet(ctx Context, slug string) (any, error)`:
  - Call `m.repo.FindBySlug(ctx.Request().Context(), slug)`
  - Return as `any`
- [x] `Module[T].MCPCreate(ctx Context, fields map[string]any) (any, error)`:
  - Marshal `fields` to JSON, unmarshal into a new `T`
  - Set `Node.ID = NewID()` if not already set
  - Set `Node.Status = Draft` if not set
  - Run `RunValidation(&item)` — return 422-equivalent error on failure
  - `m.repo.Save(ctx.Request().Context(), &item)`
  - `dispatchAfter(ctx, m.signals[AfterCreate], &item)`
  - Invalidate cache
  - Return `&item, nil`
- [x] `Module[T].MCPUpdate(ctx Context, slug string, fields map[string]any) (any, error)`:
  - `m.repo.FindBySlug(ctx.Request().Context(), slug)` → item
  - Marshal `fields` to JSON; unmarshal into a temporary `T`; copy non-zero
    fields from the temporary value onto item using reflection (preserve Node.ID
    and Node.Status)
  - `RunValidation(&item)` — return error on failure
  - `m.repo.Save(ctx.Request().Context(), &item)`
  - `dispatchAfter(ctx, m.signals[AfterUpdate], &item)`
  - Invalidate cache
  - Return `&item, nil`
- [x] `Module[T].MCPPublish(ctx Context, slug string) error`:
  - Find item; check `item.Status != Published`
  - Set `Status = Published`, `PublishedAt = time.Now().UTC()`
  - Save; `dispatchAfter(AfterPublish)`; trigger sitemap+feed; invalidate cache
- [x] `Module[T].MCPSchedule(ctx Context, slug string, at time.Time) error`:
  - Find item; set `Status = Scheduled`, `ScheduledAt = &at`
  - Save; invalidate cache
- [x] `Module[T].MCPArchive(ctx Context, slug string) error`:
  - Find item; set `Status = Archived`
  - Save; invalidate cache; trigger sitemap
- [x] `Module[T].MCPDelete(ctx Context, slug string) error`:
  - Find item (to get ID); `m.repo.Delete(ctx.Request().Context(), item.Node.ID)`
  - `dispatchAfter(AfterDelete)`; invalidate cache; trigger sitemap
- [x] Compile-time interface check in `module.go`:
  ```go
  var _ MCPModule = (*Module[struct{ Node }])(nil)
  ```

#### 1.3 — Amendment A49: `forge.go` (forge core)

- [x] Add `mcpModules []MCPModule` field to `App` struct
- [x] In `App.Content()`: after registering the module, check if any option is
  `mcpOption`; if yes, append `r.(MCPModule)` to `a.mcpModules`
- [x] Add `App.MCPModules() []MCPModule`:
  ```go
  // MCPModules returns all content modules registered with forge.MCP(...).
  // forge-mcp calls this to build its resource and tool registry.
  func (a *App) MCPModules() []MCPModule { return a.mcpModules }
  ```
- [x] Verify type assertion `r.(MCPModule)` compiles (Module[T] satisfies the
  interface from 1.2)

#### 1.4 — forge-mcp module scaffold

- [x] Create `forge-mcp/go.mod`:
  ```
  module github.com/forge-cms/forge-mcp

  go 1.24.0

  require github.com/forge-cms/forge v0.0.0

  replace github.com/forge-cms/forge => ../
  ```
- [x] Add `use ./forge-mcp` to `go.work`
- [x] Create `forge-mcp/mcp.go` with package `forgemcp`:
  - Package comment: "Package forgemcp implements an MCP (Model Context Protocol)
    server for Forge applications. It exposes content modules registered with
    forge.MCP(...) as MCP resources and tools, enabling AI assistants to query
    and manage content through a structured protocol."
  - `Server` struct:
    ```go
    type Server struct {
        modules []forge.MCPModule
        auth    forge.AuthFunc // nil = stdio (no auth); set for SSE
    }
    ```
  - `New(app *forge.App) *Server` constructor: calls `app.MCPModules()`, stores slice
  - `Server.moduleByPrefix(prefix string) (forge.MCPModule, bool)` — lookup helper
  - `Server.allResources(ctx forge.Context) []mcpResource` (internal) — iterates
    modules with MCPRead, calls MCPList(ctx, forge.Published), builds resource list
  - `mcpResource` internal type: `{URI, Name, Description, MimeType string}`
  - `mcpToolDefs(m forge.MCPModule) []mcpTool` (internal) — for modules with
    MCPWrite, builds tool definitions from MCPMeta + MCPSchema
  - `mcpTool` internal type: `{Name, Description string, InputSchema map[string]any}`
  - `inputSchema(fields []forge.MCPField) map[string]any` — converts `[]MCPField`
    to a JSON Schema object (`"type": "object"`, `"properties"`, `"required"`)

#### Verification — Step 1

- [x] `go build ./...` — no errors (forge core + forge-mcp)
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestMCP ./...` — all green
- [x] Amendment A49 documented in `DECISIONS.md` (index row + body)
- [x] `BACKLOG.md` — step table row and summary checkbox updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 10.B — Read path (depends on Step 1)

### Step 2 — `forge-mcp/resource.go`

**Depends on:** Step 1 (`forge-mcp/mcp.go`, Amendment A49)
**Decisions:** Amendment A49, Decision 14 (content lifecycle)
**Files:** `forge-mcp/resource.go` (new), extended `forge-mcp/mcp_test.go`

#### 2.1 — Internal types (`resource.go`)

- [x] Define `resourceContent` internal type:
  ```go
  type resourceContent struct {
      URI      string `json:"uri"`
      MimeType string `json:"mimeType"`
      Text     string `json:"text"` // JSON-encoded content item
  }
  ```
- [x] Define `resourceTemplate` internal type (for `resources/templates/list`):
  ```go
  type resourceTemplate struct {
      URITemplate string `json:"uriTemplate"` // e.g. "forge://posts/{slug}"
      Name        string `json:"name"`
      Description string `json:"description"`
      MimeType    string `json:"mimeType"`
  }
  ```

#### 2.2 — `handleResourcesList` (`resource.go`)

- [x] `func (s *Server) handleResourcesList(ctx forge.Context) any`:
  - Delegates to `allResources(ctx)` (already in `mcp.go`); wraps in
    `map[string]any{"resources": ...}`
  - `allResources` already calls `MCPList(ctx, forge.Published)` — no
    additional lifecycle logic needed here

#### 2.3 — `handleResourcesTemplatesList` (`resource.go`)

- [x] `func (s *Server) handleResourcesTemplatesList() any`:
  - Iterate modules with `MCPRead`; build one `resourceTemplate` per module
  - `URITemplate: "forge:/" + prefix + "/{slug}"`
  - Return `map[string]any{"resourceTemplates": templates}`

#### 2.4 — `parseResourceURI` helper (`resource.go`)

- [x] `func (s *Server) parseResourceURI(uri string) (forge.MCPModule, string, bool)`:
  - For each MCPRead module: try `strings.CutPrefix(uri, "forge:/"+prefix+"/")`
  - If ok and slug non-empty and slug contains no `/` → return `(m, slug, true)`
  - Returns `(nil, "", false)` for bad URI, unknown prefix, or extra path segments

#### 2.5 — `handleResourcesRead` (`resource.go`)

- [x] `func (s *Server) handleResourcesRead(ctx forge.Context, params json.RawMessage) (any, *jsonRPCError)`:
  - Unmarshal params → `struct{ URI string \`json:"uri"\` }`
  - Call `parseResourceURI` → `-32001` if not found
  - Call `m.MCPGet(ctx, slug)` → `-32001` on error
  - Type-assert to `interface{ GetStatus() forge.Status }` → `-32001` if
    status ≠ `forge.Published`
  - `json.Marshal(item)` → return
    `map[string]any{"contents": []resourceContent{{URI, MimeType, Text}}}`

#### 2.6 — Dispatch hook (`forge-mcp/mcp.go`, single line)

- [x] Add `handleResourceMethod` to `resource.go`:
  ```go
  func (s *Server) handleResourceMethod(ctx forge.Context, req jsonRPCRequest) (jsonRPCResponse, bool)
  ```
  Handles `resources/list`, `resources/templates/list`, `resources/read`; returns
  `(response, true)` if matched, `(zero, false)` otherwise.
- [x] In `handle` (`mcp.go`), insert before the `default` case:
  ```go
  if r, ok := s.handleResourceMethod(ctx, req); ok { return r }
  ```

#### 2.7 — Flag E: `MCPGet` godoc (`mcp.go` forge core)

- [x] Add to the `MCPGet` comment in the `MCPModule` interface:
  "MCPGet does not filter by lifecycle status — it returns the item
  regardless of status. Callers are responsible for enforcing lifecycle
  rules (e.g. forge-mcp checks Published before including in a response)."

#### 2.8 — Fixture upgrade + tests (`forge-mcp/mcp_test.go`)

- [x] Rename `testPost` → `testMCPPost`; upgrade struct:
  ```go
  type testMCPPost struct {
      forge.Node
      Title  string `forge:"required,min=3"`
      Body   string `forge:"required,min=10"`
      Rating int
      Tags   string `json:"tags"`
  }
  ```
- [x] Update all existing references (`newTestApp`, `TestNewServer`) from
  `testPost` / `*testPost` → `testMCPPost` / `*testMCPPost`
- [x] Add `seedPost(repo, slug, status, title, body)` internal helper that
  creates and saves a `testMCPPost` with the given fields via `repo.Save`
- [x] `TestMCPResourcesList` — 1 Published + 1 Draft; assert only Published
  URI appears; assert URI format = `forge://posts/{slug}`
- [x] `TestMCPResourcesRead_published` — read Published item by URI; assert
  `contents[0].text` round-trips to correct `title` value (JSON round-trip
  pattern per Flag D)
- [x] `TestMCPResourcesRead_draft` — read Draft by URI; assert `-32001` error
- [x] `TestMCPResourcesTemplatesList` — 2 MCPRead modules; assert 2 templates
  with correct `uriTemplate` format

#### Verification — Step 2

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestMCPResources ./...` — all 4 new tests green
- [x] Full `go test ./...` — no regressions
- [x] `BACKLOG.md` — step row updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 10.C — Write path (depends on Step 2)

### Step 3 — `forge-mcp/tool.go`

**Depends on:** Step 2
**Decisions:** Amendment A49, Decision 4 (role hierarchy), Decision 13 (validation)
**Files:** `forge-mcp/tool.go` (new), extended `forge-mcp/mcp_test.go`

#### 3.1 — Tool naming convention

Tools are named `{operation}_{type_snake}`, e.g. `create_blog_post`,
`publish_blog_post`. Type name is derived from `MCPMeta.TypeName` converted to
`lower_snake_case`.

- [ ] `toolName(operation, typeName string) string` helper:
  convert camelCase TypeName to snake_case (e.g. `BlogPost` → `blog_post`,
  `MCPPost` → `mcp_post`; consecutive uppercase letters = single word),
  prepend operation

#### 3.2 — `tools/list` handler

- [ ] `Server.handleToolsList() any`:
  - For each module with `MCPWrite` in its operations:
    - Emit tools: `create_{type}`, `update_{type}`, `publish_{type}`,
      `schedule_{type}`, `archive_{type}`, `delete_{type}`
    - For `create_{type}`: inputSchema from `inputSchema(m.MCPSchema())`
    - For `update_{type}`: inputSchema adds a required `"slug"` string field
      plus all schema fields (none required — partial update semantics)
    - For `publish_{type}`, `archive_{type}`, `delete_{type}`:
      inputSchema = `{"type":"object","properties":{"slug":{"type":"string"}},"required":["slug"]}`
    - For `schedule_{type}`: inputSchema adds required `"slug"` and required
      `"scheduled_at"` (type `"string"`, format `"date-time"`)
  - Return `map[string]any{"tools": tools}`

#### 3.3 — `tools/call` dispatcher

- [ ] `Server.handleToolsCall(ctx forge.Context, name string, args map[string]any) (any, error)`:
  - Split `name` on first `_` to get `operation` and `typeSnake`
  - Find module where `snakeCase(m.MCPMeta().TypeName) == typeSnake`;
    return MCP error -32602 (invalid params) if not found
  - Route to operation:
    - `"create"` → `m.MCPCreate(ctx, args)` → return serialised item
    - `"update"` → extract `slug` from args; call `m.MCPUpdate(ctx, slug, args)`
    - `"publish"` → extract `slug`; call `m.MCPPublish(ctx, slug)`
    - `"schedule"` → extract `slug` and `scheduled_at` (parse RFC3339); call
      `m.MCPSchedule(ctx, slug, t)`
    - `"archive"` → `m.MCPArchive(ctx, slug)`
    - `"delete"` → `m.MCPDelete(ctx, slug)`
  - On forge validation error (`forge.ErrValidation`): return MCP error -32602
    with the validation message as `"data"`
  - On forge not-found error: return MCP error -32001

#### 3.4 — Role enforcement

- [ ] `Server.authorise(ctx forge.Context, op MCPOperation) error`:
  - `MCPRead` requires at minimum `forge.Guest` (read is public)
  - `MCPWrite` requires at minimum `forge.Author`
  - Check via `forge.HasRole(ctx.User().Role, requiredRole)` — same check as
    the HTTP write handler
  - Return `forge.ErrForbidden` if insufficient

#### 3.5 — Tests

- [ ] `TestMCPToolsList` — register a module with MCPWrite; assert all 6 tools
  appear with correct names and input schemas
- [ ] `TestMCPToolsCall_create` — call `create_{type}` with valid fields; assert
  item created in repo with Draft status
- [ ] `TestMCPToolsCall_create_validation` — call `create_{type}` missing a
  required field; assert MCP error -32602 returned
- [ ] `TestMCPToolsCall_publish` — create a Draft item; call `publish_{type}`;
  assert status is Published and PublishedAt is non-zero
- [ ] `TestMCPToolsCall_schedule` — call `schedule_{type}` with a future time;
  assert status is Scheduled and ScheduledAt is set
- [ ] `TestMCPToolsCall_archive` — publish then archive; assert status is Archived
- [ ] `TestMCPToolsCall_delete` — create then delete; assert FindBySlug returns
  not-found error
- [ ] `TestMCPToolsCall_forbidden` — call `create_{type}` with a Guest context;
  assert MCP error is returned

#### Verification — Step 3

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestMCPTools ./...` — all green
- [ ] `BACKLOG.md` — updated
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 10.D — Transport (depends on Step 3)

### Step 4 — `forge-mcp/transport.go`

**Depends on:** Step 3
**Decisions:** Amendment A49
**Files:** `forge-mcp/transport.go` (new), extended `forge-mcp/mcp_test.go`

The MCP protocol uses JSON-RPC 2.0 messages. Two transports are provided:
- **stdio** — newline-delimited JSON on stdin/stdout (for local AI tools such
  as Claude Desktop and Cursor)
- **SSE** — Server-Sent Events over HTTP for remote authenticated connections;
  client POSTs requests to `/mcp/message`, server pushes responses as SSE

#### 4.1 — JSON-RPC types

- [ ] Define internal types in `transport.go`:
  ```go
  type jsonRPCRequest struct {
      JSONRPC string          `json:"jsonrpc"`
      ID      any             `json:"id"`
      Method  string          `json:"method"`
      Params  json.RawMessage `json:"params,omitempty"`
  }
  type jsonRPCResponse struct {
      JSONRPC string `json:"jsonrpc"`
      ID      any    `json:"id,omitempty"`
      Result  any    `json:"result,omitempty"`
      Error   *jsonRPCError `json:"error,omitempty"`
  }
  type jsonRPCError struct {
      Code    int    `json:"code"`
      Message string `json:"message"`
      Data    any    `json:"data,omitempty"`
  }
  ```
- [ ] `Server.handle(ctx forge.Context, req jsonRPCRequest) jsonRPCResponse`:
  dispatches to `initialize`, `resources/list`, `resources/templates/list`,
  `resources/read`, `tools/list`, `tools/call`; maps errors to JSON-RPC error
  codes; returns `jsonrpc: "2.0"` on all responses

#### 4.2 — `initialize` handler

- [ ] `Server.handleInitialize() any`:
  returns `map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "forge-mcp", "version": "1.0.0"}, "capabilities": map[string]any{"resources": map[string]any{"subscribe": false, "listChanged": false}, "tools": map[string]any{"listChanged": false}}}`

#### 4.3 — stdio transport

- [ ] `Server.ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error`:
  - Creates a `forge.NewBackgroundContext` (background context uses
    `forge.Admin` role for stdio — the process runs locally and the operator
    is trusted)
  - Reads newline-delimited JSON from `in` in a loop
  - Parses each line as `jsonRPCRequest`
  - Calls `s.handle(forgeCtx, req)` — result or error
  - Encodes `jsonRPCResponse` to `out` as a single JSON line + `\n`
  - Returns when `ctx` is cancelled or `in` is closed
  - Production usage: `server.ServeStdio(ctx, os.Stdin, os.Stdout)`

#### 4.4 — SSE transport

- [ ] `Server.Handler() http.Handler`:
  - Returns a `http.ServeMux` with two routes:
    - `GET /mcp` — SSE endpoint; upgrades to SSE, sends `event: open\ndata: {}\n\n`
      then blocks until the connection closes
    - `POST /mcp/message` — accepts a `jsonRPCRequest` as JSON body; authenticates
      the caller using `Authorization: Bearer {token}` via `forge.BearerHMAC`
      (server must be constructed with `WithSecret(secret string)` option or read
      from `forge.Config`); constructs a `forge.Context` with the authenticated
      user's role; calls `s.handle(ctx, req)`; returns `jsonRPCResponse` as JSON
  - SSE response headers: `Content-Type: text/event-stream`,
    `Cache-Control: no-cache`, `Connection: keep-alive`
- [ ] `WithSecret(secret string) ServerOption` constructor option:
  sets the HMAC secret used by the SSE transport to verify Bearer tokens

#### 4.5 — Tests

- [ ] `TestMCPServeStdio_roundtrip` — construct `Server`; create two
  `io.Pipe` pairs (`inR/inW` for input, `outR/outW` for output); call
  `server.ServeStdio(ctx, inR, outW)` in a goroutine; write
  `initialize` JSON line to `inW`; read response from `outR`; assert
  valid `protocolVersion` in result; cancel ctx to shut down
- [ ] `TestMCPServeStdio_resourcesList` — seed a published item, send
  `resources/list` over stdio, assert item appears in response
- [ ] `TestMCPHandler_initialize` — POST `initialize` to `/mcp/message`;
  assert 200 + correct result
- [ ] `TestMCPHandler_toolsCall_unauthenticated` — POST without Bearer token;
  assert 401 or JSON-RPC error

#### Verification — Step 4

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestMCPServe ./...` — all green
- [ ] `BACKLOG.md` — updated
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Layer 10.E — Documentation (depends on Step 4)

### Step 5 — `forge-mcp/README.md`

**Depends on:** Step 4
**Decisions:** Decision 19
**Files:** `forge-mcp/README.md` (new), extended `forge-mcp/mcp_test.go`
(integration group), `integration_full_test.go` (forge core cross-milestone
group G22)

Note on `integration_full_test.go`: `forge` core cannot import `forge-mcp`,
so G22 exercises the **forge core side of Amendment A49** — specifically that
`Module[T].MCPSchema()`, `App.MCPModules()`, and the `MCPModule` interface all
behave correctly in isolation. The end-to-end MCP communication tests live in
`forge-mcp/mcp_test.go`.

#### 5.1 — `forge-mcp/README.md`

- [ ] Quick start: install, register a module with `forge.MCP(forge.MCPWrite)`,
  create an `MCPServer`, call `ServeStdio`
- [ ] Claude Desktop configuration example (`claude_desktop_config.json`)
- [ ] Cursor MCP configuration example
- [ ] SSE remote configuration with Bearer token
- [ ] Table: which operations are available under MCPRead vs MCPWrite
- [ ] Note on lifecycle enforcement: MCPRead exposes only Published content;
  MCPWrite respects the full Draft → Scheduled → Published → Archived lifecycle
- [ ] Note on role enforcement: SSE transport authenticates using the same
  `forge.BearerHMAC` tokens as the REST API
- [ ] Note on zero-dependency design: no external MCP SDK

#### 5.2 — Example in README verified by example_test.go

- [ ] Add `ExampleMCPServer` to `forge-mcp/mcp_test.go` (compile test):
  ```go
  func ExampleNew() {
      app := forge.New(forge.MustConfig(forge.Config{...}))
      // app.Content(..., forge.MCP(forge.MCPWrite))
      srv := forgemcp.New(app)
      _ = srv
      // Output:
  }
  ```

#### 5.3 — `integration_full_test.go` G22 (forge core)

Cross-milestone group exercising A49 on the forge core side:

- [ ] `TestFull_G22_MCPModuleInterface` — register two modules, one with
  `forge.MCP(forge.MCPRead)` and one with `forge.MCP(forge.MCPWrite)`; call
  `app.MCPModules()`; assert two modules returned; verify `MCPMeta()` returns
  correct Prefix and Operations for each; verify `MCPSchema()` returns at least
  one field with correct Required flag from a `forge:"required"` struct tag
- [ ] `TestFull_G22_MCPCreatePublishLifecycle` — use `MCPCreate` + `MCPPublish`
  directly (no transport) through the `MCPModule` interface; assert item
  transitions correctly and appears in `MCPList` filtered by Published

#### 5.4 — ARCHITECTURE.md update

- [ ] Add `forge-mcp/` entry to the package structure section
- [ ] Document `MCPModule` interface location (`mcp.go`) and implementor
  (`module.go`)
- [ ] Document `App.MCPModules()` addition
- [ ] Update the "planned files" → "implemented files" table for M10

#### 5.5 — README.md badge

- [ ] Update the MCP section badge in the root `README.md` from
  `🔲 Coming in Milestone 10` to `✅ Available`

#### Verification — Step 5

- [ ] `go build ./...` — no errors (all modules)
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test ./...` — all green, including G22 groups and forge-mcp tests
- [ ] `forge-mcp/README.md` — quick start instructions are accurate
- [ ] Root `README.md` — MCP badge updated
- [ ] `ARCHITECTURE.md` — reflects forge-mcp module
- [ ] `BACKLOG.md` — M10 marked ✅ Done; top-level Progress table updated
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 10

- [ ] `forge-mcp` compiles with zero external dependencies
- [ ] All five steps verified green
- [ ] An AI assistant can connect to a local Forge app via stdio and call
  `tools/call` to create and publish a content item
- [ ] An AI assistant can connect to a remote Forge app via SSE using a Bearer
  token and perform CRUD operations with role enforcement
- [ ] `integration_full_test.go` G22 groups pass
- [ ] Root `README.md` MCP section badge updated to ✅ Available
- [ ] `ARCHITECTURE.md` reflects the forge-mcp module
- [ ] `CHANGELOG.md` has a `[1.1.0]` entry in forge core (new exported
  types: `MCPModule`, `MCPMeta`, `MCPField`; no breaking changes) and a
  `[1.0.0]` entry in `forge-mcp/CHANGELOG.md` for the initial release
- [ ] `go test ./...` is green across all workspace modules
