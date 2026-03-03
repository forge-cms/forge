package forge

import (
	"strings"
	"time"
	"unicode/utf8"
)

// — Image —————————————————————————————————————————————————————————————————

// Image is a typed image reference. Width and Height are required for optimal
// Open Graph rendering and Twitter Card display. The zero value (empty URL)
// renders no image tags — safe to leave unset.
type Image struct {
	URL    string // absolute or root-relative
	Alt    string // accessibility and SEO description
	Width  int    // pixels; required for og:image:width
	Height int    // pixels; required for og:image:height
}

// — Alternate —————————————————————————————————————————————————————————————

// Alternate is an hreflang entry for internationalised pages.
// Reserved for v2 — Forge always generates an empty Alternates slice in v1.
type Alternate struct {
	Locale string // BCP 47 language tag, e.g. "en-GB"
	URL    string // absolute URL for this locale
}

// — Breadcrumb ————————————————————————————————————————————————————————————

// Breadcrumb is a single step in a breadcrumb trail. Build slices using
// the Crumb constructor and the Crumbs helper.
type Breadcrumb struct {
	Label string // human-readable label
	URL   string // root-relative or absolute URL
}

// Crumb returns a single Breadcrumb entry.
// Use with Crumbs to build Head.Breadcrumbs:
//
//	forge.Crumbs(
//	    forge.Crumb("Home",  "/"),
//	    forge.Crumb("Posts", "/posts"),
//	    forge.Crumb(p.Title, "/posts/"+p.Slug),
//	)
func Crumb(label, url string) Breadcrumb { return Breadcrumb{Label: label, URL: url} }

// Crumbs collects Breadcrumb entries for use in Head.Breadcrumbs.
func Crumbs(crumbs ...Breadcrumb) []Breadcrumb { return crumbs }

// — Rich-result type constants —————————————————————————————————————————————

// Rich result type constants for Head.Type. Each maps to a schema.org type
// used to generate JSON-LD structured data (see schema.go).
const (
	Article      = "Article"      // blog posts and news articles
	Product      = "Product"      // e-commerce product pages
	FAQPage      = "FAQPage"      // frequently asked questions
	HowTo        = "HowTo"        // step-by-step guides
	Event        = "Event"        // events with dates and locations
	Recipe       = "Recipe"       // recipes with ingredients and steps
	Review       = "Review"       // reviews with star ratings
	Organization = "Organization" // company or about pages
)

// — Head ——————————————————————————————————————————————————————————————————

// Head carries all SEO and social metadata for a content page.
// Define it on your content type via the Headable interface.
// Forge uses the Head to populate HTML <head> tags, JSON-LD structured data,
// sitemaps, RSS feeds, and AI endpoints.
//
// All fields are optional: the zero value is safe and produces a minimal page header.
type Head struct {
	Title       string       // page title; used in <title>, og:title, and JSON-LD
	Description string       // meta description; recommended max 160 characters
	Author      string       // author name; used in <meta name="author"> and JSON-LD
	Published   time.Time    // publication date; zero value omits date tags
	Modified    time.Time    // last-modified date; zero value omits date tags
	Image       Image        // primary image; zero URL omits all image tags
	Type        string       // rich result type (Article, Product, etc.); empty omits JSON-LD
	Canonical   string       // canonical URL; empty omits the canonical tag
	Breadcrumbs []Breadcrumb // breadcrumb trail; empty omits BreadcrumbList JSON-LD
	Alternates  []Alternate  // hreflang entries; always empty in v1
	NoIndex     bool         // true renders <meta name="robots" content="noindex">
}

// — Headable ——————————————————————————————————————————————————————————————

// Headable is implemented by content types that provide their own SEO metadata.
// Forge calls Head when building HTML responses, sitemaps, and AI endpoints.
// Use HeadFunc on a module to override the content type's own Head method.
type Headable interface{ Head() Head }

// — HeadFunc option ———————————————————————————————————————————————————————

// headFuncOption stores a module-level head override function.
type headFuncOption[T any] struct{ fn func(Context, T) Head }

func (headFuncOption[T]) isOption() {}

// HeadFunc returns an Option that overrides a content type's Head method at
// the module level. The function receives the current request context and the
// content item; its return value takes precedence over the content type's own
// Head() implementation.
//
//	app.Content(&BlogPost{},
//	    forge.At("/posts"),
//	    forge.HeadFunc(func(ctx forge.Context, p *BlogPost) forge.Head {
//	        return forge.Head{Title: p.Title + " — " + ctx.SiteName()}
//	    }),
//	)
func HeadFunc[T any](fn func(Context, T) Head) Option { return headFuncOption[T]{fn: fn} }

// — Excerpt ———————————————————————————————————————————————————————————————

// Excerpt returns a plain-text summary truncated at the last word boundary
// within maxLen characters. A Unicode ellipsis ("…") is appended when the
// text is truncated. Use it to populate Head.Description.
//
//	forge.Excerpt(p.Body, 160)
func Excerpt(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}
	// Find the byte offset immediately after the maxLen-th rune.
	bytePos := 0
	for i := 0; i < maxLen; i++ {
		_, size := utf8.DecodeRuneInString(text[bytePos:])
		bytePos += size
	}
	truncated := text[:bytePos]
	// Only truncate further when we're mid-word (next byte is not a space).
	if bytePos < len(text) && text[bytePos] != ' ' {
		if idx := strings.LastIndex(truncated, " "); idx > 0 {
			truncated = truncated[:idx]
		}
	}
	return truncated + "…"
}

// — URL ————————————————————————————————————————————————————————————————————

// URL joins path segments into a root-relative URL. It collapses duplicate
// slashes, ensures a leading slash, and trims any trailing slash (the root "/"
// is preserved).
//
//	forge.URL("/posts/", p.Slug)  →  "/posts/my-slug"
func URL(parts ...string) string {
	joined := strings.Join(parts, "/")
	// Collapse consecutive slashes.
	for strings.Contains(joined, "//") {
		joined = strings.ReplaceAll(joined, "//", "/")
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	joined = strings.TrimRight(joined, "/")
	if joined == "" {
		return "/"
	}
	return joined
}
