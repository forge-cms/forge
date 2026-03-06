package forge

// integration_full_test.go — cross-milestone integration suite (M1–M5).
//
// Each test exercises behaviour that requires at least two milestone components
// working together. No test in this file duplicates coverage from
// integration_test.go (which covers single-module M4 scenarios).
//
// Groups:
//   G1 — Multi-module routing (M2)
//   G2 — Role-based access via inline middleware (M1 + M2)
//   G3 — Signal fire-through across modules (M1 + M2)
//   G4 — Content negotiation: two modules, mixed template configuration (M2 + M4)
//   G5 — forge:head + schema helpers through real render (M3 + M4)
//   G6 — SEO wiring: robots.txt + sitemap registration (M2 + M3)
//   G7 — Error template fallback across two modules (M2 + M4)
//   G8 — TemplateData end-to-end (M3 + M4)
//   G9 — Social + SitemapConfig (M5 + M3)
//   G10 — AI indexing + content negotiation (M5 + M4)
//   G11 — RSS feed + AfterPublish signal (M5 + M1)
//   G12 — Full M5 stack (M5 + M3 + M4)

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// — Helpers ———————————————————————————————————————————————————————————————

// fullTestApp is the shared state for two-module integration tests.
type fullTestApp struct {
	app      *App
	handler  http.Handler
	postRepo *MemoryRepo[*testPost]
	pageRepo *MemoryRepo[*testMDPost]
	postsMod *Module[*testPost]
	pagesMod *Module[*testMDPost]
}

// newFullTestApp creates an App with two modules:
//   - posts (*testPost at /posts) with postOpts appended after Repo + At("/posts")
//   - pages (*testMDPost at /pages) with pageOpts appended after Repo + At("/pages")
//
// Templates are parsed and Handler() is called before returning.
func newFullTestApp(t *testing.T, postOpts []Option, pageOpts []Option) *fullTestApp {
	t.Helper()
	postRepo := NewMemoryRepo[*testPost]()
	pageRepo := NewMemoryRepo[*testMDPost]()

	pOpts := append([]Option{Repo(postRepo), At("/posts")}, postOpts...)
	gOpts := append([]Option{Repo(pageRepo), At("/pages")}, pageOpts...)

	m1 := NewModule((*testPost)(nil), pOpts...)
	m2 := NewModule((*testMDPost)(nil), gOpts...)

	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Content(m1)
	app.Content(m2)

	if err := m1.parseTemplates(); err != nil {
		t.Fatalf("newFullTestApp: parseTemplates posts: %v", err)
	}
	if err := m2.parseTemplates(); err != nil {
		t.Fatalf("newFullTestApp: parseTemplates pages: %v", err)
	}

	return &fullTestApp{
		app:      app,
		handler:  app.Handler(),
		postRepo: postRepo,
		pageRepo: pageRepo,
		postsMod: m1,
		pagesMod: m2,
	}
}

// fullSeedPost saves a published *testPost with the given slug and title.
func fullSeedPost(t *testing.T, repo *MemoryRepo[*testPost], slug, title string) *testPost {
	t.Helper()
	p := &testPost{Node: Node{ID: NewID(), Slug: slug, Status: Published}, Title: title}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("fullSeedPost: %v", err)
	}
	return p
}

// fullSeedPage saves a published *testMDPost with the given slug and title.
func fullSeedPage(t *testing.T, repo *MemoryRepo[*testMDPost], slug, title string) *testMDPost {
	t.Helper()
	p := &testMDPost{
		Node:  Node{ID: NewID(), Slug: slug, Status: Published},
		Title: title,
		Body:  "Hello world",
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("fullSeedPage: %v", err)
	}
	return p
}

// writeErrTemplate creates errors/{status}.html inside dir with the given body.
func writeErrTemplate(t *testing.T, dir string, status int, body string) {
	t.Helper()
	errDir := filepath.Join(dir, "errors")
	if err := os.MkdirAll(errDir, 0755); err != nil {
		t.Fatalf("writeErrTemplate mkdir: %v", err)
	}
	name := filepath.Join(errDir, fmt.Sprintf("%d.html", status))
	if err := os.WriteFile(name, []byte(body), 0644); err != nil {
		t.Fatalf("writeErrTemplate write: %v", err)
	}
}

// — G1: Multi-module routing (M2) ————————————————————————————————————————

