# Forge — Milestone 4 Backlog (v0.4.0)

HTML rendering via html/template: TemplateData, template loading with startup
validation, forge:head partial, five template helpers, and a cross-component
integration test suite covering the largest gaps in existing coverage.

---

## Progress

| Step | File | Status | Completed |
|------|------|--------|-----------|
| 1 | templatedata.go | ✅ Done | 2026-03-05 |
| 2 | templates.go | ✅ Done | 2026-03-05 |
| 3 | templatehelpers.go | ✅ Done | 2026-03-05 |
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

### Step 2 — templates.go ✅ 2026-03-05
<!-- collapsed — see git log for detail -->

---

## Layer 3 — Template helpers (depends on templates.go)

### Step 3 — templatehelpers.go ✅ 2026-03-05
<!-- collapsed — see git log for detail -->

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
