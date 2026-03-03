# Forge — Milestone 3 Backlog (v0.3.0)

SEO metadata, breadcrumbs, JSON-LD structured data, event-driven sitemaps,
and robots.txt — defined once on the content type, rendered correctly everywhere.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | head.go | ✅ Done | 2026-03-03 |
| 2 | schema.go | 🔲 Not started | — |
| 3 | sitemap.go | 🔲 Not started | — |
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

- [ ] Define internal `ldBase struct` with `Context string \`json:"@context"\`` and `Type string \`json:"@type"\`` — embedded in all schema structs
- [ ] All schema structs use `encoding/json` — no external dependencies
- [ ] Godoc on package-level: "schema.go provides Go types for Google-supported JSON-LD rich results. Use SchemaFor to generate a <script type=\"application/ld+json\"> block from a forge.Head."

#### 2.2 — Article schema

- [ ] Define `type articleSchema struct` with fields mapped to schema.org/Article:
  `Headline`, `Description`, `Author` (personSchema), `DatePublished`, `DateModified`, `Image` (imageSchema), `URL`
- [ ] Define `type personSchema struct { Type string; Name string }`
- [ ] Define `type imageSchema struct` with URL, Width, Height

#### 2.3 — Product schema

- [ ] Define `type productSchema struct`: Name, Description, Image (imageSchema), URL

#### 2.4 — FAQPage schema

- [ ] Define `type faqPageSchema struct`: MainEntity []faqEntrySchema
- [ ] Define `type faqEntrySchema struct`: Name (question), AcceptedAnswer answerSchema
- [ ] Define `type answerSchema struct`: Text string
- [ ] Note: FAQPage requires content-type to implement `FAQEntries() []forge.FAQEntry` — define `type FAQEntry struct { Question, Answer string }` and `type FAQProvider interface { FAQEntries() []FAQEntry }`

#### 2.5 — HowTo schema

- [ ] Define `type howToSchema struct`: Name, Description, Step []howToStepSchema
- [ ] Define `type howToStepSchema struct`: Name, Text string
- [ ] Define `type HowToStep struct { Name, Text string }` and `type HowToProvider interface { HowToSteps() []HowToStep }`

#### 2.6 — Event schema

- [ ] Define `type eventSchema struct`: Name, Description, StartDate, EndDate, Location (placeSchema), URL, Image (imageSchema)
- [ ] Define `type placeSchema struct`: Name, Address string
- [ ] Define `type EventDetails struct { StartDate, EndDate time.Time; Location, Address string }` and `type EventProvider interface { EventDetails() EventDetails }`

#### 2.7 — Recipe schema

- [ ] Define `type recipeSchema struct`: Name, Description, RecipeIngredient []string, RecipeInstructions []howToStepSchema, Author (personSchema), Image (imageSchema)
- [ ] Define `type RecipeDetails struct { Ingredients []string; Steps []HowToStep }` and `type RecipeProvider interface { RecipeDetails() RecipeDetails }`

#### 2.8 — Review schema

- [ ] Define `type reviewSchema struct`: Name, ReviewBody, Author (personSchema), ReviewRating (ratingSchema)
- [ ] Define `type ratingSchema struct`: RatingValue float64; BestRating float64; WorstRating float64
- [ ] Define `type ReviewDetails struct { Body string; Rating, BestRating, WorstRating float64 }` and `type ReviewProvider interface { ReviewDetails() ReviewDetails }`

#### 2.9 — Organization schema

- [ ] Define `type organizationSchema struct`: Name, URL, Logo (imageSchema), Description
- [ ] Define `type OrganizationDetails struct { Name, URL, Description string; Logo Image }` and `type OrganizationProvider interface { OrganizationDetails() OrganizationDetails }`

#### 2.10 — BreadcrumbList schema

- [ ] Define `type breadcrumbListSchema struct`: ItemListElement []breadcrumbItemSchema
- [ ] Define `type breadcrumbItemSchema struct`: Position int; Name string; ID (URL) string
- [ ] BreadcrumbList is generated automatically from `Head.Breadcrumbs` — no extra interface needed

#### 2.11 — SchemaFor function

- [ ] Define `func SchemaFor(head Head, content any) string`
- [ ] Returns a `<script type="application/ld+json">...</script>` string (empty string if Head.Type is empty)
- [ ] Selection logic (switch on `head.Type`):
  - `Article` → articleSchema populated from Head; if content implements `Headable`, fields already in Head
  - `Product` → productSchema
  - `FAQPage` → requires content to implement `FAQProvider`; returns empty if not
  - `HowTo` → requires `HowToProvider`
  - `Event` → requires `EventProvider`
  - `Recipe` → requires `RecipeProvider`
  - `Review` → requires `ReviewProvider`
  - `Organization` → requires `OrganizationProvider`
- [ ] Always appends BreadcrumbList if `len(head.Breadcrumbs) > 0` as a second JSON-LD block
- [ ] Use `json.Marshal` — no `html/template` dependency (schema.go must be usable without templates)
- [ ] Performance: one `strings.Builder`, two `json.Marshal` calls max; no reflection beyond `json.Marshal`

#### 2.12 — Tests

