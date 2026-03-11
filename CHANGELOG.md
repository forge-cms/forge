# Changelog

All notable changes to Forge are documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**API stability promise:** every exported symbol in `github.com/forge-cms/forge`
at v1.0.0 is stable. No breaking changes will be made without a new major version.
The zero-dependency policy and zero-reflection-at-request-time guarantee are
treated as part of the stability promise.

**Architectural rationale:** see [DECISIONS.md](DECISIONS.md) for the reasoning
behind every design choice. Amendments that changed existing behaviour are
cross-referenced below by their Amendment ID.

---

## [Unreleased]

Changes planned for v2 and beyond are tracked in [BACKLOG.md](BACKLOG.md)
under Milestone 10 and the v2+ Roadmap section.

---

## [1.0.1] — 2026-03-11

Error handling pipeline hardening. All six `http.Error` bypass sites removed;
four missing sentinels added; `errorTemplateLookup` race fixed; `Recoverer`
stack buffer increased. No breaking API changes. (Amendments A29–A32)

### Added

- `ErrBadRequest` (400 `bad_request`), `ErrNotAcceptable` (406 `not_acceptable`),
  `ErrRequestTooLarge` (413 `request_too_large`), `ErrTooManyRequests`
  (429 `too_many_requests`) sentinel errors — complete the framework's own
  HTTP status vocabulary (A29)
- `setErrorTemplateLookup` / `runErrorTemplateLookup` internal helpers that
  wrap `errorTemplateLookup` in a `sync.RWMutex`, eliminating the data race
  between `App.Handler()` start-up and in-flight requests (A29)
- `ERROR_HANDLING.md` — authoritative strategy document for error handling;
  required reading before any code that calls `WriteError` or adds a sentinel

### Fixed

- `respond()` used a direct type assertion `err.(*ValidationError)` instead of
  `errors.As`; a wrapped `*ValidationError` would have silently produced a 422
  response without field details (A29)
- `writeContent` had no `*http.Request`, forcing 406 responses via `http.Error`
  (plain text, no `X-Request-ID`); now receives `r *http.Request` and calls
  `WriteError(w, r, ErrNotAcceptable)` (A30)
- JSON decode failures in `createHandler` and `updateHandler` called
  `http.Error` (plain text, no `X-Request-ID`, always 400); now calls
  `WriteError` with `ErrRequestTooLarge` (413) when `*http.MaxBytesError` is
  detected, otherwise `ErrBadRequest` (400) (A30)
- `renderListHTML` and `renderShowHTML` called `http.Error` for nil template;
  now calls `WriteError(w, r, ErrNotAcceptable)` (A31)
- `RateLimit` called `http.Error` for 429 rate-limit responses (plain text, no
  `X-Request-ID`); now calls `WriteError(w, r, ErrTooManyRequests)` (A32)
- `Recoverer` stack capture buffer was 4096 bytes; deep stacks (recursive
  templates, chained middleware) were silently truncated; increased to 32 KB (A32)

---

## [1.0.0] — 2026-03-08

v1.0.0 stabilisation: test coverage audit, benchmarks, godoc pass, and three
reference example applications.

### Added

- `go test ./... -cover` coverage raised to ≥ 85%; targeted additions for
  `App.RedirectStore`, `TrustedProxy`, `CacheStore.Sweep`, `RedirectStore.Len`,
  `stripMarkdown`, `forgeLLMSEntries`
- `benchmarks_test.go`: 17 benchmarks covering hot paths across M1–M8;
  results in [BENCHMARKS.md](BENCHMARKS.md)
- Godoc improved on `type App` and all `App.*` methods (A18–A26); `SQLRepo[T]`
  method comments brought to parity with `MemoryRepo[T]`
- `example/blog/`: standalone blog — `Post` type, `SitemapConfig`, `Social`,
  `FeedConfig`, `AIIndex`, `On[*Post](AfterPublish)`, scheduled publishing
- `example/docs/`: standalone docs site — `Doc` type, `Headable`,
  `Markdownable`, `AIDocSummary`, `AIIndex(LLMsTxt, LLMsTxtFull, AIDoc)`,
  `RobotsConfig{AIScraper: AskFirst}`
- `example/api/`: standalone JSON API — `Resource` type, `Authenticate` +
  `BearerHMAC`, `Auth(Read(Guest), Write(Editor))`, `On[T](BeforeCreate)`,
  `Redirects`, `SecurityHeaders`, `RateLimit`

### Changed (Amendment A27)

- `middleware.go`: `Authenticate(auth AuthFunc) func(http.Handler) http.Handler`
  — middleware that populates `Context.User()` from an `AuthFunc` on every
  request; enables `Module[T]` role enforcement in production. Pairs with
  `BearerHMAC`, `CookieSession`, or `AnyAuth`.

---

## [0.8.0] — 2026-03-07

