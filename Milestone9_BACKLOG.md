# Forge — Milestone 9 Backlog (v1.0.0)

v1.0.0 stabilisation: coverage audit, benchmarks, godoc pass, example apps, CHANGELOG.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | coverage audit (test additions) | ✅ Done | 2026-03-08 |
| 2 | benchmarks_test.go | ✅ Done | 2026-03-08 |
| 3 | forge.go + storage.go (godoc) | ✅ Done | 2026-03-08 |
| 4 | example/blog/ | ✅ Done | 2026-03-08 |
| 5 | example/docs/ | ✅ Done | 2026-03-08 |
| 6 | example/api/ | 🔲 Not started | — |
| 7 | CHANGELOG.md + integration_full_test.go G21 | 🔲 Not started | — |

---

## Coverage audit results (pre-Step-1)

Measured with `go test -coverprofile=coverage.out github.com/forge-cms/forge`
followed by `go tool cover -func=coverage.out`.

**Overall total: 84.5%** — above the 80% target.
After Step 1 additions: **87.1%** — above the 85% Step 1 target.

All files have at least one test file. The 0%-coverage functions fall into two groups:

| Group | Examples | Action |
|-------|----------|--------|
| Unexported `isOption()` marker methods | every Option type | None — interface satisfaction, never externally called |
| Public APIs with no test path | `App.RedirectStore()`, `TrustedProxy`, `CacheStore.Sweep`, `RedirectStore.Len` | Add targeted unit tests in Step 1 |
| Internal helpers only reachable via DB | `RedirectStore.Load/Save/Remove`, `columnForField` | Left for M10/integration with a real DB; documented here as known gap |
| Template func `forgeLLMSEntries` | `templatehelpers.go:195` | Add coverage via direct `TemplateFuncMap` call in Step 1 |
| `module.go:stripMarkdown` | plain-text content negotiation path | Add `Accept: text/plain` request test in Step 1 |

---

## Layer 9 — Stabilisation

### Step 1 — Coverage audit: targeted test additions

**Depends on:** all M1–M8 files
**Decisions:** none
**Files:** additions to `forge_test.go`, `middleware_test.go`, `redirects_test.go`,
`module_test.go`, `templatehelpers_test.go`
**One-step-one-file exception:** this step is a documentation-only coverage audit
plus small additions to existing test files. No new implementation file is created.
The primary new artifact is the coverage table above in this backlog file.

#### 1.1 — forge_test.go additions

- [x] `TestApp_RedirectStore` — call `App.RedirectStore()` on a fresh `New(cfg)` app;
  assert non-nil after `app.Content(m)` with a `Redirects(...)` module

#### 1.2 — middleware_test.go additions

- [x] `TestTrustedProxy_setsRealIP` — build a `TrustedProxy([]string{"10.0.0.0/8"})`
  middleware, send a request with `X-Forwarded-For: 10.0.0.5`, assert
  `r.RemoteAddr` is rewritten to the forwarded IP
- [x] `TestTrustedProxy_untrustedIgnored` — send `X-Forwarded-For` from an
  untrusted CIDR; assert `RemoteAddr` is unchanged
- [x] `TestInMemoryCache_Sweep` — add two entries to a `CacheStore`, call `Sweep()`,
  assert both entries are evicted

#### 1.3 — redirects_test.go additions

- [x] `TestRedirectStore_Len` — add 3 entries via `Add()`, assert `Len() == 3`

#### 1.4 — module_test.go additions

- [x] `TestModule_plainTextContentNeg` — register a module, send a request with
  `Accept: text/plain`, assert response body has no markdown syntax
  (exercises `stripMarkdown`)

#### 1.5 — templatehelpers_test.go additions