- [ ] `TestSchemaFor_Article` — non-empty output, valid JSON, correct @type
- [ ] `TestSchemaFor_FAQPage` — needs FAQProvider implementation; correct question/answer structure
- [ ] `TestSchemaFor_BreadcrumbList` — appended when Breadcrumbs non-empty
- [ ] `TestSchemaFor_EmptyType` — returns empty string
- [ ] `TestSchemaFor_UnknownType` — returns empty string (graceful)
- [ ] `BenchmarkSchemaFor_Article` — baseline alloc count

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestSchemaFor ./...` — all green
- [ ] `BACKLOG.md` — step table row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

---

## Layer 3 — Sitemaps (depends on head.go, node.go, signals.go)

### Step 3 — sitemap.go

**Depends on:** head.go (Headable), node.go (Status, Published, PublishedAt), signals.go (Signal constants), module.go (Option interface)
**Decisions:** Decision 9 (Sitemap strategy — fragment + index, event-driven regeneration)
**Files:** `sitemap.go`, `sitemap_test.go`

**Amendment required before implementation:**
`sitemap.go` needs to add `SitemapConfig` as a module option stored in `moduleConfig`, and register
the fragment sitemap handler in `Module[T].Register()`. This requires adding fields to `module.go`.
Draft the amendment, get approval, then implement both changes in this step.
The amendment to `module.go` is an allowed cross-file change within this step's scope.

#### 3.1 — SitemapConfig and ChangeFreq

- [ ] Define `type ChangeFreq string` and constants: `Always`, `Hourly`, `Daily`, `Weekly`, `Monthly`, `Yearly`, `Never`
- [ ] Define `type SitemapConfig struct`:
  - `ChangeFreq ChangeFreq` — default `Weekly` when zero
  - `Priority   float64`   — 0.0–1.0; default `0.5` when zero
- [ ] Implement `Option` interface on `SitemapConfig` (`applyOption(*moduleConfig)`) — stores config in `moduleConfig.sitemapConfig`
- [ ] Add `sitemapConfig *SitemapConfig` to `moduleConfig` in `module.go` (amendment)
- [ ] Godoc on SitemapConfig: "SitemapConfig configures the fragment sitemap for a module. Pass it to app.Content as an option."

#### 3.2 — SitemapPriority optional interface

- [ ] Define `type SitemapPrioritiser interface { SitemapPriority() float64 }`
- [ ] Godoc: "SitemapPrioritiser may be implemented by content types to provide a per-item priority override in the sitemap. If not implemented, SitemapConfig.Priority is used."

#### 3.3 — SitemapEntry and XML types

- [ ] Define `type SitemapEntry struct { Loc string; LastMod time.Time; ChangeFreq ChangeFreq; Priority float64 }`
- [ ] Define internal XML envelope structs for fragment sitemap (`<urlset>`) and index (`<sitemapindex>`) using `encoding/xml`
- [ ] Zero LastMod (time.IsZero) omits `<lastmod>` tag — use `xml:",omitempty"` with a formatted string field

#### 3.4 — Fragment sitemap generation

- [ ] Define `func WriteSitemapFragment(w io.Writer, entries []SitemapEntry) error`
  - Writes `<?xml ...>` + `<urlset xmlns="...">` + `<url>` blocks
  - Uses `xml.NewEncoder(w)` — no full document buffering
  - Returns any write error
- [ ] Define `func SitemapEntries[T any](items []T, baseURL string, cfg SitemapConfig) []SitemapEntry`
  - `T` must implement `Headable` and embed `Node` (use type constraint `interface{ Headable; GetSlug() string; GetPublishedAt() time.Time; GetStatus() Status }`)
  - Filters to `Published` only
  - Calls `item.Head().Canonical` for `Loc`; falls back to `baseURL + "/" + item.GetSlug()` if Canonical empty
  - Uses `item.GetPublishedAt()` for `LastMod`
  - Checks `SitemapPrioritiser` via type assertion for per-item override

#### 3.5 — Sitemap index generation

- [ ] Define `func WriteSitemapIndex(w io.Writer, fragmentURLs []string, lastMod time.Time) error`
  - Writes `<sitemapindex>` with one `<sitemap>` per fragment URL
  - Uses `xml.NewEncoder(w)`

#### 3.6 — In-memory sitemap store

- [ ] Define `type SitemapStore struct` with `mu sync.RWMutex` and `fragments map[string][]byte` (keyed by path, e.g. `/posts/sitemap.xml`)
- [ ] Define `func NewSitemapStore() *SitemapStore`
- [ ] Define `func (s *SitemapStore) Set(path string, data []byte)`
- [ ] Define `func (s *SitemapStore) Get(path string) ([]byte, bool)`
- [ ] Define `func (s *SitemapStore) Handler() http.Handler` — serves stored sitemap bytes as `application/xml`; 404 if not found
- [ ] Define `func (s *SitemapStore) IndexHandler(baseURL string) http.Handler` — generates the index on-the-fly from stored fragment paths

#### 3.7 — Wire sitemap handler in Module.Register (amendment to module.go)

- [ ] In `Module[T].Register(*http.ServeMux)`: if `cfg.sitemapConfig != nil`, register `GET /{prefix}/sitemap.xml` using `SitemapStore.Handler()`
- [ ] Register `GET /sitemap.xml` on the root mux using `SitemapStore.IndexHandler(baseURL)` — only once, when first module with SitemapConfig registers
- [ ] This wiring requires `Module[T]` to hold a `*SitemapStore` reference and the app's BaseURL — add these to `module.go` moduleConfig (amendment)

#### 3.8 — Tests

- [ ] `TestWriteSitemapFragment` — valid XML, correct namespace, entries filtered to Published, lastmod present
- [ ] `TestWriteSitemapIndex` — valid XML, correct number of sitemaps
- [ ] `TestSitemapStore_SetGet` — round-trip bytes
- [ ] `TestSitemapStore_Handler_notFound` — 404 for unknown path
- [ ] `TestSitemapStore_Handler_found` — correct Content-Type and body
- [ ] `BenchmarkWriteSitemapFragment` — baseline

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestWriteSitemap|TestSitemapStore ./...` — all green
- [ ] `BACKLOG.md` — step table row and summary checkbox updated
- [ ] `README.md` — no examples broken by this step
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required, or new Decision/Amendment drafted and agreed upon

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
