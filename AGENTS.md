# Forge — AI Agent Guide

This document is for AI assistants working with Forge — either building
applications or consuming a running site via MCP.

---

## For AI coding agents

You are helping a developer build a Forge application.

### Start here

Read `example/blog/main.go` first. It is the minimal complete pattern:
content type, module options, app wiring, graceful shutdown. Copy and
rename it. Everything else is additive.

### Adding a content type

```go
type Post struct {
    forge.Node
    Title string `forge:"required,min=3" db:"title"`
    Body  string `forge:"required"       db:"body"`
}
```

Rules:

- Always embed `forge.Node` — never compose it
- Use `forge:"required"` and `forge:"min=N"` for validation
- Use `db:"column_name"` for `SQLRepo` column mapping
- Avoid SQLite reserved keywords as column names (`order`, `group`, etc.)
  — use `db:"sort_order"` instead

### Wiring a module

This is the minimal wiring pattern from `example/blog/main.go`:

```go
repo := forge.NewMemoryRepo[*Post]()

m := forge.NewModule((*Post)(nil),
    forge.At("/posts"),
    forge.Repo(repo),
)

app := forge.New(forge.MustConfig(forge.Config{
    BaseURL: "http://localhost:8080",
    Secret:  []byte("change-this-secret-in-production"),
}))

app.Content(m)

if err := app.Run(":8080"); err != nil {
    log.Fatal(err)
}
```

### Adding MCP support

Add `forge.MCP(forge.MCPRead, forge.MCPWrite)` to the module options:

```go
forge.NewModule((*Post)(nil),
    forge.At("/posts"),
    forge.Repo(repo),
    forge.MCP(forge.MCPRead, forge.MCPWrite),
)
```

See `forge-mcp/README.md` for connection setup (Claude Desktop, Cursor, SSE).

### Key rules for code generation

- Zero third-party dependencies in the `forge` core package
- `forge.Context` is an interface, not a struct
- `forge.DB` is an interface, not `*sql.DB`
- All errors must implement `forge.Error` — never raw `errors.New`
- Read `ERROR_HANDLING.md` before writing any error-handling code

---

## For AI consuming agents

You are connected to a running Forge site via MCP.

### What you can do

Two operations are available depending on how the site owner configured
the modules:

- **MCPRead** — list and read published content
- **MCPWrite** — create, update, publish, schedule, archive, delete content

### Lifecycle rules

Content follows `Draft → Scheduled → Published → Archived`. You cannot
bypass this. Publishing requires an explicit `publish` tool call after
`create`.

### Role enforcement

Write operations require `Author` role or higher. The Bearer token you
were given determines your role. If an operation returns `forbidden`,
you do not have sufficient role — do not retry.

### Available tools (MCPWrite)

For each registered content type, these tools are available:

- `create_{type}` — creates a Draft
- `update_{type}` — partial update (absent fields preserved)
- `publish_{type}` — transitions to Published
- `schedule_{type}` — schedules for future publication (RFC3339 datetime)
- `archive_{type}` — transitions to Archived
- `delete_{type}` — permanent deletion

### Reading content

- `resources/list` — all Published items across all MCPRead modules
- `resources/read` — single item by URI (`forge://{prefix}/{slug}`)

### Connection setup

See `forge-mcp/README.md` for Claude Desktop, Cursor, and SSE configuration.
