package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// — Test helpers ——————————————————————————————————————————————————————————

// testPost is a minimal content type for Module tests.
// It embeds Node (providing ID, Slug, Status) and has a required Title field.
type testPost struct {
	Node
	Title string `forge:"required"`
	Body  string
}

// testMDPost is a testPost that also implements [Markdownable].
type testMDPost struct {
	Node
	Title string `forge:"required"`
	Body  string
}

func (p *testMDPost) Markdown() string { return "# " + p.Title + "\n\n" + p.Body }

// withUser injects user into the request context so [ContextFrom] picks it up.
func withUser(r *http.Request, user User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, user))
}

// authorUser returns a User with the Author role.
func authorUser() User { return User{ID: "author-1", Name: "Alice", Roles: []Role{Author}} }

// editorUser returns a User with the Editor role.
func editorUser() User { return User{ID: "editor-1", Name: "Bob", Roles: []Role{Editor}} }

// seedPost inserts a testPost into repo and returns it.
func seedPost(t *testing.T, repo Repository[*testPost], title string, status Status) *testPost {
	t.Helper()
	p := &testPost{
		Node:  Node{ID: NewID(), Slug: GenerateSlug(title), Status: status},
		Title: title,
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("seedPost: %v", err)
	}
	return p
}

// newTestModule creates a Module[*testPost] backed by the given repo.
func newTestModule(repo Repository[*testPost], opts ...Option) *Module[*testPost] {
	all := append([]Option{Repo[*testPost](repo)}, opts...)
	return NewModule((*testPost)(nil), all...)
}

// seedMDPost inserts a testMDPost into repo and returns it.
func seedMDPost(t *testing.T, repo Repository[*testMDPost], title string, status Status) *testMDPost {
	t.Helper()
	p := &testMDPost{
		Node:  Node{ID: NewID(), Slug: GenerateSlug(title), Status: status},
		Title: title,
		Body:  "Hello world",
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("seedMDPost: %v", err)
	}
	return p
}

// — List handler tests ————————————————————————————————————————————————————

func TestModuleListGuestPublishedOnly(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	seedPost(t, repo, "Published Post", Published)
	seedPost(t, repo, "Draft Post", Draft)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts", nil)
	// No user injected → GuestUser.
	m.listHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	var items []*testPost
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("items count = %d; want 1 (published only)", len(items))
	}
	if items[0].Status != Published {
		t.Errorf("item status = %q; want %q", items[0].Status, Published)
	}
}

func TestModuleListAuthorSeesAll(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	seedPost(t, repo, "Published Post", Published)
	seedPost(t, repo, "Draft Post", Draft)
	seedPost(t, repo, "Scheduled Post", Scheduled)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := withUser(httptest.NewRequest(http.MethodGet, "/testposts", nil), authorUser())
	m.listHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	var items []*testPost
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("items count = %d; want 3 (all statuses)", len(items))
	}
}

// — Show handler tests ————————————————————————————————————————————————————

func TestModuleShowPublishedGuest(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "My Post", Published)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
}

func TestModuleShowDraftGuest(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Draft Post", Draft)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	// Guests must not see draft content — respond 404, not 403.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d; want 404 (must not leak existence)", w.Code)
	}
}

func TestModuleShowDraftAuthor(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Draft Post", Draft)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := withUser(httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil), authorUser())
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200 (authors see all statuses)", w.Code)
	}
}

// — Create handler tests ——————————————————————————————————————————————————

func TestModuleCreateValidation(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := newTestModule(repo)

	// POST with missing required Title.
	body, _ := json.Marshal(map[string]string{"Body": "Hello"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/testposts", bytes.NewReader(body)),
		authorUser(),
	)
	r.Header.Set("Content-Type", "application/json")
	m.createHandler(w, r)

	// Validation failure → 422 Unprocessable Entity.
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422 (validation error)", w.Code)
	}

	// Nothing saved.
	items, _ := repo.FindAll(context.Background(), ListOptions{})
	if len(items) != 0 {
		t.Errorf("repo count = %d; want 0 (aborted on validation failure)", len(items))
	}
}

