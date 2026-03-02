# Forge — Architecture

This document describes the internal structure of Forge: how the packages
are organised, how a request flows through the system, which interfaces
are stable API contracts, and the dependency rules between packages.

Read DECISIONS.md first. This document explains *how* — DECISIONS.md explains *why*.

---

## Changelog

| Date | Change |
|------|--------|
| 2026-03-01 | Initial architecture document drafted (Milestone 1 planning) |
| 2026-03-01 | Updated to reflect Milestone 1 completion: corrected request lifecycle order, added `CacheStore`, `CSRF`, `TrustedProxy`, updated `SignToken` signature, added `ListOptions.Status`, fixed `Markdownable` location to `module.go`, marked future-milestone files as planned |
| 2026-03-02 | Milestone renumbering: M2 split into App Bootstrap (M2) and SEO & Head (M3); all subsequent milestones shifted +1 |
| 2026-03-02 | Milestone 2 Step 1: `forge.go` implemented — `Config`, `MustConfig`, `New`, `App` (`Use`/`Content`/`Handle`/`Run`/`Handler`), `Registrator` interface, graceful shutdown |
| 2026-03-02 | Milestone 2 Step P1: `forge-pgx` module implemented — `Wrap(pool)` native pgx adapter satisfying `forge.DB` |

---

## Package structure

All files are in a single package: `forge`. There are no sub-packages.
This is intentional — it eliminates circular import issues and keeps
the API surface in one place. The file names are the organisation.

### Implemented (Milestone 1 + Milestone 2)

```
github.com/forge-cms/forge/
│
├── errors.go         Error interface, sentinel errors, WriteError(), ValidationError
├── roles.go          Role type, hierarchy, HasRole(), IsRole(), built-in constants, Option interface
├── mcp.go            MCP() no-op option (v1), MCPRead/MCPWrite constants
├── node.go           Node, Status, lifecycle constants, NewID(), GenerateSlug(), UniqueSlug(), ValidateStruct()
├── context.go        Context interface, contextImpl, ContextFrom(), NewTestContext(), User, GuestUser
├── signals.go        Signal type, On[T]() option, dispatchBefore(), dispatchAfter(), debouncer
├── storage.go        DB interface, Query[T], QueryOne[T], Repository[T], MemoryRepo[T], ListOptions
├── auth.go           AuthFunc interface, BearerHMAC, CookieSession, BasicAuth, AnyAuth, SignToken
├── middleware.go     RequestLogger, Recoverer, SecurityHeaders, CORS, MaxBodySize,
│                     RateLimit, TrustedProxy, InMemoryCache, CacheStore, CSRF, Chain
├── module.go         Module[T], NewModule, Register, Markdownable, At, Cache, Auth,
│                     Middleware, Repo, On options
└── forge.go          Config, MustConfig, New, App (Use/Content/Handle/Run/Handler),
                      Registrator, httpsRedirect, graceful shutdown via SIGINT/SIGTERM

github.com/forge-cms/forge-pgx/  (separate module: ./forge-pgx/)
└── pgx.go            Wrap(*pgxpool.Pool) forge.DB — native pgx adapter
```

### Planned (future milestones)

```
├── head.go           Head struct, SEO/social metadata                      (Milestone 3)
├── templates.go      TemplateData[T], template helpers, forge:head partial (Milestone 4)
├── cookies.go        Cookie struct, categories, SetCookie, ConsentFor      (Milestone 6)
├── redirects.go      RedirectEntry, redirect table, chain collapse         (Milestone 7)
├── sitemap.go        SitemapConfig, generation, debounce goroutine         (Milestone 3)
├── rss.go            FeedConfig, Atom/RSS generation                       (Milestone 5)
├── ai.go             AIDoc, LLMsTxt, content negotiation helpers           (Milestone 5)
├── social.go         OpenGraph, TwitterCard, LinkedIn meta rendering       (Milestone 5)
└── scheduler.go      Adaptive ticker, scheduled publishing loop            (Milestone 8)
```

---

## Request lifecycle

A request arriving at a Forge app passes through these layers in order.
**Read (GET) and write (POST/PUT/DELETE) paths diverge after context creation.**

