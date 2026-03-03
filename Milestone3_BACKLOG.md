# Forge — Milestone 3 Backlog (v0.3.0)

SEO metadata, breadcrumbs, JSON-LD structured data, event-driven sitemaps,
and robots.txt — defined once on the content type, rendered correctly everywhere.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | head.go | ✅ Done | 2026-03-03 |
| 2 | schema.go | ✅ Done | 2026-03-03 |
| 3 | sitemap.go | ✅ Done | 2026-03-03 |
| 4 | robots.go | 🔲 Not started | — |

---

## Layer 1 — Head types and helpers (no dependencies beyond stdlib)

### Step 1 — head.go

**Depends on:** node.go (Node, Status), module.go (Option interface — HeadFunc option)
**Decisions:** Decision 3 (Head/SEO ownership), Decision 11 (i18n — Alternate present but empty in v1), Decision 12 (Image type)
**Files:** `head.go`, `head_test.go`

#### 1.1 — forge.Image

- [ ] Define `type Image struct` with fields: `URL string`, `Alt string`, `Width int`, `Height int`
- [ ] Godoc: "Image is a typed image reference. Width and Height are required for optimal Open Graph rendering."
- [ ] No constructor — zero value is safe (empty URL renders no image tags)

#### 1.2 — forge.Alternate

- [ ] Define `type Alternate struct` with fields: `Locale string`, `URL string`
- [ ] Godoc: "Alternate is an hreflang entry for internationalised pages. Reserved for v2 — always empty in v1."

#### 1.3 — forge.Crumb and helpers

- [ ] Define `type Crumb struct` with fields: `Label string`, `URL string`
- [ ] Define `func Crumb(label, url string) Crumb` — constructor; godoc: "Crumb returns a single breadcrumb entry."
- [ ] Note: `Crumb` is both a type and a constructor function — the constructor shadows the type at call sites; verify this compiles and is consistent with Go conventions. If a naming conflict arises, rename type to `BreadcrumbEntry` and keep `Crumb()` as the constructor; document rationale.
- [ ] Define `func Crumbs(crumbs ...Crumb) []Crumb` — godoc: "Crumbs collects breadcrumb entries for use in forge.Head."

#### 1.4 — forge.Head

- [ ] Define `type Head struct` with fields:
  - `Title       string`       — page title; used in `<title>`, `og:title`, JSON-LD
  - `Description string`       — meta description; max 160 chars recommended
  - `Author      string`       — author name; used in `<meta name="author">` and JSON-LD
  - `Published   time.Time`    — publication date; zero value omitted
  - `Modified    time.Time`    — last-modified date; zero value omitted
  - `Image       Image`        — primary image; zero URL omits all image tags
  - `Type        string`       — rich result type constant (Article, Product, etc.); empty = no JSON-LD
  - `Canonical   string`       — canonical URL; empty = no canonical tag
  - `Breadcrumbs []Crumb`      — breadcrumb trail; nil/empty omits BreadcrumbList JSON-LD
  - `Alternates  []Alternate`  — hreflang entries; always empty in v1
  - `NoIndex     bool`         — sets `<meta name="robots" content="noindex">`
- [ ] Godoc: "Head carries all SEO and social metadata for a content page. Define it on your content type via the Headable interface."

#### 1.5 — Rich result type constants

- [ ] Define untyped string constants matching README values:
  ```go
  const (
      Article      = "Article"
      Product      = "Product"
      FAQPage      = "FAQPage"
      HowTo        = "HowTo"
      Event        = "Event"
      Recipe        = "Recipe"
      Review        = "Review"
      Organization = "Organization"
  )
  ```
- [ ] Godoc on each constant: what SEO rich result it maps to

#### 1.6 — Headable interface

- [ ] Define `type Headable interface { Head() Head }`
- [ ] Godoc: "Headable is implemented by content types that provide their own SEO metadata. Forge calls Head() when building HTML responses, sitemaps, and AI endpoints."

