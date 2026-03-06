package forge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// — Test helpers ——————————————————————————————————————————————————————————

// testAIPost is a content type for AI tests that implements Markdownable.
type testAIPost struct {
	Node
	Title   string `forge:"required"`
	Summary string
	Body    string
}

func (p *testAIPost) Markdown() string { return "# " + p.Title + "\n\n" + p.Body }
func (p *testAIPost) AISummary() string {
	if p.Summary != "" {
		return p.Summary
	}
	return ""
}

// seedAIPost saves a testAIPost in repo and returns it.
func seedAIPost(t *testing.T, repo Repository[*testAIPost], title, body string, status Status) *testAIPost {
	t.Helper()
	p := &testAIPost{
		Node:  Node{ID: NewID(), Slug: GenerateSlug(title), Status: status},
		Title: title,
		Body:  body,
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("seedAIPost: %v", err)
	}
	return p
}

// aiHeadFunc is a HeadFunc for testAIPost that uses its Title.
func aiHeadFunc(_ Context, p *testAIPost) Head {
	return Head{
		Title:       p.Title,
		Description: "Desc: " + p.Title,
		Type:        Article,
	}
}

// testAICtx returns a background Context for use in AI test helpers.
func testAICtx() Context { return NewTestContext(GuestUser) }

// — TestAIIndexOption —————————————————————————————————————————————————————

func TestAIIndexOption(t *testing.T) {
	cases := []struct {
		name     string
		features []AIFeature
		wantLen  int
	}{
		{"llms_txt_only", []AIFeature{LLMsTxt}, 1},
		{"ai_doc_only", []AIFeature{AIDoc}, 1},
		{"llms_txt_full_only", []AIFeature{LLMsTxtFull}, 1},
		{"all_three", []AIFeature{LLMsTxt, LLMsTxtFull, AIDoc}, 3},
		{"empty", nil, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opt := AIIndex(tc.features...)
			ai, ok := opt.(aiIndexOption)
			if !ok {
				t.Fatalf("AIIndex() returned %T, want aiIndexOption", opt)
			}
			if got := len(ai.features); got != tc.wantLen {
				t.Errorf("len(features) = %d, want %d", got, tc.wantLen)
			}
		})
	}
}

// — TestWithoutIDOption ———————————————————————————————————————————————————

func TestWithoutIDOption(t *testing.T) {
	t.Run("returns_withoutIDOption", func(t *testing.T) {
		opt := WithoutID()
		if _, ok := opt.(withoutIDOption); !ok {
			t.Fatalf("WithoutID() returned %T, want withoutIDOption", opt)
		}
	})

	t.Run("sets_withoutID_on_module", func(t *testing.T) {
		repo := NewMemoryRepo[*testAIPost]()
		m := NewModule((*testAIPost)(nil),
			Repo(repo),
			AIIndex(AIDoc),
			WithoutID(),
		)
		if !m.withoutID {
			t.Error("m.withoutID should be true after WithoutID() option")
		}
	})

	t.Run("default_withoutID_is_false", func(t *testing.T) {
		repo := NewMemoryRepo[*testAIPost]()
		m := NewModule((*testAIPost)(nil), Repo(repo), AIIndex(AIDoc))
		if m.withoutID {
			t.Error("m.withoutID should be false by default")
		}
	})
}

// — TestLLMsTxtEndpoint ———————————————————————————————————————————————————

func TestLLMsTxtEndpoint(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "Hello World", "body text", Published)
	_ = seedAIPost(t, repo, "Draft Post", "draft body", Draft)

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

	t.Run("published_item_present", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, pub.Title) {
			t.Errorf("body missing published title %q:\n%s", pub.Title, body)
		}
		if !strings.Contains(body, "example.com") {
			t.Errorf("body missing site name:\n%s", body)
		}
	})

	t.Run("draft_item_absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		body := w.Body.String()
		if strings.Contains(body, "Draft Post") {
			t.Errorf("body contains draft item title:\n%s", body)
		}
	})
}

// — TestLLMsTxtTemplate ———————————————————————————————————————————————————

func TestLLMsTxtTemplate(t *testing.T) {
	store := NewLLMsStore("myblog.io")
	store.registerCompact()
	store.SetCompact("/posts", []LLMsEntry{
		{Title: "First Post", URL: "https://myblog.io/posts/first-post", Summary: "An intro."},
		{Title: "No Summary Post", URL: "https://myblog.io/posts/no-summary"},
	})

	r := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
	w := httptest.NewRecorder()
	store.CompactHandler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()

	cases := []struct {
		label string
		want  string
	}{
		{"site_name_header", "# myblog.io"},
		{"entry_with_summary", "- [First Post](https://myblog.io/posts/first-post): An intro."},
		{"entry_without_summary", "- [No Summary Post](https://myblog.io/posts/no-summary)"},
		{"content_type", "text/plain"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if tc.label == "content_type" {
				if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, tc.want) {
					t.Errorf("Content-Type = %q, want %q", ct, tc.want)
				}
				return
			}
			if !strings.Contains(body, tc.want) {
				t.Errorf("body missing %q:\n%s", tc.want, body)
			}
		})
	}
}

// — TestLLMsFullTxtEndpoint ———————————————————————————————————————————————

