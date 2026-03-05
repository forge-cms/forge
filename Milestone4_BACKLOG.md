# Forge — Milestone 4 Backlog (v0.4.0)

HTML rendering via html/template: TemplateData, template loading with startup
validation, forge:head partial, five template helpers, and a cross-component
integration test suite covering the largest gaps in existing coverage.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | templatedata.go | ✅ Done | 2026-03-05 |
| 2 | templates.go | 🔲 Not started | — |
| 3 | templatehelpers.go | 🔲 Not started | — |
| 4 | integration_test.go | 🔲 Not started | — |

---

## Deferred from this milestone

| Item | Reason | Target |
|------|--------|--------|
| `TemplatesWatch` | Shutdown wiring touches `templates.go` + `forge.go` (two-file amendment); pure dev convenience; adds goroutine complexity | M5 |
| `forge_llms_entries` | Correct output requires `.aidoc` URLs and `AIDoc` format from `ai.go`; ships as documented no-op stub in Step 3 | M5 |

---

## Amendments

| ID | File | Summary |
|----|------|---------|
| A6 | module.go | Template fields on `Module[T]`; `Templates`/`TemplatesOptional` options; `parseTemplates()`; HTML render branch in show/list handlers |
| A7 | errors.go | Replace inline `htmlErrorPage` fallback with `errorTemplateLookup` func var; wire from `App.Handler()` |
| A8 | forge.go | `templateModules []templateParser` on `App`; append in `Content()`; call `parseTemplates()` in `Run()` before `ListenAndServe` |

---

## Layer 1 — Template data type (no new amendments)

### Step 1 — templatedata.go ✅ 2026-03-05
<!-- collapsed — see git log for detail -->

## Layer 2 — Template loading and HTML render path (Amendments A6, A7, A8)

### Step 2 — templates.go

**Depends on:** templatedata.go, head.go, schema.go, module.go (A6), errors.go (A7), forge.go (A8)
**Decisions:** Amendment P3 (parse at startup), Decision 10 (error pages), Decision 14 (NoIndex meta), Decision 16 (error handling)
**Files:** `templates.go`, `templates_test.go`

#### 2.1 — Option types

- [ ] Define `type templatesOption struct { dir string; required bool }` implementing `isOption()`
- [ ] Define `func Templates(dir string) Option` — required; `list.html` + `show.html` must exist
- [ ] Define `func TemplatesOptional(dir string) Option` — silent no-op when dir/files absent
- [ ] Add `// TemplatesWatch is deferred to Milestone 5.` godoc stub (no implementation)

#### 2.2 — forge:head named template

- [ ] Define `const forgeHeadTmpl string` — complete `<head>` fragment:
  - `<title>{{.Title}}</title>`
  - `<meta name="description">` (omit when `.Description` empty)
  - `<link rel="canonical">` (omit when `.Canonical` empty)
  - Open Graph tags: `og:title`, `og:description`, `og:image` (when `.Image.URL` non-empty), `og:type`
  - `<meta name="robots" content="noindex, nofollow">` when `.NoIndex` (Decision 14)
  - `{{forge_meta . .}}` placeholder — requires TemplateFuncMap registered (Step 3 wires the func; safe because parse happens after func map registration)
- [ ] Usage documented in godoc: `{{template "forge:head" .Head}}`

#### 2.3 — parseTemplates() on Module[T] (Amendment A6 core)

- [ ] Add to `Module[T]` struct: `templateDir string`, `templateRequired bool`, `tplList *template.Template`, `tplShow *template.Template`, `tplMu sync.RWMutex`, `siteName string`
- [ ] Add `case templatesOption` in `NewModule` option loop: set `templateDir`, `templateRequired`
- [ ] Add `siteName string` — extracted from `baseURL` hostname via `net/url.Parse` in `setSitemap` equivalent; set during `App.Content()` wiring via a new `setSiteName(string)` method
- [ ] Implement `parseTemplates() error` on `*Module[T]`:
  - Skip if `templateDir == ""`
  - Parse `{dir}/list.html` with `template.New("list").Funcs(TemplateFuncMap()).ParseFiles(...)`
  - Register `forge:head` via `tplList.New("forge:head").Parse(forgeHeadTmpl)`
  - Repeat for `{dir}/show.html`
  - Required mode: return wrapped error if file absent
  - Optional mode: return `nil` if file absent; set `tplList`/`tplShow` only when found
  - Acquire `tplMu.Lock()` before swapping pointers

#### 2.4 — HTML render in show/list handlers (Amendment A6 render path)

- [ ] In `contentNegotiator`: set `html = true` when `m.tplShow != nil || m.tplList != nil`
- [ ] In `showHandler` HTML branch: `tplMu.RLock()`, execute `tplShow` with `NewTemplateData(ctx, item, head, m.siteName)`, `Content-Type: text/html; charset=utf-8`
- [ ] In `listHandler` HTML branch: same pattern with `tplList` and `[]T` content
- [ ] Template execution error → `WriteError(w, r, ErrInternal)` (500)

#### 2.5 — Error page wiring (Amendment A7)

