package forge

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// — Markdownable ——————————————————————————————————————————————————————————

// Markdownable is implemented by content types that render directly to Markdown.
// When T implements Markdownable, [Module] serves text/markdown responses without
// requiring forge.Templates to be configured.
type Markdownable interface{ Markdown() string }

// — contentNegotiator —————————————————————————————————————————————————————

// contentNegotiator carries the pre-compiled content-type capabilities for a
// module. Built once at [NewModule] time; never mutated after construction.
type contentNegotiator struct {
	md   bool // true if the prototype T implements Markdownable
	html bool // true if forge.Templates option is set (Milestone 3)
}

// negotiate returns the canonical content-type string to serve for r.
// The result is used to select the response serialiser. Called once per
// request on the hot path — allocation-free.
func (n contentNegotiator) negotiate(r *http.Request) string {
	a := r.Header.Get("Accept")
	if a == "" || a == "*/*" || strings.Contains(a, "application/json") {
		return "application/json"
	}
	if strings.Contains(a, "text/html") {
		return "text/html"
	}
	if strings.Contains(a, "text/markdown") {
		return "text/markdown"
	}
	if strings.Contains(a, "text/plain") {
		return "text/plain"
	}
	return "application/json"
}

// — Module Option types ———————————————————————————————————————————————————

// atOption carries the URL prefix for a Module. Use [At] to create one.
type atOption struct{ prefix string }

func (atOption) isOption() {}

// At returns an [Option] that sets the URL prefix for a module.
// The prefix must start with "/" and must not end with "/".
// Example: forge.At("/posts")
func At(prefix string) Option { return atOption{prefix: prefix} }

// moduleCacheOption carries the TTL for a module-level [CacheStore].
// Use [Cache] to create one.
type moduleCacheOption struct{ ttl time.Duration }

func (moduleCacheOption) isOption() {}

// Cache returns an [Option] that enables a per-module LRU response cache with
// the given TTL. Cached entries are flushed automatically on any create, update,
// or delete operation. The cache holds at most 1000 entries (LRU eviction).
func Cache(ttl time.Duration) Option { return moduleCacheOption{ttl: ttl} }

// middlewareModuleOption carries per-module middleware. Use [Middleware] to
// create one.
type middlewareModuleOption struct {
	mws []func(http.Handler) http.Handler
}

func (middlewareModuleOption) isOption() {}

// Middleware returns an [Option] that wraps every route in this module with the
// provided middleware. Applied in the same order as [Chain] (index 0 is outermost).
func Middleware(mws ...func(http.Handler) http.Handler) Option {
	return middlewareModuleOption{mws: mws}
}

// authOption groups per-module role options. Use [Auth] to create one.
type authOption struct{ opts []Option }

func (authOption) isOption() {}

// Auth returns an [Option] that sets the minimum role for each HTTP operation on
// this module. Accepts [Read], [Write], and [Delete] role options.
//
//	forge.Auth(
//	    forge.Read(forge.Guest),
//	    forge.Write(forge.Author),
//	    forge.Delete(forge.Editor),
//	)
func Auth(opts ...Option) Option { return authOption{opts: opts} }

// repoOption carries the [Repository] for a [Module]. Use [Repo] to create one.
// Application code never calls [Repo] directly — [App.Content] supplies it.
// In unit tests, supply it explicitly:
//
//	m := forge.NewModule(&Post{}, forge.Repo(forge.NewMemoryRepo[*Post]()))
type repoOption[T any] struct{ repo Repository[T] }

func (repoOption[T]) isOption() {}

// Repo returns an [Option] that provides the [Repository] for a [Module].
// This is called internally by App.Content. In unit tests pass a [MemoryRepo]:
//
//	forge.Repo(forge.NewMemoryRepo[*Post]())
func Repo[T any](r Repository[T]) Option { return repoOption[T]{repo: r} }

// — Node field reflection cache ———————————————————————————————————————————

