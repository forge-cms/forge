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
| 2026-03-03 | Milestone 3 Step 1: `head.go` implemented — `Head`, `Image`, `Breadcrumb`, `Alternate`, `Headable`, `HeadFunc`, `Excerpt`, `URL`, `Crumbs`; `Module[T].headFunc` field added (Amendment A1) |
| 2026-03-03 | Milestone 3 Step 2: `schema.go` implemented — `SchemaFor`, 8 JSON-LD rich result types (Article, Product, FAQPage, HowTo, Event, Recipe, Review, Organization), BreadcrumbList, 6 provider interfaces (FAQProvider, HowToProvider, EventProvider, RecipeProvider, ReviewProvider, OrganizationProvider) |
| 2026-03-03 | Milestone 3 Step 3: `sitemap.go` implemented — `SitemapConfig`, `ChangeFreq`, `SitemapNode`, `SitemapPrioritiser`, `SitemapEntry`, `SitemapStore`, `WriteSitemapFragment`, `SitemapEntries`, `WriteSitemapIndex`; Amendments A2 (node.go getters), A3 (Module sitemap wiring), A4 (App sitemap store + Handler guard) |
| 2026-03-03 | Milestone 3 Step 4: `robots.go` implemented — `CrawlerPolicy`, `Allow`/`Disallow`/`AskFirst`, `RobotsConfig`, `RobotsTxt`, `RobotsTxtHandler`; Amendment A5: `SEOOption`, `seoState`, `App.SEO()`, `robotsTxtRegistered` guard in `forge.go` |
| 2026-03-05 | Milestone 4 Step 1: `templatedata.go` implemented — `TemplateData[T]`, `NewTemplateData` constructor; `SiteName` sourced from `Config.BaseURL` hostname |
| 2026-03-05 | Milestone 4 Step 2: `templates.go` implemented — `templateParser` interface, `Templates`/`TemplatesOptional` options, `forgeHeadTmpl` const, `parseTemplates()`/`renderListHTML`/`renderShowHTML` on `Module[T]`, `bindErrorTemplates`; Amendments A6 (`module.go` template fields + HTML render path), A7 (`errors.go` `errorTemplateLookup`), A8 (`forge.go` `templateModules` + startup parse wiring) |
| 2026-03-05 | Milestone 4 Step 3: `templatehelpers.go` implemented — `forgeMeta`, `forgeDate`, `forgeMarkdown` (stdlib-only), `forgeExcerpt`, `forgeCSRFToken`, `forgeLLMSEntries` (stub), `TemplateFuncMap()`; Amendment A9 (`templates.go` `parseOneTemplate` now calls `.Funcs(TemplateFuncMap())`) |
| 2026-03-05 | Milestone 4 Step 4: `integration_test.go` implemented — 15 cross-component integration tests covering HTML render cycle, forge:head correctness, error pages (custom + fallback), CSRF token round-trip, App-level SEO/sitemap routing, and TemplateData field propagation |
| 2026-03-05 | Milestone 4 Step 5: `integration_full_test.go` implemented — 19 cross-milestone integration tests (M1–M4): multi-module routing, global middleware order, role-gated access (HasRole + inline middleware), AfterCreate/AfterDelete/cross-module signal isolation, content negotiation across two module types, forge_meta/forge_markdown/BreadcrumbList through render, sitemap URL in robots.txt, error template first-match and fallthrough, TemplateData siteName and request URL |
| 2026-03-06 | Milestone 5 Step 1: `social.go` implemented — `SocialFeature`, `OpenGraph`, `TwitterCard`, `Social()` option; Amendment A9 (`head.go`: `Tags []string`, `TwitterCardType`, `TwitterMeta`, `SocialOverrides`, `Head.Social` field); Amendment A10 (`templates.go` `forgeHeadTmpl` extended — full OG + Twitter block, `forge_rfc3339` added to `templatehelpers.go` and `TemplateFuncMap()`, Module[T].social field + case in `module.go`) |
| 2026-03-06 | Milestone 5 Step 2: `ai.go` implemented — `Markdownable` (A11: migrated from `module.go`), `AIDocSummary`, `AIFeature`, `LLMsTxt`/`LLMsTxtFull`/`AIDoc` constants, `AIIndex()` option, `WithoutID()` option, `LLMsEntry`, `LLMsTemplateData`, `LLMsStore`, `NewLLMsStore`, `extractNode`, `renderAIDoc`; `forgeLLMSEntries(data any)` wired in `templatehelpers.go` (A12); `LLMsStore` wiring in `forge.go` Content+Handler (A13); README one-liner added (A14); AIDoc URL uses `/{prefix}/{slug}/aidoc` — Go’s net/http.ServeMux does not support partial wildcard segments, so `/{slug}.aidoc` is not a valid pattern (A15: DECISIONS.md updated) || 2026-03-06 | Milestone 5 Step 3: `feed.go` implemented — `FeedConfig`, `Feed()` option (opt-in, Amendment A16: Decision 13 updated), `FeedDisabled()` option, `rssItem`/`rssChannel`/`rssRoot` XML structs, `FeedStore`, `NewFeedStore`, `buildRSSItem`, `capitalisePrefixTitle`, `guessMIMEType`, `writeRSSFeed`; `ModuleHandler` serves `/{prefix}/feed.xml`, `IndexHandler` serves `/feed.xml` aggregate (all Published items, reverse-chronological); `feedCfg`/`feedStore`/`regenerateFeed`/`setFeedStore` added to `module.go`; `feedStore`/`feedIndexRegistered` added to `forge.go` |
| 2026-03-06 | Milestone 5 Step 4: `integration_full_test.go` extended — G9–G12 cross-milestone groups appended: G9 (Social + SitemapConfig M3): OG/Twitter tags in forge:head, Draft → 404; G10 (AIIndex + M4 content negotiation): /llms.txt Published/Draft filter, /posts/{slug}/aidoc 200/404, Accept:text/markdown alongside AIDoc; G11 (Feed + M1 AfterPublish signal): /posts/feed.xml RSS 2.0, Draft excluded, AfterPublish fires within 500ms; G12 (Full M5 stack): Social+AIIndex+Feed+SitemapConfig+HeadFunc+Templates — OG/Twitter, /llms.txt, /aidoc, /feed.xml all verified. README.md: AI indexing and Social sharing badges updated from 🔲 Coming in Milestone 5 → ✅ Available. Milestone 5 complete. |
| 2026-03-06 | Amendment A17: `compressIfAccepted(w, r, body, contentType)` helper added to `ai.go`; gzip applied directly at AI endpoint handlers — `CompactHandler`, `FullHandler`, `renderAIDoc` (now takes `r *http.Request`); 1400-byte threshold; `Vary: Accept-Encoding` always set. Supersedes Decision 13 Amendment A clause 3. Tests: `TestCompressIfAccepted_gzip`, `TestCompressIfAccepted_smallBody`, `TestCompressIfAccepted_noAcceptEncoding`, `TestLLMsTxt_gzip`, `TestAIDoc_gzip`. |
| 2026-03-07 | Milestone 6 Step 1: `cookies.go` implemented — `CookieCategory` (`Necessary`/`Preferences`/`Analytics`/`Marketing`), `Cookie` struct, `SetCookie`, `SetCookieIfConsented`, `ReadCookie`, `ClearCookie`, `ConsentFor`, `GrantConsent`, `RevokeConsent`; `forge_consent` Necessary cookie stores consent state; Decision 5 enforcement: `SetCookie` panics on non-Necessary, `SetCookieIfConsented` panics on Necessary. |
| 2026-03-07 | Milestone 6 Step 2: `cookiemanifest.go` implemented — `cookieManifest`/`cookieManifestEntry` JSON types, `buildManifest`, `sameSiteName`, `ManifestAuth` option, `newCookieManifestHandler`; Amendment A18: `App.Cookies()`, `App.CookiesManifestAuth()`, `cookieDecls`/`cookieManifestOpts` fields added to `forge.go`; `GET /.well-known/cookies.json` mounted lazily in `App.Handler()`. |
| 2026-03-07 | Milestone 6 Step 3: `integration_full_test.go` extended — G13–G15 cross-milestone groups appended: G13 (M6 consent enforcement, Decision 5): SetCookie/ConsentFor/SetCookieIfConsented/GrantConsent/RevokeConsent; G14 (M6 + M2 handler pattern): consent lifecycle wired through an HTTP handler, ClearCookie expiry, Necessary always-true; G15 (M6 + M2 App + M1 BearerHMAC): manifest mounted/sorted/not-mounted-when-empty, authGuard 401/200. README.md: Cookies & Compliance badge updated from 🔲 Coming in Milestone 6 → ✅ Available. Milestone 6 complete. |
| 2026-03-07 | Milestone 7 Step 1: `storage.go` extended — `SQLRepo[T]` production `Repository[T]` backed by `forge.DB`; `Table()` `SQLRepoOption`; `camelToSnake()` + plural table-name derivation; `FindByID`/`FindBySlug` delegate to `QueryOne`; `FindAll` with status IN, ORDER BY, LIMIT/OFFSET; `Save` upsert (ON CONFLICT); `Delete` returns `ErrNotFound` when RowsAffected==0. 9 new `TestSQLRepo_*` tests + extended fake driver. Amendment A19. |
| 2026-03-07 | Milestone 7 Step 2: `redirects.go` implemented — `RedirectCode` (`Permanent`/`Gone`), `RedirectEntry` (+`IsPrefix`), `From` type, `Redirects()` module option, `RedirectStore` (exact map + prefix slice sorted longest-first, chain collapse max depth 10, `Get`/`Add`/`All`/`Len`), DB persistence (`Load`/`Save`/`Remove`), `handler()` fallback; `forge.go` Amendment A20: `redirectStore *RedirectStore`, `redirectFallbackReg`, `New()` init, `Content()` extracts `redirectsOption`, `Handler()` mounts `"/"` fallback, `App.Redirect()`, `App.RedirectStore()`. 13 new `TestRedirectStore_*`/`TestApp_Redirect_*` tests. |
| 2026-03-07 | Milestone 7 Step 3: `redirectmanifest.go` implemented — `redirectManifestEntry`/`redirectManifest` JSON types, `buildRedirectManifest` (delegates to `store.All()` for sorted entries), `newRedirectManifestHandler` (serialises per-request from live store, reuses `manifestAuthOption`, `Cache-Control: no-store`); `forge.go` Amendment A21: `redirectManifestReg bool`, `GET /.well-known/redirects.json` always mounted in `Handler()`. 8 new `TestRedirectManifest_*` tests. |

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
│                     GetSlug(), GetPublishedAt(), GetStatus() getter methods (Amendment A2)
├── context.go        Context interface, contextImpl, ContextFrom(), NewTestContext(), User, GuestUser
├── signals.go        Signal type, On[T]() option, dispatchBefore(), dispatchAfter(), debouncer
├── storage.go        DB interface, Query[T], QueryOne[T], Repository[T], MemoryRepo[T], ListOptions
├── auth.go           AuthFunc interface, BearerHMAC, CookieSession, BasicAuth, AnyAuth, SignToken
├── middleware.go     RequestLogger, Recoverer, SecurityHeaders, CORS, MaxBodySize,
│                     RateLimit, TrustedProxy, InMemoryCache, CacheStore, CSRF, Chain
├── module.go         Module[T], NewModule, Register, At, Cache, Auth,
                      Middleware, Repo, On, SitemapConfig, AIIndex, WithoutID,
                      Feed, FeedDisabled options;
                      setSitemap, regenerateSitemap, setAIRegistry, regenerateAI, aiDocHandler;
                      setFeedStore, regenerateFeed;
                      aiFeatures, llmsStore, withoutID, feedCfg, feedStore fields
