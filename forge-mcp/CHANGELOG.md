# Changelog — forge-mcp

All notable changes to the `forge-mcp` module are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.3] — 2026-03-18

Fix `list_{type}s` response format: wrap slice in `{"items": [...]}` object.

### Fixed

- `tool.go`: `list` case in `handleToolsCall` now returns
  `map[string]any{"items": items}` instead of a raw `[]any`; a bare JSON
  array result caused MCP protocol validation errors in clients that
  interpret array-valued tool results as batch responses

---

## [1.0.2] — 2026-03-18

Admin read tools for MCPWrite modules.

### Added

- `mcp.go`: `mcpAdminReadToolDefs` generates two tools per MCPWrite module:
  `list_{type}s` (all items, optional `status` filter) and `get_{type}`
  (single item by slug); both tools return items at any lifecycle status
- `tool.go`: `authoriseEditor` role check (Editor or Admin); `moduleForAdminList`
  resolves the plural typeSnake used by `list_{type}s` tool names; `list` and
  `get` cases in `handleToolsCall`; `handleToolsList` updated to include admin
  read tools alongside write tools

---

## [1.0.1] — 2026-03-17

`inputSchema` and `inputSchemaUpdate` now emit the correct JSON Schema for
`[]string` fields (Amendment A52-2).

### Fixed

- `mcp.go`: `inputSchema` and `inputSchemaUpdate` now emit
  `{"type":"array","items":{"type":"string"}}` for fields with `Type == "array"`;
  previously emitted bare `{"type":"array"}` without an `items` declaration, and
  incorrectly applied `minLength`/`maxLength`/`enum` constraints to array fields
  (Amendment A52-2)

---

## [1.0.0] — 2026-03-17

Initial release of `forge-mcp` — MCP support for Forge apps (Milestone 10).

### Added

- `mcp.go`: `Server` struct; `New(app, opts...)` constructor; `ServerOption`
  interface; `WithSecret(secret []byte)` option; `handle` JSON-RPC dispatcher;
  `handleInitialize`; JSON-RPC wire types (`jsonRPCRequest`, `jsonRPCResponse`,
  `jsonRPCError`); `mcpTool`, `mcpResource`, `allResources`, `mcpToolDefs`,
  `inputSchema`, `inputSchemaUpdate` helpers; `hasMCPOp`, `slugOf`, `snakeCase`
  utilities
- `resource.go`: `handleResourceMethod`, `handleResourcesList`,
  `handleResourcesTemplatesList`, `handleResourcesRead`, `parseResourceURI`;
  `mcpResource`, `resourceContent`, `resourceTemplate` wire types;
  Published-only lifecycle enforcement for MCP resources/read
- `tool.go`: `handleToolMethod`, `handleToolsList`, `handleToolsCall`
  dispatcher (create/update/publish/schedule/archive/delete); `toolName`,
  `parseToolName`, `moduleForType`, `authorise`, `errorFor`, `stringArg`
  helpers; Author-level role enforcement; idempotent publish; delete response
  `{"deleted":true,"slug":...}`
- `transport.go`: `ServeStdio(ctx context.Context, in io.Reader, out io.Writer)`
  for local stdio transport (Claude Desktop, Cursor, CLI tools); `Handler()`
  returning an `http.Handler` with SSE keepalive (`GET /mcp`) and authenticated
  JSON-RPC endpoint (`POST /mcp/message`) for remote SSE transport; 1 MiB
  request body limit; HMAC Bearer token authentication via `forge.VerifyBearerToken`