#### 1.7 — HeadFunc option

- [ ] Define `type headFuncOption[T any] struct { fn func(Context, T) Head }`
- [ ] Define `func HeadFunc[T any](fn func(Context, T) Head) Option` — returns a typed `headFuncOption`
- [ ] Implement `Option` interface (`applyOption(*moduleConfig)`) on `headFuncOption` — stores the function as `any` in `moduleConfig.headFunc`
- [ ] Add `headFunc any` field to `moduleConfig` in `module.go` — **Amendment required**: this crosses into module.go. Draft Amendment before implementing.
- [ ] Godoc: "HeadFunc overrides the content type's Head() method at the module level. The returned Head wins over the content type's own Head() implementation."

#### 1.8 — Excerpt helper

- [ ] Define `func Excerpt(text string, maxLen int) string`
- [ ] Behaviour: strip leading/trailing whitespace; if `len(text) <= maxLen` return `text`; otherwise truncate at the last word boundary before `maxLen` and append `"…"` (UTF-8 ellipsis, 3 bytes; ensure total byte length ≤ maxLen+3)
- [ ] No reflection, no allocations beyond the result string
- [ ] Godoc: "Excerpt returns a plain-text summary truncated at the last word boundary within maxLen characters. Use it to populate Head.Description."

#### 1.9 — URL helper

- [ ] Define `func URL(parts ...string) string`
- [ ] Behaviour: join parts with `/`, collapse `//`, ensure leading `/`, no trailing `/` (unless root)
- [ ] Godoc: "URL joins path segments into a root-relative URL. Use it to build Head.Canonical."

#### 1.10 — Tests

- [ ] `TestExcerpt` — table-driven: empty string, shorter than maxLen, exact length, truncation at word boundary, no word boundary (hard truncate), Unicode multibyte chars
- [ ] `TestURL` — table-driven: single segment, multiple segments, trailing slash input, empty parts
- [ ] `TestCrumbs` — verifies slice construction and zero value safety
- [ ] `TestHead_zeroValueSafe` — zero-value `Head{}` and `Image{}` must not panic when accessed
- [ ] `TestHeadFunc_implementsOption` — compile-time check that `HeadFunc[any](nil)` satisfies `Option`
- [ ] Benchmark `BenchmarkExcerpt` — must allocate ≤ 1 time per call

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestExcerpt|TestURL|TestCrumbs|TestHead|TestHeadFunc ./...` — all green
- [x] `go test -bench BenchmarkExcerpt ./...` — 1 alloc/op ✅
- [x] `BACKLOG.md` — step table row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required

---

## Layer 2 — JSON-LD structured data (depends on head.go)

### Step 2 — schema.go

**Depends on:** head.go (Image, Head, rich type constants)
**Decisions:** Decision 3 (Head.Type drives JSON-LD selection)
**Files:** `schema.go`, `schema_test.go`

#### 2.1 — JSON-LD base types

- [x] Define internal `ldBase struct` with `Context string \`json:"@context"\`` and `Type string \`json:"@type"\`` — embedded in all schema structs
- [x] All schema structs use `encoding/json` — no external dependencies
- [x] Godoc on package-level: "schema.go provides Go types for Google-supported JSON-LD rich results. Use SchemaFor to generate a <script type=\"application/ld+json\"> block from a forge.Head."

#### 2.2 — Article schema

- [x] Define `type articleSchema struct` with fields mapped to schema.org/Article:
  `Headline`, `Description`, `Author` (personSchema), `DatePublished`, `DateModified`, `Image` (imageSchema), `URL`
- [x] Define `type personSchema struct { Type string; Name string }`
- [x] Define `type imageSchema struct` with URL, Width, Height

#### 2.3 — Product schema

- [x] Define `type productSchema struct`: Name, Description, Image (imageSchema), URL