│                     (Markdownable migrated to ai.go — Amendment A11)
├── forge.go          Config, MustConfig, New, App (Use/Content/Handle/Run/Handler/SEO),
│                     Registrator, SEOOption, seoState, httpsRedirect,
│                     graceful shutdown via SIGINT/SIGTERM;
│                     SitemapStore wiring in Content+Handler (Amendment A4);
│                     SEO option loop, robotsTxtRegistered guard in Handler (Amendment A5);
│                     LLMsStore wiring in Content+Handler, llmsTxtRegistered +
                      llmsFullTxtRegistered guards (Amendment A13);
                      FeedStore wiring in Content+Handler, feedIndexRegistered guard (A16)
└── head.go           Head (Title, Description, Author, Published, Modified, Image, Type,
                      Canonical, Tags, Breadcrumbs, Alternates, Social, NoIndex),
                      Image, Breadcrumb, Alternate, Headable, HeadFunc[T],
                      Excerpt, URL, Crumbs, Crumb, rich-result constants,
                      TwitterCardType (Summary/SummaryLargeImage/AppCard/PlayerCard),
                      TwitterMeta, SocialOverrides
└── schema.go         SchemaFor, FAQProvider, HowToProvider, EventProvider,
                      RecipeProvider, ReviewProvider, OrganizationProvider,
                      FAQEntry, HowToStep, EventDetails, RecipeDetails,
                      ReviewDetails, OrganizationDetails