```
HTTP Request
    │
    ▼
┌─────────────────────────────────┐
│  Global middleware chain        │  RequestLogger, Recoverer, SecurityHeaders,
│  (app.Use order, planned)       │  CORS, MaxBodySize, RateLimit, CSRF
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  net/http ServeMux router       │  Go 1.22 pattern matching, path parameters
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  forge.Context creation         │  ContextFrom(w, r)
│                                 │  Sets X-Request-ID (UUID v7 if absent)
│                                 │  Extracts User resolved by auth middleware
└────────────────┬────────────────┘
                 │
    ▼ GET / read only
┌─────────────────────────────────┐
│  Cache check                    │  forge.Cache(ttl) per-module LRU
│                                 │  HIT → write X-Cache: HIT, return immediately
│                                 │  MISS → continue (X-Cache: MISS set on response)
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Role check                     │  ctx.User().HasRole(required)
│                                 │  Insufficient role → 403
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Storage fetch                  │  repo.FindBySlug / repo.FindAll
│                                 │  Not found → 404
└────────────────┬────────────────┘
                 │
    ▼ GET / read only
┌─────────────────────────────────┐
│  Lifecycle enforcement          │  non-Published + Guest → 404
│                                 │  (404 intentional — do not leak draft existence)
└────────────────┬────────────────┘
                 │
    ▼ POST / PUT / DELETE only
┌─────────────────────────────────┐
│  Input decode + validation      │  json.Decode → auto-ID/Slug → RunValidation
│                                 │  Validation failure → 422
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  BeforeX signals                │  Synchronous. Can abort with error → 500.
│                                 │  BeforeCreate / BeforeUpdate / BeforeDelete
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Storage operation              │  repo.Save / repo.Delete
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  AfterX signals                 │  Asynchronous (goroutine). Cannot abort.
│                                 │  AfterCreate/Update/Delete/Publish/Unpublish/Archive
└────────────────┬────────────────┘
                 │
    ▼
┌─────────────────────────────────┐
│  Content negotiation            │  application/json → JSON (default)
│                                 │  text/html       → 406 until Milestone 3
│                                 │  text/markdown   → Markdown() or 406
│                                 │  text/plain      → stripped text
│                                 │  Vary: Accept always set
└────────────────┬────────────────┘
                 │
    ▼
HTTP Response  (X-Request-ID always set)
```

---

## Stable interfaces (public API contracts)

These interfaces are the extension points for users of Forge.
They must not change in v1.x without a deprecation cycle.

### Implemented (Milestone 1)

```go
// Markdownable — implement to enable text/markdown content negotiation.
// Declared in module.go.
type Markdownable interface {
    Markdown() string
}

// Validatable — implement to run custom validation after struct-tag validation
type Validatable interface {
    Validate() error
}

// AuthFunc — implement to provide a custom authentication scheme.
// Forge provides BearerHMAC, CookieSession, BasicAuth, and AnyAuth.
type AuthFunc interface {
    authenticate(*http.Request) (User, bool)
}

// Repository[T] — implement to provide a custom storage backend
type Repository[T any] interface {
    FindByID(ctx context.Context, id string) (T, error)
    FindBySlug(ctx context.Context, slug string) (T, error)
    FindAll(ctx context.Context, opts ListOptions) ([]T, error)
    Save(ctx context.Context, node T) error
    Delete(ctx context.Context, id string) error
}

// Context — the request context passed to all hooks and handlers.
// Implemented as an interface (not a struct) to enable testing without HTTP.
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

// DB — satisfied by *sql.DB, *sql.Tx, and pgx adapters
type DB interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Registrator — implemented by *Module[T]; pass to App.Content for type-safe registration
type Registrator interface {
    Register(mux *http.ServeMux)
}
```

### Key exported functions and types (Milestone 1 + Milestone 2 Step 1)