- [x] `TestForgeLLMSEntries_viaFuncMap` — retrieve `forge_llms_entries` from
  `TemplateFuncMap()`, call it with a mock `LLMsStore`, assert the returned
  slice is non-nil

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -count=1 ./...` — all green
- [x] `go test "-coverprofile=coverage.out" github.com/forge-cms/forge` then
  `go tool cover "-func=coverage.out" | Select-String "total:"` — ≥ 85%
- [x] `BACKLOG.md` — step 1 row updated; `Milestone9_BACKLOG.md` step 1 ticked
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 2 — `benchmarks_test.go`

**Depends on:** Step 1 (coverage baseline established)
**Decisions:** none
**Files:** `benchmarks_test.go` (new file — no paired implementation file; benchmarks
supplement existing tested code per M9 one-file exception noted in this backlog)

#### 2.1 — Auth benchmarks

- [x] `BenchmarkSignToken` — benchmark `SignToken` with a fixed `User` and 1h TTL
- [x] `BenchmarkBearerHMAC_verify` — sign a token then benchmark the verify path via
  `BearerHMAC(secret)(handler)` with a pre-signed bearer header

#### 2.2 — Cookie benchmarks

- [x] `BenchmarkConsentFor_granted` — create a `ConsentFor` cookie context and call
  `ConsentFor(ctx, Analytics)` in a tight loop

#### 2.3 — Redirect benchmarks

- [x] `BenchmarkRedirectStore_Get_exact` — seed 100 exact-match entries, benchmark
  `Get("/posts/article-50")`
- [x] `BenchmarkRedirectStore_Get_prefix` — seed 50 prefix entries, benchmark
  `Get("/posts/article-50/old-name")`

#### 2.4 — Scheduler benchmarks

- [x] `BenchmarkScheduler_tick_noop` — call `tick()` on a scheduler backed by an empty
  `MemoryRepo`; measures overhead of processScheduled with no work to do

#### 2.5 — Feed benchmarks

- [x] `BenchmarkFeedBuild` — seed a `FeedStore` with 20 published items and call
  `regenerateFeed` or the equivalent internal path to measure RSS generation cost

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go test -run "^$" -bench "Benchmark" -benchtime=3s ./...` — all benchmarks run
  and produce ns/op output (no panics, no failures)
- [x] `go test -count=1 ./...` — full suite still green
- [x] `BACKLOG.md` — step 2 row updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 3 — `forge.go` + `storage.go` (godoc pass)

**Depends on:** Step 2
**Decisions:** none
**Files:** `forge.go`, `storage.go`
**One-step-two-file exception:** these are documentation additions only — no
implementation changes, no new test surface. Grouped in one step because neither
file requires a new test file.

#### 3.1 — forge.go godoc additions

- [x] Add `// App is the central registry...` struct-level godoc to `type App struct`
- [x] Audit and add godoc to any App method added in Amendments A18–A26 that is
  missing a doc comment: `Cookies()`, `CookieManifestAuth()`, `Redirect()`,
  `RedirectStore()`, `RedirectManifestAuth()`
- [x] Verify `Config`, `MustConfig`, `New`, `Run`, `Handler`, `Use`, `Content`,
  `Handle`, `SEO` all have godoc

#### 3.2 — storage.go godoc additions

- [x] Audit `SQLRepo[T]` methods: `FindByID`, `FindBySlug`, `FindAll`, `Save`,
  `Delete` — add godoc to any that are missing, matching the style of `MemoryRepo[T]`
- [x] Audit `NewSQLRepo`, `Table`, `SQLRepoOption` for godoc completeness

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `go doc github.com/forge-cms/forge App` — shows struct-level godoc
- [x] `go doc github.com/forge-cms/forge SQLRepo` — shows type-level godoc
- [x] `go test -count=1 ./...` — full suite still green
- [x] `BACKLOG.md` — step 3 row updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 4 — `example/blog/`

**Depends on:** Step 3 (final API surface locked before example authoring)
**Decisions:** Decision 4 (rendering model), Decision 9 (sitemap), Decision 13 (feeds)
**Files:** `example/blog/main.go`, `example/blog/go.mod`,
`example/blog/templates/list.html`, `example/blog/templates/show.html`
*(Note: Forge requires `list.html`/`show.html`; backlog names `index.html`/`post.html` were corrected at implementation time.)*

#### 4.1 — Content type and seeding

- [x] Define `Post` struct embedding `forge.Node` with fields `Title string`,
  `Body string`, `Tags []string`
- [x] Implement `forge.Headable` on `Post` returning `forge.Head` with title, excerpt,
  URL, OpenGraph card
- [x] Implement `forge.Markdownable` on `Post` returning markdown representation
- [x] Seed 8 posts: 6 Published, 1 Draft, 1 Scheduled (2 minutes in the future)

#### 4.2 — Module wiring