└── sitemap.go        SitemapConfig, ChangeFreq, SitemapEntry, SitemapNode,
                      SitemapPrioritiser, SitemapStore, SitemapEntries[T],
                      WriteSitemapFragment, WriteSitemapIndex
└── robots.go         CrawlerPolicy (Allow/Disallow/AskFirst), RobotsConfig,
                      RobotsTxt, RobotsTxtHandler
└── templatedata.go   TemplateData[T], NewTemplateData
└── templates.go      templateParser, Templates, TemplatesOptional, forgeHeadTmpl, parseTemplates,
                      renderListHTML, renderShowHTML, errorTemplate, bindErrorTemplates;
                      Amendment A6 (Module[T] template fields + HTML render path),
                      Amendment A7 (errorTemplateLookup in errors.go),
                      Amendment A8 (templateModules + startup wiring in forge.go)
└── templatehelpers.go forgeMeta, forgeDate, forgeRFC3339, forgeMarkdown, forgeExcerpt, forgeCSRFToken,
                      forgeLLMSEntries(data any), TemplateFuncMap();
                      Amendment A9 (parseOneTemplate uses .Funcs(TemplateFuncMap()));
                      forge_rfc3339 added (M5 Step 1) for article:published_time in forge:head;
                      forgeLLMSEntries wired to real implementation (Amendment A12)