Scheduled publishing: automatic `Scheduled→Published` transition with signal
dispatch, sitemap regeneration, and feed rebuild.

### Added

- `scheduler.go`: `Scheduler` type, adaptive ticker (next-due interval, 60 s
  fallback), `schedulableModule` interface
- `module.go`: `Module[T].processScheduled` — transitions Scheduled items whose
  `ScheduledAt` is past to Published, assigns `PublishedAt`, fires `AfterPublish`,
  triggers sitemap and feed regeneration (Amendment A25)
- `forge.go`: `App` starts the scheduler before `ListenAndServe` and stops it
  after `srv.Shutdown` (Amendment A26)
- `NewBackgroundContext(host string) Context` — zero-value Context for use
  outside the HTTP request cycle, e.g. in scheduler callbacks (Amendment A24)

### Changed (Amendments A23, A25)

- `node.go`: `Node` time fields (`PublishedAt`, `ScheduledAt`, `CreatedAt`,
  `UpdatedAt`) carry `db:"..."` struct tags for `SQLRepo[T]` column mapping
  (Amendment A23)

---

## [0.7.0] — 2026-03-07

Production-ready SQL repository, redirect enforcement, chain collapse, and the
`/.well-known/redirects.json` inspect endpoint.

### Added

- `storage.go`: `SQLRepo[T any]` — production `Repository[T]` backed by
  `forge.DB`; struct-tag column mapping cached in `sync.Map`; `Table()` option
  for custom table names; full CRUD + `FindAll`/`FindBySlug` (Amendment A19)
- `redirects.go`: `RedirectCode` (`Permanent`, `Temporary`, `Gone`),
  `RedirectEntry`, `From` named type, `Redirects` module option, `RedirectStore`
  with O(1) exact + prefix lookups, chain collapse, optional DB persistence,
  `App.Redirect()`, `RedirectStore.Len()` (Amendments A20, A21)
- `redirectmanifest.go`: `/.well-known/redirects.json` always mounted, live
  serialisation of `RedirectStore`, `App.RedirectManifestAuth()` (Amendment A22)
- `forge.go`: `"/"` fallback handler wired from `redirectStore.handler()`
  (Amendment A20)

---

## [0.6.0] — 2026-03-07

Cookie consent enforcement and `/.well-known/cookies.json` compliance manifest.

### Added

- `cookies.go`: `CookieCategory` (`Necessary`, `Preferences`, `Analytics`,
  `Marketing`), `Cookie`, `SetCookie`, `SetCookieIfConsented`, `ReadCookie`,
  `ClearCookie`, `GrantConsent`, `RevokeConsent`, `ConsentFor`
- `cookiemanifest.go`: `/.well-known/cookies.json` typed JSON manifest,
  `ManifestAuth` option, `App.Cookies()`, `App.CookiesManifestAuth()`
  (Amendment A18)

---

## [0.5.0] — 2026-03-06

Open Graph, Twitter Cards, AI indexing (llms.txt + AIDoc), and opt-in RSS feeds.

### Added

- `social.go`: `Social` module option, `OpenGraph`, `TwitterCard`, card-type
  constants, `SocialOverrides`; `forge:head` partial renders OG and Twitter Card
  `<meta>` tags automatically when Social is registered
- `ai.go`: `AIIndex` module option; `LLMsTxt`, `LLMsTxtFull`, `AIDoc` flags;
  `LLMsStore`; `/llms.txt` compact index; `/llms-full.txt` full markdown corpus
  (requires `Markdownable`); `/{prefix}/{slug}/aidoc` per-item endpoint
  (requires `Markdownable`); `AIDocSummary` interface; `WithoutID` option;
  gzip compression in AI handlers (Amendment A17)
- `feed.go`: `Feed` module option, `FeedConfig`, `FeedDisabled`;
  `/{prefix}/feed.xml` per-module RSS 2.0; `/feed.xml` aggregate index;
  signal-driven regeneration (Amendment A16)

### Changed

- `Markdownable` interface (`Markdown() string`) moved from `module.go` to
  `ai.go`; consumed by `/llms-full.txt` and `/{slug}/aidoc`

---

## [0.4.0] — 2026-03-05

HTML rendering, template helpers, and content negotiation.

### Added

- `templatedata.go`: `TemplateData[T any]` (`Content`, `Head`, `User`,
  `Request`, `SiteName`), `NewTemplateData`
- `templates.go`: `Templates(dir)` / `TemplatesOptional(dir)` module options;
  `forge:head` partial (title, meta description, canonical, OG, Twitter Card,
  JSON-LD); error page template `forge:error`; HTML render path for list + show
- `templatehelpers.go`: `forge_meta`, `forge_date`, `forge_rfc3339`,
  `forge_markdown`, `forge_excerpt`, `forge_csrf_token`, `forge_llms_entries`;
  `TemplateFuncMap()` export

### Changed (Amendments A6, A7, A8, P3)