// TestFull_multiModuleRouting verifies that two modules registered at different
// prefixes route independently without cross-contamination.
func TestFull_multiModuleRouting(t *testing.T) {
	postDir := intTmpDir(t, `<p>posts-list</p>`, `<p>post:{{.Content.Title}}</p>`)
	fa := newFullTestApp(t,
		[]Option{Templates(postDir)},
		nil, // pages: JSON only
	)
	fullSeedPost(t, fa.postRepo, "hello-post", "Hello Post")
	fullSeedPage(t, fa.pageRepo, "about", "About Page")

	t.Run("posts_show_html", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/posts/hello-post", nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		fa.handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Hello Post") {
			t.Errorf("posts: expected 'Hello Post', got: %s", w.Body.String())
		}
	})

	t.Run("pages_show_json", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/pages/about", nil)
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		fa.handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q; want application/json", ct)
		}
	})

	t.Run("posts_slug_does_not_hit_pages", func(t *testing.T) {
		// /posts/about should 404 — About Page is in /pages, not /posts.
		r := httptest.NewRequest("GET", "/posts/about", nil)
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		fa.handler.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Errorf("status = %d; want 404 (slug belongs to pages, not posts)", w.Code)
		}
	})
}

// TestFull_customHandleRoute verifies that App.Handle registers a route that
// coexists with module routes.
func TestFull_customHandleRoute(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	app.Content(m)
	handler := app.Handler()

	// Custom route works.
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("/health status = %d; want 200", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("/health body = %q; want ok", w.Body.String())
	}

	// Module route still works.
	r2 := httptest.NewRequest("GET", "/posts", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r2)
	if w2.Code != 200 {
		t.Errorf("/posts status = %d; want 200", w2.Code)
	}
}

// TestFull_globalMiddlewareOrder verifies that App.Use applies middleware in
// registration order — first Use argument is the outermost wrapper.
func TestFull_globalMiddlewareOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, name)
				mu.Unlock()
				next.ServeHTTP(w, r)
			})
		}
	}

	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Use(record("A"), record("B"))
	app.Content(m)
	handler := app.Handler()

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()

	if len(got) < 2 {
		t.Fatalf("expected ≥2 middleware invocations, got %d: %v", len(got), got)
	}
	if got[0] != "A" || got[1] != "B" {
		t.Errorf("middleware order = %v; want [A B ...]", got)
	}
}

// — G2: Role-based access via inline middleware (M1 + M2) —————————————————

// TestFull_roleCheck_denies verifies that an inline middleware using HasRole
// rejects unauthenticated requests with 403.
func TestFull_roleCheck_denies(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))

	// Inline role-guard middleware built with M1's HasRole + M1's Context.
	requireEditor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ContextFrom(w, r)
			if !HasRole(ctx.User().Roles, Editor) {
				WriteError(w, r, ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	app.Use(requireEditor)
	app.Content(m)
	handler := app.Handler()

	// No user → GuestUser → denied.
	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Errorf("status = %d; want 403 (no role)", w.Code)
	}
}

// TestFull_roleCheck_allows verifies that a request with a matching role
// passes the inline middleware guard.
func TestFull_roleCheck_allows(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))

	requireEditor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ContextFrom(w, r)
			if !HasRole(ctx.User().Roles, Editor) {
				WriteError(w, r, ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	app.Use(requireEditor)
	app.Content(m)
	handler := app.Handler()

	// Editor role → allowed.
	r := httptest.NewRequest("GET", "/posts", nil)
	r = withUser(r, editorUser())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("status = %d; want 200 (editor role)", w.Code)
	}
}

// — G3: Signal fire-through (M1 + M2) ————————————————————————————————————

