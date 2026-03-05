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
| 4 | integration_test.go | ✅ Done | 2026-03-05 |

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

### Step 4 — integration_test.go ✅ 2026-03-05
<!-- collapsed — see git log for detail -->

---

## Completion criteria for Milestone 4

- [x] `go build ./...` — clean
- [x] `go vet ./...` — clean
- [x] `gofmt -l .` — nothing
- [x] `go test ./...` — all green
- [x] All exported symbols have godoc
- [x] `Accept: text/html` on a module with `Templates` registered → full HTML with `<title>` and `forge:head`
- [x] `Accept: application/json` on same route still returns JSON
- [x] `Accept: text/html` on module without `Templates` → 406
- [x] Missing template at startup → `App.Run()` returns error immediately
- [x] `forge:head` partial renders `noindex` meta for non-Published content (Decision 14)
- [x] `WriteError` renders custom `templates/errors/{status}.html` when present
- [x] CSRF middleware + `forge_csrf_token` helper → token round-trip tested
- [x] `TemplatesWatch` deferred and documented with reason + target milestone
- [x] `forge_llms_entries` ships as documented no-op stub
- [x] Post-milestone DRY/performance/security review completed
- [x] Retrospective completed