func TestModuleCreateSuccess(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	m := newTestModule(repo)

	body, _ := json.Marshal(map[string]string{"Title": "Hello World", "Body": "Content here"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/testposts", bytes.NewReader(body)),
		authorUser(),
	)
	r.Header.Set("Content-Type", "application/json")
	m.createHandler(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201\nbody: %s", w.Code, w.Body.String())
	}

	var created testPost
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if created.ID == "" {
		t.Error("ID must be set on created item")
	}
	if created.Slug == "" {
		t.Error("Slug must be set on created item")
	}

	// Verify it was saved.
	items, _ := repo.FindAll(context.Background(), ListOptions{})
	if len(items) != 1 {
		t.Errorf("repo count = %d; want 1", len(items))
	}
}

// — Update handler tests ——————————————————————————————————————————————————

func TestModuleUpdateForbiddenGuest(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Existing Post", Published)

	m := newTestModule(repo)
	body, _ := json.Marshal(map[string]string{"Title": "Updated"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/testposts/"+p.Slug, bytes.NewReader(body))
	r.SetPathValue("slug", p.Slug)
	// No user → GuestUser; default writeRole is Author.
	m.updateHandler(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", w.Code)
	}
}

// — Delete handler tests ——————————————————————————————————————————————————

func TestModuleDeleteForbiddenAuthor(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "My Post", Published)

	m := newTestModule(repo) // default deleteRole = Editor
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest(http.MethodDelete, "/testposts/"+p.Slug, nil),
		authorUser(), // Author < Editor → forbidden
	)
	r.SetPathValue("slug", p.Slug)
	m.deleteHandler(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d; want 403", w.Code)
	}
}

// — Content negotiation tests —————————————————————————————————————————————

func TestModuleContentNegotiationJSON(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "My Post", Published)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.Header.Set("Accept", "application/json")
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q; want application/json", ct)
	}
	vary := w.Header().Get("Vary")
	if vary == "" {
		t.Error("Vary header must be set")
	}
}

func TestModuleContentNegotiationMarkdown(t *testing.T) {
	repo := NewMemoryRepo[*testMDPost]()
	title := "Markdown Post"
	p := seedMDPost(t, repo, title, Published)

	m := NewModule((*testMDPost)(nil), Repo[*testMDPost](repo))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testmdposts/"+p.Slug, nil)
	r.Header.Set("Accept", "text/markdown")
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200\nbody: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type = %q; want text/markdown; charset=utf-8", ct)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("markdown body must not be empty")
	}
}

func TestModuleContentNegotiationMarkdownUnsupported(t *testing.T) {
	// testPost does NOT implement Markdownable.
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Plain Post", Published)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.Header.Set("Accept", "text/markdown")
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	// text/markdown unsupported → 406 Not Acceptable.
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("status = %d; want 406", w.Code)
	}
}

func TestModuleContentNegotiationHTML(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "HTML Post", Published)

	m := newTestModule(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.Header.Set("Accept", "text/html")
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	// No templates registered in Step 10 → 406.
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("status = %d; want 406 (no templates registered)", w.Code)
	}
}

// — Cache tests ———————————————————————————————————————————————————————————

func TestModuleCacheMISS(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Cached Post", Published)

	m := newTestModule(repo, Cache(5*time.Minute))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
	r.SetPathValue("slug", p.Slug)
	m.showHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	if got := w.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("X-Cache = %q; want MISS", got)
	}
}