// nodeFields holds the struct field index paths for the three required Node fields.
// Index paths are []int because fields may be in embedded structs (e.g. Node.ID).
type nodeFields struct{ id, slug, status []int }

// nodeFieldCache caches field index paths keyed by reflect.Type.
var nodeFieldCache sync.Map

// getNodeFields returns cached field index paths for t's underlying struct type.
// Traverses pointer types automatically. Panics (at construction time, not
// request time) when ID, Slug, or Status fields are absent.
func getNodeFields(t reflect.Type) nodeFields {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if v, ok := nodeFieldCache.Load(t); ok {
		return v.(nodeFields)
	}
	idF, idOK := t.FieldByName("ID")
	slugF, slugOK := t.FieldByName("Slug")
	statusF, statusOK := t.FieldByName("Status")
	if !idOK || !slugOK || !statusOK {
		panic("forge: content type must embed forge.Node (missing ID, Slug, or Status field)")
	}
	f := nodeFields{id: idF.Index, slug: slugF.Index, status: statusF.Index}
	nodeFieldCache.Store(t, f)
	return f
}

// elemValue dereferences any number of pointer types and returns the
// underlying struct reflect.Value.
func elemValue(v any) reflect.Value {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	return rv
}

// nodeStatusOf returns the Status field of v.
func nodeStatusOf(v any) Status {
	rv := elemValue(v)
	f := getNodeFields(rv.Type())
	return rv.FieldByIndex(f.status).Interface().(Status)
}

// nodeSlugOf returns the Slug field of v.
func nodeSlugOf(v any) string {
	rv := elemValue(v)
	f := getNodeFields(rv.Type())
	return rv.FieldByIndex(f.slug).String()
}

// nodeIDOf returns the ID field of v.
func nodeIDOf(v any) string {
	rv := elemValue(v)
	f := getNodeFields(rv.Type())
	return rv.FieldByIndex(f.id).String()
}

// setNodeID sets the ID field on the pointed-to struct. v must be a pointer.
func setNodeID(v any, id string) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	f := getNodeFields(rv.Type())
	rv.FieldByIndex(f.id).SetString(id)
}

// setNodeSlug sets the Slug field on the pointed-to struct. v must be a pointer.
func setNodeSlug(v any, slug string) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	f := getNodeFields(rv.Type())
	rv.FieldByIndex(f.slug).SetString(slug)
}

// autoSlug inspects v for a non-empty string field to derive a URL slug from.
// Checks for "Title", "Name", "Headline" first, then any string field tagged
// forge:"required". Returns "" if no suitable field is found.
func autoSlug(rv reflect.Value) string {
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	t := rv.Type()
	for _, name := range []string{"Title", "Name", "Headline"} {
		if idx, ok := fieldIndex(t, name); ok {
			if f := t.Field(idx); f.Type.Kind() == reflect.String {
				if s := rv.Field(idx).String(); s != "" {
					return GenerateSlug(s)
				}
			}
		}
	}
	for i := 0; i < t.NumField(); i++ {
		fi := t.Field(i)
		if fi.Type.Kind() == reflect.String && strings.Contains(fi.Tag.Get("forge"), "required") {
			if s := rv.Field(i).String(); s != "" {
				return GenerateSlug(s)
			}
		}
	}
	return ""
}

// fieldIndex returns the index of the named field in t, or -1.
func fieldIndex(t reflect.Type, name string) (int, bool) {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Name == name {
			return i, true
		}
	}
	return -1, false
}