- Templates parsed once at `app.Run()` / `app.Handler()` startup; missing
  template files cause fast-fail (Amendment P3)
- `forge:head` emits `BreadcrumbList` JSON-LD when `Head.Breadcrumbs` is
  non-empty (Amendment A8)
- Error pages rendered via `forge:error` when available; fallback to
  `WriteError` plain text (Amendments A6, A7)

---

## [0.3.0] — 2026-03-03

SEO metadata, JSON-LD structured data, per-module sitemaps, and robots.txt.

### Added

- `head.go`: `Head`, `Image`, `Excerpt`, `Crumb`, `Crumbs`, `Breadcrumb`,
  `Headable` interface, `HeadFunc` module option
- `schema.go`: JSON-LD types — `Article`, `Product`, `FAQPage`, `HowTo`,
  `Event`, `Recipe`, `Review`, `Organization`, `BreadcrumbList`; `SchemaOf`
  serialises to `<script type="application/ld+json">`
- `sitemap.go`: per-module `/{prefix}/sitemap.xml`, `/sitemap.xml` aggregate
  index, `SitemapConfig` option, `SitemapStore`, `SitemapPrioritiser`
  interface, debounce-driven async regeneration (Amendment P1)
- `robots.go`: auto-generated `robots.txt`, `RobotsConfig`, `AskFirst` /
  `Disallow` AI-crawler policy constants, `App.SEO()`

---

## [0.2.0] — 2026-03-02

App bootstrap, HTTP server, graceful shutdown, and the `forge-pgx` companion
module.

### Added

- `forge.go`: `Config`, `MustConfig`, `New`, `App` (`Use`, `Content`, `Handle`,
  `Run`, `Handler`), `Registrator` interface, graceful shutdown on
  `SIGINT`/`SIGTERM`
- `forge-pgx` (`github.com/forge-cms/forge-pgx`): `forgepgx.Wrap(*pgxpool.Pool)
  forge.DB` — pgx/v5 adapter; no generated code, no ORM

---

## [0.1.0] — 2026-03-01

Foundation: the minimum needed to build a real application.
Zero third-party dependencies. All types in package `forge`.

### Added

- `errors.go`: `Error` interface, `ValidationError`, `Err`, `Require`,
  `WriteError`; sentinels `ErrNotFound`, `ErrGone`, `ErrForbidden`, `ErrUnauth`,
  `ErrConflict`
- `roles.go`: `Role`, `Guest`/`Author`/`Editor`/`Admin` (levels 10/20/30/40 —
  Amendment R1), `HasRole`, `IsRole`, `NewRole`, `Read`/`Write`/`Delete` options
- `node.go`: `Node`, `Status` (`Draft`, `Scheduled`, `Published`, `Archived`),
  `NewID` (UUID v7 — Amendment S1), `GenerateSlug`, `UniqueSlug`, `RunValidation`
- `context.go`: `User` (Amendment R3), `GuestUser`, `Context` interface,
  `ContextFrom`, `NewTestContext`
- `signals.go`: `Signal`, signal constants (`BeforeCreate`, `AfterCreate`,
  `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`, `AfterPublish`),
  `On[T]` generic option (Amendment S2), debouncer
- `storage.go`: `DB` interface, `Query[T]`, `QueryOne[T]`, `Repository[T]`
  interface, `MemoryRepo[T]`, `ListOptions`
- `auth.go`: `AuthFunc` interface (Amendment S8), `BearerHMAC`, `CookieSession`,
  `BasicAuth` (production warning — Amendment S7), `AnyAuth`, `SignToken`
  (ttl-aware — Amendment S10)
- `middleware.go`: `RequestLogger`, `Recoverer`, `CORS`, `MaxBodySize`,
  `RateLimit` (with `TrustedProxy` — Amendment S12), `SecurityHeaders`,
  `InMemoryCache`, `CacheStore`, `CSRF` (Amendments S6, S11), `Chain`
- `module.go`: `Module[T any]` (Amendment M3), `NewModule`, `At`, `Cache`,
  `Auth`, `Middleware`, `Repo`, `On`; content negotiation (`application/json`,
  `text/html`, `text/markdown`); per-module LRU; lifecycle enforcement
- `mcp.go`: `MCPOperation`, `MCPRead`/`MCPWrite`, `MCP()` no-op placeholder
  (reserved for Milestone 10)

---

## Version policy

Forge uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **MAJOR** — breaking change to any exported symbol in `github.com/forge-cms/forge`
- **MINOR** — new exported symbols; backward-compatible amendments
- **PATCH** — bug fixes with no API change

v1.0.0 and all future v1.x releases maintain full backward compatibility.
A v2 will be introduced as a separate import path
(`github.com/forge-cms/forge/v2`) following Go module conventions.

See [DECISIONS.md](DECISIONS.md) for the architectural rationale behind every
design choice in this release.
