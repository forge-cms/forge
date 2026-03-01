# Forge — Backlog

This document is the living roadmap for Forge.
All decisions affecting architecture are locked in DECISIONS.md.
This document is about *what to build* and in what order.

---

## Status

| Phase | Contents | Status |
|-------|----------|--------|
| Architecture | All 22 decisions | ✅ Locked |
| Documentation | README, DECISIONS, BACKLOG, CHECKLIST | ✅ Done |
| Core implementation | forge.Node, Module, Context, Auth... | 🔲 Not started |
| Example app | example/blog | 🔲 Not started |
| Tests | Full test suite | 🔲 Not started |
| Launch | Domain, Sponsors | 🔲 Not started |

---

## Milestone 1 — Core (v0.1.0)

The minimum needed for a developer to build something real.

### forge.Node
- [ ] `ID` — UUID v7 generation via `crypto/rand`
- [ ] `Slug` — auto-generation from first `forge:"required"` string field
- [ ] Slug sanitisation — whitelist `[a-z0-9-]`, max 200 chars, collision suffix
- [ ] `Status` — Draft / Published / Scheduled / Archived
- [ ] `PublishedAt`, `ScheduledAt`, `CreatedAt`, `UpdatedAt`
- [ ] Struct tag validation — `required`, `min`, `max`, `email`, `url`, `slug`, `oneof`
- [ ] `Validate() error` interface — runs after tag validation

