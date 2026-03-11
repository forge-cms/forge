// Package main is a self-contained Forge JSON API — a Go resource curator —
// demonstrating authentication, role-based authorisation, validation hooks,
// and legacy URL redirects with no HTML templates.
//
// All responses are JSON (content negotiation is automatic; no Accept header
// required). There are no templates — the module serves JSON natively.
//
// # Authentication
//
// Two roles are pre-seeded so the role hierarchy is visible:
//
//	Author (forge.Author)  — can read Published resources (GET list + show)
//	Editor (forge.Editor)  — can also create and update resources (POST / PUT)
//
// NOTE: tokens are hardcoded for demonstration only.
// In production use forge.SignToken to issue signed HMAC tokens.
//
// Tokens are printed at server startup. Use them with curl:
//
//	# Public read — no token required
//	curl http://localhost:8082/resources
//	curl http://localhost:8082/resources/go-language-spec
//
//	# Legacy redirect (301 → /resources/go-language-spec)
//	curl -L http://localhost:8082/resources/go-spec
//
//	# Create — requires Editor token
//	curl -X POST http://localhost:8082/resources \
//	  -H "Authorization: Bearer <editor-token>" \
//	  -H "Content-Type: application/json" \
//	  -d '{"title":"My Resource","url":"https://example.com","description":"A great Go resource"}'
//
//	# Update — requires Editor token
//	curl -X PUT http://localhost:8082/resources/my-resource \
//	  -H "Authorization: Bearer <editor-token>" \
//	  -H "Content-Type: application/json" \
//	  -d '{"title":"My Resource","url":"https://example.com","description":"Updated description"}'
//
// Run with:
//
//	cd example/api && go run .
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/forge-cms/forge"
)

// secret is the HMAC signing key shared between forge.BearerHMAC (which
// validates incoming tokens) and forge.SignToken (which issued them).
// In production read this from an environment variable or secrets store —
// never hard-code it.
const secret = "forge-api-example-secret-32bytes!"

// NOTE: tokens are hardcoded for demonstration only.
// In production use forge.SignToken to issue signed HMAC tokens.
var (
	editorToken string // forge.Editor role — can create and update
	authorToken string // forge.Author role — read-only (below Editor threshold)
)

func init() {
	var err error

	// Forge: SignToken signs a User into a bearer token verified by BearerHMAC.
	// ttl=0 means the token never expires — set a finite duration in production.
	editorToken, err = forge.SignToken(
		forge.User{ID: "editor-1", Name: "Alice", Roles: []forge.Role{forge.Editor}},
		secret, 0,
	)
	if err != nil {
		log.Fatalf("sign editor token: %v", err)
	}

	authorToken, err = forge.SignToken(
		forge.User{ID: "author-1", Name: "Bob", Roles: []forge.Role{forge.Author}},
		secret, 0,
	)
	if err != nil {
		log.Fatalf("sign author token: %v", err)
	}
}

// Resource is the content type for a curated Go community resource.
// Embedding forge.Node provides ID, Slug, Status, and timestamp fields.
type Resource struct {
	forge.Node

	Title       string `forge:"required,min=3,max=200"`
	URL         string `forge:"required"`
	Description string `forge:"required,min=10"`
	Tags        []string
}

// Head implements [forge.Headable] so Resource satisfies [forge.SitemapNode].
// Forge calls Head() when building the sitemap fragment at /resources/sitemap.xml.
// Returning a zero Canonical is fine — regenerateSitemap derives it from
// Config.BaseURL + slug automatically.
func (r *Resource) Head() forge.Head {
	return forge.Head{Title: r.Title}
}

// Markdown returns a plain-text summary of the resource suitable for AI index.
// Implementing this method makes Resource satisfy [forge.Markdownable], which
// enables the /llms-full.txt corpus endpoint via AIIndex(LLMsTxtFull).
func (r *Resource) Markdown() string {
	return "# " + r.Title + "\n\n" + r.Description + "\n\nURL: " + r.URL
}