- [x] `forge.NewModule[*Post]` with options:
  - `forge.At("/posts")`
  - `forge.Repo(repo)`
  - `forge.SitemapConfig{}`
  - `forge.Social(forge.OpenGraph, forge.TwitterCard)` *(constants, not struct literals)*
  - `forge.Feed(forge.FeedConfig{...})` *(Feed wraps FeedConfig)*
  - `forge.AIIndex(forge.LLMsTxt, forge.LLMsTxtFull, forge.AIDoc)`
  - `forge.On[*Post](forge.AfterPublish, ...)` logging hook
- [x] Wire module to app via `app.Content(m)`
- [x] `app.SEO(&forge.RobotsConfig{Sitemaps: true})` — allow all crawlers, append sitemap

#### 4.3 — App and templates

- [x] `forge.MustConfig(forge.Config{BaseURL: "http://localhost:8080", Secret: [32]byte{...}})`
- [x] `forge.Templates("templates")` pointing at `templates/list.html` and `templates/show.html`
- [x] `list.html`: list of posts with title, date, excerpt, tag list
- [x] `show.html`: full post body rendered with `{{ forge_markdown .Content.Body }}`; both use `{{ template "forge:head" .Head }}`
- [x] `go.mod` uses `require github.com/forge-cms/forge` + `replace` directive; `go.work` updated with `use ./example/blog`

#### 4.4 — Inline comments

- [x] Each non-obvious Forge feature is annotated with a `// Forge:` comment explaining why
- [x] `main.go` has a top-level block comment explaining the app's purpose and the
  features it demonstrates

#### Verification

- [x] `go build ./...` — no errors
- [x] `go vet ./...` — clean (run from `example/blog/`)
- [x] `gofmt -l .` — returns nothing (run from `example/blog/`)
- [x] `go build .` from `example/blog/` — binary compiles with zero errors
- [x] `BACKLOG.md` — step 4 row updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 5 — `example/docs/`

**Depends on:** Step 4 (establishes example module pattern)
**Decisions:** Decision 7 (AIDoc), Decision 8 (llms.txt)
**Files:** `example/docs/main.go`, `example/docs/go.mod`,
`example/docs/templates/base.html`, `example/docs/templates/index.html`,
`example/docs/templates/doc.html`

#### 5.1 — Content type and seeding

- [x] Define `Doc` struct embedding `forge.Node` with fields `Title string`,
  `Body string`, `Section string`
- [x] Implement `forge.Headable` returning head with breadcrumbs
  (`forge.Crumbs(forge.Crumb("Docs", "/docs"), ...)`)
- [x] Implement `forge.Markdownable` and `forge.AIDocSummary`
- [x] Seed 6 Published docs across 2 sections

#### 5.2 — Module wiring

- [x] `forge.NewModule[*Doc]` with options:
  - `forge.At("/docs")`
  - `forge.Repo(repo)`
  - `forge.AIIndex(forge.LLMsTxt, forge.LLMsTxtFull, forge.AIDoc)`
  - `forge.RobotsConfig{AIScraper: forge.AskFirst}` on the app
- [x] `app.Content(m)` + `app.SEO(robots)`

#### 5.3 — App and templates

- [x] `list.html` with `{{template "forge:head" .Head}}` (Forge uses list.html, not base.html+index.html)
- [x] `list.html` grouped by Section via `$section` variable
- [x] `show.html` with full body, breadcrumb nav, link to `/docs/{slug}/aidoc`
- [x] `go.mod` with `replace` directive

#### 5.4 — Inline comments

- [x] Each AIDoc/llms.txt feature annotated with a `// Forge:` comment

#### Verification

- [x] `go build .` from `example/docs/` — compiles
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — returns nothing
- [x] `BACKLOG.md` — step 5 row updated
- [x] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 6 — `example/api/`

**Depends on:** Step 5
**Decisions:** Decision 2 (storage), Decision 15 (roles), Decision 17 (redirects)
**Files:** `example/api/main.go`, `example/api/go.mod`

#### 6.1 — Content type and seeding

- [ ] Define `Article` struct embedding `forge.Node` with fields `Title string`,
  `Body string`, `AuthorID string`
- [ ] Seed 6 Published articles, 1 Archived (with redirect from old slug)

#### 6.2 — Module wiring