```go
// App bootstrap (forge.go)
type Config struct {
    BaseURL      string        // required: canonical site URL, e.g. "https://example.com"
    Secret       []byte        // required: min 16 bytes; used for HMAC tokens and cookies
    DB           DB            // optional: *sql.DB or forgepgx.Wrap(pool)
    HTTPS        bool          // optional: enable HTTP→HTTPS redirect
    ReadTimeout  time.Duration // optional: default 5 s
    WriteTimeout time.Duration // optional: default 10 s
    IdleTimeout  time.Duration // optional: default 120 s
}
func MustConfig(cfg Config) Config           // validates Config; panics with descriptive msg
func New(cfg Config) *App                    // creates App; applies default timeouts

func (a *App) Use(mws ...func(http.Handler) http.Handler)  // append global middleware
func (a *App) Handle(pattern string, h http.Handler)       // register raw handler
func (a *App) Content(v any, opts ...Option)               // register *Module[T] or untyped module
func (a *App) Handler() http.Handler                       // compose all routes + middleware
func (a *App) Run(addr string) error                       // listen; graceful shutdown on SIGINT/SIGTERM

// SignToken — ttl=0 means no expiry; ttl>0 embeds exp claim, rejected after expiry
func SignToken(user User, secret string, ttl time.Duration) (string, error)

// CSRF — double-submit cookie protection; wrap CookieSession-authenticated routes only
func CSRF(auth AuthFunc) func(http.Handler) http.Handler

// RateLimit — pass TrustedProxy() when running behind nginx/Caddy/CloudFlare
func RateLimit(n int, d time.Duration, opts ...Option) func(http.Handler) http.Handler
func TrustedProxy() Option

// CacheStore — exported LRU cache backing forge.Cache() and forge.InMemoryCache()
type CacheStore struct{ /* unexported */ }
func NewCacheStore(ttl time.Duration, max int) *CacheStore
func (c *CacheStore) Flush()  // invalidate all entries (called on write operations)
func (c *CacheStore) Sweep()  // remove expired entries (called by background ticker)

// ListOptions — Status filter is applied inside the repository layer
type ListOptions struct {
    Page    int
    PerPage int
    OrderBy string
    Desc    bool
    Status  []Status // nil/empty = all statuses; non-empty = exact match filter
}
```

### Planned (future milestones)

```go
// Headable — implement to control SEO, social, and AI metadata  (head.go, Milestone 3)
type Headable interface {
    Head() Head
}

// AIDocSummary — optional; custom AIDoc summary field           (ai.go, Milestone 5)
type AIDocSummary interface {
    AIDocSummary() string
}

// SitemapPriority — optional; per-item sitemap priority         (sitemap.go, Milestone 3)
type SitemapPriority interface {
    SitemapPriority() float64
}
```

---

## Internal dependency rules

To prevent circular imports and keep the package coherent, these rules apply.
Files marked *planned* do not exist yet.

```
errors.go       — no internal dependencies (foundation layer)
roles.go        — no internal dependencies (foundation layer)
mcp.go          — no internal dependencies
node.go         — depends on: errors
context.go      — depends on: roles, node
auth.go         — depends on: errors, roles, context, node
signals.go      — depends on: context, errors
storage.go      — depends on: node, errors
middleware.go   — depends on: errors, context, auth, node
module.go       — depends on: node, context, signals, storage, errors, middleware

── planned ──────────────────────────────────────────────────────────────────
head.go         — no internal dependencies                              (Milestone 3)
forge.go        — depends on: all of the above                          (Milestone 2)
templates.go    — depends on: head, context, node                       (Milestone 4)
cookies.go      — depends on: errors                                    (Milestone 6)
redirects.go    — depends on: node, errors                              (Milestone 7)
sitemap.go      — depends on: node, signals                             (Milestone 3)
rss.go          — depends on: node, signals, head                       (Milestone 5)
ai.go           — depends on: node, head                                (Milestone 5)
social.go       — depends on: head                                      (Milestone 5)
scheduler.go    — depends on: node, signals, storage                    (Milestone 8)
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

## Scheduler *(planned — Milestone 8)*

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

## Redirect table *(planned — Milestone 7)*

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

## Template data shape *(planned — Milestone 4)*

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

// Token for test requests — ttl=0 means no expiry
tok, _ := forge.SignToken(user, "test-secret", 0)

// Module integration test via httptest — no app.Run() required
repo := forge.NewMemoryRepo[*Post]()
m := forge.NewModule((*Post)(nil), forge.Repo(repo))
mux := http.NewServeMux()
m.Register(mux)
w := httptest.NewRecorder()
r := httptest.NewRequest(http.MethodGet, "/posts", nil)
mux.ServeHTTP(w, r)
```

Use `net/http/httptest` with `m.Register(mux)` for module integration tests.
Use `forge.NewTestContext()` with direct signal handler calls for unit tests.
`forge.App` / `app.Handler()` will be available from Milestone 2.