- [ ] Add `var errorTemplateLookup func(status int) *template.Template` to `errors.go`
- [ ] In `respond()` HTML path: if `errorTemplateLookup != nil`, call it; if template non-nil, execute with `struct{ Status int; Message string; RequestID string }{}` and return
- [ ] Fall through to inline `htmlErrorPage` constant when lookup nil or template nil

#### 2.6 — Startup parse wiring (Amendment A8)

- [ ] Define `type templateParser interface { parseTemplates() error }` in `templates.go`
- [ ] Add `templateModules []templateParser` to `App` struct in `forge.go`
- [ ] In `App.Content()`: after `Register()`, if module implements `templateParser`, append
- [ ] In `App.Content()`: if module implements `interface{ setSiteName(string) }`, call with hostname from `cfg.BaseURL`
- [ ] In `App.Run()`: iterate `templateModules`, call `parseTemplates()`; return error on first failure (fail-fast, Amendment P3)
- [ ] In `App.Handler()`: set `errorTemplateLookup` to a closure over `templateModules` that searches for `errors/{status}.html` in each module's `templateDir`

#### 2.7 — Tests

- [ ] `TestTemplates_missingList` — required mode, list.html absent → error from parseTemplates
- [ ] `TestTemplates_missingShow` — required mode, show.html absent → error from parseTemplates
- [ ] `TestTemplatesOptional_missingDir` — optional mode, no dir → nil error, tplList/tplShow nil
- [ ] `TestTemplates_forgeHeadRegistered` — parsed template set contains "forge:head" template
- [ ] `TestTemplates_noIndexMeta` — execute show template with `Head{NoIndex: true}` → `noindex` in output
- [ ] `TestTemplates_errorPage_custom` — `errors/404.html` present → executed by `errorTemplateLookup`
- [ ] `TestTemplates_errorPage_fallback` — no template dir → inline HTML fallback

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestTemplates ./...` — all green
- [ ] `BACKLOG.md` — step table row and summary checkbox updated
- [ ] `README.md` — no examples broken

---

## Layer 3 — Template helpers (depends on templates.go)

### Step 3 — templatehelpers.go

**Depends on:** templates.go, head.go (SchemaFor, Excerpt), auth.go (CSRFCookieName), schema.go
**Decisions:** Decision 8 (llms.txt — stub only)
**Files:** `templatehelpers.go`, `templatehelpers_test.go`

#### 3.1 — forge_meta

- [ ] `func forgeMeta(head Head, content any) template.HTML` — calls `SchemaFor(head, content)`, returns `template.HTML`
- [ ] Godoc: `{{forge_meta .Head .Content}}`

#### 3.2 — forge_date

- [ ] `func forgeDate(t time.Time) string` — format `"2 January 2006"`; zero time returns `""`
- [ ] Godoc: `{{.Content.PublishedAt | forge_date}}`

#### 3.3 — forge_markdown

- [ ] `func forgeMarkdown(s string) template.HTML` — stdlib-only Markdown converter:
  - `#`–`######` headings → `<h1>`–`<h6>`
  - `**text**` → `<strong>`
  - `*text*` → `<em>`
  - `` `code` `` → `<code>`
  - `[text](url)` → `<a href="url">text</a>`
  - `- ` / `* ` list items → `<ul><li>` blocks
  - Blank-line paragraph separation → `<p>` wrapping
  - Process in correct order: links before bold/italic (avoids partial matches)
- [ ] Returns `template.HTML` (not double-escaped)
- [ ] Godoc: `{{.Content.Body | forge_markdown}}`

#### 3.4 — forge_excerpt

- [ ] `func forgeExcerpt(maxLen int, s string) template.HTML` — wraps `Excerpt(s, maxLen)`
- [ ] Pipeline order: `maxLen` is first arg (partial application), `s` comes from pipe
- [ ] Returns `template.HTML`
- [ ] Godoc: `{{.Content.Body | forge_excerpt 120}}`

#### 3.5 — forge_csrf_token

- [ ] `func forgeCSRFToken(r *http.Request) template.HTML` — reads `r.Cookie(CSRFCookieName)`
- [ ] When present: returns `template.HTML` of `<input type="hidden" name="csrf_token" value="{token}">`
- [ ] When absent: returns `template.HTML("")`
- [ ] Godoc: `{{forge_csrf_token .Request}}`

#### 3.6 — forge_llms_entries (deferred stub)

- [ ] `func forgeLLMSEntries() template.HTML { return "" }`
- [ ] Godoc comment: `// TODO(ai): implement in Milestone 5 — requires ai.go for AIDoc URL generation. Returns empty string until then.`

#### 3.7 — TemplateFuncMap

- [ ] `func TemplateFuncMap() template.FuncMap` — returns map with all six helpers
- [ ] Keys: `"forge_meta"`, `"forge_date"`, `"forge_markdown"`, `"forge_excerpt"`, `"forge_csrf_token"`, `"forge_llms_entries"`
- [ ] Godoc the function

#### 3.8 — Tests

