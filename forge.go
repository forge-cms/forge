package forge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Default timeout values applied by New when the caller does not set them
// in Config. These are conservative production defaults.
const (
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

// Config holds the application-wide configuration passed to [New].
//
// BaseURL and Secret are required; all other fields are optional.
//
// Timeouts default to 5 s (read), 10 s (write), and 120 s (idle) when left as
// zero. Set them explicitly to override.
type Config struct {
	// BaseURL is the canonical URL of the site, e.g. "https://example.com"
	// (no trailing slash). Required.
	BaseURL string

	// Secret is the HMAC signing key used by [BearerHMAC], [CookieSession], and
	// [SignToken]. It must be at least 16 bytes. Required.
	Secret []byte

	// DB is the database connection used by content modules.
	// It accepts *sql.DB, *sql.Tx, or any value that satisfies [DB].
	// Optional — leave nil to use in-memory repositories only.
	DB DB

	// HTTPS forces an HTTP→HTTPS redirect for all plain-HTTP requests when
	// true. The App handler checks r.TLS and the X-Forwarded-Proto header so
	// this works correctly behind a reverse proxy. Optional.
	HTTPS bool

	// ReadTimeout is the maximum time to read the full request, including the
	// body. Defaults to 5 s. Optional.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum time to write the full response. Defaults to
	// 10 s. Optional.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum keep-alive idle time between requests.
	// Defaults to 120 s. Optional.
	IdleTimeout time.Duration
}

// MustConfig validates cfg and returns it unchanged.
//
// Panics with a descriptive message if:
//   - Config.BaseURL is empty or not a valid absolute URL
//   - Config.Secret is fewer than 16 bytes
//
// Typical usage:
//
//	app := forge.New(forge.MustConfig(forge.Config{
//	    BaseURL: os.Getenv("BASE_URL"),
//	    Secret:  []byte(os.Getenv("SECRET")),
//	}))
func MustConfig(cfg Config) Config {
	if cfg.BaseURL == "" {
		panic(`forge: Config.BaseURL is required (e.g. "https://example.com")`)
	}
	u, err := url.ParseRequestURI(cfg.BaseURL)
	if err != nil || !u.IsAbs() {
		panic(fmt.Sprintf("forge: Config.BaseURL %q is not a valid absolute URL: %v", cfg.BaseURL, err))
	}
	if len(cfg.Secret) < 16 {
		panic("forge: Config.Secret must be at least 16 bytes")
	}
	return cfg
}

// Registrator is implemented by any value that can register its HTTP routes on
// a [http.ServeMux]. [*Module] satisfies this interface automatically.
//
// Pass a pre-built [*Module] to [App.Content] to register it:
//
//	posts := forge.NewModule[*Post](&Post{}, forge.Repo(repo))
//	app.Content(posts)
type Registrator interface {
	Register(mux *http.ServeMux)
}

// SEOOption is implemented by any value that modifies the app-level SEO
// configuration. Pass SEOOption values to [App.SEO]:
//
//	app.SEO(&forge.RobotsConfig{AIScraper: forge.AskFirst, Sitemaps: true})
type SEOOption interface {
	applySEO(*seoState)
}

// seoState holds the app-level SEO configuration applied via [App.SEO].
type seoState struct {
	robots *RobotsConfig
}

// App is the top-level application. Obtain one with [New].
type App struct {
	cfg                    Config
	mux                    *http.ServeMux
	middleware             []func(http.Handler) http.Handler
	sitemapStore           *SitemapStore    // non-nil when at least one module has SitemapConfig
	sitemapIndexRegistered bool             // true once GET /sitemap.xml is registered
	seo                    seoState         // app-level SEO configuration set via SEO()
	robotsTxtRegistered    bool             // true once GET /robots.txt is registered
	templateModules        []templateParser // modules with HTML templates; parsed at Run() time
	llmsStore              *LLMsStore       // non-nil when at least one module uses AIIndex
	llmsTxtRegistered      bool             // true once GET /llms.txt is registered
	llmsFullTxtRegistered  bool             // true once GET /llms-full.txt is registered
	feedStore              *FeedStore       // non-nil when at least one module uses Feed(...)
	feedIndexRegistered    bool             // true once GET /feed.xml is registered
	cookieDecls            []Cookie         // registered via Cookies(); drives /.well-known/cookies.json
	cookieManifestOpts     []Option         // options for the manifest handler (e.g. ManifestAuth)
	cookieManifestReg      bool             // true once GET /.well-known/cookies.json is registered
	redirectStore          *RedirectStore   // runtime redirect table; always non-nil after New()
	redirectFallbackReg    bool             // true once "/" fallback handler is registered
	redirectManifestReg    bool             // true once GET /.well-known/redirects.json is registered
}

// New creates a new [App] from cfg.
//
// Default timeouts are applied if the corresponding Config fields are zero:
// ReadTimeout 5 s, WriteTimeout 10 s, IdleTimeout 120 s.
//
// New does not validate the Config. Use [MustConfig] to catch configuration
// errors at startup:
//
//	app := forge.New(forge.MustConfig(cfg))
func New(cfg Config) *App {
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	return &App{
		cfg:           cfg,
		mux:           http.NewServeMux(),
		redirectStore: NewRedirectStore(),
	}
}

// Use appends one or more global middleware to the App's middleware stack.
//
// Middleware is applied in the order it is added: the first call to Use wraps
// the outermost layer. Use may be called multiple times; all calls are
// additive.
//
//	app.Use(forge.RequestLogger(), forge.Recoverer(), forge.SecurityHeaders())
func (a *App) Use(mws ...func(http.Handler) http.Handler) {
	a.middleware = append(a.middleware, mws...)
}

// Handle registers a raw [http.Handler] at the given pattern on the App's
// internal mux. The pattern follows the same rules as [http.ServeMux].
//
// Use Handle for endpoints that are not managed by a [Module]:
//
//	app.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.WriteHeader(http.StatusOK)
//	}))
func (a *App) Handle(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

// Content registers a content module with the App.
//
// If v implements [Registrator] (which [*Module] does), its Register method is
// called directly and opts are ignored. This is the idiomatic path:
//
//	posts := forge.NewModule[*Post](&Post{}, forge.Repo(repo), forge.At("/posts"))
//	app.Content(posts)
//
// If v does not implement [Registrator], Content calls [NewModule][any](v,
// opts...) and registers the result. In this case forge.Repo must be supplied
// as a repoOption[any] — type safety is lost. Prefer the [Registrator] path
// for all production code.
func (a *App) Content(v any, opts ...Option) {
	// Extract any redirect options regardless of Registrator path.
	for _, o := range opts {
		if ro, ok := o.(redirectsOption); ok {
			a.redirectStore.Add(RedirectEntry{
				From:     string(ro.from),
				To:       ro.to,
				Code:     Permanent,
				IsPrefix: true,
			})
		}
	}
	if r, ok := v.(Registrator); ok {
		r.Register(a.mux)
		if sm, ok := r.(interface{ setSitemap(*SitemapStore, string) }); ok {
			if a.sitemapStore == nil {
				a.sitemapStore = NewSitemapStore()
			}
			sm.setSitemap(a.sitemapStore, a.cfg.BaseURL)
		}
		if tp, ok := r.(templateParser); ok {
			a.templateModules = append(a.templateModules, tp)
		}
		if sn, ok := r.(interface{ setSiteName(string) }); ok {
			u, _ := url.Parse(a.cfg.BaseURL)
			sn.setSiteName(u.Hostname())
		}
		if ai, ok := r.(interface{ setAIRegistry(*LLMsStore, string) }); ok {
			if a.llmsStore == nil {
				u, _ := url.Parse(a.cfg.BaseURL)
				a.llmsStore = NewLLMsStore(u.Hostname())
			}
			ai.setAIRegistry(a.llmsStore, a.cfg.BaseURL)
		}
		if fd, ok := r.(interface{ setFeedStore(*FeedStore, string) }); ok {
			if a.feedStore == nil {
				u, _ := url.Parse(a.cfg.BaseURL)
				a.feedStore = NewFeedStore(u.Hostname(), a.cfg.BaseURL)
			}
			fd.setFeedStore(a.feedStore, a.cfg.BaseURL)
		}
		return
	}
	m := NewModule(v, opts...)
	m.Register(a.mux)
}

// httpsRedirect returns a middleware that redirects plain-HTTP requests to
// their HTTPS equivalents with a 301 Moved Permanently response.
//
// The redirect is skipped when the request is already over TLS (r.TLS != nil)
// or when the X-Forwarded-Proto header equals "https" (reverse-proxy scenario).
func httpsRedirect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				next.ServeHTTP(w, r)
				return
			}
			target := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
	}
}