// stripMarkdown removes common Markdown formatting from s for text/plain output.
func stripMarkdown(s string) string {
	s = strings.NewReplacer(
		"**", "", "__", "", "*", "", "_", "",
		"### ", "", "## ", "", "# ",
		"",
	).Replace(s)
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '[' {
			closeIdx := strings.Index(s[i:], "](")
			if closeIdx > 0 {
				out.WriteString(s[i+1 : i+closeIdx])
				rest := s[i+closeIdx+2:]
				if parenEnd := strings.Index(rest, ")"); parenEnd >= 0 {
					i = i + closeIdx + 2 + parenEnd + 1
					continue
				}
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// — Module ————————————————————————————————————————————————————————————————

// Module is the core routing and lifecycle unit for a content type T.
// T must embed [Node] — its struct must have exported ID, Slug, and Status fields.
// Use [NewModule] to construct; Registration onto a ServeMux is done via [Register].
// App.Content handles both steps automatically.
type Module[T any] struct {
	prefix      string
	repo        Repository[T]
	readRole    Role
	writeRole   Role
	deleteRole  Role
	signals     map[Signal][]signalHandler
	cache       *CacheStore // nil when no Cache option was given
	middlewares []func(http.Handler) http.Handler
	neg         contentNegotiator
	debounce    *debouncer
	debounceMu  sync.Mutex
	debounceCtx Context // last Context seen; used by debounced dispatch
	proto       reflect.Type
}

// NewModule constructs a [Module] for content type T.
//
// proto is a representative value of T (typically a nil pointer: (*Post)(nil))
// used to derive the default URL prefix and to detect capabilities.
//
// Required options (supplied automatically by App.Content):
//   - [Repo]: provides the Repository[T]
//
// Optional options:
//   - [At]: override URL prefix (default: "/"+lowercase(TypeName)+"s")
//   - [Auth]: set per-operation role requirements
//   - [Cache]: enable per-module LRU response cache
//   - [Middleware]: wrap all routes with the given middleware
//   - [On]: register signal handlers
//
// Panics if no [Repo] option is present — this is a programming error caught
// at startup, never at request time.
func NewModule[T any](proto T, opts ...Option) *Module[T] {
	t := reflect.TypeOf(proto)

	// Validate Node fields eagerly — fail at startup, not request time.
	getNodeFields(t)

	// Determine default URL prefix from the type name.
	typeName := t.Name()
	if t.Kind() == reflect.Ptr {
		typeName = t.Elem().Name()
	}
	prefix := "/" + strings.ToLower(typeName) + "s"

	m := &Module[T]{
		prefix:     prefix,
		readRole:   Guest,
		writeRole:  Author,
		deleteRole: Editor,
		signals:    make(map[Signal][]signalHandler),
		proto:      t,
	}

	// Detect Markdownable capability.
	_, m.neg.md = any(proto).(Markdownable)

	// Parse options.
	var repoFound bool
	for _, o := range opts {
		switch v := o.(type) {
		case atOption:
			m.prefix = v.prefix
		case moduleCacheOption:
			m.cache = NewCacheStore(v.ttl, 1000)
		case middlewareModuleOption:
			m.middlewares = v.mws
		case authOption:
			for _, ao := range v.opts {
				if ro, ok := ao.(roleOption); ok {
					switch ro.signal {
					case "read":
						m.readRole = ro.role
					case "write":
						m.writeRole = ro.role
					case "delete":
						m.deleteRole = ro.role
					}
				}
			}
		case signalOption:
			m.signals[v.signal] = append(m.signals[v.signal], v.handler)
		}
		// repoOption[T] requires a concrete type assertion — handled separately.
		if ro, ok := o.(repoOption[T]); ok {
			m.repo = ro.repo
			repoFound = true
		}
	}

	if !repoFound {
		panic("forge: Module[T] requires a Repository; use forge.Repo(...) or App.Content")
	}

	// Background sweep for module cache.
	if m.cache != nil {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				m.cache.Sweep()
			}
		}()
	}

	// Debouncer for SitemapRegenerate (Amendment P1: 2-second delay).
	m.debounce = newDebouncer(2*time.Second, func() {
		m.debounceMu.Lock()
		ctx := m.debounceCtx
		m.debounceMu.Unlock()
		if ctx != nil {
			dispatchAfter(ctx, m.signals[SitemapRegenerate], nil)
		}
	})

	return m
}

// Register mounts the five standard routes for this module onto mux.
// Called automatically by App.Content.
//
//	GET    /{prefix}          → list
//	GET    /{prefix}/{slug}   → show
//	POST   /{prefix}          → create
//	PUT    /{prefix}/{slug}   → update
//	DELETE /{prefix}/{slug}   → delete
func (m *Module[T]) Register(mux *http.ServeMux) {
	list := http.HandlerFunc(m.listHandler)
	show := http.HandlerFunc(m.showHandler)
	create := http.HandlerFunc(m.createHandler)
	update := http.HandlerFunc(m.updateHandler)
	del := http.HandlerFunc(m.deleteHandler)

	if len(m.middlewares) > 0 {
		list = http.HandlerFunc(Chain(list, m.middlewares...).ServeHTTP)
		show = http.HandlerFunc(Chain(show, m.middlewares...).ServeHTTP)
		create = http.HandlerFunc(Chain(create, m.middlewares...).ServeHTTP)
		update = http.HandlerFunc(Chain(update, m.middlewares...).ServeHTTP)
		del = http.HandlerFunc(Chain(del, m.middlewares...).ServeHTTP)
	}

	mux.Handle("GET "+m.prefix, list)
	mux.Handle("GET "+m.prefix+"/{slug}", show)
	mux.Handle("POST "+m.prefix, create)
	mux.Handle("PUT "+m.prefix+"/{slug}", update)
	mux.Handle("DELETE "+m.prefix+"/{slug}", del)
}

// — Cache helpers —————————————————————————————————————————————————————————

// cacheKey returns the cache key for r.
func cacheKey(r *http.Request) string {
	return r.Method + " " + r.URL.RequestURI() + " " + r.Header.Get("Accept")
}

// serveCached returns true if the response was served from the module cache.
// Sets X-Cache: HIT and writes the cached response to w.
func (m *Module[T]) serveCached(w http.ResponseWriter, r *http.Request) bool {
	if m.cache == nil {
		return false
	}
	key := cacheKey(r)
	m.cache.mu.Lock()
	entry := m.cache.get(key)
	m.cache.mu.Unlock()
	if entry == nil {
		return false
	}
	for k, vals := range entry.header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("X-Cache", "HIT")
	w.WriteHeader(entry.status)
	w.Write(entry.body) //nolint:errcheck
	return true
}

// storeCache stores a captured response in the module cache if status is 200.
func (m *Module[T]) storeCache(r *http.Request, rec *cacheRecorder) {
	if m.cache == nil || rec.status != http.StatusOK {
		return
	}
	h := make(http.Header)
	for k, v := range rec.ResponseWriter.Header() {
		h[k] = v
	}
	e := &cacheEntry{
		key:     cacheKey(r),
		body:    rec.body,
		header:  h,
		status:  rec.status,
		expires: time.Now().Add(m.cache.ttl),
	}
	m.cache.mu.Lock()
	m.cache.set(e)
	m.cache.mu.Unlock()
}

// invalidateCache flushes the module cache after a write operation.
func (m *Module[T]) invalidateCache() {
	if m.cache != nil {
		m.cache.Flush()
	}
}

// — Content-type helpers ——————————————————————————————————————————————————

// writeContent serialises v to w using ct content-type.
// Sets Vary: Accept and Content-Type on every response.
func writeContent(w http.ResponseWriter, ct string, v any) {
	w.Header().Set("Vary", "Accept")
	switch ct {
	case "text/html":
		http.Error(w, "HTML templates not registered", http.StatusNotAcceptable)
	case "text/markdown":
		if md, ok := v.(Markdownable); ok {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(md.Markdown())) //nolint:errcheck
		} else {
			http.Error(w, "text/markdown not supported for this content type", http.StatusNotAcceptable)
		}
	case "text/plain":
		var body string
		if md, ok := v.(Markdownable); ok {
			body = stripMarkdown(md.Markdown())
		} else {
			b, _ := json.Marshal(v)
			body = string(b)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body)) //nolint:errcheck
	default: // application/json
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(v) //nolint:errcheck
	}
}