// TestFull_signalOnCreate verifies that AfterCreate fires when an item is
// created through the module's create handler.
func TestFull_signalOnCreate(t *testing.T) {
	var fired atomic.Int32
	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil),
		Repo(repo),
		At("/posts"),
		On(AfterCreate, func(_ Context, _ *testPost) error {
			fired.Add(1)
			return nil
		}),
	)

	body, _ := json.Marshal(map[string]string{"Title": "Signal Post"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest("POST", "/posts", bytes.NewReader(body)),
		authorUser(),
	)
	m.createHandler(w, r)

	if w.Code != 201 {
		t.Fatalf("create: status = %d; body: %s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fired.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fired.Load() == 0 {
		t.Error("AfterCreate did not fire within 500ms")
	}
}

// TestFull_signalOnDelete verifies that AfterDelete fires when an item is
// deleted through the module's delete handler.
func TestFull_signalOnDelete(t *testing.T) {
	var fired atomic.Int32
	repo := NewMemoryRepo[*testPost]()
	p := fullSeedPost(t, repo, "delete-me", "Delete Me")

	m := NewModule((*testPost)(nil),
		Repo(repo),
		At("/posts"),
		On(AfterDelete, func(_ Context, _ *testPost) error {
			fired.Add(1)
			return nil
		}),
	)

	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest("DELETE", "/posts/"+p.Slug, nil),
		editorUser(),
	)
	r.SetPathValue("slug", p.Slug)
	m.deleteHandler(w, r)

	if w.Code != 204 {
		t.Fatalf("delete: status = %d; body: %s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fired.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fired.Load() == 0 {
		t.Error("AfterDelete did not fire within 500ms")
	}
}

// TestFull_signalCrossModuleIsolation verifies that signals registered on one
// module do not fire on another module in the same App.
func TestFull_signalCrossModuleIsolation(t *testing.T) {
	var postsFired, pagesFired atomic.Int32

	postRepo := NewMemoryRepo[*testPost]()
	pageRepo := NewMemoryRepo[*testMDPost]()

	postsMod := NewModule((*testPost)(nil),
		Repo(postRepo),
		At("/posts"),
		On(AfterCreate, func(_ Context, _ *testPost) error {
			postsFired.Add(1)
			return nil
		}),
	)
	pagesMod := NewModule((*testMDPost)(nil),
		Repo(pageRepo),
		At("/pages"),
		On(AfterCreate, func(_ Context, _ *testMDPost) error {
			pagesFired.Add(1)
			return nil
		}),
	)

	// Trigger create on posts module only.
	body, _ := json.Marshal(map[string]string{"Title": "Posts Only"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest("POST", "/posts", bytes.NewReader(body)),
		authorUser(),
	)
	postsMod.createHandler(w, r)

	if w.Code != 201 {
		t.Fatalf("create posts: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Wait for async signals.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if postsFired.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Posts AfterCreate must have fired.
	if postsFired.Load() == 0 {
		t.Error("posts AfterCreate did not fire within 500ms")
	}
	// Pages AfterCreate must NOT have fired.
	time.Sleep(20 * time.Millisecond) // small extra window to catch false fires
	if pagesFired.Load() != 0 {
		t.Errorf("pages AfterCreate fired = %d; want 0 (signal must not leak to other module)", pagesFired.Load())
	}

	// Suppress unused variable warning.
	_ = pagesMod
}

// — G4: Content negotiation, two modules (M2 + M4) ————————————————————————

// TestFull_jsonModule_noTemplates verifies that a module without templates
// returns 406 when the client requests text/html.
func TestFull_jsonModule_noTemplates(t *testing.T) {
	fa := newFullTestApp(t, nil, nil) // neither module has templates
	fullSeedPage(t, fa.pageRepo, "about", "About")

	r := httptest.NewRequest("GET", "/pages/about", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	fa.handler.ServeHTTP(w, r)
	if w.Code != 406 {
		t.Errorf("status = %d; want 406 (no HTML templates)", w.Code)
	}
}

// TestFull_jsonModule_jsonWorks verifies that the same template-less module
// continues to serve JSON correctly.
func TestFull_jsonModule_jsonWorks(t *testing.T) {
	fa := newFullTestApp(t, nil, nil)
	fullSeedPage(t, fa.pageRepo, "home", "Home")

	r := httptest.NewRequest("GET", "/pages/home", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	fa.handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("status = %d; want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q; want application/json", w.Header().Get("Content-Type"))
	}
}

// TestFull_htmlModule_templateFallback verifies that TemplatesOptional with
// only list.html present returns 200 for the list route and 406 for the show
// route (no show.html).
func TestFull_htmlModule_templateFallback(t *testing.T) {
	// list.html exists; show.html does not.
	dir := intTmpDir(t, `<p>list: {{len .Content}} items</p>`, "")

	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"), TemplatesOptional(dir))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Content(m)
	if err := m.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	handler := app.Handler()

	fullSeedPost(t, repo, "opt-post", "Optional Post")

	t.Run("list_200", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/posts", nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Errorf("list status = %d; want 200; body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("show_406", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/posts/opt-post", nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 406 {
			t.Errorf("show status = %d; want 406 (no show.html)", w.Code)
		}
	})
}

// — G5: forge:head + schema through real render (M3 + M4) —————————————————

// TestFull_schemaForThroughTemplate verifies that forge_meta in a real
// template render produces a JSON-LD <script> block for Article content.
func TestFull_schemaForThroughTemplate(t *testing.T) {
	const tpl = `<!DOCTYPE html><html><head>{{forge_meta .Head .Content}}</head><body></body></html>`
	dir := intTmpDir(t, `<p>list</p>`, tpl)
	_, handler, repo := intSetup(t,
		Templates(dir),
		HeadFunc(func(_ Context, p *testPost) Head {
			return Head{Title: p.Title, Type: Article}
		}),
	)
	intSeed(t, repo, "schema-post", "Schema Post")

	r := httptest.NewRequest("GET", "/posts/schema-post", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `application/ld+json`) {
		t.Errorf("expected JSON-LD script tag in body, got:\n%s", body)
	}
	if !strings.Contains(body, `"Article"`) {
		t.Errorf("expected Article schema type in body, got:\n%s", body)
	}
}

// TestFull_forgeMarkdownInTemplate verifies that forge_markdown converts
// Markdown syntax to HTML inside a rendered template.
func TestFull_forgeMarkdownInTemplate(t *testing.T) {
	const tpl = `{{.Content.Body | forge_markdown}}`
	pageDir := intTmpDir(t, `<p>list</p>`, tpl)

	pageRepo := NewMemoryRepo[*testMDPost]()
	m := NewModule((*testMDPost)(nil), Repo(pageRepo), At("/pages"), Templates(pageDir))
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Content(m)
	if err := m.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	handler := app.Handler()

	p := &testMDPost{
		Node:  Node{ID: NewID(), Slug: "bold-page", Status: Published},
		Title: "Bold Page",
		Body:  "**bold text** and *italic*",
	}
	if err := pageRepo.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/pages/bold-page", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "<strong>bold text</strong>") {
		t.Errorf("expected <strong>bold text</strong> in body, got:\n%s", body)
	}
	if !strings.Contains(body, "<em>italic</em>") {
		t.Errorf("expected <em>italic</em> in body, got:\n%s", body)
	}
}

// TestFull_breadcrumbs verifies that a non-empty Head.Breadcrumbs causes
// forge_meta to append a BreadcrumbList JSON-LD block after the primary schema.
func TestFull_breadcrumbs(t *testing.T) {
	const tpl = `{{forge_meta .Head .Content}}`
	dir := intTmpDir(t, `<p>list</p>`, tpl)
	_, handler, repo := intSetup(t,
		Templates(dir),
		HeadFunc(func(_ Context, p *testPost) Head {
			return Head{
				Title: p.Title,
				Type:  Article,
				Breadcrumbs: Crumbs(
					Crumb("Home", "https://example.com"),
					Crumb("Posts", "https://example.com/posts"),
				),
			}
		}),
	)
	intSeed(t, repo, "crumb-post", "Crumb Post")

	r := httptest.NewRequest("GET", "/posts/crumb-post", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "BreadcrumbList") {
		t.Errorf("expected BreadcrumbList in body, got:\n%s", body)
	}
	// Two separate ld+json script blocks — Article + BreadcrumbList.
	count := strings.Count(body, `application/ld+json`)
	if count < 2 {
		t.Errorf("expected 2 ld+json script blocks, got %d:\n%s", count, body)
	}
}

// — G6: SEO wiring (M2 + M3) ——————————————————————————————————————————————

// TestFull_sitemapAppendsInRobots verifies that App.SEO with Sitemaps: true
// produces a robots.txt containing a Sitemap directive.
func TestFull_sitemapAppendsInRobots(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := NewModule((*testPost)(nil), Repo(repo), At("/posts"), SitemapConfig{})
	app := New(MustConfig(Config{
		BaseURL: "https://example.com",
		Secret:  []byte("16bytessecretkey"),
	}))
	app.Content(m)
	app.SEO(&RobotsConfig{Sitemaps: true})
	handler := app.Handler()

	r := httptest.NewRequest("GET", "/robots.txt", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("GET /robots.txt status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Sitemap:") {
		t.Errorf("expected 'Sitemap:' directive in robots.txt, got:\n%s", body)
	}
	if !strings.Contains(body, "https://example.com/sitemap.xml") {
		t.Errorf("expected sitemap URL in robots.txt, got:\n%s", body)
	}
}

// — G7: Error template fallback across two modules (M2 + M4) ——————————————

// TestFull_errorTemplate_firstMatch verifies that when both modules have an
// errors/404.html, the first registered module's template is used.
func TestFull_errorTemplate_firstMatch(t *testing.T) {
	origLookup := errorTemplateLookup
	t.Cleanup(func() { errorTemplateLookup = origLookup })

	postDir := intTmpDir(t, `<p>list</p>`, `<p>show</p>`)
	pageDir := intTmpDir(t, `<p>list</p>`, `<p>show</p>`)
	writeErrTemplate(t, postDir, 404, `<p>posts-first-match</p>`)
	writeErrTemplate(t, pageDir, 404, `<p>pages-second</p>`)

	fa := newFullTestApp(t,
		[]Option{Templates(postDir)},
		[]Option{Templates(pageDir)},
	)

	r := httptest.NewRequest("GET", "/posts/no-such", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	fa.handler.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("status = %d; want 404", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "posts-first-match") {
		t.Errorf("expected first module's error template, got:\n%s", body)
	}
	if strings.Contains(body, "pages-second") {
		t.Errorf("unexpected second module's template in first-match scenario:\n%s", body)
	}
}

// TestFull_errorTemplate_fallsThrough verifies that when the first module has
// no errors/ directory, the second module's error template is used.
func TestFull_errorTemplate_fallsThrough(t *testing.T) {
	origLookup := errorTemplateLookup
	t.Cleanup(func() { errorTemplateLookup = origLookup })

	postDir := intTmpDir(t, `<p>list</p>`, `<p>show</p>`)
	// postDir intentionally has no errors/ subdirectory.
	pageDir := intTmpDir(t, `<p>list</p>`, `<p>show</p>`)
	writeErrTemplate(t, pageDir, 404, `<p>pages-fallthrough-404</p>`)

	fa := newFullTestApp(t,
		[]Option{Templates(postDir)},
		[]Option{Templates(pageDir)},
	)

	r := httptest.NewRequest("GET", "/posts/no-such", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	fa.handler.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("status = %d; want 404", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "pages-fallthrough-404") {
		t.Errorf("expected second module's error template (fallthrough), got:\n%s", body)
	}
}

// — G8: TemplateData end-to-end (M3 + M4) ————————————————————————————————

// TestFull_templateData_siteName verifies that Config.BaseURL hostname is
// propagated to {{.SiteName}} inside a rendered HTML template.
func TestFull_templateData_siteName(t *testing.T) {
	const tpl = `site:{{.SiteName}}`
	dir := intTmpDir(t, `<p>list</p>`, tpl)
	_, handler, repo := intSetup(t, Templates(dir))
	intSeed(t, repo, "sn-post", "SiteName Post")

	r := httptest.NewRequest("GET", "/posts/sn-post", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	// Config.BaseURL is "https://example.com" → hostname is "example.com".
	if !strings.Contains(w.Body.String(), "site:example.com") {
		t.Errorf("expected site:example.com in body, got: %s", w.Body.String())
	}
}

// TestFull_templateData_requestURL verifies that the live *http.Request is
// accessible in the template and {{.Request.URL.Path}} matches the request.
func TestFull_templateData_requestURL(t *testing.T) {
	const tpl = `path:{{.Request.URL.Path}}`
	dir := intTmpDir(t, `list:{{.Request.URL.Path}}`, tpl)
	_, handler, repo := intSetup(t, Templates(dir))
	intSeed(t, repo, "url-post", "URL Post")

	r := httptest.NewRequest("GET", "/posts/url-post", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "path:/posts/url-post") {
		t.Errorf("expected path:/posts/url-post in body, got: %s", w.Body.String())
	}
}

// — G9: Social + SitemapConfig (M5 + M3) ——————————————————————————————————

// TestFull_social_ogTagsInHTML verifies that a module configured with
// Social(OpenGraph, TwitterCard) and SitemapConfig{} (M3) renders og:title and
// twitter:card meta tags inside forge:head when HeadFunc returns a non-empty Title.
func TestFull_social_ogTagsInHTML(t *testing.T) {
	const show = `<!DOCTYPE html><html><head>{{template "forge:head" .Head}}</head></html>`
	dir := intTmpDir(t, `<p>list</p>`, show)
	_, handler, repo := intSetup(t,
		Social(OpenGraph, TwitterCard),
		SitemapConfig{},
		HeadFunc(func(_ Context, p *testPost) Head {
			return Head{Title: p.Title, Description: "A test post"}
		}),
		Templates(dir),
	)
	intSeed(t, repo, "og-post", "OG World")

	r := httptest.NewRequest("GET", "/posts/og-post", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `property="og:title"`) {
		t.Errorf("og:title missing from body:\n%s", body)
	}
	if !strings.Contains(body, `name="twitter:card"`) {
		t.Errorf("twitter:card missing from body:\n%s", body)
	}
}

// TestFull_social_draftReturns404 verifies that a Draft post returns 404
// when Social and SitemapConfig options are active — lifecycle is enforced
// regardless of which M5 options are present.
func TestFull_social_draftReturns404(t *testing.T) {
	dir := intTmpDir(t, `<p>list</p>`, `<p>{{.Content.Title}}</p>`)
	_, handler, repo := intSetup(t,
		Social(OpenGraph, TwitterCard),
		SitemapConfig{},
		Templates(dir),
	)
	// Seed a draft post directly (intSeed always seeds Published).
	draft := &testPost{
		Node:  Node{ID: NewID(), Slug: "og-draft", Status: Draft},
		Title: "Draft OG Post",
	}
	if err := repo.Save(context.Background(), draft); err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	r := httptest.NewRequest("GET", "/posts/og-draft", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d (want 404); body: %s", w.Code, w.Body.String())
	}
}

// — G10: AI indexing + content negotiation (M5 + M4) ————————————————————

// TestFull_ai_llmsTxt_publishedPresent verifies that /llms.txt lists Published
// items and excludes Draft items when AIIndex(LLMsTxt) is configured.
func TestFull_ai_llmsTxt_publishedPresent(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "AI World", "body text", Published)
	_ = seedAIPost(t, repo, "AI Draft", "draft body", Draft)

	store := NewLLMsStore("example.com")
	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(LLMsTxt),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(store, "https://example.com")
	m.regenerateAI(testAICtx())

	mux := http.NewServeMux()
	m.Register(mux)
	mux.Handle("GET /llms.txt", store.CompactHandler())

	r := httptest.NewRequest("GET", "/llms.txt", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, pub.Title) {
		t.Errorf("published title %q missing from /llms.txt:\n%s", pub.Title, body)
	}
	if strings.Contains(body, "AI Draft") {
		t.Errorf("/llms.txt should not contain Draft item:\n%s", body)
	}
}

// TestFull_ai_aiDoc_published verifies that /posts/{slug}/aidoc returns 200
// for a Published item and contains the AIDoc v1 header.
func TestFull_ai_aiDoc_published(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "AIDoc Published", "body text", Published)

	store := NewLLMsStore("example.com")
	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(AIDoc),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(store, "https://example.com")

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/"+pub.Slug+"/aidoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "+++aidoc+v1+++") {
		t.Errorf("body missing AIDoc header:\n%s", w.Body.String())
	}
}

// TestFull_ai_aiDoc_draftReturns404 verifies that /posts/{slug}/aidoc returns
// 404 for a Draft item — lifecycle enforcement on the AIDoc endpoint.
func TestFull_ai_aiDoc_draftReturns404(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	draft := seedAIPost(t, repo, "AIDoc Draft", "body text", Draft)

	store := NewLLMsStore("example.com")
	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(AIDoc),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(store, "https://example.com")

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/"+draft.Slug+"/aidoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d (want 404); body: %s", w.Code, w.Body.String())
	}
}

// TestFull_ai_markdownContentNeg verifies that Accept: text/markdown returns
// the Markdown() body alongside the AIDoc option being active (M4 + M5).
func TestFull_ai_markdownContentNeg(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "Markdown Negotiation", "markdown body text", Published)

	store := NewLLMsStore("example.com")
	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(AIDoc),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(store, "https://example.com")

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/"+pub.Slug, nil)
	r.Header.Set("Accept", "text/markdown")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/markdown") {
		t.Errorf("Content-Type = %q; want text/markdown", ct)
	}
	if !strings.Contains(w.Body.String(), pub.Title) {
		t.Errorf("markdown body missing post title %q:\n%s", pub.Title, w.Body.String())
	}
}

// — G11: RSS feed + AfterPublish signal (M5 + M1) ————————————————————————

// TestFull_feed_publishedInFeed verifies that /posts/feed.xml returns a valid
// RSS 2.0 document containing Published items.
func TestFull_feed_publishedInFeed(t *testing.T) {
	repo := NewMemoryRepo[*testFeedPost]()
	pub := seedFeedPost(t, repo, "Feed World", Published)

	store := NewFeedStore("example.com", "https://example.com")
	m := NewModule((*testFeedPost)(nil),
		Repo(repo),
		At("/posts"),
		Feed(FeedConfig{Title: "Integration Blog"}),
		HeadFunc(feedHeadFunc),
	)
	m.setFeedStore(store, "https://example.com")
	m.regenerateFeed(testFeedCtx())

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/feed.xml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `version="2.0"`) {
		t.Errorf("body missing RSS version attribute:\n%s", body)
	}
	if !strings.Contains(body, pub.Title) {
		t.Errorf("published title %q missing from feed:\n%s", pub.Title, body)
	}
}