- [ ] `forge.NewModule[*Article]` with:
  - `forge.At("/articles")`
  - `forge.Repo(repo)`
  - `forge.Auth(forge.BearerHMAC(secret))` — write endpoints require auth
  - `forge.Read(forge.Guest)` — public read
  - `forge.Write(forge.Editor)` — Editor+ to write
  - `forge.On(forge.BeforeCreate, ...)` — validation hook
  - `forge.Redirects(forge.From("/old-articles/{slug}"))` — legacy path
- [ ] `app.Use(forge.Chain(forge.SecurityHeaders, forge.RateLimit(100)))` — global middleware
- [ ] `app.Content(m)`

#### 6.3 — App wiring

- [ ] JSON-only app — no `forge.Templates` option
- [ ] `go.mod` with `replace` directive
- [ ] Top-of-file comment with: how to get a signed token for testing, example curl commands

#### 6.4 — Inline comments

- [ ] Role check pattern, redirect setup, content negotiation all annotated

#### Verification

- [ ] `go build .` from `example/api/` — compiles
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `BACKLOG.md` — step 6 row updated
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

### Step 7 — `CHANGELOG.md` + `integration_full_test.go` G21

**Depends on:** Step 6 (all implementation complete before changelog)
**Decisions:** none
**Files:** `CHANGELOG.md` (new), `integration_full_test.go` (append G21)

#### 7.1 — CHANGELOG.md

- [ ] Create `CHANGELOG.md` in Keep a Changelog format
  (https://keepachangelog.com/en/1.1.0/)
- [ ] Add `[Unreleased]` header for post-v1 work
- [ ] Add `[1.0.0] — 2026-03-08` section with sub-sections:
  - **Added** — one bullet per milestone (M1–M9) in plain English
  - **Notes** — API stability promise: all exported symbols in `forge` package
    are stable as of v1.0.0; breaking changes require a new major version
- [ ] Add `[0.8.0]` through `[0.1.0]` sections (one per milestone, brief)

#### 7.2 — integration_full_test.go G21

- [ ] Append `// — G21: Full v1.0.0 stack (M1+M2+M3+M5+M7+M8) ----` group header
- [ ] `TestFull_v1_fullStack` — wire a single app with:
  - `Module[*testPost]` using `Repo`, `At`, `Auth(BearerHMAC)`, `SitemapConfig{}`,
    `FeedConfig{}`, `AIIndex(LLMsTxt)`, `Redirects(From(...))`
  - One scheduled item (past-due) and one published item
  - Call `processScheduled` to publish the scheduled item (M8)
  - Assert: `GET /posts` → 200 JSON (M2)
  - Assert: `GET /sitemap.xml` → 200 (M3)
  - Assert: `GET /posts/feed.xml` → 200 (M5)
  - Assert: `GET /llms.txt` → 200 (M5)
  - Assert: `GET /.well-known/redirects.json` → 200 (M7)
  - Assert: scheduler-published item appears in `GET /posts` response (M8+M2 cross-check)

#### 7.3 — BACKLOG.md + README final review

- [ ] `BACKLOG.md` — M9 milestone row marked ✅ Done; all step rows ✅ Done
- [ ] `README.md` — confirm all milestone badges are ✅ Available; add v1.0.0
  release notice at top if not present
- [ ] `ARCHITECTURE.md` — add M9 changelog entry

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -count=1 ./...` — full suite green
- [ ] `go test "-coverprofile=coverage.out" github.com/forge-cms/forge` then
  `go tool cover "-func=coverage.out" | Select-String "total:"` — ≥ 85%
- [ ] `CHANGELOG.md` exists and `[1.0.0]` section is present
- [ ] `BACKLOG.md` — M9 milestone row and all step rows ✅ Done
- [ ] Review `ARCHITECTURE.md` and `DECISIONS.md` — no new decisions required,
      or new Decision/Amendment drafted and agreed upon

---

## Completion criteria for Milestone 9

- [ ] `go test "-coverprofile=coverage.out" github.com/forge-cms/forge` total ≥ 85%
- [ ] `benchmarks_test.go` — 7 new benchmarks covering M5–M8 hot paths
- [ ] All exported symbols in `forge.go` and `storage.go` have godoc comments
- [ ] `example/blog/`, `example/docs/`, `example/api/` each compile standalone
- [ ] `CHANGELOG.md` — `[1.0.0]` section present in Keep a Changelog format
- [ ] `integration_full_test.go` — G21 cross-milestone group appended and passing
- [ ] `go test ./...` green; `go vet ./...` clean; `gofmt -l .` empty