#### 2.4 — FAQPage schema

- [x] Define `type faqPageSchema struct`: MainEntity []faqEntrySchema
- [x] Define `type faqEntrySchema struct`: Name (question), AcceptedAnswer answerSchema
- [x] Define `type answerSchema struct`: Text string
- [x] Note: FAQPage requires content-type to implement `FAQEntries() []forge.FAQEntry` — define `type FAQEntry struct { Question, Answer string }` and `type FAQProvider interface { FAQEntries() []FAQEntry }`

#### 2.5 — HowTo schema

- [x] Define `type howToSchema struct`: Name, Description, Step []howToStepSchema
- [x] Define `type howToStepSchema struct`: Name, Text string
- [x] Define `type HowToStep struct { Name, Text string }` and `type HowToProvider interface { HowToSteps() []HowToStep }`

#### 2.6 — Event schema

- [x] Define `type eventSchema struct`: Name, Description, StartDate, EndDate, Location (placeSchema), URL, Image (imageSchema)
- [x] Define `type placeSchema struct`: Name, Address string
- [x] Define `type EventDetails struct { StartDate, EndDate time.Time; Location, Address string }` and `type EventProvider interface { EventDetails() EventDetails }`

#### 2.7 — Recipe schema

- [x] Define `type recipeSchema struct`: Name, Description, RecipeIngredient []string, RecipeInstructions []howToStepSchema, Author (personSchema), Image (imageSchema)
- [x] Define `type RecipeDetails struct { Ingredients []string; Steps []HowToStep }` and `type RecipeProvider interface { RecipeDetails() RecipeDetails }`

#### 2.8 — Review schema

- [x] Define `type reviewSchema struct`: Name, ReviewBody, Author (personSchema), ReviewRating (ratingSchema)
- [x] Define `type ratingSchema struct`: RatingValue float64; BestRating float64; WorstRating float64
- [x] Define `type ReviewDetails struct { Body string; Rating, BestRating, WorstRating float64 }` and `type ReviewProvider interface { ReviewDetails() ReviewDetails }`

#### 2.9 — Organization schema

- [x] Define `type organizationSchema struct`: Name, URL, Logo (imageSchema), Description
- [x] Define `type OrganizationDetails struct { Name, URL, Description string; Logo Image }` and `type OrganizationProvider interface { OrganizationDetails() OrganizationDetails }`

#### 2.10 — BreadcrumbList schema

- [x] Define `type breadcrumbListSchema struct`: ItemListElement []breadcrumbItemSchema
- [x] Define `type breadcrumbItemSchema struct`: Position int; Name string; ID (URL) string
- [x] BreadcrumbList is generated automatically from `Head.Breadcrumbs` — no extra interface needed

#### 2.11 — SchemaFor function

- [x] Define `func SchemaFor(head Head, content any) string`
- [x] Returns a `<script type="application/ld+json">...</script>` string (empty string if Head.Type is empty)
- [x] Selection logic (switch on `head.Type`):
  - `Article` → articleSchema populated from Head; if content implements `Headable`, fields already in Head
  - `Product` → productSchema
  - `FAQPage` → requires content to implement `FAQProvider`; returns empty if not
  - `HowTo` → requires `HowToProvider`
  - `Event` → requires `EventProvider`
  - `Recipe` → requires `RecipeProvider`
  - `Review` → requires `ReviewProvider`
  - `Organization` → requires `OrganizationProvider`
- [x] Always appends BreadcrumbList if `len(head.Breadcrumbs) > 0` as a second JSON-LD block
- [x] Use `json.Marshal` — no `html/template` dependency (schema.go must be usable without templates)
- [x] Performance: one `strings.Builder`, two `json.Marshal` calls max; no reflection beyond `json.Marshal`