func main() {
	repo := forge.NewMemoryRepo[*Resource]()
	seed(repo)

	// Forge: BearerHMAC validates Authorization: Bearer <token> on every request.
	// It returns an AuthFunc — pair it with forge.Authenticate to populate
	// Context.User() so module role checks (forge.Auth) evaluate correctly.
	auth := forge.BearerHMAC(secret)

	m := forge.NewModule((*Resource)(nil),
		forge.At("/resources"),
		forge.Repo(repo),

		// Forge: Auth(Read(Guest), Write(Editor)) sets the minimum role for each
		// operation class. Guest means no token required to read. Editor means
		// a valid Editor-or-higher token is required to create or update.
		forge.Auth(
			forge.Read(forge.Guest),   // GET /resources and GET /resources/{slug} — public
			forge.Write(forge.Editor), // POST /resources and PUT /resources/{slug} — Editor+
		),

		// Forge: BeforeCreate fires synchronously before the item is saved.
		// Returning a non-nil error aborts the create and sends a 422 response.
		// forge.Err creates a forge.ValidationError carrying a field-level message.
		forge.On(forge.BeforeCreate, func(_ forge.Context, r *Resource) error {
			if !strings.HasPrefix(r.URL, "http://") && !strings.HasPrefix(r.URL, "https://") {
				return forge.Err("url", "must start with http:// or https://")
			}
			return nil
		}),

		// Forge: SitemapConfig{} registers a sitemap fragment at /resources/sitemap.xml.
		// The app-level sitemap index at /sitemap.xml aggregates all module fragments.
		forge.SitemapConfig{},

		// Forge: Feed(FeedConfig{}) registers an RSS 2.0 feed at /resources/feed.xml.
		forge.Feed(forge.FeedConfig{Title: "Forge API — Resources", Description: "Curated Go community resources"}),

		// Forge: AIIndex registers AI discovery endpoints.
		//   /llms.txt          — compact content index (llmstxt.org format)
		//   /llms-full.txt     — full markdown corpus (requires Markdownable)
		forge.AIIndex(forge.LLMsTxt, forge.LLMsTxtFull),
	)

	app := forge.New(forge.MustConfig(forge.Config{
		BaseURL: "http://localhost:8082",
		Secret:  []byte(secret),
	}))

	// Forge: Authenticate(auth) runs BearerHMAC on every inbound request and
	// stores the User in the request context so Context.User() returns it.
	// Without this middleware, ctx.User() always returns GuestUser and the
	// write-role check in forge.Auth would never pass.
	//
	// SecurityHeaders() adds HSTS, X-Frame-Options, X-Content-Type-Options,
	// Referrer-Policy, and Content-Security-Policy to every response.
	//
	// RateLimit(100, time.Second) caps each IP at 100 requests per second.
	// Pass forge.TrustedProxy() as a third argument when running behind nginx
	// or Caddy so the real client IP is read from X-Forwarded-For.
	app.Use(
		forge.Authenticate(auth),
		forge.SecurityHeaders(),
		forge.RateLimit(100, time.Second),
	)

	// Forge: Redirects(From("/resources/go-spec"), "/resources/go-language-spec")
	// registers the 301 rule in the RedirectStore so it appears in the manifest
	// at /.well-known/redirects.json. The explicit Handle below registers the same
	// redirect as a fixed mux route so it takes priority over GET /resources/{slug}.
	app.Content(m,
		forge.Redirects(
			forge.From("/resources/go-spec"),
			"/resources/go-language-spec",
		),
	)

	// Explicit route so the redirect fires before GET /resources/{slug}.
	// Fixed-path patterns beat wildcard patterns in Go 1.22's mux.
	app.Handle("GET /resources/go-spec", http.RedirectHandler("/resources/go-language-spec", http.StatusMovedPermanently))

	// Forge: SEO(&RobotsConfig{Sitemaps: true}) registers GET /robots.txt and
	// appends the sitemap index URL to its Sitemap: directives.
	app.SEO(&forge.RobotsConfig{Sitemaps: true})

	// Welcome page — inline HTML so the API example stays template-free.
	app.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Forge API Example</title>
<style>
body{font-family:system-ui,sans-serif;max-width:740px;margin:40px auto;padding:0 24px;color:#1a1a1a}
h1{font-size:1.8rem;margin-bottom:.25rem}p{color:#555;line-height:1.6}
h2{margin:2rem 0 .5rem}ul{padding-left:1.2rem}li{margin:.4rem 0}
code{background:#f4f4f4;padding:2px 6px;border-radius:4px;font-size:.9em}
a{color:#2563eb}
footer{margin-top:3rem;padding-top:1rem;border-top:1px solid #e5e7eb;font-size:.85rem;color:#888}
footer a{color:#2563eb}
</style>
</head>
<body>
<h1>Forge API Example</h1>
<p>A headless JSON API built with Forge. Demonstrates authentication, role-based authorisation, validation hooks, and legacy URL redirects.</p>
<h2>Endpoints</h2>
<ul>
  <li><a href="/resources"><code>GET /resources</code></a> — list published resources (public)</li>
  <li><code>GET /resources/{slug}</code> — single resource (public)</li>
  <li><code>POST /resources</code> — create resource (Editor+)</li>
  <li><code>PUT /resources/{slug}</code> — update resource (Editor+)</li>
  <li><code>DELETE /resources/{slug}</code> — delete resource (Admin)</li>
  <li><code>GET /resources/go-spec</code> — legacy redirect → <code>/resources/go-language-spec</code></li>
</ul>
<h2>AI &amp; Discovery</h2>
<ul>
  <li><a href="/llms.txt"><code>GET /llms.txt</code></a> — compact AI index</li>
  <li><a href="/llms-full.txt"><code>GET /llms-full.txt</code></a> — full AI corpus</li>
  <li><a href="/resources/sitemap.xml"><code>GET /resources/sitemap.xml</code></a> — sitemap fragment</li>
  <li><a href="/resources/feed.xml"><code>GET /resources/feed.xml</code></a> — RSS feed</li>
  <li><a href="/.well-known/redirects.json"><code>GET /.well-known/redirects.json</code></a> — redirect manifest</li>
</ul>
<h2>Authentication</h2>
<p>Send an <code>Authorization: Bearer &lt;token&gt;</code> header. Editor and Author tokens are printed in the server startup log.</p>
<footer>Built with <a href="https://github.com/forge-cms/forge">Forge</a> · <a href="/robots.txt">robots.txt</a></footer>
</body></html>`)
	}))

	log.Println("Forge API — http://localhost:8082")
	log.Println("")
	log.Println("  Editor token (Alice):", editorToken)
	log.Println("  Author token (Bob):  ", authorToken)
	log.Println("")
	log.Println("  Home:                http://localhost:8082/")
	log.Println("  Resources:           http://localhost:8082/resources")
	log.Println("  Legacy redirect:     http://localhost:8082/resources/go-spec  → 301")
	log.Println("  Redirects manifest:  http://localhost:8082/.well-known/redirects.json")
	log.Println("  robots.txt:          http://localhost:8082/robots.txt")
	log.Println("  Sitemap:             http://localhost:8082/resources/sitemap.xml")
	log.Println("  RSS feed:            http://localhost:8082/resources/feed.xml")
	log.Println("  AI index:            http://localhost:8082/llms.txt")
	log.Println("  AI corpus:           http://localhost:8082/llms-full.txt")

	if err := app.Run(":8082"); err != nil {
		log.Fatal(err)
	}
}

// seed inserts 8 Published resources, 1 Draft, and 1 Scheduled into repo.
// In production these come from a SQL database via forge.SQLRepo[T].
func seed(repo forge.Repository[*Resource]) {
	now := time.Now().UTC()
	ctx := context.Background()

	type spec struct {
		slug   string
		title  string
		url    string
		desc   string
		tags   []string
		status forge.Status
		days   int        // days ago (for PublishedAt)
		sched  *time.Time // non-nil for Scheduled
	}

	schedTime := now.Add(48 * time.Hour) // scheduled 2 days in the future

	resources := []spec{
		{
			slug:   "go-language-spec",
			title:  "The Go Language Specification",
			url:    "https://go.dev/ref/spec",
			desc:   "The official specification for the Go programming language. The authoritative reference for parsers, compilers, and anyone building Go tooling.",
			tags:   []string{"spec", "language", "reference"},
			status: forge.Published,
			days:   90,
		},
		{
			slug:   "effective-go",
			title:  "Effective Go",
			url:    "https://go.dev/doc/effective_go",
			desc:   "A guide to writing clear, idiomatic Go code. Covers formatting, names, control flow, data structures, methods, interfaces, and concurrency.",
			tags:   []string{"guide", "style", "idioms"},
			status: forge.Published,
			days:   85,
		},
		{
			slug:   "the-go-blog",
			title:  "The Go Blog",
			url:    "https://go.dev/blog",
			desc:   "Official blog from the Go team at Google. Covers language features, toolchain releases, community news, and deep technical articles.",
			tags:   []string{"blog", "official", "news"},
			status: forge.Published,
			days:   80,
		},
		{
			slug:   "pkg-go-dev",
			title:  "pkg.go.dev",
			url:    "https://pkg.go.dev",
			desc:   "The official Go package index and documentation host. Search for any public Go module, browse docs, and explore the import graph.",
			tags:   []string{"packages", "docs", "modules"},
			status: forge.Published,
			days:   75,
		},
		{
			slug:   "gopls",
			title:  "gopls — The Go Language Server",
			url:    "https://pkg.go.dev/golang.org/x/tools/gopls",
			desc:   "The official language server for Go, consumed by VS Code, Neovim, Emacs, and other editors. Provides completion, diagnostics, and refactoring.",
			tags:   []string{"tooling", "lsp", "editor"},
			status: forge.Published,
			days:   70,
		},
		{
			slug:   "go-playground",
			title:  "The Go Playground",
			url:    "https://go.dev/play",
			desc:   "Run and share Go code snippets in the browser with no installation required. Useful for reproducing bugs, sharing examples, and learning.",
			tags:   []string{"playground", "tools", "learning"},
			status: forge.Published,
			days:   65,
		},
		{
			slug:   "concurrency-is-not-parallelism",
			title:  "Concurrency is not Parallelism — Rob Pike",
			url:    "https://go.dev/talks/2012/waza.slide",
			desc:   "Rob Pike's landmark 2012 talk distinguishing concurrency (structuring programs) from parallelism (executing computations simultaneously). Essential Go philosophy.",
			tags:   []string{"concurrency", "talk", "rob-pike"},
			status: forge.Published,
			days:   60,
		},
		{
			slug:   "go-memory-model",
			title:  "The Go Memory Model",
			url:    "https://go.dev/ref/mem",
			desc:   "Specifies the conditions under which reads of a variable in one goroutine can be guaranteed to observe values written by a different goroutine.",
			tags:   []string{"concurrency", "memory", "reference"},
			status: forge.Published,
			days:   55,
		},
		{
			// Draft — not publicly visible; demonstrates the Draft lifecycle state.
			slug:   "go-tour",
			title:  "A Tour of Go",
			url:    "https://go.dev/tour",
			desc:   "An interactive introduction to Go in your browser. Covers the basics of the language with hands-on exercises — the recommended starting point for beginners.",
			tags:   []string{"learning", "interactive", "beginner"},
			status: forge.Draft,
			days:   0,
		},
		{
			// Scheduled — will become Published at sched time; demonstrates the Scheduled lifecycle state.
			slug:   "go-proverbs",
			title:  "Go Proverbs — Rob Pike",
			url:    "https://go-proverbs.github.io",
			desc:   "Rob Pike's collection of Go proverbs from GopherFest 2015. Pithy design principles that capture the philosophy of idiomatic Go.",
			tags:   []string{"proverbs", "philosophy", "rob-pike"},
			status: forge.Scheduled,
			days:   0,
			sched:  &schedTime,
		},
	}

	for _, r := range resources {
		var pub time.Time
		if r.days > 0 {
			pub = now.Add(-time.Duration(r.days) * 24 * time.Hour)
		}
		node := forge.Node{
			ID:          forge.NewID(),
			Slug:        r.slug,
			Status:      r.status,
			PublishedAt: pub,
			ScheduledAt: r.sched,
		}
		res := &Resource{
			Node:        node,
			Title:       r.title,
			URL:         r.url,
			Description: r.desc,
			Tags:        r.tags,
		}
		if err := repo.Save(ctx, res); err != nil {
			log.Fatalf("seed %q: %v", r.slug, err)
		}
	}
}