- [ ] `TestForgeDate_formatted` — non-zero time formats correctly
- [ ] `TestForgeDate_zero` — zero time returns `""`
- [ ] `TestForgeMeta_withSchema` — Article type → `<script type="application/ld+json">` present
- [ ] `TestForgeMeta_noSchema` — empty Type → empty string
- [ ] `TestForgeMarkdown_heading` — `# Title` → `<h1>Title</h1>`
- [ ] `TestForgeMarkdown_bold` — `**text**` → `<strong>text</strong>`
- [ ] `TestForgeMarkdown_link` — `[text](url)` → `<a href="url">text</a>`
- [ ] `TestForgeMarkdown_list` — `- item` → `<ul><li>item</li></ul>`
- [ ] `TestForgeMarkdown_paragraph` — blank-line separation → `<p>` blocks
- [ ] `TestForgeExcerpt_pipeline` — truncates at word boundary
- [ ] `TestForgeCSRFToken_present` — cookie present → `<input>` tag returned
- [ ] `TestForgeCSRFToken_absent` — no cookie → empty string
- [ ] `TestTemplateFuncMap_keys` — all six keys present in map
- [ ] `BenchmarkForgeMarkdown` — 500-word body baseline

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestForge|TestTemplate|BenchmarkForge ./...` — all green
- [ ] `BACKLOG.md` — step table row and summary checkbox updated
- [ ] `README.md` — no examples broken

---

## Layer 4 — Integration tests (depends on all M4 files)

### Step 4 — integration_test.go

**Depends on:** templatedata.go, templates.go, templatehelpers.go, module.go, forge.go, errors.go, middleware.go
**Decisions:** All M4 decisions
**Files:** `integration_test.go` only

#### 4.1 — HTML render cycle

- [ ] `TestIntegration_showHTML` — full App request → `<title>` in body, `text/html` content-type
- [ ] `TestIntegration_listHTML` — list route, Accept text/html → 200
- [ ] `TestIntegration_json_unaffected` — JSON still works on same route after Templates registered
- [ ] `TestIntegration_htmlFallback_noTemplates` — no Templates option → Accept text/html → 406

#### 4.2 — forge:head correctness

- [ ] `TestIntegration_forgeHead_noIndex` — `Head.NoIndex = true` → `noindex` in rendered output
- [ ] `TestIntegration_forgeHead_canonical` — `Head.Canonical` non-empty → `<link rel="canonical">` present
- [ ] `TestIntegration_forgeHead_jsonLD` — `Head.Type = "Article"` → JSON-LD `<script>` in output

#### 4.3 — Error pages

- [ ] `TestIntegration_errorPage_custom` — `errors/404.html` in tmpdir → rendered on 404
- [ ] `TestIntegration_errorPage_fallback` — no template → inline fallback HTML contains status code

#### 4.4 — CSRF (existing gap in middleware_test.go)

- [ ] `TestIntegration_csrf_tokenInForm` — CSRF middleware + forge_csrf_token helper → token in `<input>`
- [ ] `TestIntegration_csrf_rejectMissing` — POST without X-CSRF-Token header → 403

#### 4.5 — Existing App-level gaps

- [ ] `TestIntegration_seo_robotsTxt` — `App.SEO(&RobotsConfig{})` + `App.Handler()` → GET /robots.txt → 200
- [ ] `TestIntegration_sitemap_index` — module with `SitemapConfig{}` → GET /sitemap.xml → 200 after Handler()

#### 4.6 — TemplateData correctness

- [ ] `TestIntegration_templateData_user` — authenticated request → `TemplateData.User.ID` non-empty in template
- [ ] `TestIntegration_templateData_head` — `HeadFunc` return value reflected in rendered `<title>`

#### Verification

- [ ] `go build ./...` — no errors
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — returns nothing
- [ ] `go test -v -run TestIntegration ./...` — all green
- [ ] `go test -count=1 ./...` — full suite green
- [ ] `BACKLOG.md` + `Milestone4_BACKLOG.md` updated — M4 row ✅
- [ ] `ARCHITECTURE.md` updated — M4 files in Implemented, removed from Planned
- [ ] `README.md` — no examples broken

---

## Completion criteria for Milestone 4

- [ ] `go build ./...` — clean
- [ ] `go vet ./...` — clean
- [ ] `gofmt -l .` — nothing
- [ ] `go test ./...` — all green
- [ ] All exported symbols have godoc
- [ ] `Accept: text/html` on a module with `Templates` registered → full HTML with `<title>` and `forge:head`
- [ ] `Accept: application/json` on same route still returns JSON
- [ ] `Accept: text/html` on module without `Templates` → 406
- [ ] Missing template at startup → `App.Run()` returns error immediately
- [ ] `forge:head` partial renders `noindex` meta for non-Published content (Decision 14)
- [ ] `WriteError` renders custom `templates/errors/{status}.html` when present
- [ ] CSRF middleware + `forge_csrf_token` helper → token round-trip tested
- [ ] `TemplatesWatch` deferred and documented with reason + target milestone
- [ ] `forge_llms_entries` ships as documented no-op stub
- [ ] Post-milestone DRY/performance/security review completed
- [ ] Retrospective completed
