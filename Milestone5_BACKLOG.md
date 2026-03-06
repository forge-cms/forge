# Forge — Milestone 5 Backlog (v0.5.0)

Open Graph, Twitter Cards, `/llms.txt`, AIDoc endpoints, and RSS feeds.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | social.go | ✅ Done | 2026-03-06 |
| 2 | ai.go | 🔲 Not started | — |
| 3 | feed.go | 🔲 Not started | — |
| 4 | integration_full_test.go | 🔲 Not started | — |

---

## Layer 5 — Social & AI (depends on head.go, templates.go, module.go, templatehelpers.go)

### Step 1 — social.go

**Depends on:** `head.go`, `templates.go`  
**Decisions:** Decision 3 (zero-dependency), Decision 18 (Head as single metadata source)  
**Amendments required before implementation:**
- **A9** — Add `Social SocialOverrides` field to `Head` struct in `head.go`
- **A10** — Extend `forge:head` partial in `templates.go` to render `og:*` / `twitter:*` tags  
**Files:** `social.go`, `social_test.go`

#### 1.1 — Amendment A9: extend Head struct

- [x] Propose and agree Amendment A9: add `Social SocialOverrides` to `Head` in `head.go`
- [x] Add `SocialOverrides` struct with `Twitter TwitterMeta` field to `head.go`
- [x] Add `TwitterMeta` struct with `Card TwitterCardType`, `Creator string` fields to `head.go`
- [x] Add `TwitterCardType` type and constants: `Summary`, `SummaryLargeImage`, `AppCard`, `PlayerCard`
- [x] Add `Tags []string` to `Head` struct (implementation prerequisite for `article:tag` OG rendering and RSS categories)
- [x] Add godoc to all new exported types in `head.go`
- [x] Verify `head.go` compiles clean after A9

#### 1.2 — Amendment A10: extend forge:head partial

- [x] Propose and agree Amendment A10: extend `forge:head` template in `templates.go`
- [x] Add OG block to `forge:head`: `og:title`, `og:description`, `og:url`, `og:type`, `og:image`, `og:image:width`, `og:image:height`, `article:published_time`, `article:author`, `article:tag` (range over Tags)
- [x] Add Twitter block to `forge:head`: `twitter:card`, `twitter:title`, `twitter:description`, `twitter:image`, `twitter:creator` (from `Social.Twitter.Creator` if set)
- [x] OG/Twitter block guarded by `{{if .Title}}`; article:published_time guarded by `{{if gt .Published.Year 1}}`
- [x] Add `forgeRFC3339` helper to `templatehelpers.go` and `TemplateFuncMap()` (avoids backtick/quote escaping in the template constant; used for article:published_time)
- [x] Update `templatehelpers_test.go` `TestTemplateFuncMap_keys` to expect 7 keys
- [x] Update `templates_test.go` `TestTemplates_noIndexMeta` to use `TemplateFuncMap()` (forge:head now requires FuncMap)
- [x] Verify `templates.go` compiles clean after A10

#### 1.3 — Social option types

- [x] Define `SocialFeature` type (`type SocialFeature int`)
- [x] Define `OpenGraph SocialFeature` and `TwitterCard SocialFeature` constants
- [x] Define `Social(features ...SocialFeature) Option` — stores requested features on the module
- [x] `Social()` is declarative; `forge:head` always renders OG/Twitter when `Head.Title` is set — no per-request opt-in required
- [x] Add `social []SocialFeature` field to `Module[T]` struct and `case socialOption` to option switch in `module.go`
- [x] Add godoc to all exported symbols

#### 1.4 — Tests