// writeContentCached writes a negotiated response through a cacheRecorder if
// caching is enabled, then stores the response in the module cache on 200.
func (m *Module[T]) writeContentCached(w http.ResponseWriter, r *http.Request, ct string, v any) {
	if m.cache != nil {
		w.Header().Set("X-Cache", "MISS")
		rec := newCacheRecorder(w)
		writeContent(rec, ct, v)
		m.storeCache(r, rec)
	} else {
		writeContent(w, ct, v)
	}
}

// — Lifecycle helper ——————————————————————————————————————————————————————

// isVisible reports whether item should be visible to the calling user.
// Guests see only Published content. Authors and above see all statuses.
func isVisible(item any, user User) bool {
	if nodeStatusOf(item) == Published {
		return true
	}
	return user.HasRole(Author)
}

// — debounce helper ———————————————————————————————————————————————————————

func (m *Module[T]) triggerSitemap(ctx Context) {
	m.debounceMu.Lock()
	m.debounceCtx = ctx
	m.debounceMu.Unlock()
	m.debounce.Trigger()
}

// — newItemPtr allocates a *struct for T via reflection. ——————————————————

// newItemPtr returns a pointer to a newly allocated zero value of T's
// underlying struct type. Used for JSON decoding.
func (m *Module[T]) newItemPtr() (reflect.Value, reflect.Type) {
	t := m.proto
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return reflect.New(t), t
}