**Implementation notes:**
- Internal LD types named `ldNode`, `ldPerson`, `ldImage`, `ldArticle`, `ldProduct`, `ldFAQPage`, `ldHowTo`, `ldEvent`, `ldRecipe`, `ldReview`, `ldOrganization`, `ldBreadcrumbList`, `ldBreadcrumbItem`
- `HowToStep` reused between HowTo and Recipe (no duplication)
- Article and Product populated from Head directly — no provider interface needed

#### 2.12 — Tests

- [x] `TestSchemaFor_Article` — non-empty output, valid JSON, correct @type
- [x] `TestSchemaFor_FAQPage` — needs FAQProvider implementation; correct question/answer structure
- [x] `TestSchemaFor_BreadcrumbList` — appended when Breadcrumbs non-empty
- [x] `TestSchemaFor_EmptyType` — returns empty string
- [x] `TestSchemaFor_UnknownType` — returns empty string (graceful)
- [x] `BenchmarkSchemaFor_Article` — baseline alloc count (5975 ns/op, 11 allocs/op)

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestSchemaFor ./...` — all green (14/14)
- [x] `BACKLOG.md` — step table row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required

---

## Layer 3 — Sitemaps (depends on head.go, node.go, signals.go)

### Step 3 — sitemap.go

**Depends on:** head.go (Headable), node.go (Status, Published, PublishedAt), signals.go (Signal constants, debouncer), module.go (Option interface, Module[T])
**Decisions:** Decision 9 (fragment + index, event-driven regeneration)
**Files:** `sitemap.go`, `sitemap_test.go`

**Amendments agreed before implementation:**
- **A2** — `node.go`: add `GetSlug() string`, `GetPublishedAt() time.Time`, `GetStatus() Status` getter methods to `Node`
- **A3** — `module.go`: add `sitemapCfg *SitemapConfig`, `sitemapStore *SitemapStore`, `baseURL string` fields to `Module[T]`; add `setSitemap` method; add `case SitemapConfig` to option loop; register sitemap route in `Register`; hook debouncer to `regenerateSitemap`
- **A4** — `forge.go`: add `sitemapStore *SitemapStore` to `App`; wire `setSitemap` in `Content`; register `GET /sitemap.xml` in `Handler`

#### 3.1 — Amendment A2: Node getter methods (node.go)

- [x] Add `func (n *Node) GetSlug() string { return n.Slug }` with godoc
- [x] Add `func (n *Node) GetPublishedAt() time.Time { return n.PublishedAt }` with godoc
- [x] Add `func (n *Node) GetStatus() Status { return n.Status }` with godoc
- [x] `go build ./...` clean after change

#### 3.2 — Amendment A3: module.go wiring

- [x] Add fields to `Module[T]` struct: `sitemapCfg *SitemapConfig`, `sitemapStore *SitemapStore`, `baseURL string`
- [x] Add `func (m *Module[T]) setSitemap(store *SitemapStore, baseURL string)` method
- [x] Add `case SitemapConfig` to `NewModule` option loop: store pointer copy in `m.sitemapCfg`
- [x] In `Register`: when `m.sitemapCfg != nil && m.sitemapStore != nil`, register `"GET " + m.prefix + "/sitemap.xml"` → `m.sitemapStore.Handler()`
- [x] Replace debouncer no-op fn with `m.regenerateSitemap()`; implement `regenerateSitemap()` as private method: list repo, call `SitemapEntries`, write to `bytes.Buffer`, call `m.sitemapStore.Set`; skip when repo or store is nil
- [x] `go build ./...` clean after change

#### 3.3 — Amendment A4: forge.go wiring

- [x] Add `sitemapStore *SitemapStore` field to `App` struct
- [x] In `App.Content`: after `r.Register`, type-assert `r` against `interface{ setSitemap(*SitemapStore, string) }`; lazily init store; call `setSitemap`
- [x] In `App.Handler`: if `a.sitemapStore != nil`, register `"GET /sitemap.xml"` → `a.sitemapStore.IndexHandler(a.cfg.BaseURL)` before returning
- [x] `go build ./...` clean after change

**Implementation note:** Added `sitemapIndexRegistered bool` guard to `App` to prevent panic on duplicate route registration when `Handler()` is called multiple times (e.g. in tests).

#### 3.4 — ChangeFreq and SitemapConfig (sitemap.go)

- [x] Define `type ChangeFreq string`
- [x] Define constants: `Always`, `Hourly`, `Daily`, `Weekly`, `Monthly`, `Yearly`, `Never` with godoc
- [x] Define `type SitemapConfig struct { ChangeFreq ChangeFreq; Priority float64 }`
- [x] Implement `isOption()` on value receiver `SitemapConfig`
- [x] Godoc: "SitemapConfig configures the per-module sitemap fragment. ChangeFreq defaults to Weekly; Priority defaults to 0.5."

#### 3.5 — SitemapPrioritiser and SitemapNode (sitemap.go)

- [x] Define `type SitemapPrioritiser interface { SitemapPriority() float64 }` with godoc
- [x] Define `type SitemapNode interface { Headable; GetSlug() string; GetPublishedAt() time.Time; GetStatus() Status }` with godoc

#### 3.6 — SitemapEntry and internal XML structs (sitemap.go)

- [x] Define `type SitemapEntry struct { Loc string; LastMod time.Time; ChangeFreq ChangeFreq; Priority float64 }` with godoc
- [x] Define internal `xmlURLSet`, `xmlURL`, `xmlSitemapIndex`, `xmlSitemapRef` with `encoding/xml` tags
- [x] `LastMod` in XML structs stored as `string` (date-only, `omitempty`); zero `time.Time` → empty string → tag omitted

#### 3.7 — WriteSitemapFragment and SitemapEntries (sitemap.go)

- [x] Define `func WriteSitemapFragment(w io.Writer, entries []SitemapEntry) error` — manual XML header + `xml.NewEncoder`; streaming
- [x] Define `func SitemapEntries[T SitemapNode](items []T, baseURL string, cfg SitemapConfig) []SitemapEntry`
  - Skip non-`Published` items
  - `Loc`: `item.Head().Canonical`; fallback `strings.TrimRight(baseURL, "/") + "/" + item.GetSlug()`
  - `ChangeFreq`: `cfg.ChangeFreq` if non-zero, else `Weekly`
  - `Priority`: `SitemapPrioritiser` override → `cfg.Priority` if > 0 → `0.5`

#### 3.8 — WriteSitemapIndex (sitemap.go)

- [x] Define `func WriteSitemapIndex(w io.Writer, fragmentURLs []string, lastMod time.Time) error` — streaming `<sitemapindex>`; valid on empty slice

#### 3.9 — SitemapStore (sitemap.go)

- [x] `type SitemapStore struct { mu sync.RWMutex; fragments map[string][]byte }` with godoc
- [x] `func NewSitemapStore() *SitemapStore`
- [x] `func (s *SitemapStore) Set(path string, data []byte)` — stores copy
- [x] `func (s *SitemapStore) Get(path string) ([]byte, bool)`
- [x] `func (s *SitemapStore) Paths() []string` — sorted keys
- [x] `func (s *SitemapStore) Handler() http.Handler` — serves by `r.URL.Path`; 404 if absent; `application/xml; charset=utf-8`
- [x] `func (s *SitemapStore) IndexHandler(baseURL string) http.Handler` — on each request calls `Paths()`, builds full URLs, calls `WriteSitemapIndex`

#### 3.10 — Tests (sitemap_test.go)

- [x] `TestWriteSitemapFragment` — valid XML, namespace, `<loc>`, date-only `<lastmod>`
- [x] `TestWriteSitemapFragment_empty` — valid `<urlset/>`, no error
- [x] `TestWriteSitemapFragment_zeroLastMod` — `<lastmod>` absent when zero
- [x] `TestSitemapEntries_filtersUnpublished` — only Published items returned
- [x] `TestSitemapEntries_canonicalLoc` — non-empty `Canonical` used as `Loc`
- [x] `TestSitemapEntries_slugFallback` — empty `Canonical` → `baseURL + "/" + slug`
- [x] `TestSitemapEntries_customPriority` — `SitemapPrioritiser` value wins
- [x] `TestWriteSitemapIndex` — valid XML, correct `<sitemap>` count
- [x] `TestSitemapStore_SetGet` — byte round-trip
- [x] `TestSitemapStore_Handler_notFound` — 404
- [x] `TestSitemapStore_Handler_found` — 200, `application/xml`, correct body
- [x] `TestSitemapStore_IndexHandler` — two fragments → valid index with both URLs
- [x] `BenchmarkWriteSitemapFragment` — 100-entry baseline (5919 ns/op, 224 allocs/op)

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -v -run TestWriteSitemap|TestSitemapStore|TestSitemapEntries ./...` — all green (12/12)
- [x] `go test -bench BenchmarkWriteSitemap -benchmem ./...` — recorded (5919 ns/op)
- [x] `go test -count=1 ./...` — full suite green
- [x] `BACKLOG.md` — step table row and summary checkbox updated
- [x] `README.md` — no examples broken by this step
- [x] `ARCHITECTURE.md` — `sitemap.go` moved from Planned to Implemented; `SitemapPrioritiser` name corrected
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required

