# Forge — Backlog

High-level roadmap for all milestones. This file tracks progress at the milestone
and step level. For sub-task detail, verification blocks, and implementation notes
see the corresponding `Milestone{N}_BACKLOG.md` file.

All architectural decisions are locked in `DECISIONS.md`.

---

## Progress

| Milestone | Description | Status |
|-----------|-------------|--------|
| M1 | Core (v0.1.0) | ✅ Done |
| M2 | App Bootstrap (v0.2.0) | ✅ Done |
| M3 | SEO & Head (v0.3.0) | ✅ Done |
| M4 | Templates & Rendering (v0.4.0) | ✅ Done |
| M5 | Social & AI (v0.5.0) | ✅ Done |
| M6 | Cookies & Compliance (v0.6.0) | ✅ Done |
| M7 | Redirects (v0.7.0) | ✅ Done |
| M8 | Scheduled publishing (v0.8.0) | ✅ Done |
| M9 | v1.0.0 stabilisation | ✅ Done |
| M10 | MCP support (v2) | 🔲 Not started |

---

## Milestone 1 — Core (v0.1.0)

The minimum needed for a developer to build something real.
**Detail:** [Milestone1_BACKLOG.md](Milestone1_BACKLOG.md)

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
| 11 | forge.go | ⏸ Deferred — see M2 | — |
| P1 | forge-pgx | ⏸ Deferred — see M2 | — |

- [x] Step 1 — `errors.go`: forge.Error interface, sentinel errors, ValidationError, Require, WriteError
- [x] Step 2 — `roles.go`: Role type, built-in roles, level hierarchy, HasRole, IsRole, NewRole builder, Read/Write/Delete options
- [x] Step 3 — `mcp.go`: MCPOperation type, MCPRead/MCPWrite constants, MCP() no-op Option (reserved for v2)
- [x] Step 4 — `node.go`: Node struct, Status type, NewID (UUID v7), GenerateSlug, UniqueSlug, struct tag validation engine, RunValidation
- [x] Step 5 — `context.go`: User struct, GuestUser, Context interface, contextImpl, ContextFrom, NewTestContext
- [x] Step 6 — `signals.go`: Signal type and constants, On[T] generic option, dispatchBefore, dispatchAfter, debouncer
- [x] Step 7 — `storage.go`: DB interface, Query[T], QueryOne[T], Repository[T], MemoryRepo[T], ListOptions
- [x] Step 8 — `auth.go`: BearerHMAC, CookieSession (+ CSRF), BasicAuth, AnyAuth, SignToken
- [x] Step 9 — `middleware.go`: RequestLogger, Recoverer, CORS, MaxBodySize, RateLimit, SecurityHeaders, InMemoryCache, Chain
- [x] Step 10 — `module.go`: Module[T any], At/Cache/Auth/Middleware/Repo options, lifecycle, content negotiation, signals, per-module LRU
- [ ] ⏸ Step 11 — `forge.go`: Config, MustConfig, New, App (Use/Content/Handle/Run/Handler), graceful shutdown — **Deferred to Milestone 2 Step 1** (module.go scope grew to cover all routing; App bootstrapping is a natural M2 entry point)
- [ ] ⏸ Step P1 — `forge-pgx` (separate module): forgepgx.Wrap(pool) thin adapter for pgx/v5 native pool — **Deferred to Milestone 2**

---

## Milestone 2 — App Bootstrap (v0.2.0)

