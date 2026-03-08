package forge

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

// — Option types ——————————————————————————————————————————————————————————

// templateParser is implemented by [*Module] once a Templates or
// TemplatesOptional option is provided. [App.Content] appends each
// implementing module to its internal list; [App.Run] calls
// parseTemplates on all of them before the server starts.
type templateParser interface {
	parseTemplates() error
}

// templatesOption carries the directory path and required flag for HTML
// templates. Created by [Templates] or [TemplatesOptional].
type templatesOption struct {
	dir      string
	required bool
}

func (templatesOption) isOption() {}

// Templates returns an [Option] that sets the directory containing HTML
// templates for a module. The directory must contain list.html and show.html;
// if either file is absent [App.Run] returns an error before the server starts.
//
// Template files are parsed once at startup. The expected layout is:
//
//	{dir}/list.html        — rendered for GET /{prefix}
//	{dir}/show.html        — rendered for GET /{prefix}/{slug}
//	{dir}/errors/404.html  — (optional) custom error page for 404 responses
//
// Use [TemplatesOptional] during development when template files are added
// incrementally.
func Templates(dir string) Option { return templatesOption{dir: dir, required: true} }

// TemplatesOptional returns an [Option] that sets the template directory but
// treats absent files as a silent no-op. HTML content negotiation is only
// enabled for a handler when its corresponding template file is found.
//
// Use this during development when templates are added incrementally.
func TemplatesOptional(dir string) Option { return templatesOption{dir: dir, required: false} }

// TemplatesWatch is deferred to Milestone 5. It will provide hot-reload of
// template files during development without restarting the server.

// — forge:head template ———————————————————————————————————————————————————

// forgeHeadTmpl is the named template injected into every parsed template set
// as "forge:head". Developers invoke it inside their own <head> element:
//
//	{{template "forge:head" .Head}}
//
// The template receives a [Head] value and renders: title, description meta,
// canonical link, Open Graph tags, and a robots noindex tag when [Head.NoIndex]
// is true. JSON-LD is not emitted here — place {{forge_meta .Head .Content}}
// in the template body to control JSON-LD placement and schema type.
const forgeHeadTmpl = `{{define "forge:head"}}<title>{{.Title}}</title>
{{- if .Description}}
<meta name="description" content="{{.Description}}">
{{- end}}
{{- if .Canonical}}
<link rel="canonical" href="{{.Canonical}}">
{{- end}}
{{- if .Title}}
<meta property="og:title" content="{{.Title}}">
{{- if .Description}}
<meta property="og:description" content="{{.Description}}">
{{- end}}
{{- if .Canonical}}
<meta property="og:url" content="{{.Canonical}}">
{{- end}}
{{- if .Image.URL}}
<meta property="og:image" content="{{.Image.URL}}">
{{- if gt .Image.Width 0}}
<meta property="og:image:width" content="{{.Image.Width}}">
<meta property="og:image:height" content="{{.Image.Height}}">
{{- end}}
{{- end}}
<meta property="og:type" content="{{if .Type}}{{.Type}}{{else}}website{{end}}">
{{- if eq .Type "Article"}}
{{- if gt .Published.Year 1}}
<meta property="article:published_time" content="{{forge_rfc3339 .Published}}">
{{- end}}
{{- if .Author}}
<meta property="article:author" content="{{.Author}}">
{{- end}}
{{- range .Tags}}
<meta property="article:tag" content="{{.}}">
{{- end}}
{{- end}}
{{- if .Social.Twitter.Card}}
<meta name="twitter:card" content="{{.Social.Twitter.Card}}">
{{- else if .Image.URL}}
<meta name="twitter:card" content="summary_large_image">
{{- else}}
<meta name="twitter:card" content="summary">
{{- end}}
<meta name="twitter:title" content="{{.Title}}">
{{- if .Description}}
<meta name="twitter:description" content="{{.Description}}">
{{- end}}
{{- if .Image.URL}}
<meta name="twitter:image" content="{{.Image.URL}}">
{{- end}}
{{- if .Social.Twitter.Creator}}
<meta name="twitter:creator" content="{{.Social.Twitter.Creator}}">
{{- end}}
{{- end}}
{{- if .NoIndex}}
<meta name="robots" content="noindex, nofollow">
{{- end}}
{{end}}`

// — Module template methods ————————————————————————————————————————————————

// setSiteName stores the site name (the hostname from [Config.BaseURL]) so it
// can be passed to [NewTemplateData] during HTML rendering.
// Called by [App.Content] as part of module wiring.
func (m *Module[T]) setSiteName(name string) {
	m.siteName = name
}