- [x] `TestSocialOption` — all four combinations (OpenGraph, TwitterCard, both, empty) return `socialOption`
- [x] `TestForgeHeadOGRendering` — Head with Title+Image produces `og:title`, `og:image`, `og:image:width`, `og:url`, `twitter:title`, `twitter:image`, default `twitter:card`
- [x] `TestForgeHeadTwitterRendering` — explicit `Social.Twitter.Card` and Creator propagate; no-image fallback produces `summary`
- [x] `TestForgeHeadArticleMeta` — `Type: Article` produces `article:author`, `article:tag` ×2, `article:published_time`, correct `og:type`
- [x] `TestForgeHeadNoOGWithoutTitle` — OG and Twitter blocks absent when `Head.Title` is empty
- [x] All tests table-driven with `t.Run`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestSocial ./...` — all green
- [x] `go test -v -run TestForgeHead ./...` — all green
- [x] `go test ./...` — full suite green
- [x] `BACKLOG.md` — step 1 row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] `README.md` — no section badges update needed (Social badge updated in Pre-M5 commit; remains 🔲 until Step 4 finalises the feature)
- [x] `integration_full_test.go` — N/A (final step only)
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — ARCHITECTURE.md updated; no new Decisions required

---

### Step 2 — ai.go

**Depends on:** `social.go`, `module.go`, `templatehelpers.go`  
**Decisions:** Decision 3 (zero-dependency), Decision 18 (Head as metadata source), Decision 19 (MCP reserved)  
**Amendments required before implementation:**
- **A11** — Migrate `Markdownable` interface from `module.go` to `ai.go`
- **A12** — Wire `forgeLLMSEntries()` stub in `templatehelpers.go` to the real implementation in `ai.go`  
**Files:** `ai.go`, `ai_test.go`

#### 2.1 — Amendment A11: migrate Markdownable

- [ ] Propose and agree Amendment A11: move `Markdownable` interface definition from `module.go` to `ai.go`
- [ ] Copy `Markdownable` interface to `ai.go` (same `forge` package — no import change needed)
- [ ] Remove `Markdownable` from `module.go`
- [ ] Verify no compilation errors — all usages still resolve within the same package

#### 2.2 — Amendment A12: wire forgeLLMSEntries

- [ ] Propose and agree Amendment A12: replace the `forgeLLMSEntries()` stub in `templatehelpers.go` with a real implementation that calls the AI registry
- [ ] Define `aiRegistry` package-level store in `ai.go` (sync.Map keyed by module prefix)
- [ ] `forgeLLMSEntries(data any)` in `templatehelpers.go` calls into `ai.go` to produce the lllms.txt entries string
- [ ] Verify stub replacement compiles and returns non-empty string when modules are registered with `AIIndex`

#### 2.3 — AIIndex option and types

- [ ] Define `AIFeature` type
- [ ] Define `LLMsTxt AIFeature` and `AIDoc AIFeature` constants
- [ ] Define `AIIndex(features ...AIFeature) Option` — registers the module in the AI registry
- [ ] Define `AIDocSummary` interface: `AISummary() string` (short human-readable summary for llms.txt)
- [ ] `Markdownable` is defined here (migrated from module.go via A11)
- [ ] Add `WithoutID() Option` — suppresses UUID from AIDoc output (for content types where ID leakage is undesirable)

#### 2.4 — /llms.txt endpoint

- [ ] `app.Handle("GET /llms.txt", ...)` registered at startup when at least one module uses `AIIndex(LLMsTxt)`
- [ ] If `templates/llms.txt` exists, render it as a template with `TemplateData`; otherwise use the built-in format
- [ ] Built-in format: site name header, description, per-module section listing Published items with title, slug, and summary (via `AISummary()` if available)
- [ ] Only `Published` content appears; Draft/Scheduled/Archived are excluded
- [ ] Regenerate on `AfterPublish` and `AfterArchive` signals (debounced, same pattern as sitemap)
- [ ] `forgeLLMSEntries(data)` template helper reads from `aiRegistry` — used in custom templates

#### 2.5 — /{prefix}/{slug}.aidoc endpoint

- [ ] Registered per-module when `AIIndex(AIDoc)` is set
- [ ] Returns AIDoc format (text/plain, gzip at transport layer via standard Go `http.ResponseWriter`)
- [ ] AIDoc format:
  ```
  +++aidoc+v1+++
  type:     {Head.Type}
  id:       {Node.ID}      (omitted when WithoutID() is set)
  slug:     {Node.Slug}
  title:    {Head.Title}
  author:   {Head.Author}  (omitted if empty)
  created:  {Node.CreatedAt YYYY-MM-DD}
  modified: {Node.UpdatedAt YYYY-MM-DD}
  tags:     [{Head.Tags comma-separated}]  (omitted if empty)
  summary:  {AISummary() if Markdownable, else Excerpt(description, 200)}
  +++
  {Markdown() if Markdownable, else plain JSON body}
  ```
- [ ] Returns 404 for non-Published content (same lifecycle enforcement as normal GET)
- [ ] Content-Type: `text/plain; charset=utf-8`

#### 2.6 — Tests

- [ ] `TestAIIndexOption` — AIIndex(LLMsTxt), AIIndex(AIDoc), AIIndex(LLMsTxt, AIDoc) register without error
- [ ] `TestLLMsTxtEndpoint` — GET /llms.txt returns 200 with Published items; Draft items absent
- [ ] `TestLLMsTxtTemplate` — custom `templates/llms.txt` is rendered when present
- [ ] `TestAIDocEndpoint` — GET /posts/hello.aidoc returns correct AIDoc format for Published item
- [ ] `TestAIDocNotFound` — Draft item returns 404 on .aidoc endpoint
- [ ] `TestAIDocWithoutID` — WithoutID() suppresses `id:` line from AIDoc output
- [ ] `TestMarkdownable` — content type implementing Markdown() returns markdown body in AIDoc
- [ ] All tests table-driven with `t.Run`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestAI ./...` — all green
- [ ] `go test -v -run TestLLMS ./...` — all green
- [ ] `go test -v -run TestAIDoc ./...` — all green
- [ ] `BACKLOG.md` — step 2 row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] `README.md` — section status badges updated if this step ships a documented feature
- [ ] `integration_full_test.go` — new cross-milestone groups added (final step of each milestone only)
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

