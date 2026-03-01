# Forge — Architecture

This document describes the internal structure of Forge: how the packages
are organised, how a request flows through the system, which interfaces
are stable API contracts, and the dependency rules between packages.

Read DECISIONS.md first. This document explains *how* — DECISIONS.md explains *why*.

---

## Package structure

```
github.com/forge-cms/forge/
│
├── forge.go          App, Config, New(), MustConfig()
├── node.go           Node, Status, lifecycle constants
├── module.go         Module[T], Option type, routing, lifecycle enforcement
├── context.go        Context interface, contextImpl, ContextFrom(), NewTestContext()
├── auth.go           BearerHMAC, CookieSession, BasicAuth, User, SignToken
├── roles.go          Role type, hierarchy, HasRole(), Is(), built-in constants
├── head.go           Head struct, Image, Excerpt(), URL(), Crumbs(), Crumb()
├── errors.go         Error interface, sentinel errors, WriteError(), ValidationError
├── storage.go        Query[T], QueryOne[T], Repository[T], MemoryRepo[T], ListOptions
├── signals.go        Signal type, On() option, signal dispatch
├── middleware.go     RequestLogger, Recoverer, SecurityHeaders, CORS, RateLimit...
├── cookies.go        Cookie struct, categories, SetCookie, ConsentFor, manifest
├── redirects.go      RedirectEntry, redirect table, chain collapse
├── sitemap.go        SitemapConfig, generation, debounce goroutine
├── rss.go            FeedConfig, Atom/RSS generation
├── ai.go             AIDoc, LLMsTxt, Markdownable interface, content negotiation
├── social.go         OpenGraph, TwitterCard, LinkedIn meta tag rendering
├── templates.go      TemplateData[T], template helpers, forge:head partial
├── scheduler.go      Adaptive ticker, scheduled publishing loop
└── mcp.go            MCP() no-op option (v1), MCPRead/MCPWrite constants
```

All files are in a single package: `forge`. There are no sub-packages.
This is intentional — it eliminates circular import issues and keeps
the API surface in one place. The file names are the organisation.

---

## Request lifecycle

A request arriving at a Forge app passes through these layers in order:

```
HTTP Request
    │
    ▼
┌─────────────────────────────────┐
│  Global middleware chain        │  RequestLogger, Recoverer, SecurityHeaders,
│  (app.Use order)                │  CORS, MaxBodySize, RateLimit
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  net/http ServeMux router       │  Pattern matching, path parameters
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  forge.Context creation         │  RequestID (UUID v7), User (from auth), Locale
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Auth resolution                │  BearerHMAC → CookieSession → Guest
│                                 │  First match wins
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Redirect table lookup          │  Only on potential 404 — not on every request
│                                 │  Match → 301 or 410 immediately
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Module handler dispatch        │
│  (list / show / create /        │
│   update / delete)              │
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Lifecycle enforcement          │  Draft/Scheduled/Archived + Guest → 404
│                                 │  Draft + Author (own content) → allowed
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Role check                     │  Insufficient role → 403
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Cache check (read operations)  │  LRU hit → return cached response
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  BeforeX signals                │  Synchronous. Can abort with error.
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Storage operation              │  Caller's Repository[T] implementation
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  AfterX signals                 │  Asynchronous (goroutine). Cannot abort.
│                                 │  Sitemap/feed regeneration fires here.
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Content negotiation            │  Accept: application/json → JSON
│                                 │  Accept: text/html → template
│                                 │  Accept: text/markdown → Markdown()
│                                 │  Accept: text/plain → stripped text
└────────────────┬────────────────┘
                 │
    ▼
HTTP Response  (X-Request-ID always set)
```

---

## Stable interfaces (public API contracts)

These interfaces are the extension points for users of Forge.
They must not change in v1.x without a deprecation cycle.

```go
// Headable — implement to control SEO, social, and AI metadata
type Headable interface {
    Head() Head
}

// Markdownable — implement to enable text/markdown content negotiation
type Markdownable interface {
    Markdown() string
}

// Validatable — implement to run custom validation after struct-tag validation
type Validatable interface {
    Validate() error
}

// Repository[T] — implement to provide a custom storage backend
type Repository[T any] interface {
    FindByID(ctx context.Context, id string) (T, error)
    FindBySlug(ctx context.Context, slug string) (T, error)
    FindAll(ctx context.Context, opts ListOptions) ([]T, error)
    Save(ctx context.Context, node T) error
    Delete(ctx context.Context, id string) error
}

// Context — the request context passed to all hooks and handlers
// Implemented as an interface (not a struct) to enable testing without HTTP
type Context interface {
    context.Context
    User() User
    Locale() string
    SiteName() string
    RequestID() string
    Request() *http.Request
    Response() http.ResponseWriter
}

// Error — all Forge errors implement this
type Error interface {
    error
    HTTPStatus() int
    Code() string
    Public() string
}

// AIDocSummary — optional; implement for a custom AIDoc summary field
type AIDocSummary interface {
    AIDocSummary() string
}

// SitemapPriority — optional; implement to control per-item sitemap priority
type SitemapPriority interface {
    SitemapPriority() float64
}
```

---

## Internal dependency rules

To prevent circular imports and keep the package coherent, these rules apply:

```
errors.go       — no internal dependencies (foundation layer)
node.go         — depends on: errors
roles.go        — no internal dependencies
context.go      — depends on: roles
auth.go         — depends on: errors, roles, context
signals.go      — depends on: context, errors
storage.go      — depends on: node, errors
module.go       — depends on: node, context, auth, signals, storage, errors
head.go         — no internal dependencies
cookies.go      — depends on: errors
redirects.go    — depends on: node, errors
sitemap.go      — depends on: node, signals
rss.go          — depends on: node, signals, head
ai.go           — depends on: node, head
social.go       — depends on: head
templates.go    — depends on: head, context, node
middleware.go   — depends on: errors, context
scheduler.go    — depends on: node, signals, storage
mcp.go          — depends on: (nothing — no-op in v1)
forge.go        — depends on: all of the above
```

The dependency graph has no cycles. `errors.go` and `roles.go` are the only
true foundation files — everything else can depend on them freely.

---

## forge.Node embedding

Every content type embeds `forge.Node`. Embedding (not composition) is required
because Forge uses reflection to access Node fields directly:

```go
// forge reads these fields by name via reflection — do not rename them
type Node struct {
    ID          string
    Slug        string
    Status      Status
    PublishedAt time.Time
    ScheduledAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

The reflection access is cached on first use via `sync.Map` — field lookup
is O(1) after the first request for any given type.

---

## Signal dispatch

Signals are dispatched synchronously (BeforeX) or asynchronously (AfterX).

```
BeforeCreate / BeforeUpdate / BeforeDelete
    → run in request goroutine
    → return error → operation aborted, error returned to client
    → panic → recovered, logged, 500 returned

AfterCreate / AfterUpdate / AfterDelete / AfterPublish / AfterUnpublish / AfterArchive
    → run in new goroutine (go dispatch(...))
    → errors logged, never returned to client
    → panic recovered and logged

SitemapRegenerate
    → fired by AfterPublish, AfterUnpublish, AfterArchive, AfterDelete
    → debounced 2 seconds — coalesces bursts of changes
    → runs sitemap + feed regeneration
```

---

## Scheduler

The scheduled publishing loop runs as a goroutine started by `app.Run()`.

```
On startup:
    query storage for the next scheduled item (MIN(scheduled_at) WHERE status = 'scheduled')
    if found: set timer to time.Until(scheduled_at)
    if not found: set fallback ticker to 60 seconds

On tick:
    query all items WHERE status = 'scheduled' AND scheduled_at <= now
    for each: set status = published, set published_at = now
              fire AfterPublish signal (async)
    recalculate next scheduled item → reset timer

On shutdown:
    wait for in-progress tick to complete (max 5 seconds)
    then exit
```

---

## Content negotiation

A single endpoint responds differently based on the `Accept` header:

```
Accept: application/json     → JSON response (default for API clients)
Accept: text/html            → rendered template
Accept: text/markdown        → calls Markdown() if implemented, else 406
Accept: text/plain           → stripped plaintext version
```

The `Accept` header check uses pre-compiled content-type matching per module,
not string comparison on every request.

---

## Redirect table

The redirect table is a flat key-value store keyed by `FromPath`.
It lives alongside the content — in the same database, same transaction.

Redirect lookups happen only on requests that would otherwise produce a 404.
The resolution order:

```
1. Try to find a published node with this slug in this module
2. If not found: check redirect table for this path
3. If found in redirect table: serve 301 or 410
4. If not found anywhere: serve 404
```

This means redirect lookup adds zero overhead to successful requests.

---

## Cache

The LRU cache is per-module, not global. Each `forge.Cache(ttl)` call
creates an independent cache for that module.

```
Cache key:   "{method}:{path}:{accept-header}"
Cache value: serialised HTTP response (status + headers + body)
Max entries: 1000 per module (configurable)
Eviction:    LRU when max entries reached
TTL:         hard expiry per entry
Invalidation: AfterCreate / AfterUpdate / AfterDelete signals clear the module cache
```

`X-Cache: HIT` and `X-Cache: MISS` headers are always set.

---

## Storage and the forge.DB interface

Forge defines a minimal `forge.DB` interface internally:

```go
type DB interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

This interface is satisfied by:
- `*sql.DB` (standard library) — zero additional dependency
- `*sql.Tx` — transactions work automatically
- `forgepgx.Wrap(pool)` — native pgx pool adapter (~2.5× faster for PostgreSQL)
- Any custom type that implements the three methods

`forge.Query[T]` and `forge.QueryOne[T]` accept `forge.DB`, not `*sql.DB`.
This means switching drivers requires changing exactly one value in `forge.Config`.

The `forge-pgx` adapter lives at `github.com/forge-cms/forge-pgx` — a separate
module. It imports both `forge` and `pgx/v5`. Forge core never imports pgx.

---

## Template data shape

```go
// show handler
TemplateData[T] {
    Content  T             // the single content item
    Head     forge.Head    // from item.Head() merged with module HeadFunc
    User     forge.User    // current user — zero value if Guest
    Request  *http.Request
}

// list handler
TemplateData[[]T] {
    Content  []T           // slice of items
    Head     forge.Head    // from module HeadFunc
    User     forge.User
    Request  *http.Request
}
```

---

## Testing

Every public interface has a test double:

```go
// In-memory repository — no database needed
repo := forge.NewMemoryRepo[*BlogPost]()

// Test context — no HTTP needed
ctx := forge.NewTestContext(forge.User{
    ID:    "test-user",
    Roles: []forge.Role{forge.Editor},
})

// Test app — full app without listening on a port
app := forge.New(forge.Config{
    BaseURL: "http://localhost",
    Secret:  []byte("test-secret-32-bytes-minimum-xx"),
    Env:     forge.Test,
})
handler := app.Handler() // returns http.Handler, does not call app.Run()
```

Use `net/http/httptest` with `app.Handler()` for integration tests.
Use `forge.NewTestContext()` with direct hook calls for unit tests.
