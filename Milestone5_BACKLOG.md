# Forge — Milestone 5 Backlog (v0.5.0)

Open Graph, Twitter Cards, `/llms.txt`, AIDoc endpoints, and RSS feeds.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | social.go | ✅ Done | 2026-03-06 |
| 2 | ai.go | ✅ Done | 2026-03-06 |
| 3 | feed.go | ✅ Done | 2026-03-06 |
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
- **A15** — AIDoc URL: `/{prefix}/{slug}/aidoc` (not `.aidoc`); Go 1.22 ServeMux disallows partial wildcard segments  
**Files:** `ai.go`, `ai_test.go`

#### 2.1 — Amendment A11: migrate Markdownable

- [x] Propose and agree Amendment A11: move `Markdownable` interface definition from `module.go` to `ai.go`
- [x] Copy `Markdownable` interface to `ai.go` (same `forge` package — no import change needed)
- [x] Remove `Markdownable` from `module.go`
- [x] Verify no compilation errors — all usages still resolve within the same package

#### 2.2 — Amendment A12: wire forgeLLMSEntries

- [x] Propose and agree Amendment A12: replace the `forgeLLMSEntries()` stub in `templatehelpers.go` with a real implementation
- [x] `forgeLLMSEntries(data any) template.HTML` in `templatehelpers.go` formats `LLMsTemplateData.Entries`
- [x] Verify stub replacement compiles and returns non-empty string when entries are present in `LLMsTemplateData`

#### 2.3 — AIIndex option and types

- [x] Define `AIFeature` type
- [x] Define `LLMsTxt AIFeature` constant — enables /llms.txt compact index
- [x] Define `LLMsTxtFull AIFeature` constant — enables /llms-full.txt full markdown corpus (opt-in)
- [x] Define `AIDoc AIFeature` constant — enables /{prefix}/{slug}/aidoc per-item endpoints
- [x] Define `AIIndex(features ...AIFeature) Option` — registers the module in the AI registry
- [x] Define `AIDocSummary` interface: `AISummary() string` (short human-readable summary for llms.txt)
- [x] `Markdownable` is defined here (migrated from module.go via A11)
- [x] Add `WithoutID() Option` — suppresses UUID from AIDoc output (for content types where ID leakage is undesirable)

#### 2.4 — /llms.txt endpoint

- [x] `GET /llms.txt` registered in `App.Handler()` when `LLMsStore.HasCompact()` is true
- [x] Built-in format: site name header + per-item entries as `- [Title](URL): Summary`
- [x] Only `Published` content appears; Draft/Scheduled/Archived are excluded
- [x] Regenerate debounced on any write event (same debouncer as sitemap, `regenerateAI` called from callback)
- [x] `forgeLLMSEntries(data any)` template helper formats `LLMsTemplateData.Entries` for custom templates

#### 2.5 — /llms-full.txt endpoint

- [x] `GET /llms-full.txt` registered in `App.Handler()` when `LLMsStore.HasFull()` is true
- [x] Corpus header format: `# {SiteName} — Full Content Corpus` followed by `> Generated by Forge on {YYYY-MM-DD} | Only published content | {N} items`
- [x] Per-item format: `## {Title}`, `URL: {canonical}`, `Published: {YYYY-MM-DD}`, blank line, `Markdown()` if Markdownable else `Head.Description`, `---`
- [x] Only `Published` content appears; Draft/Scheduled/Archived excluded
- [x] Regenerate on any write event (debounced, same pattern as sitemap and /llms.txt)
- [x] Content-Type: `text/plain; charset=utf-8`

#### 2.6 — /{prefix}/{slug}/aidoc endpoint

- [x] Registered per-module when `AIIndex(AIDoc)` is set, at `/{prefix}/{slug}/aidoc` (Amendment A15)
- [x] Returns AIDoc format (text/plain)
- [x] AIDoc format:
  ```
  +++aidoc+v1+++
  type:     {Head.Type or "article"}
  id:       {Node.ID}      (omitted when WithoutID() is set)
  slug:     {Node.Slug}
  title:    {Head.Title}
  author:   {Head.Author}  (omitted if empty)
  created:  {Node.CreatedAt YYYY-MM-DD}
  modified: {Node.UpdatedAt YYYY-MM-DD}
  tags:     [{Head.Tags comma-separated}]  (omitted if empty)
  summary:  {AISummary() non-empty, else Head.Description, else Excerpt(Markdown,120)}
  +++
  {Markdown() if Markdownable, else JSON body}
  ```