---

## Layer 4 — Robots.txt (depends on sitemap.go, forge.go)

### Step 4 — robots.go

**Depends on:** sitemap.go (SitemapStore for sitemap URL append), forge.go (Config.BaseURL, App.SEO method)
**Decisions:** Decision 9 (robots.txt references sitemap URL at bottom)
**Files:** `robots.go`, `robots_test.go`

**Amendment required before implementation:**
`robots.go` needs `App.SEO(opts ...SEOOption)` on `forge.App` (in `forge.go`).
Draft and agree the amendment before implementing. The `SEOOption` interface and
the `App.SEO` method will be added to `forge.go` as part of this step.

#### 4.1 — CrawlerPolicy type

- [ ] Define `type CrawlerPolicy string` and constants:
  - `Allow    CrawlerPolicy = "allow"`   — allow all crawlers (default)
  - `Disallow CrawlerPolicy = "disallow"` — disallow all AI crawlers
  - `AskFirst CrawlerPolicy = "ask-first"` — respectful policy: disallow GPTBot, CCBot, anthropic-ai, Claude-Web, PerplexityBot; allow all others
- [ ] Godoc on `AskFirst`: "AskFirst blocks known AI training crawlers while permitting AI assistants that respect the robots.txt contract. Recommended for sites that wish to be indexed by AI search but not scraped for training."