func TestModuleCacheHIT(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	p := seedPost(t, repo, "Cached Post", Published)

	m := newTestModule(repo, Cache(5*time.Minute))

	do := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/testposts/"+p.Slug, nil)
		r.SetPathValue("slug", p.Slug)
		m.showHandler(w, r)
		return w
	}

	w1 := do()
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Fatal("first request should be MISS")
	}

	w2 := do()
	if got := w2.Header().Get("X-Cache"); got != "HIT" {
		t.Errorf("X-Cache = %q; want HIT on second identical request", got)
	}
	if w2.Body.String() != w1.Body.String() {
		t.Error("cached body does not match original body")
	}
}

func TestModuleCacheInvalidatedOnCreate(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()
	_ = seedPost(t, repo, "Existing Post", Published)

	m := newTestModule(repo, Cache(5*time.Minute))

	// Warm the list cache.
	warmReq := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/testposts", nil)
		m.listHandler(w, r)
		return w
	}
	w1 := warmReq()
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Fatal("first request should be MISS")
	}
	w2 := warmReq()
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Fatal("second request should be HIT")
	}

	// Create a new post → cache must be invalidated.
	body, _ := json.Marshal(map[string]string{"Title": "New Post"})
	cw := httptest.NewRecorder()
	cr := withUser(
		httptest.NewRequest(http.MethodPost, "/testposts", bytes.NewReader(body)),
		authorUser(),
	)
	m.createHandler(cw, cr)
	if cw.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", cw.Code)
	}

	// Next list request should be MISS (cache was flushed).
	w3 := warmReq()
	if got := w3.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("X-Cache = %q; want MISS after create (cache should be invalidated)", got)
	}
}

// — Signal tests ——————————————————————————————————————————————————————————

func TestModuleSignalBeforeCreateAborts(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()

	abortErr := Err("title", "forbidden title")
	hook := On(BeforeCreate, func(ctx Context, p *testPost) error {
		return abortErr
	})

	m := newTestModule(repo, hook)
	body, _ := json.Marshal(map[string]string{"Title": "Forbidden"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/testposts", bytes.NewReader(body)),
		authorUser(),
	)
	m.createHandler(w, r)

	// BeforeCreate error → handler must not save and must return an error response.
	if w.Code == http.StatusCreated {
		t.Error("BeforeCreate error should abort the create operation")
	}

	items, _ := repo.FindAll(context.Background(), ListOptions{})
	if len(items) != 0 {
		t.Error("nothing should be saved when BeforeCreate returns an error")
	}
}

func TestModuleSignalAfterCreateFires(t *testing.T) {
	repo := NewMemoryRepo[*testPost]()

	var fired atomic.Int32
	hook := On(AfterCreate, func(ctx Context, p *testPost) error {
		fired.Add(1)
		return nil
	})

	m := newTestModule(repo, hook)
	body, _ := json.Marshal(map[string]string{"Title": "Hello"})
	w := httptest.NewRecorder()
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/testposts", bytes.NewReader(body)),
		authorUser(),
	)
	m.createHandler(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d\n%s", w.Code, w.Body.String())
	}

	// AfterCreate is asynchronous — allow time for the goroutine to run.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fired.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if fired.Load() == 0 {
		t.Error("AfterCreate hook did not fire within 500ms")
	}
}

// — Benchmark ——————————————————————————————————————————————————————————————

// BenchmarkModuleRequest measures the hot path for a cached GET show request.
func BenchmarkModuleRequest(b *testing.B) {
	repo := NewMemoryRepo[*testPost]()
	p := &testPost{
		Node:  Node{ID: NewID(), Slug: "bench-post", Status: Published},
		Title: "Benchmark Post",
		Body:  "Some body content for benchmarking.",
	}
	_ = repo.Save(context.Background(), p)

	m := NewModule((*testPost)(nil),
		Repo[*testPost](repo),
		Cache(5*time.Minute),
	)

	// Warm the cache.
	w0 := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/testposts/bench-post", nil)
	r0.SetPathValue("slug", "bench-post")
	m.showHandler(w0, r0)

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/testposts/bench-post", nil)
		r.SetPathValue("slug", "bench-post")
		m.showHandler(w, r)
	}
}