### Step 3 — feed.go

**Depends on:** `ai.go`, `sitemap.go`, `module.go`, `signals.go`  
**Decisions:** Decision 3 (zero-dependency), Decision 10 (event-driven regeneration)  
**Files:** `feed.go`, `feed_test.go`

#### 3.1 — FeedConfig types

- [ ] Define `FeedConfig` struct with fields: `Title string`, `Description string`, `Language string` (default `"en"`)
- [ ] Define `Feed(config FeedConfig) Option` — enables RSS for the module
- [ ] Define `FeedDisabled() Option` — explicit opt-out (for modules where RSS makes no sense)
- [ ] RSS is opt-in per module — no feed generated unless `Feed(...)` is set
- [ ] Add godoc to all exported symbols

#### 3.2 — RSS generation

- [ ] `feedStore` package-level store (sync.Map keyed by module prefix, same pattern as `sitemapStore`)
- [ ] Per-module RSS at `/{prefix}/feed.xml` — generated at startup and on signals
- [ ] App-level index at `/feed.xml` — lists all module feeds (only if at least one module has `Feed` set)
- [ ] RSS 2.0 format: `<rss version="2.0">`, `<channel>` with title/description/link/language, `<item>` per Published node
- [ ] `<item>` fields: `<title>`, `<link>` (canonical from Head), `<description>` (Head.Description), `<pubDate>` (RFC1123Z from PublishedAt), `<guid isPermaLink="true">` (canonical URL), `<enclosure>` for Image if present, `<category>` per tag
- [ ] Content-Type: `application/rss+xml; charset=utf-8`
- [ ] Use zero-dependency XML generation (format string or `encoding/xml` from stdlib)

#### 3.3 — Signal-driven regeneration

- [ ] Subscribe to `AfterPublish` and `AfterArchive` signals on every module where `Feed` is set
- [ ] Debounce regeneration (same `debounce` helper used by sitemap — reuse, do not duplicate)
- [ ] Regeneration updates both the module fragment and the app-level index
- [ ] Startup generation: generate all feeds after all modules are registered (same pattern as sitemap)

#### 3.4 — Lifecycle enforcement

- [ ] Only `Published` content appears in feeds
- [ ] `Draft`, `Scheduled`, and `Archived` nodes are excluded (same filter as sitemap)
- [ ] Sitemap lifecycle table in README already covers RSS — no README update needed for this rule

#### 3.5 — Tests