└── social.go         SocialFeature, OpenGraph, TwitterCard, Social() option
└── ai.go             Markdownable (migrated from module.go, A11), AIDocSummary,
                      AIFeature, LLMsTxt, LLMsTxtFull, AIDoc constants,
                      AIIndex() option, WithoutID() option,
                      LLMsEntry, LLMsTemplateData, LLMsStore, NewLLMsStore,
                      extractNode, renderAIDoc, hasAIFeature
└── feed.go           FeedConfig, Feed() option (opt-in, A16), FeedDisabled() option,
                      FeedStore, NewFeedStore, buildRSSItem, capitalisePrefixTitle,
                      guessMIMEType, writeRSSFeed;
                      ModuleHandler → /{prefix}/feed.xml;
                      IndexHandler → /feed.xml aggregate (reverse-chronological)
└── integration_test.go 15 integration tests: HTML render cycle, forge:head, error pages,
                      CSRF round-trip, App-level SEO/sitemap, TemplateData correctness
└── integration_full_test.go 19 cross-milestone tests (M1–M4): multi-module routing,
                      global middleware order, role guards, AfterCreate/AfterDelete/isolation,
                      content negotiation, forge_meta/forge_markdown/BreadcrumbList,
                      sitemap in robots.txt, error template first-match + fallthrough,
                      TemplateData siteName + request URL

github.com/forge-cms/forge-pgx/  (separate module: ./forge-pgx/)
└── pgx.go            Wrap(*pgxpool.Pool) forge.DB — native pgx adapter
```

### Planned (future milestones)

```
├── storage.go (extend) SQLRepo[T] — production Repository[T] backed by forge.DB;
│                     Table() SQLRepoOption; auto-derived table names (snake_case plural);
│                     FindByID/FindBySlug/FindAll/Save/Delete; reuses dbFields cache;
│                     $N SQL placeholders (Decision 23); Amendment A19       (Milestone 7)
├── redirects.go      RedirectCode (MovedPermanently/Gone), RedirectEntry (+IsPrefix),
│                     From type, Redirects() option, RedirectStore (exact + prefix
│                     lookup, chain collapse, DB persistence via Load/Save/Remove),
│                     App.Redirect(), "/" fallback wiring (Amendment A20)      (Milestone 7)
├── redirectmanifest.go  buildRedirectManifest, newRedirectManifestHandler;
│                     GET /.well-known/redirects.json (always mounted, live JSON);
│                     reuses ManifestAuth option (Amendment A21)               (Milestone 7)
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

// SitemapPrioritiser — optional; per-item sitemap priority   (sitemap.go, Milestone 3)
type SitemapPrioritiser interface {
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
cookies.go      — depends on: errors (none — stdlib net/http only)
├── cookiemanifest.go — depends on: cookies, forge.go (Amendment A18)
redirects.go    — depends on: errors, storage (forge.DB), forge.go (A20)       (Milestone 7)
├── redirectmanifest.go — depends on: redirects, cookiemanifest (manifestAuthOption), forge.go (A21)
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