- [x] Returns 404 for non-Published content (same lifecycle enforcement as normal GET)
- [x] Content-Type: `text/plain; charset=utf-8`

#### 2.7 — Tests

- [x] `TestAIIndexOption` — AIIndex(LLMsTxt), AIIndex(AIDoc), AIIndex(LLMsTxt, LLMsTxtFull, AIDoc) register without error
- [x] `TestWithoutIDOption` — WithoutID() returns withoutIDOption and sets m.withoutID on module
- [x] `TestLLMsTxtEndpoint` — GET /llms.txt returns 200 with Published items; Draft items absent
- [x] `TestLLMsTxtTemplate` — LLMsStore.CompactHandler formats entries in llmstxt.org convention
- [x] `TestLLMsFullTxtEndpoint` — GET /llms-full.txt returns 200 with Published items in full corpus format
- [x] `TestLLMsFullTxtFallback` — /llms-full.txt returns 404 when no module registers LLMsTxtFull
- [x] `TestLLMsFullTxtHeader` — corpus header matches `# {SiteName} — Full Content Corpus` and `> Generated by Forge…`
- [x] `TestAIDocEndpoint` — GET /posts/{slug}/aidoc returns correct AIDoc format for Published item
- [x] `TestAIDocNotFound` — Draft item returns 404 on /aidoc endpoint
- [x] `TestAIDocWithoutID` — WithoutID() suppresses `id:` line from AIDoc output
- [x] All tests table-driven with `t.Run`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestAI ./...` — all green
- [x] `go test -v -run TestLLMs ./...` — all green (both compact and full endpoints)
- [x] `go test -v -run TestAIDoc ./...` — all green
- [x] `BACKLOG.md` — step 2 row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] `README.md` — LLMsTxtFull one-liner opt-in example added (Amendment A14)
- [x] `integration_full_test.go` — N/A (final step only)
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — Amendment A15 (AIDoc URL) drafted and documented

---

### Step 3 — feed.go

**Depends on:** `ai.go`, `sitemap.go`, `module.go`, `signals.go`  
**Decisions:** Decision 3 (zero-dependency), Decision 10 (event-driven regeneration), Decision 13 (RSS feeds)  
**Amendments required before implementation:**
- **A16** — Decision 13 updated: RSS is opt-in (`Feed(FeedConfig{...})` required), not auto-generated. `FeedDisabled()` retained as explicit marker.  
**Files:** `feed.go`, `feed_test.go`

#### 3.1 — Amendment A16 + FeedConfig types

- [x] Draft and agree Amendment A16 in `DECISIONS.md` — Decision 13: change auto-generate to opt-in; document rationale (no surprise endpoints on admin/API-only modules)
- [x] Define `FeedConfig` struct: `Title string`, `Description string`, `Language string`
- [x] Define `feedOption struct{ cfg FeedConfig }` + `func (feedOption) isOption()` + `Feed(cfg FeedConfig) Option`
- [x] Define `feedDisabledOption struct{}` + `func (feedDisabledOption) isOption()` + `FeedDisabled() Option`
- [x] Add godoc to all exported symbols

#### 3.2 — RSS XML types (`encoding/xml`)

- [x] Define `rssGUID struct{ IsPermaLink bool \`xml:\"isPermaLink,attr\"\`; Value string \`xml:\",chardata\"\`` }`
- [x] Define `rssEnclosure struct{ URL, Length, Type string }` with `xml` attr tags
- [x] Define `rssItem struct` with `xml` struct tags: `Title`, `Link`, `Description`, `PubDate` (RFC1123Z), `GUID rssGUID`, `Enclosure *rssEnclosure` (omitempty), `Author` (omitempty), `Categories []string \`xml:\"category\"\``
- [x] Define `rssChannel struct`: `Title`, `Link`, `Description`, `Language`, `LastBuildDate`, `Items []rssItem \`xml:\"item\"\``
- [x] Define `rssRoot struct{ XMLName xml.Name \`xml:\"rss\"\`; Version string \`xml:\"version,attr\"\`; Channel rssChannel \`xml:\"channel\"\`` }`
- [x] `buildRSSItem(head Head, n Node, baseURL string) rssItem` — maps fields; canonical URL = Head.Canonical else baseURL+prefix+"/"+slug; `<enclosure>` when Head.Image.URL non-empty; `<category>` per tag; pubDate = Node.PublishedAt.Format(time.RFC1123Z)

#### 3.3 — FeedStore

- [x] Define `FeedStore struct`: `sync.RWMutex`, `siteName string`, `baseURL string`, `fragments map[string][]rssItem` (keyed by prefix), `configs map[string]FeedConfig`
- [x] `NewFeedStore(siteName, baseURL string) *FeedStore`
- [x] `(s *FeedStore) Set(prefix string, cfg FeedConfig, items []rssItem)` — stores fragment under lock
- [x] `(s *FeedStore) HasFeeds() bool` — returns len(s.fragments) > 0; used by `App.Handler()` guard
- [x] `(s *FeedStore) ModuleHandler(prefix string) http.Handler` — marshals stored items for that prefix; `Content-Type: application/rss+xml; charset=utf-8`
- [x] `(s *FeedStore) IndexHandler() http.Handler` — merges all fragments, sorts by PubDate descending, marshals into one aggregate channel; `Content-Type: application/rss+xml; charset=utf-8`
- [x] Channel title for module handler: `FeedConfig.Title` if non-empty, else `capitalise(strings.TrimLeft(prefix, "/"))` (ASCII helper, zero-dep)
- [x] Channel title for index: `siteName` (hostname)

#### 3.4 — module.go wiring (A16)

- [x] Add `feedCfg *FeedConfig` and `feedStore *FeedStore` fields to `Module[T]` struct
- [x] Add `case feedOption: cfg := v.cfg; m.feedCfg = &cfg` and `case feedDisabledOption: m.feedCfg = nil` to the option switch in `NewModule`
- [x] Add `m.regenerateFeed(ctx)` to the debouncer callback after `m.regenerateAI(ctx)`
- [x] Add `GET /{prefix}/feed.xml` route in `Register()` when `m.feedCfg != nil && m.feedStore != nil`
- [x] Add `setFeedStore(store *FeedStore, baseURL string)` method — sets `m.feedStore`, sets `m.baseURL` if not already set, calls `store.Set(m.prefix, *m.feedCfg, nil)` to register the prefix
- [x] Add `regenerateFeed(ctx Context)` method — guards on `m.feedStore == nil || m.feedCfg == nil`; queries Published; builds `[]rssItem`; calls `m.feedStore.Set(m.prefix, *m.feedCfg, items)`

#### 3.5 — forge.go wiring (A16)

- [x] Add `feedStore *FeedStore` and `feedIndexRegistered bool` to `App` struct
- [x] In `Content()`: add interface check block `interface{ setFeedStore(*FeedStore, string) }` — lazy-inits `NewFeedStore(hostname, baseURL)` on first call; calls `setFeedStore`
- [x] In `Handler()`: add guard — `if a.feedStore != nil && a.feedStore.HasFeeds() && !a.feedIndexRegistered { ... mount GET /feed.xml ... }`

#### 3.6 — Tests

- [x] `TestFeedOption` — `Feed(FeedConfig{...})` sets feedCfg on module; `FeedDisabled()` clears feedCfg
- [x] `TestFeedEndpoint` — GET /posts/feed.xml returns 200, `Content-Type: application/rss+xml`, `<rss version="2.0">`
- [x] `TestFeedContainsPublishedOnly` — Published item in feed body; Draft item absent
- [x] `TestFeedFields` — title, link, description, pubDate, guid, author, category correct in `<item>`
- [x] `TestFeedEnclosure` — item with non-empty `Head.Image.URL` produces `<enclosure>` element
- [x] `TestFeedIndexEndpoint` — GET /feed.xml merges items from all Feed-enabled modules
- [x] `TestFeedDefaultTitle` — empty `FeedConfig.Title` defaults to capitalised prefix (e.g. `/posts` → `"Posts"`)
- [x] All tests table-driven with `t.Run`

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestFeed ./...` — all green
- [x] `BACKLOG.md` — step 3 row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] `README.md` — section status badges updated if this step ships a documented feature
- [x] `integration_full_test.go` — N/A (final step only)
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — Amendment A16 drafted and agreed upon

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
- [ ] Assert GET /{slug}/aidoc returns correct AIDoc format for Published item
- [ ] Assert GET /{slug}/aidoc returns 404 for Draft item
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