// Handler returns the composed http.Handler that serves all registered routes
// behind the global middleware stack.
//
// When Config.HTTPS is true, an HTTP→HTTPS redirect middleware is prepended
// before all user-supplied middleware.
//
// Handler is called automatically by [App.Run]. Call it directly when you need
// to hand the handler to your own server (e.g. for testing or embedding):
//
//	srv := &http.Server{Handler: app.Handler()}
func (a *App) Handler() http.Handler {
	if a.sitemapStore != nil && !a.sitemapIndexRegistered {
		a.sitemapIndexRegistered = true
		a.mux.Handle("GET /sitemap.xml", a.sitemapStore.IndexHandler(a.cfg.BaseURL))
	}
	if a.seo.robots != nil && !a.robotsTxtRegistered {
		a.robotsTxtRegistered = true
		a.mux.Handle("GET /robots.txt", RobotsTxtHandler(*a.seo.robots, a.cfg.BaseURL))
	}
	if a.llmsStore != nil && a.llmsStore.HasCompact() && !a.llmsTxtRegistered {
		a.llmsTxtRegistered = true
		a.mux.Handle("GET /llms.txt", a.llmsStore.CompactHandler())
	}
	if a.llmsStore != nil && a.llmsStore.HasFull() && !a.llmsFullTxtRegistered {
		a.llmsFullTxtRegistered = true
		a.mux.Handle("GET /llms-full.txt", a.llmsStore.FullHandler())
	}
	if a.feedStore != nil && a.feedStore.HasFeeds() && !a.feedIndexRegistered {
		a.feedIndexRegistered = true
		a.mux.Handle("GET /feed.xml", a.feedStore.IndexHandler())
	}
	if len(a.cookieDecls) > 0 && !a.cookieManifestReg {
		a.cookieManifestReg = true
		u, _ := url.Parse(a.cfg.BaseURL)
		a.mux.Handle("GET /.well-known/cookies.json",
			newCookieManifestHandler(u.Hostname(), a.cookieDecls, a.cookieManifestOpts...),
		)
	}
	if !a.redirectManifestReg {
		a.redirectManifestReg = true
		u2, _ := url.Parse(a.cfg.BaseURL)
		a.mux.Handle("GET /.well-known/redirects.json",
			newRedirectManifestHandler(u2.Hostname(), a.redirectStore),
		)
	}
	if !a.redirectFallbackReg {
		a.redirectFallbackReg = true
		a.mux.Handle("/", a.redirectStore.handler())
	}
	if len(a.templateModules) > 0 {
		bindErrorTemplates(a.templateModules)
	}
	mws := a.middleware
	if a.cfg.HTTPS {
		mws = append([]func(http.Handler) http.Handler{httpsRedirect()}, mws...)
	}
	return Chain(a.mux, mws...)
}