// ptrToT converts a *struct reflect.Value back into T.
func ptrToT[T any](pv reflect.Value, proto reflect.Type) T {
	var item T
	rv := reflect.ValueOf(&item).Elem()
	if rv.Kind() == reflect.Ptr {
		rv.Set(pv)
	} else {
		rv.Set(pv.Elem())
	}
	return item
}

// — Handlers ——————————————————————————————————————————————————————————————

// listHandler serves GET /{prefix}.
func (m *Module[T]) listHandler(w http.ResponseWriter, r *http.Request) {
	ctx := ContextFrom(w, r)

	if m.serveCached(w, r) {
		return
	}

	items, err := m.repo.FindAll(ctx, ListOptions{})
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// Lifecycle filter: guests see only Published content.
	if !ctx.User().HasRole(Author) {
		filtered := items[:0]
		for _, item := range items {
			if nodeStatusOf(item) == Published {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	ct := m.neg.negotiate(r)
	m.writeContentCached(w, r, ct, items)
}

// showHandler serves GET /{prefix}/{slug}.
func (m *Module[T]) showHandler(w http.ResponseWriter, r *http.Request) {
	ctx := ContextFrom(w, r)

	if m.serveCached(w, r) {
		return
	}

	slug := r.PathValue("slug")
	item, err := m.repo.FindBySlug(ctx, slug)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// Lifecycle enforcement: non-Published → 404 for Guest.
	// 404 is intentional — do not leak the existence of non-public content.
	if !isVisible(item, ctx.User()) {
		WriteError(w, r, ErrNotFound)
		return
	}

	ct := m.neg.negotiate(r)
	m.writeContentCached(w, r, ct, item)
}

// createHandler serves POST /{prefix}.
func (m *Module[T]) createHandler(w http.ResponseWriter, r *http.Request) {
	ctx := ContextFrom(w, r)

	if !ctx.User().HasRole(m.writeRole) {
		WriteError(w, r, ErrForbidden)
		return
	}

	pv, elemType := m.newItemPtr()
	if err := json.NewDecoder(r.Body).Decode(pv.Interface()); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Assign a new ID and auto-generate a slug if not provided.
	f := getNodeFields(elemType)
	id := NewID()
	pv.Elem().FieldByIndex(f.id).SetString(id)
	if pv.Elem().FieldByIndex(f.slug).String() == "" {
		slug := autoSlug(pv.Elem())
		if slug == "" {
			slug = id[:8]
		}
		pv.Elem().FieldByIndex(f.slug).SetString(slug)
	}

	item := ptrToT[T](pv, m.proto)

	// Validate before any persistence.
	if err := RunValidation(item); err != nil {
		WriteError(w, r, err)
		return
	}

	// BeforeCreate hooks (synchronous — first error aborts).
	if err := dispatchBefore(ctx, m.signals[BeforeCreate], item); err != nil {
		WriteError(w, r, err)
		return
	}

	if err := m.repo.Save(ctx, item); err != nil {
		WriteError(w, r, err)
		return
	}

	// AfterCreate hooks (asynchronous).
	dispatchAfter(ctx, m.signals[AfterCreate], item)

	// Status-based publish hooks.
	if nodeStatusOf(item) == Published {
		dispatchAfter(ctx, m.signals[AfterPublish], item)
	}

	m.invalidateCache()
	m.triggerSitemap(ctx)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item) //nolint:errcheck
}

// updateHandler serves PUT /{prefix}/{slug}.
func (m *Module[T]) updateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := ContextFrom(w, r)

	if !ctx.User().HasRole(m.writeRole) {
		WriteError(w, r, ErrForbidden)
		return
	}

	slug := r.PathValue("slug")
	existing, err := m.repo.FindBySlug(ctx, slug)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// Allocate a new item and decode the request body into it.
	pv, elemType := m.newItemPtr()
	if err := json.NewDecoder(r.Body).Decode(pv.Interface()); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Preserve the original ID and Slug — URL is the source of truth.
	f := getNodeFields(elemType)
	pv.Elem().FieldByIndex(f.id).SetString(nodeIDOf(existing))
	pv.Elem().FieldByIndex(f.slug).SetString(slug)

	item := ptrToT[T](pv, m.proto)

	if err := RunValidation(item); err != nil {
		WriteError(w, r, err)
		return
	}

	// Save previous status for post-save signal dispatch.
	prevStatus := nodeStatusOf(existing)
	newStatus := nodeStatusOf(item)

	if err := dispatchBefore(ctx, m.signals[BeforeUpdate], item); err != nil {
		WriteError(w, r, err)
		return
	}

	if err := m.repo.Save(ctx, item); err != nil {
		WriteError(w, r, err)
		return
	}

	dispatchAfter(ctx, m.signals[AfterUpdate], item)

	// Status-transition hooks.
	if prevStatus != Published && newStatus == Published {
		dispatchAfter(ctx, m.signals[AfterPublish], item)
	} else if prevStatus == Published && newStatus != Published {
		dispatchAfter(ctx, m.signals[AfterUnpublish], item)
	}
	if newStatus == Archived {
		dispatchAfter(ctx, m.signals[AfterArchive], item)
	}

	m.invalidateCache()
	m.triggerSitemap(ctx)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(item) //nolint:errcheck
}

// deleteHandler serves DELETE /{prefix}/{slug}.
func (m *Module[T]) deleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := ContextFrom(w, r)

	if !ctx.User().HasRole(m.deleteRole) {
		WriteError(w, r, ErrForbidden)
		return
	}

	slug := r.PathValue("slug")
	existing, err := m.repo.FindBySlug(ctx, slug)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	if err := dispatchBefore(ctx, m.signals[BeforeDelete], existing); err != nil {
		WriteError(w, r, err)
		return
	}

	id := nodeIDOf(existing)
	if err := m.repo.Delete(ctx, id); err != nil {
		WriteError(w, r, err)
		return
	}

	dispatchAfter(ctx, m.signals[AfterDelete], existing)

	m.invalidateCache()
	m.triggerSitemap(ctx)

	w.WriteHeader(http.StatusNoContent)
}