// TestFull_feed_draftAbsent verifies that Draft items are excluded from the
// RSS feed at /posts/feed.xml.
func TestFull_feed_draftAbsent(t *testing.T) {
	repo := NewMemoryRepo[*testFeedPost]()
	_ = seedFeedPost(t, repo, "Feed Published", Published)
	_ = seedFeedPost(t, repo, "Feed Draft Title", Draft)

	store := NewFeedStore("example.com", "https://example.com")
	m := NewModule((*testFeedPost)(nil),
		Repo(repo),
		At("/posts"),
		Feed(FeedConfig{Title: "Blog"}),
		HeadFunc(feedHeadFunc),
	)
	m.setFeedStore(store, "https://example.com")
	m.regenerateFeed(testFeedCtx())

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/feed.xml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if strings.Contains(w.Body.String(), "Feed Draft Title") {
		t.Errorf("feed should not contain Draft item:\n%s", w.Body.String())
	}
}

// TestFull_feed_afterPublishSignalFires verifies that the AfterPublish signal
// fires when a Draft item is transitioned to Published while Feed is active
// (M5 + M1 cross-milestone).
func TestFull_feed_afterPublishSignalFires(t *testing.T) {
	var fired atomic.Int32
	repo := NewMemoryRepo[*testFeedPost]()

	// Seed a Draft post that will be updated to Published.
	draft := &testFeedPost{
		Node:  Node{ID: NewID(), Slug: "signal-feed", Status: Draft},
		Title: "Signal Feed Post",
	}
	if err := repo.Save(context.Background(), draft); err != nil {
		t.Fatalf("seed draft: %v", err)
	}

	store := NewFeedStore("example.com", "https://example.com")
	m := NewModule((*testFeedPost)(nil),
		Repo(repo),
		At("/posts"),
		Feed(FeedConfig{Title: "Blog"}),
		HeadFunc(feedHeadFunc),
		On(AfterPublish, func(_ Context, _ *testFeedPost) error {
			fired.Add(1)
			return nil
		}),
	)
	m.setFeedStore(store, "https://example.com")

	body, _ := json.Marshal(map[string]any{"Title": "Signal Feed Post", "Status": "published"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest("PUT", "/posts/signal-feed", bytes.NewReader(body)),
		editorUser(),
	)
	r.SetPathValue("slug", "signal-feed")
	m.updateHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("update: status = %d; body: %s", w.Code, w.Body.String())
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fired.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fired.Load() == 0 {
		t.Error("AfterPublish did not fire within 500ms")
	}
}

// — G12: Full M5 stack (M5 + M3 + M4) ———————————————————————————————————

// TestFull_fullM5_htmlHasOGTags verifies that a module with all M5 options
// (Social, AIIndex, Feed, SitemapConfig) plus M3/M4 (HeadFunc, Templates)
// still renders og:title and twitter:card correctly in forge:head.
func TestFull_fullM5_htmlHasOGTags(t *testing.T) {
	const show = `<!DOCTYPE html><html><head>{{template "forge:head" .Head}}</head></html>`
	dir := intTmpDir(t, `<p>list</p>`, show)

	aiStore := NewLLMsStore("example.com")
	feedSt := NewFeedStore("example.com", "https://example.com")
	repo := NewMemoryRepo[*testAIPost]()

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		Social(OpenGraph, TwitterCard),
		AIIndex(LLMsTxt, AIDoc),
		Feed(FeedConfig{Title: "Full M5 Blog"}),
		SitemapConfig{},
		HeadFunc(aiHeadFunc),
		Templates(dir),
	)
	m.setAIRegistry(aiStore, "https://example.com")
	m.setFeedStore(feedSt, "https://example.com")

	pub := seedAIPost(t, repo, "Full M5 Post", "body text", Published)
	m.regenerateAI(testAICtx())
	m.regenerateFeed(testAICtx())

	if err := m.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	mux := http.NewServeMux()
	m.Register(mux)
	mux.Handle("GET /llms.txt", aiStore.CompactHandler())

	r := httptest.NewRequest("GET", "/posts/"+pub.Slug, nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `property="og:title"`) {
		t.Errorf("og:title missing from full M5 HTML:\n%s", body)
	}
	if !strings.Contains(body, `name="twitter:card"`) {
		t.Errorf("twitter:card missing from full M5 HTML:\n%s", body)
	}
}