// SEO applies one or more app-level SEO options.
//
// Call SEO before [App.Handler] or [App.Run] so the configuration is applied
// before routes are registered. SEO may be called multiple times; later calls
// override earlier values for the same option type.
//
//	app.SEO(&forge.RobotsConfig{AIScraper: forge.AskFirst, Sitemaps: true})
func (a *App) SEO(opts ...SEOOption) {
	for _, opt := range opts {
		opt.applySEO(&a.seo)
	}
}

// Cookies registers cookie declarations for the compliance manifest at
// /.well-known/cookies.json. Call once at startup with all cookies the
// application may set.
//
// Duplicate declarations (same Name) are silently deduplicated; the first
// declaration with a given name wins.
//
// Optionally pass [ManifestAuth] to restrict the manifest endpoint to
// authenticated requests:
//
//	app.Cookies(
//	    forge.Cookie{Name: "session", Category: forge.Necessary, ...},
//	    forge.Cookie{Name: "prefs",   Category: forge.Preferences, ...},
//	)
func (a *App) Cookies(decls ...Cookie) {
	seen := make(map[string]struct{}, len(a.cookieDecls))
	for _, d := range a.cookieDecls {
		seen[d.Name] = struct{}{}
	}
	for _, d := range decls {
		if _, ok := seen[d.Name]; !ok {
			a.cookieDecls = append(a.cookieDecls, d)
			seen[d.Name] = struct{}{}
		}
	}
}