// parseTemplates loads list.html and show.html from the module's template
// directory, registers the forge:head named partial in both template sets, and
// stores them thread-safely under tplMu.
//
// Called by [App.Run] before the server starts. Returns a descriptive error
// when a required file is absent or a template fails to parse. Returns nil
// immediately when no template directory is configured.
func (m *Module[T]) parseTemplates() error {
	if m.templateDir == "" {
		return nil
	}

	listPath := filepath.Join(m.templateDir, "list.html")
	showPath := filepath.Join(m.templateDir, "show.html")

	tplList, err := parseOneTemplate(listPath, m.templateRequired)
	if err != nil {
		return fmt.Errorf("forge: templates for %s: %w", m.prefix, err)
	}

	tplShow, err := parseOneTemplate(showPath, m.templateRequired)
	if err != nil {
		return fmt.Errorf("forge: templates for %s: %w", m.prefix, err)
	}

	m.tplMu.Lock()
	m.tplList = tplList
	m.tplShow = tplShow
	if tplList != nil || tplShow != nil {
		m.neg.html = true
	}
	m.tplMu.Unlock()

	return nil
}

// parseOneTemplate parses a single HTML template file and registers the
// forge:head sub-template in the returned set.
//
// When required is false and the file does not exist, (nil, nil) is returned.
// When required is true and the file does not exist, an error is returned.
func parseOneTemplate(path string, required bool) (*template.Template, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if required {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, nil
	}

	tpl, err := template.New(filepath.Base(path)).Funcs(TemplateFuncMap()).ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}

	// Register the forge:head partial into this template set so every
	// developer template can call {{template "forge:head" .Head}}.
	if _, err := tpl.Parse(forgeHeadTmpl); err != nil {
		return nil, fmt.Errorf("register forge:head in %s: %w", filepath.Base(path), err)
	}

	return tpl, nil
}

// renderListHTML renders tplList with a TemplateData[[]T] payload.
// If tplList is nil the request receives a 406 Not Acceptable response.
// Template execution errors produce a 500; the response buffer is flushed
// only on success so Content-Type is not written on error.
func (m *Module[T]) renderListHTML(w http.ResponseWriter, r *http.Request, ctx Context, items []T) {
	m.tplMu.RLock()
	tpl := m.tplList
	m.tplMu.RUnlock()

	if tpl == nil {
		http.Error(w, "HTML templates not registered", http.StatusNotAcceptable)
		return
	}

	data := NewTemplateData(ctx, items, Head{}, m.siteName)
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		WriteError(w, r, fmt.Errorf("forge: list template execution: %w", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes()) //nolint:errcheck
}

// renderShowHTML renders tplShow with a TemplateData[T] payload.
// Head is resolved via [resolveHead]: HeadFunc takes priority, then [Headable],
// then a zero Head. If tplShow is nil the request receives a 406 Not Acceptable response.
func (m *Module[T]) renderShowHTML(w http.ResponseWriter, r *http.Request, ctx Context, item T) {
	m.tplMu.RLock()
	tpl := m.tplShow
	m.tplMu.RUnlock()

	if tpl == nil {
		http.Error(w, "HTML templates not registered", http.StatusNotAcceptable)
		return
	}

	head := m.resolveHead(ctx, item)
	data := NewTemplateData(ctx, item, head, m.siteName)
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		WriteError(w, r, fmt.Errorf("forge: show template execution: %w", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes()) //nolint:errcheck
}

// errorTemplate searches the module's template directory for
// errors/{status}.html and parses it on demand. Returns nil when the file
// is absent or the module has no template directory.
//
// Called lazily by the [errorTemplateLookup] closure set in [App.Handler].
func (m *Module[T]) errorTemplate(status int) *template.Template {
	if m.templateDir == "" {
		return nil
	}
	path := filepath.Join(m.templateDir, "errors", fmt.Sprintf("%d.html", status))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	tpl, err := template.New(filepath.Base(path)).ParseFiles(path)
	if err != nil {
		return nil
	}
	return tpl
}

// bindErrorTemplates sets the package-level [errorTemplateLookup] closure.
// It iterates the given modules searching for errors/{status}.html in each
// module's template directory and returns the first match found.
//
// Called once by [App.Handler] when at least one module has a template
// directory registered.
func bindErrorTemplates(modules []templateParser) {
	errorTemplateLookup = func(status int) *template.Template {
		for _, tp := range modules {
			type errorTemplater interface {
				errorTemplate(int) *template.Template
			}
			if m, ok := tp.(errorTemplater); ok {
				if tpl := m.errorTemplate(status); tpl != nil {
					return tpl
				}
			}
		}
		return nil
	}
}