func TestLLMsFullTxtEndpoint(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "Markdown Article", "Full markdown body here.", Published)
	_ = seedAIPost(t, repo, "Hidden Draft", "draft content", Draft)

	store := NewLLMsStore("example.com")
	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/articles"),
		AIIndex(LLMsTxtFull),
		HeadFunc(aiHeadFunc),
	)
	m.setAIRegistry(store, "https://example.com")
	m.regenerateAI(testAICtx())

	mux := http.NewServeMux()
	mux.Handle("GET /llms-full.txt", store.FullHandler())

	t.Run("published_item_present", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/llms-full.txt", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, pub.Title) {
			t.Errorf("body missing published title %q:\n%s", pub.Title, body)
		}
		if !strings.Contains(body, pub.Body) {
			t.Errorf("body missing markdown body %q:\n%s", pub.Body, body)
		}
	})

	t.Run("draft_item_absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/llms-full.txt", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		body := w.Body.String()
		if strings.Contains(body, "Hidden Draft") {
			t.Errorf("body contains draft item:\n%s", body)
		}
	})
}

// — TestLLMsFullTxtFallback ———————————————————————————————————————————————

func TestLLMsFullTxtFallback(t *testing.T) {
	// Store has no module registered with LLMsTxtFull —
	// /llms-full.txt should never be mounted, so the mux returns 404.
	store := NewLLMsStore("example.com")
	store.registerCompact() // only compact registered, not full

	mux := http.NewServeMux()
	// App.Handler only mounts /llms-full.txt when HasFull() == true.
	if store.HasFull() {
		mux.Handle("GET /llms-full.txt", store.FullHandler())
	}

	r := httptest.NewRequest(http.MethodGet, "/llms-full.txt", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when no LLMsTxtFull module registered", w.Code)
	}
}

// — TestLLMsFullTxtHeader —————————————————————————————————————————————————

func TestLLMsFullTxtHeader(t *testing.T) {
	store := NewLLMsStore("acmecorp.com")
	store.registerFull()
	store.SetFull("/posts", "## Example\nURL: https://acmecorp.com/posts/example\n\ncontent\n---\n\n")
	// prime compact so total count = 1
	store.SetCompact("/posts", []LLMsEntry{{Title: "Example", URL: "https://acmecorp.com/posts/example"}})

	r := httptest.NewRequest(http.MethodGet, "/llms-full.txt", nil)
	w := httptest.NewRecorder()
	store.FullHandler().ServeHTTP(w, r)

	body := w.Body.String()
	cases := []struct {
		label string
		want  string
	}{
		{"site_name_corpus_header", "# acmecorp.com \u2014 Full Content Corpus"},
		{"generated_by_forge", "> Generated by Forge on"},
		{"only_published_content", "Only published content"},
		{"item_count", "1 items"},
		{"content_included", "## Example"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if !strings.Contains(body, tc.want) {
				t.Errorf("body missing %q:\n%s", tc.want, body)
			}
		})
	}
}

// — TestAIDocEndpoint —————————————————————————————————————————————————————

func TestAIDocEndpoint(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "My First Post", "Markdown **body** content.", Published)

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(AIDoc),
		HeadFunc(aiHeadFunc),
	)
	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest(http.MethodGet, "/posts/"+pub.Slug+"/aidoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()

	cases := []struct {
		label string
		want  string
	}{
		{"aidoc_header", "+++aidoc+v1+++"},
		{"type_field", "type:     Article"},
		{"id_field", "id:       " + pub.ID},
		{"slug_field", "slug:     " + pub.Slug},
		{"title_field", "title:    My First Post"},
		{"summary_from_aisummary", "summary:"},
		{"body_separator", "+++"},
		{"markdown_body", "Markdown **body** content."},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if !strings.Contains(body, tc.want) {
				t.Errorf("body missing %q:\n%s", tc.want, body)
			}
		})
	}
}

// — TestAIDocNotFound —————————————————————————————————————————————————————

func TestAIDocNotFound(t *testing.T) {
	cases := []struct {
		name   string
		status Status
	}{
		{"draft", Draft},
		{"archived", Archived},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewMemoryRepo[*testAIPost]()
			p := seedAIPost(t, repo, "Hidden "+tc.name, "body", tc.status)

			m := NewModule((*testAIPost)(nil),
				Repo(repo),
				At("/posts"),
				AIIndex(AIDoc),
			)
			mux := http.NewServeMux()
			m.Register(mux)

			r := httptest.NewRequest(http.MethodGet, "/posts/"+p.Slug+"/aidoc", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404 for %s item", w.Code, tc.name)
			}
		})
	}
}

// — TestAIDocWithoutID ————————————————————————————————————————————————————

func TestAIDocWithoutID(t *testing.T) {
	repo := NewMemoryRepo[*testAIPost]()
	pub := seedAIPost(t, repo, "Public Post", "some body text", Published)

	m := NewModule((*testAIPost)(nil),
		Repo(repo),
		At("/posts"),
		AIIndex(AIDoc),
		WithoutID(),
		HeadFunc(aiHeadFunc),
	)
	mux := http.NewServeMux()
	m.Register(mux)

	r := httptest.NewRequest(http.MethodGet, "/posts/"+pub.Slug+"/aidoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()

	t.Run("id_suppressed", func(t *testing.T) {
		if strings.Contains(body, "id:") {
			t.Errorf("body contains id: field when WithoutID() is set:\n%s", body)
		}
	})

	t.Run("slug_present", func(t *testing.T) {
		if !strings.Contains(body, "slug:     "+pub.Slug) {
			t.Errorf("body missing slug field:\n%s", body)
		}
	})

	t.Run("id_value_absent", func(t *testing.T) {
		if strings.Contains(body, pub.ID) {
			t.Errorf("body contains ID value %q when WithoutID() is set:\n%s", pub.ID, body)
		}
	})
}