// TestFull_fullM5_llmsTxt verifies that /llms.txt is populated with Published
// items when the full M5 option set is active.
func TestFull_fullM5_llmsTxt(t *testing.T) {
	aiStore := NewLLMsStore("example.com")
	feedSt := NewFeedStore("example.com", "https://example.com")
	repo := NewMemoryRepo[*testAIPost]()

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		Social(OpenGraph, TwitterCard),
		AIIndex(LLMsTxt, AIDoc),
		Feed(FeedConfig{Title: "Full M5 Blog"}),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(aiStore, "https://example.com")
	m.setFeedStore(feedSt, "https://example.com")

	pub := seedAIPost(t, repo, "LLMs Entry", "body text", Published)
	m.regenerateAI(testAICtx())

	mux := http.NewServeMux()
	m.Register(mux)
	mux.Handle("GET /llms.txt", aiStore.CompactHandler())

	r := httptest.NewRequest("GET", "/llms.txt", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), pub.Title) {
		t.Errorf("published title %q missing from /llms.txt:\n%s", pub.Title, w.Body.String())
	}
}

// TestFull_fullM5_aiDoc verifies that /posts/{slug}/aidoc returns a valid
// AIDoc response when the full M5 option set is active.
func TestFull_fullM5_aiDoc(t *testing.T) {
	aiStore := NewLLMsStore("example.com")
	feedSt := NewFeedStore("example.com", "https://example.com")
	repo := NewMemoryRepo[*testAIPost]()

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		Social(OpenGraph, TwitterCard),
		AIIndex(LLMsTxt, AIDoc),
		Feed(FeedConfig{Title: "Full M5 Blog"}),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(aiStore, "https://example.com")
	m.setFeedStore(feedSt, "https://example.com")

	pub := seedAIPost(t, repo, "AIDoc Full M5", "body text", Published)

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/"+pub.Slug+"/aidoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "+++aidoc+v1+++") {
		t.Errorf("body missing AIDoc header:\n%s", w.Body.String())
	}
}

// TestFull_fullM5_feed verifies that /posts/feed.xml returns a valid RSS 2.0
// document with Published items when the full M5 option set is active.
func TestFull_fullM5_feed(t *testing.T) {
	aiStore := NewLLMsStore("example.com")
	feedSt := NewFeedStore("example.com", "https://example.com")
	repo := NewMemoryRepo[*testAIPost]()

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		Social(OpenGraph, TwitterCard),
		AIIndex(LLMsTxt, AIDoc),
		Feed(FeedConfig{Title: "Full M5 Blog"}),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(aiStore, "https://example.com")
	m.setFeedStore(feedSt, "https://example.com")

	pub := seedAIPost(t, repo, "Feed Full M5", "body text", Published)
	m.regenerateFeed(testAICtx())

	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest("GET", "/posts/feed.xml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `version="2.0"`) {
		t.Errorf("body missing RSS version:\n%s", body)
	}
	if !strings.Contains(body, pub.Title) {
		t.Errorf("published title %q missing from feed:\n%s", pub.Title, body)
	}
}