// CookiesManifestAuth sets the [AuthFunc] that guards /.well-known/cookies.json.
// Call before [App.Handler] or [App.Run].
//
//	app.CookiesManifestAuth(forge.BearerHMAC(secret, forge.Editor))
func (a *App) CookiesManifestAuth(auth AuthFunc) {
	a.cookieManifestOpts = append(a.cookieManifestOpts, ManifestAuth(auth))
}

// Redirect registers a manual redirect rule. Chain collapse is applied
// automatically: if from already redirects to an intermediate path and this
// call adds a rule for that intermediate path, the chain is collapsed (A→B
// + B→C = A→C). Maximum collapse depth is 10 (Decision 24).
//
// To issue a 301 Moved Permanently:
//
//	app.Redirect("/old-path", "/new-path", forge.Permanent)
//
// To issue a 410 Gone (pass an empty destination):
//
//	app.Redirect("/removed", "", forge.Gone)
func (a *App) Redirect(from, to string, code RedirectCode) {
	a.redirectStore.Add(RedirectEntry{From: from, To: to, Code: code})
}

// RedirectStore returns the App's [RedirectStore], which can be used to load
// persisted redirects from a database at startup, or to save/remove entries
// at runtime:
//
//	if err := app.RedirectStore().Load(ctx, db); err != nil {
//	    log.Fatal(err)
//	}
func (a *App) RedirectStore() *RedirectStore {
	return a.redirectStore
}

// Run starts the HTTP server on addr (e.g. ":8080") and blocks until
// SIGINT or SIGTERM is received.
//
// On receiving a signal, Run initiates a graceful shutdown with a 5-second
// deadline, waits for active connections to drain, and returns nil.
// Non-shutdown errors from ListenAndServe are returned directly.
//
//	if err := app.Run(":8080"); err != nil {
//	    log.Fatal(err)
//	}
func (a *App) Run(addr string) error {
	// Parse HTML templates before starting the server. Fail fast so startup
	// errors are obvious rather than surfacing as 406 responses at request time.
	for _, tp := range a.templateModules {
		if err := tp.parseTemplates(); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      a.Handler(),
		ReadTimeout:  a.cfg.ReadTimeout,
		WriteTimeout: a.cfg.WriteTimeout,
		IdleTimeout:  a.cfg.IdleTimeout,
	}

	// serveErr receives the result of ListenAndServe.
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe()
	}()

	// Block until an OS signal or a fatal server error.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		// Server failed before any signal — return the error directly.
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Signal received — begin graceful shutdown.
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return nil
}
