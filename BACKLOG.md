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
| M5 | Social & AI (v0.5.0) | 🔲 Not started |
| M6 | Cookies & Compliance (v0.6.0) | 🔲 Not started |
| M7 | Redirects (v0.7.0) | 🔲 Not started |
| M8 | Scheduled publishing (v0.8.0) | 🔲 Not started |
| M9 | v1.0.0 stabilisation | 🔲 Not started |
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
| 4 | integration_full_test.go | 🔲 Not started | — |

- [x] Step 1 — `social.go`: Social option, OpenGraph, TwitterCard, card types, SocialOverrides, forge:head OG/Twitter rendering
- [x] Step 2 — `ai.go`: AIIndex option, LLMsTxt, LLMsTxtFull (full markdown corpus, opt-in), AIDoc format, AIDocSummary and Markdownable interfaces, WithoutID option, /llms.txt, /llms-full.txt and /{prefix}/{slug}/aidoc endpoints — **Note:** `Markdownable` migrates here from `module.go` (Amendment A11)
- [x] Step 3 — `feed.go`: opt-in RSS 2.0 per module (Feed option), FeedConfig, FeedDisabled, /{prefix}/feed.xml + /feed.xml aggregate, signal-driven regeneration (Amendment A16)
- [ ] Step 4 — `integration_full_test.go`: cross-milestone groups G9–G12 (Social+SEO, AI+content negotiation, RSS+signals, full M5 stack) + README badge updates

---

## Milestone 6 — Cookies & Compliance (v0.6.0)

Typed cookie declarations, category-enforced consent, compliance manifest.
**Detail:** Milestone6_BACKLOG.md *(not yet created)*

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | cookies.go | 🔲 Not started | — |
| 2 | cookiemanifest.go | 🔲 Not started | — |

- [ ] Step 1 — `cookies.go`: Cookie struct, Necessary/Preferences/Analytics/Marketing categories, SetCookie, SetCookieIfConsented, ReadCookie, ClearCookie, ConsentFor, app.Cookies
- [ ] Step 2 — `cookiemanifest.go`: /.well-known/cookies.json endpoint, ManifestAuth option, consent state storage

---

## Milestone 7 — Redirects (v0.7.0)

Automatic redirect tracking, 410 Gone, chain collapse, inspect endpoint.
**Detail:** Milestone7_BACKLOG.md *(not yet created)*

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | redirects.go | 🔲 Not started | — |
| 2 | redirectmanifest.go | 🔲 Not started | — |

- [ ] Step 1 — `redirects.go`: RedirectEntry, auto-create on slug/prefix rename, 410 on archive/delete, chain collapse, Redirects(From) bulk option, app.Redirect manual route
- [ ] Step 2 — `redirectmanifest.go`: /.well-known/redirects.json inspect endpoint (Editor+)

---

## Milestone 8 — Scheduled publishing (v0.8.0)

Adaptive ticker, Scheduled→Published transition, AfterPublish signal.
**Detail:** Milestone8_BACKLOG.md *(not yet created)*

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | scheduler.go | 🔲 Not started | — |

- [ ] Step 1 — `scheduler.go`: adaptive ticker, fallback polling (60s), Scheduled→Published transition, PublishedAt assignment, AfterPublish signal, sitemap+feed trigger, graceful shutdown coordination

---

## Milestone 9 — v1.0.0 stabilisation

Test coverage, benchmarks, godoc audit, example apps.
**Detail:** Milestone9_BACKLOG.md *(not yet created)*

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | test suite | 🔲 Not started | — |
| 2 | benchmarks | 🔲 Not started | — |
| 3 | godoc audit | 🔲 Not started | — |
| 4 | example apps | 🔲 Not started | — |
| 5 | CHANGELOG.md | 🔲 Not started | — |

- [ ] Step 1 — Full test suite: all packages, minimum 80% coverage
- [ ] Step 2 — Benchmark suite: request throughput, cache hit rate, template render time
- [ ] Step 3 — godoc audit: all exported symbols documented
- [ ] Step 4 — Example apps: example/blog, example/docs, example/api
- [ ] Step 5 — CHANGELOG.md, semantic versioning policy, API stability promise

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