- [ ] `TestFeedOption` — Feed(FeedConfig{...}) registers without error; FeedDisabled() sets opt-out flag
- [ ] `TestFeedEndpoint` — GET /posts/feed.xml returns 200 with RSS 2.0 document
- [ ] `TestFeedContainsPublishedOnly` — Draft item absent from feed; Published item present
- [ ] `TestFeedSignalRegeneration` — publishing a new item triggers feed regeneration
- [ ] `TestFeedIndexEndpoint` — GET /feed.xml lists all module feeds when multiple modules have Feed set
- [ ] `TestFeedEnclosure` — item with Image produces `<enclosure>` element
- [ ] All tests table-driven with `t.Run`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestFeed ./...` — all green
- [ ] `BACKLOG.md` — step 3 row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] `README.md` — section status badges updated if this step ships a documented feature
- [ ] `integration_full_test.go` — new cross-milestone groups added (final step of each milestone only)
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

### Step 4 — integration_full_test.go (cross-milestone groups G9–G12 + README badges)

**Depends on:** `social.go`, `ai.go`, `feed.go`  
**Decisions:** All M5 decisions  
**Files:** `integration_full_test.go` (extend only — append new groups, never renumber existing)

#### 4.1 — G9: Social + SEO cross-milestone group

- [ ] G9 exercises `forge.Social(forge.OpenGraph, forge.TwitterCard)` on a module that also has `forge.SitemapConfig` (M3) set
- [ ] Assert rendered HTML contains both `<meta property="og:title"` and existing JSON-LD from M3 schema
- [ ] Assert that a Draft item does NOT produce OG tags (lifecycle enforcement)

#### 4.2 — G10: AI indexing + content negotiation cross-milestone group

- [ ] G10 exercises `forge.AIIndex(forge.LLMsTxt, forge.AIDoc)` on a module from M2 (App Bootstrap)
- [ ] Assert GET /llms.txt returns 200 with Published item and excludes Draft item
- [ ] Assert GET /{slug}.aidoc returns correct AIDoc format for Published item
- [ ] Assert GET /{slug}.aidoc returns 404 for Draft item
- [ ] Assert `Accept: text/markdown` still works (M4 content negotiation) alongside AIDoc

#### 4.3 — G11: RSS feed + signals cross-milestone group

- [ ] G11 exercises `forge.Feed(forge.FeedConfig{...})` on a module that also uses M1 signals
- [ ] Assert GET /posts/feed.xml returns 200 with RSS 2.0 document after setup
- [ ] Assert publishing a new item (via signal) causes feed to include the new item
- [ ] Assert Draft item absent from feed; assert `AfterPublish` signal fires

#### 4.4 — G12: Full M5 stack cross-milestone group

- [ ] G12 wires a single module with Social + AIIndex + Feed together with M3 SEO + M4 Templates
- [ ] Assert forge:head renders OG, Twitter, JSON-LD, canonical, and robots meta in a single HTML response
- [ ] Assert /llms.txt, /{slug}.aidoc, and /posts/feed.xml all return valid responses
- [ ] Assert the sitemap (M3) and RSS feed (M5) both contain the same Published item

#### 4.5 — README badge updates

- [ ] Update `README.md` `## Social sharing` badge: `🔲 **Coming in Milestone 5**` → `✅ **Available**`
- [ ] Update `README.md` `## AI indexing` badge: `🔲 **Coming in Milestone 5**` → `✅ **Available**`
- [ ] Verify no other section badges need updating for M5 features (Cookies M6, Redirects M7, Scheduled M8 remain 🔲)

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestIntegrationFull ./...` — all G1–G12 green
- [ ] `BACKLOG.md` — step 4 row and summary checkbox updated; M5 milestone row marked ✅
- [ ] `README.md` — Social and AI indexing badges updated to ✅ Available
- [ ] `README.md` — section status badges updated if this step ships a documented feature
- [ ] `integration_full_test.go` — G9–G12 added and all passing
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 5

- [ ] All four steps pass `go test ./...` — green
- [ ] `forge:head` renders `og:*` and `twitter:*` tags when `Head.Title` is set
- [ ] `/llms.txt` serves Published content index; Draft/Scheduled/Archived absent
- [ ] `/{prefix}/{slug}.aidoc` returns AIDoc format for Published; 404 for non-Published
- [ ] `/{prefix}/feed.xml` and `/feed.xml` return RSS 2.0; regenerate on publish
- [ ] `Markdownable` is in `ai.go`, not `module.go`
- [ ] `forgeLLMSEntries()` is wired to real implementation (no longer a stub)
- [ ] `integration_full_test.go` — new cross-milestone groups G9–G12 added and all passing
- [ ] `README.md` — Social sharing and AI indexing section badges updated to `✅ **Available**`