#### 4.2 — RobotsConfig

- [ ] Define `type RobotsConfig struct`:
  - `Disallow   []string`       — paths to disallow (e.g. `"/admin"`)
  - `Sitemaps   bool`           — if true, appends `Sitemap: <baseURL>/sitemap.xml` at end of robots.txt
  - `AIScraper  CrawlerPolicy`  — crawler policy; zero value is `Allow`
- [ ] Implement `SEOOption` interface on `RobotsConfig` (see 4.5 below)

#### 4.3 — SEOOption interface and App.SEO (amendment to forge.go)

- [ ] Define `type SEOOption interface { applySEO(*seoState) }` in `robots.go` (or `forge.go` — decide at amendment time; prefer `forge.go` for discoverability)
- [ ] Define internal `seoState struct { robots *RobotsConfig; sitemap *SitemapConfig }` — holds app-level SEO config
- [ ] Add `seo seoState` field to `App` struct in `forge.go` (amendment)
- [ ] Add `func (a *App) SEO(opts ...SEOOption)` to `forge.go` (amendment)
- [ ] Implement `SEOOption` on `*RobotsConfig`: `func (c *RobotsConfig) applySEO(s *seoState) { s.robots = c }`
- [ ] Implement `SEOOption` on `*SitemapConfig`: `func (c *SitemapConfig) applySEO(s *seoState) { s.sitemap = c }` — enables `app.SEO(forge.SitemapConfig{...})` syntax