A developer can write `forge.New(cfg)`, wire up modules, and run a real server.
**Detail:** [Milestone2_BACKLOG.md](Milestone2_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | forge.go | ✅ Done | 2026-03-02 |
| P1 | forge-pgx | ✅ Done | 2026-03-02 |

- [x] Step 1 — `forge.go`: Config, MustConfig, New, App (Use/Content/Handle/Run/Handler), Registrator, graceful shutdown — *deferred from M1 Step 11*
- [x] Step P1 — `forge-pgx` (separate module): forgepgx.Wrap(pool) thin adapter for pgx/v5 native pool — *deferred from M1 Step P1*

---

## Milestone 3 — SEO & Head (v0.3.0)

Metadata, structured data, sitemaps, and robots.txt.
**Detail:** [Milestone3_BACKLOG.md](Milestone3_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | head.go | ✅ Done | 2026-03-03 |
| 2 | schema.go | ✅ Done | 2026-03-03 |
| 3 | sitemap.go | ✅ Done | 2026-03-03 |
| 4 | robots.go | ✅ Done | 2026-03-03 |

- [x] Step 1 — `head.go`: Head and Image structs, Excerpt, URL builder, Crumbs, Headable interface, HeadFunc option
- [x] Step 2 — `schema.go`: JSON-LD structured data types (Article, Product, FAQPage, HowTo, Event, Recipe, Review, Organization, BreadcrumbList)
- [x] Step 3 — `sitemap.go`: per-module fragment sitemaps, sitemap index, SitemapConfig option, SitemapStore, debounce-driven regeneration
- [x] Step 4 — `robots.go`: auto-generated robots.txt, RobotsConfig, AskFirst AI crawler policy, sitemap URL append

---

## Milestone 4 — Templates & Rendering (v0.4.0)

HTML rendering, template helpers, content negotiation.
**Detail:** [Milestone4_BACKLOG.md](Milestone4_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | templatedata.go | ✅ Done | 2026-03-05 |
| 2 | templates.go | ✅ Done | 2026-03-05 |
| 3 | templatehelpers.go | ✅ Done | 2026-03-05 |
| 4 | integration_test.go | ✅ Done | 2026-03-05 |
| 5 | integration_full_test.go | ✅ Done | 2026-03-05 |

- [x] Step 1 — `templatedata.go`: TemplateData[T] struct, NewTemplateData constructor — Content, Head, User, Request, SiteName
- [x] Step 2 — `templates.go`: Templates/TemplatesOptional options, forge:head partial, startup parse, HTML render path, error pages (Amendments A6/A7/A8)
- [x] Step 3 — `templatehelpers.go`: forge_meta, forge_date, forge_markdown, forge_excerpt, forge_csrf_token, forge_llms_entries (stub)
- [x] Step 4 — `integration_test.go`: full HTML render cycle, forge:head, error pages, CSRF, App-level SEO/sitemap gaps
- [x] Step 5 — `integration_full_test.go`: cross-milestone suite (M1–M4) — multi-module routing, roles, signals, content negotiation, schema, SEO, error pages, TemplateData

---

## Milestone 5 — Social & AI (v0.5.0)

Open Graph, Twitter Cards, llms.txt, AIDoc, RSS feeds.
**Detail:** [Milestone5_BACKLOG.md](Milestone5_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | social.go | ✅ Done | 2026-03-06 |
| 2 | ai.go | ✅ Done | 2026-03-06 |
| 3 | feed.go | ✅ Done | 2026-03-06 |
| 4 | integration_full_test.go | ✅ Done | 2026-03-06 |

- [x] Step 1 — `social.go`: Social option, OpenGraph, TwitterCard, card types, SocialOverrides, forge:head OG/Twitter rendering
- [x] Step 2 — `ai.go`: AIIndex option, LLMsTxt, LLMsTxtFull (full markdown corpus, opt-in), AIDoc format, AIDocSummary and Markdownable interfaces, WithoutID option, /llms.txt, /llms-full.txt and /{prefix}/{slug}/aidoc endpoints — **Note:** `Markdownable` migrates here from `module.go` (Amendment A11)
- [x] Step 3 — `feed.go`: opt-in RSS 2.0 per module (Feed option), FeedConfig, FeedDisabled, /{prefix}/feed.xml + /feed.xml aggregate, signal-driven regeneration (Amendment A16)
- [x] Step 4 — `integration_full_test.go`: cross-milestone groups G9–G12 (Social+SEO, AI+content negotiation, RSS+signals, full M5 stack) + README badge updates

---

## Milestone 6 — Cookies & Compliance (v0.6.0)

Typed cookie declarations, category-enforced consent, compliance manifest.
**Detail:** [Milestone6_BACKLOG.md](Milestone6_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | cookies.go | ✅ Done | 2026-03-07 |
| 2 | cookiemanifest.go | ✅ Done | 2026-03-07 |
| 3 | integration_full_test.go | ✅ Done | 2026-03-07 |

- [x] Step 1 — `cookies.go`: CookieCategory, Necessary/Preferences/Analytics/Marketing, Cookie struct, SetCookie, SetCookieIfConsented, ReadCookie, ClearCookie, GrantConsent, RevokeConsent, ConsentFor
- [x] Step 2 — `cookiemanifest.go`: cookieManifest JSON type, buildManifest, newCookieManifestHandler, ManifestAuth option, App.Cookies() + wiring in forge.go
- [x] Step 3 — `integration_full_test.go`: cross-milestone groups G13–G15 (consent enforcement, consent lifecycle + M1 roles, full M6 stack + manifest) + README badge

---

## Milestone 7 — Redirects (v0.7.0)

Production `SQLRepo[T]`, automatic redirect tracking, 410 Gone, chain collapse,
optional DB persistence, and `/.well-known/redirects.json` inspect endpoint.
**Detail:** [Milestone7_BACKLOG.md](Milestone7_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | storage.go | ✅ Done | 2026-03-07 |
| 2 | redirects.go | ✅ Done | 2026-03-07 |
| 3 | redirectmanifest.go | ✅ Done | 2026-03-07 |
| 4 | integration_full_test.go | ✅ Done | 2026-03-07 |

- [x] Step 1 — `storage.go`: `SQLRepo[T]` production `Repository[T]` backed by `forge.DB`; `Table()` SQLRepoOption; auto-derived table names; full CRUD using `dbFields` cache (Amendment A19)
- [x] Step 2 — `redirects.go`: `RedirectCode`, `RedirectEntry` (+`IsPrefix`), `From`/`Redirects` option, `RedirectStore` (exact + prefix lookup, chain collapse, DB persistence via `Load`/`Save`/`Remove`), `App.Redirect()`, `"/"` fallback wiring in `forge.go` (Amendment A20)
- [x] Step 3 — `redirectmanifest.go`: `/.well-known/redirects.json` — always mounted, live serialisation, reuses `ManifestAuth` option (Amendment A21)
- [x] Step 4 — `integration_full_test.go`: cross-milestone groups G16–G18 (redirect enforcement, prefix rewrite + M2, full M7 stack + manifest + M6 ManifestAuth) + README badge updates

---

## Milestone 8 — Scheduled publishing (v0.8.0)

Adaptive ticker, Scheduled→Published transition, AfterPublish signal.
**Detail:** [Milestone8_BACKLOG.md](Milestone8_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | scheduler.go | ✅ Done | 2026-03-07 |
| 2 | integration_full_test.go | ✅ Done | 2026-03-07 |

- [x] Step 1 — `scheduler.go`: adaptive ticker, fallback 60s, Scheduled→Published, PublishedAt assignment, AfterPublish signal, sitemap+feed trigger, graceful shutdown; + Amendments A23 (db tags on Node), A24 (NewBackgroundContext), A25 (Module.processScheduled), A26 (forge.go wiring)
- [x] Step 2 — `integration_full_test.go`: cross-milestone groups G19–G20 (scheduler end-to-end with MemoryRepo + M1 signals, scheduler + App + sitemap M8+M3+M2) + README badge update

---

## Milestone 9 — v1.0.0 stabilisation

Test coverage, benchmarks, godoc audit, example apps.
**Detail:** [Milestone9_BACKLOG.md](Milestone9_BACKLOG.md)

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | coverage audit (test additions) | ✅ Done | 2026-03-08 |
| 2 | benchmarks_test.go | ✅ Done | 2026-03-08 |
| 3 | forge.go + storage.go (godoc) | ✅ Done | 2026-03-08 |
| 4 | example/blog/ | ✅ Done | 2026-03-08 |
| 5 | example/docs/ | ✅ Done | 2026-03-08 |
| 6 | example/api/ | ✅ Done | 2026-03-08 |
| 7 | CHANGELOG.md + integration_full_test.go G21 | ✅ Done | 2026-03-08 |
| 8 | example_test.go | ✅ Done | 2026-03-08 |

- [x] Step 1 — coverage audit: targeted test additions to reach ≥ 85%; covers `App.RedirectStore`, `TrustedProxy`, `CacheStore.Sweep`, `RedirectStore.Len`, `stripMarkdown`, `forgeLLMSEntries`
- [x] Step 2 — `benchmarks_test.go`: 7 benchmarks for M5–M8 hot paths (auth sign/verify, consent, redirect lookup ×2, scheduler tick noop, feed build)
- [x] Step 3 — godoc pass: `forge.go` `type App` struct comment + all A18–A26 methods; `storage.go` `SQLRepo[T]` method parity with `MemoryRepo[T]`
- [x] Step 4 — `example/blog/`: standalone module with Post type, MemoryRepo, SitemapConfig, Social, FeedConfig, AIIndex, scheduled publishing
- [x] Step 5 — `example/docs/`: standalone module with Doc type, AIIndex/LLMsTxtFull/AIDoc, RobotsConfig AskFirst, breadcrumbs
- [x] Step 6 — `example/api/`: standalone module with Resource type, BearerHMAC+Authenticate, role-based access, BeforeCreate validation, legacy redirects, SecurityHeaders/RateLimit, JSON-only
- [x] Step 7 — `CHANGELOG.md` (Keep a Changelog, v0.1.0–v1.0.0) + `integration_full_test.go` G21 (full v1.0.0 stack: M1+M2+M3+M5+M7+M8)
- [x] Step 8 — `example_test.go`: 7 compile-verified README Example functions (ExampleNewModule, ExampleAuth, ExampleAuthenticate, ExampleAIIndex, ExampleSocial, ExampleOn, ExampleRobotsConfig); README compile test rule added to copilot-instructions.md

**Amendment A42** — `Config.Version` field + `App.Health()` endpoint — ✅ Done 2026-03-12

**Amendment A44** — `dbFields`: flatten embedded struct fields in `storage.go` (`dbField.index` `int` → `[]int`, add `collectDBFields` recursive helper) — ✅ Done 2026-03-15 (v1.0.7)

**Amendment A45** — Default `BearerHMAC` auth wired in `New()` via `Config.Auth` field — ✅ Done 2026-03-15 (v1.0.8)

**Amendment A46** — Minimal Markdown→HTML renderer in `TemplateFuncMap` (`forge_markdown` upgrade) — ✅ Done 2026-03-15 (v1.0.9)

---

## Milestone 10 — MCP support (v2)

Implementation of Decision 19. Syntax reserved in v1 via mcp.go.
**Detail:** Milestone10_BACKLOG.md *(not yet created)*

- [ ] Step 1 — `forge-mcp` module scaffold: MCPServer, resource schema auto-generation from Node + struct tags
- [ ] Step 2 — MCPRead: expose Published content as readable MCP resources with lifecycle enforcement
- [ ] Step 3 — MCPWrite: expose Create/Update/Delete/Publish as MCP tools with role checks
- [ ] Step 4 — Transport: stdio (local AI tools) + SSE (remote, authenticated)
- [ ] Step 5 — Documentation: connecting Claude/Cursor/Copilot to a Forge app

---

## Phase 2 — Production foundation

See [VISION.md](VISION.md) for context. Steps are ordered by dependency
and practical value for forge-cms.dev.

- **Health endpoint + error reporter interface** — `GET /_health` ✅ Done (Amendment A42); `forge.ErrorReporter` interface (plug in third-party error tracking or custom webhooks via `app.Use(...)`) still pending
- **Shared template partials** — `forge.Templates` currently parses a single file; nav/footer must be duplicated across list.html and show.html; add partial directory support or `{{template "include"}}` mechanism; discovered during forge-site templates sprint
- **`forge:head` public helper** — `forgeHeadTmpl` is package-private; `forge.Handle` home handlers cannot use `forge:head`; expose as `forge.HeadPartial(head Head) template.HTML` or equivalent
- **`forge.New` MustConfig enforcement** — `forge.New(forge.Config{...})` without `MustConfig` silently accepts invalid config; make `New` call `MustConfig` internally so validation is not opt-in
- **`forge.AppSchema{}`** — `forge.Handle` routes have no content type and cannot use `SchemaFor`; static pages (home, about) cannot generate Organization or WebSite JSON-LD without hardcoding; add `forge.AppSchema{}` via `app.SEO()` for app-level structured data; discovered during forge-site rich results testing (Amendment S9)
- **`forge.OGDefaults{}`** — no app-level fallback for `og:image`, `twitter:site`, `twitter:creator`; developers must hardcode these in templates; add via `app.SEO()` so defaults are injected automatically; discovered during forge-site OG implementation (Amendment S9)
- **DDoS mitigation for heavy AI endpoints** — `llms-full.txt` and per-item `/{prefix}/{slug}/aidoc` are CPU-intensive under load: large Markdown payloads, optionally gzip-compressed per request. Proposed approach (in priority order): (1) pre-compute and cache the compressed response at generation time — serve from buffer on every request, no per-request compression cost, fits Forge's existing event-driven philosophy; (2) per-endpoint rate limiting with a lower limit on `llms-full.txt` and `/{prefix}/{slug}/aidoc` than on standard content endpoints; (3) concurrent request cap on heavy endpoints, independent of rate limit. Relates to: `forge.RateLimit`, `ai.go` (`compressIfAccepted`).

---

## v2+ Roadmap (not yet planned)

These topics require a new Tier 1 decision round before planning begins.

- **i18n** — locale-aware URLs, hreflang tags, per-locale content
- **Forge AI** — content assistant built on MCP + AIDoc + llms.txt
- **Admin UI** — `forge-studio` as a separate package
- **Search** — SQLite FTS5 integration, `forge.Searchable` interface
- **Webhooks** — outbound HTTP on content events
- **Multi-tenancy** — multiple sites from one instance
- **GraphQL** — auto-generated schema from content types
- **Edge/CDN** — surrogate keys, automatic CDN purge
- **Image resizing** — `forge-images` as a separate package
- **Forge Cloud** — managed hosting, dual-license introduction
- **Database migrations** — `forge migrate` CLI or migration interface
- **Publish-time validation** — `forge:"required_when=published"` tag or `OnPublish` interface; enforces field requirements on `Published` transition without requiring manual `Validate()` implementation; needed before forge-admin
- **Token revocation** — `forge.SignToken` TTL=0 is permanent; only revocation is rotating `Config.Secret` (invalidates all tokens); needs per-token revocation list backed by `forge.DB` or short default TTL + refresh; required before Forge Cloud

---

## Known issues (unfiled)

These are confirmed bugs or sharp edges discovered during real-world usage.
Each will be resolved as a patch or Phase 2 item.

- [ ] **Health endpoint HTTPS redirect** — `forge.Config{HTTPS: true}` causes `GET /_health` to return 301, breaking tools like Caddy's `health_uri` check that follow internal routes and expect 200. Internal health checks should bypass the HTTPS redirect. (Relates to Amendment A42)
- [ ] **SQLite reserved keyword guard** — `SQLRepo` generates unquoted column names; reserved keywords such as `order` cause SQL syntax errors at runtime. Consider quoting all column names in generated SQL (`"order"`), or document the restriction clearly so developers know to avoid reserved words in struct field names.