### forge.Signals (early — everything depends on this)
- [ ] `forge.Signal` type
- [ ] Built-in signals: `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `AfterPublish`, `AfterUnpublish`, `AfterArchive`, `SitemapRegenerate`
- [ ] `forge.On(signal, handler)` — module option
- [ ] `BeforeX` signals can abort the operation by returning an error
- [ ] `AfterX` signals run asynchronously

### forge.Context
- [ ] Embeds `context.Context`
- [ ] Implemented as an interface (Decision 21) — enables testing without HTTP
- [ ] `User() forge.User`
- [ ] `Locale() string` — returns `"en"` in v1
- [ ] `SiteName() string`
- [ ] `RequestID() string`
- [ ] `Request() *http.Request`
- [ ] `Response() http.ResponseWriter`
- [ ] `forge.ContextFrom(r *http.Request) forge.Context`
- [ ] `forge.NewTestContext(user forge.User) forge.Context` — for unit tests

### forge.Error hierarchy
- [ ] `forge.Error` interface — `Code()`, `HTTPStatus()`, `Public()`
- [ ] Sentinel errors — `ErrNotFound`, `ErrGone`, `ErrForbidden`, `ErrUnauth`, `ErrConflict`
- [ ] `forge.Err(field, message)` — ValidationError with field details
- [ ] `forge.Require(errs...)` — collect multiple ValidationErrors
- [ ] `forge.WriteError(w, r, err)` — single call, correct HTTP response
- [ ] Error response format — JSON with `code`, `message`, `request_id`, `fields`
- [ ] HTML error pages — `templates/errors/{status}.html`

### Auth
- [ ] `forge.BearerHMAC(secret)` — HMAC-SHA256 token validation
- [ ] `forge.CookieSession(name, secret)` — cookie-based auth + auto CSRF
- [ ] `forge.BasicAuth(user, pass)` — dev only + production startup warning
- [ ] `forge.AnyAuth(fns...)` — accept bearer or cookie, first match wins
- [ ] `forge.SignToken(user, secret)` — generate signed token
- [ ] `forge.User` — `ID`, `Name`, `Roles`
- [ ] `user.HasRole(role)` — hierarchical check (Admin includes Editor includes Author)
- [ ] `user.Is(role)` — exact match only
- [ ] CSRF token generation and validation in `CookieSession`

### Roles
- [ ] `forge.Admin`, `forge.Editor`, `forge.Author`, `forge.Guest` — built-in constants
- [ ] Hierarchical inheritance — `HasRole(Editor)` returns true for Admin
- [ ] `forge.Role("custom").Below(Editor).Above(Author)` — custom roles
- [ ] `app.Roles(...)` — register custom roles
- [ ] `forge.Read(role)`, `forge.Write(role)`, `forge.Delete(role)` — per-module auth

### Storage (Decision 22)
- [ ] `forge.DB` interface — `QueryContext`, `ExecContext`, `QueryRowContext`
- [ ] `forge.Query[T](db forge.DB, sql, args...)` — list query with struct scanning
- [ ] `forge.QueryOne[T](db forge.DB, sql, args...)` — single item query
- [ ] Satisfied by `*sql.DB`, `*sql.Tx`, and `forgepgx.Wrap(pool)` out of the box
- [ ] Struct field mapping — `db` tag, then field name lowercased
- [ ] Reflection cache — `sync.Map`, scan struct fields once per type
- [ ] `forge.Repository[T]` interface — for MemoryRepo and test doubles
- [ ] `forge.NewMemoryRepo[T]()` — in-memory implementation for tests
- [ ] `forge.ListOptions` — `Page`, `PerPage`, `OrderBy`, `Desc`, `Offset()`

### forge-pgx (parallel to Milestone 1 — separate module)
- [ ] `github.com/forge-cms/forge-pgx` — new repository under forge-cms org
- [ ] `forgepgx.Wrap(pool *pgxpool.Pool) forge.DB` — native pool adapter
- [ ] ~25 lines — thin translation layer, no business logic
- [ ] Tests against a real PostgreSQL instance
- [ ] README with performance comparison vs stdlib

### forge.Module[T]
- [ ] `app.Content(&T{}, opts...)` registration
- [ ] Auto-routing: GET list, GET show, POST create, PUT update, DELETE delete
- [ ] Lifecycle enforcement — Draft/Scheduled/Archived → 404 for Guest
- [ ] Content negotiation — JSON / HTML / markdown / plain text
- [ ] `forge.At(prefix)` option
- [ ] `forge.Cache(ttl)` option — LRU, max 1000 entries
- [ ] `forge.Middleware(...)` option
- [ ] `forge.MCP(...)` option — no-op in v1, reserved for v2 (Decision 19)

### App
- [ ] `forge.New(config)` — top-level builder, calls MustConfig internally
- [ ] `app.Use(middleware)` — global middleware
- [ ] `app.Content(...)` — register content module
- [ ] `app.Handle(pattern, handler)` — custom route
- [ ] `app.HandleFunc(pattern, fn)` — custom route function
- [ ] `app.Run(addr)` — start with graceful shutdown (SIGINT/SIGTERM, 30s timeout)
- [ ] `app.Handler()` — return http.Handler without starting server

### Configuration (Decision 20)
- [ ] `forge.Config` struct — `BaseURL`, `Secret`, `Env`, `Logger`, `LogLevel`
- [ ] `forge.Development`, `forge.Production`, `forge.Test` — env constants
- [ ] Auto-read `FORGE_ENV` → `Config.Env`
- [ ] Auto-read `FORGE_BASE_URL` → `Config.BaseURL` (fallback)
- [ ] Auto-read `FORGE_SECRET` → `Config.Secret` (fallback)
- [ ] Auto-read `FORGE_LOG_LEVEL` → `Config.LogLevel` (fallback)
- [ ] Auto-read `PORT` → used by `app.Run("")`
- [ ] `forge.MustConfig(cfg)` — startup validation with precise error messages
- [ ] FORGE_SECRET warning if not set in production
- [ ] FORGE_SECRET warning if under 32 bytes
- [ ] `app.Run("")` → uses `PORT` → falls back to `:8080`

### Middleware
- [ ] `forge.RequestLogger()` — structured slog output + request_id
- [ ] `forge.Recoverer()` — panic → 500, never crash
- [ ] `forge.CORS(origin)` — CORS headers
- [ ] `forge.MaxBodySize(n)` — request body limit
- [ ] `forge.RateLimit(n, duration)` — per-IP rate limiting
- [ ] `forge.SecurityHeaders()` — HSTS, CSP, X-Frame-Options, Referrer-Policy
- [ ] `forge.InMemoryCache(ttl, opts...)` — LRU cache, max entries, X-Cache header
- [ ] `forge.Chain(h, middlewares...)` — composition helper

---

## Milestone 2 — SEO & Head (v0.2.0)

### forge.Head
- [ ] `forge.Head` struct — all fields from README
- [ ] `forge.Image` struct — URL, Alt, Width, Height
- [ ] `forge.Excerpt(text, maxLen)` — smart truncation at word boundary
- [ ] `forge.URL(parts...)` — URL builder
- [ ] `forge.Crumbs(...)` / `forge.Crumb(label, url)` — breadcrumb builder
- [ ] `forge.Headable` interface — `Head() forge.Head`
- [ ] `forge.HeadFunc(fn)` — module-level Head override

### Structured data (JSON-LD)
- [ ] `forge.Article` — BlogPosting schema
- [ ] `forge.Product` — Product schema
- [ ] `forge.FAQPage` — FAQPage schema
- [ ] `forge.HowTo` — HowToStep schema
- [ ] `forge.Event` — Event schema
- [ ] `forge.Recipe` — Recipe schema
- [ ] `forge.Review` — Review schema
- [ ] `forge.Organization` — Organization schema
- [ ] BreadcrumbList — auto-generated from `Head.Breadcrumbs`

### Sitemap
- [ ] Per-module sitemap fragment (`/posts/sitemap.xml`)
- [ ] Sitemap index merger (`/sitemap.xml`)
- [ ] Event-driven regeneration via Signal
- [ ] Debounce — 2 seconds, async goroutine
- [ ] `forge.SitemapConfig` — BaseURL, ChangeFreq, Priority
- [ ] `forge.SitemapPriority()` — optional interface per content type

### Robots.txt
- [ ] `app.SEO(forge.RobotsConfig{...})` — auto-generated robots.txt
- [ ] `forge.AskFirst` AI scraper policy
- [ ] Auto-append sitemap URL

---

## Milestone 3 — Templates & Rendering (v0.3.0)

Moved before Social/AI — needed to build example apps and validate the API.

### Template system
- [ ] `forge.Templates(dir)` — parse at startup, fail fast
- [ ] `forge.TemplatesWatch(dir)` — hot-reload in development
- [ ] `forge.TemplatesOptional(dir)` — no startup error if dir missing
- [ ] `templates/{type}/list.html` and `show.html` convention
- [ ] `templates/errors/{status}.html` — custom error pages
- [ ] `{{template "forge:head" .Head}}` — built-in head partial
- [ ] `forge_meta`, `forge_date`, `forge_markdown`, `forge_excerpt` template helpers
- [ ] `forge_csrf_token` — CSRF token helper for forms
- [ ] `forge_llms_entries` — llms.txt template helper
- [ ] `forge.TemplateData[T]` — `Content`, `Head`, `User`, `Request`

---

## Milestone 4 — Social & AI (v0.4.0)

### Social sharing
- [ ] `forge.Social(platforms...)` — module option
- [ ] `forge.OpenGraph` — all `og:` meta tags including article tags
- [ ] `forge.TwitterCard` — all `twitter:` meta tags
- [ ] `forge.SummaryLargeImage`, `forge.Summary` — Twitter card types
- [ ] `forge.SocialOverrides` — per-platform overrides
- [ ] `forge.LinkedIn` — LinkedIn-specific tags

### AI indexing
- [ ] `forge.AIIndex(options...)` — module option
- [ ] `forge.LLMsTxt` — auto-generated `/llms.txt`
- [ ] `/llms-full.txt` — with content summaries
- [ ] Template override — `templates/llms.txt`
- [ ] `forge.AIDoc` — `.aidoc` endpoint per content item
- [ ] `forge.WithoutID` — suppress UUID in AIDoc
- [ ] AIDoc format — `+++aidoc+v1+++` delimiter, all fields
- [ ] `forge.AIDocSummary()` — optional interface for custom summary
- [ ] Content negotiation — `Accept: text/markdown`, `Accept: text/plain`
- [ ] `forge.Markdownable` interface — `Markdown() string`

### RSS feeds
- [ ] Auto-generated `/feed.xml` per module
- [ ] Published content only
- [ ] `forge.FeedConfig` — title, description, author
- [ ] `forge.Feed(forge.Disabled)` — opt-out
- [ ] Regeneration via same Signal as sitemap

---

## Milestone 5 — Cookies & Compliance (v0.5.0)

- [ ] `forge.Cookie` struct — all fields from README
- [ ] `forge.Necessary`, `forge.Preferences`, `forge.Analytics`, `forge.Marketing`
- [ ] `forge.SetCookie(w, r, cookie, value)` — Necessary only
- [ ] `forge.SetCookieIfConsented(w, r, cookie, value) bool`
- [ ] `forge.ReadCookie(r, cookie) (string, bool)`
- [ ] `forge.ClearCookie(w, cookie)`
- [ ] `forge.ConsentFor(r, category) bool`
- [ ] `app.Cookies(cookies...)` — registration
- [ ] `/.well-known/cookies.json` — compliance manifest endpoint
- [ ] `forge.ManifestAuth(role)` — access control on manifest
- [ ] Consent state stored in Necessary cookie

---

## Milestone 6 — Redirects (v0.6.0)

- [ ] `forge.RedirectEntry` struct
- [ ] Auto-create on slug rename
- [ ] Auto-create on prefix change
- [ ] `410 Gone` on archive and delete
- [ ] `404` on Draft and Scheduled (does not leak existence)
- [ ] Redirect chain collapse — A→B→C becomes A→C
- [ ] `forge.Redirects(forge.From(prefix))` — bulk redirect
- [ ] `app.Redirect(from, to, type)` — manual redirect
- [ ] `/.well-known/redirects.json` — inspect endpoint (Editor+)

---

## Milestone 7 — Scheduled publishing (v0.7.0)

- [ ] Adaptive ticker — `time.Until(nextScheduledAt)`
- [ ] Fallback polling — 60 seconds if nothing scheduled
- [ ] Transition Scheduled → Published
- [ ] Set `PublishedAt` automatically
- [ ] Fire `AfterPublish` Signal
- [ ] Trigger sitemap + feed regeneration
- [ ] Graceful shutdown — wait for in-progress publish cycle

---

## Milestone 8 — v1.0.0 stabilisation

- [ ] Full test suite — all packages, minimum 80% coverage
- [ ] Benchmark suite — request throughput, cache hit rate, template render time
- [ ] godoc documentation on all exported symbols
- [ ] Example apps:
      `example/blog` — blog with posts and tags
      `example/docs` — documentation site
      `example/api` — pure API without templates
- [ ] CHANGELOG.md created
- [ ] Semantic versioning policy documented
- [ ] API stability promise — no breaking changes in v1.x

---

## Milestone 9 — MCP support (v2)

Implementation of Decision 19. Syntax already reserved in v1.

- [ ] `forge.MCPServer` — MCP server started with `app.Run()`
- [ ] Auto-generated resource schema from `forge.Node` + struct tags
- [ ] `forge.MCPRead` — expose content as readable MCP resources
- [ ] `forge.MCPWrite` — expose Create/Update/Delete/Publish as MCP tools
- [ ] Lifecycle enforcement in MCP — Draft not visible to Guest via MCP
- [ ] Auth in MCP — same role system as HTTP endpoints
- [ ] Rate limiting on MCP endpoints
- [ ] Transport: stdio (local AI tools) + SSE (remote, authenticated)
- [ ] `forge-mcp` as separate package (preserves zero-deps in core)
- [ ] Documentation: "Connecting Claude/Cursor/Copilot to your Forge app"

---

## v2+ Roadmap (not yet planned)

These topics may not be implemented without a new Tier 1 decision round.

- **i18n** — locale-aware URLs, hreflang tags, per-locale content
- **Forge AI** — content assistant built on MCP + AIDoc + llms.txt. Paid product via Forge Cloud. Architecturally impossible without Forge's content semantics.
- **Admin UI** — `forge-studio` as a separate package
- **Search** — SQLite FTS5 integration, `forge.Searchable` interface
- **Webhooks** — outbound HTTP on content events
- **Multi-tenancy** — multiple sites from one instance
- **GraphQL** — auto-generated schema from content types
- **Edge/CDN** — surrogate keys, automatic CDN purge
- **Image resizing** — `forge-images` as a separate package
- **Forge Cloud** — managed hosting, dual-license introduction
- **Database migrations** — `forge migrate` CLI or migration interface