#### 4.4 — Robots.txt generation

- [ ] Define `func RobotsTxt(cfg RobotsConfig, baseURL string) string`
  - Writes a well-formed robots.txt string
  - `User-agent: *` block with Disallow entries
  - If `AIScraper == AskFirst`: add separate disallow blocks for GPTBot, CCBot, anthropic-ai, Claude-Web, PerplexityBot
  - If `AIScraper == Disallow`: add `User-agent: *` / `Disallow: /` (applied only to AI-identified crawlers — research current list)
  - If `cfg.Sitemaps && baseURL != ""`: append `Sitemap: <baseURL>/sitemap.xml`
  - Returns the full robots.txt content as a string

#### 4.5 — RobotsTxt HTTP handler

- [ ] Define `func RobotsTxtHandler(cfg RobotsConfig, baseURL string) http.HandlerFunc`
  - Pre-generates the robots.txt string at construction time (not per-request)
  - Serves with `Content-Type: text/plain; charset=utf-8`
  - Cache-Control: `max-age=86400` (1 day)
- [ ] Wire in `App.Run` / `App.Handler` (via amendment to forge.go): if `app.seo.robots != nil`, register `GET /robots.txt` with `RobotsTxtHandler(*app.seo.robots, app.cfg.BaseURL)`

#### 4.6 — Tests

- [ ] `TestRobotsTxt_default` — no disallow, no AI policy: only `User-agent: *\nDisallow:\n`
- [ ] `TestRobotsTxt_disallowPaths` — correct disallow entries
- [ ] `TestRobotsTxt_askFirst` — GPTBot, CCBot, anthropic-ai, Claude-Web, PerplexityBot disallowed; `User-agent: *` allows all
- [ ] `TestRobotsTxt_sitemapAppended` — `Sitemap:` line present when Sitemaps=true
- [ ] `TestRobotsTxtHandler_contentType` — response is `text/plain; charset=utf-8`
- [ ] `TestApp_SEO_implementsOption` — compile check: `RobotsConfig{}` and `SitemapConfig{}` satisfy `SEOOption`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestRobots|TestApp_SEO ./...` — all green
- [ ] `BACKLOG.md` — step table row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step (check `app.SEO(forge.RobotsConfig{...})` example)
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 3

- [ ] `go build ./...` — no errors, no warnings
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test ./...` — all tests green
- [ ] All exported symbols have godoc comments
- [ ] `forge.Head{...}` with all fields renders correctly (tested via SchemaFor and RobotsTxt helpers)
- [ ] `forge.Excerpt` truncates at word boundary with ≤ 1 alloc/op
- [ ] `forge.URL` joins paths correctly (no double slashes, leading slash)
- [ ] `forge.SchemaFor` returns valid JSON-LD for all supported types; empty string for unsupported
- [ ] Fragment sitemaps contain only Published content; index merges all fragments
- [ ] `RobotsTxt` with `AskFirst` disallows known AI training crawlers
- [ ] `App.SEO(forge.SitemapConfig{...}, forge.RobotsConfig{...})` compiles and registers handlers
- [ ] `ARCHITECTURE.md` updated: head.go, schema.go, sitemap.go, robots.go symbols added
- [ ] `README.md` — SEO section examples (`Head()`, `app.SEO(...)`) verified against implementation
- [ ] Post-milestone DRY/performance/security review completed and findings resolved
- [ ] Any deferred steps documented in target milestone with reason
- [ ] Retrospective completed before milestone gate commit
