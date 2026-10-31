package gothic

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Framework defaults for the HTML document. MountApp emits these unless an app
// overrides them.
const (
	DefaultLang  = "en"
	DefaultTitle = "Gothic App"
	DefaultType  = "website"

	DefaultIconPath    = "/_gothic/icon.svg"
	DefaultIconPNGPath = "/_gothic/icon-512.png"
	DefaultTouchPath   = "/_gothic/apple-touch-icon.png"
	DefaultImagePath   = "/_gothic/og.png"
)

// Icon is a document icon link.
type Icon struct {
	// Href is the icon URL. When it is root relative, browsers resolve it
	// against the served page.
	Href string
	// Type is the MIME type. It is inferred from the extension when empty.
	Type string
	// Sizes is the value for the sizes attribute, such as "512x512".
	Sizes string
	// Rel is the link relation. It defaults to "icon". Use
	// "apple-touch-icon" for iOS home screen icons.
	Rel string
}

// Document describes the HTML shell around an app view: the head metadata,
// icon links, and stylesheets. The zero value is usable and gets the framework
// defaults, including a generated social card at DefaultImagePath.
type Document struct {
	// Lang is the html lang attribute. It defaults to DefaultLang.
	Lang string
	// Title is the page title. It defaults to DefaultTitle.
	Title string
	// Description becomes the meta description and og:description.
	Description string
	// SiteName becomes og:site_name. It defaults to Title.
	SiteName string
	// Type is og:type. It defaults to DefaultType.
	Type string
	// ThemeColor becomes the theme-color meta tag when set.
	ThemeColor string
	// Image is og:image. Relative values are resolved to absolute URLs using
	// the request. It defaults to DefaultImagePath. Set NoImage or use
	// WithImage("") to suppress the default card.
	Image string
	// URL is og:url. It defaults to the absolute request path.
	URL string
	// Icons are the favicon and touch icon links. A nil slice gets the
	// framework default set. Use an empty non-nil slice to emit no icons.
	Icons []Icon
	// Styles are stylesheet hrefs, in order.
	Styles []string
	// ExtraHead is inserted verbatim at the end of the head. Treat it as
	// trusted markup.
	ExtraHead string
	// NoImage suppresses the default framework social card.
	NoImage bool

	docFunc func(*http.Request) Document
}

// DefaultDocument returns the framework document defaults.
func DefaultDocument() Document {
	return Document{
		Lang:  DefaultLang,
		Title: DefaultTitle,
		Type:  DefaultType,
		Image: DefaultImagePath,
		Icons: defaultIcons(),
	}
}

func defaultIcons() []Icon {
	return []Icon{
		{Href: DefaultIconPath, Type: "image/svg+xml"},
		{Href: DefaultIconPNGPath, Type: "image/png", Sizes: "512x512"},
		{Href: DefaultTouchPath, Rel: "apple-touch-icon", Sizes: "180x180"},
	}
}

// AppOption customizes the document that MountApp renders.
type AppOption func(*Document)

// WithDocument replaces the document configuration. Zero fields still fall
// back to the framework defaults at render time.
func WithDocument(doc Document) AppOption {
	return func(d *Document) {
		*d = doc
	}
}

// WithDocumentFunc returns a per-request document that is merged over the
// server and mount configuration. Non-zero fields and non-nil slices win.
func WithDocumentFunc(fn func(*http.Request) Document) AppOption {
	return func(d *Document) {
		d.docFunc = fn
	}
}

// WithTitle sets the page title and og:title.
func WithTitle(title string) AppOption {
	return func(d *Document) { d.Title = title }
}

// WithDescription sets the meta description and og:description.
func WithDescription(description string) AppOption {
	return func(d *Document) { d.Description = description }
}

// WithSiteName sets og:site_name.
func WithSiteName(name string) AppOption {
	return func(d *Document) { d.SiteName = name }
}

// WithLanguage sets the html lang attribute.
func WithLanguage(lang string) AppOption {
	return func(d *Document) { d.Lang = lang }
}

// WithThemeColor sets the theme-color meta tag.
func WithThemeColor(color string) AppOption {
	return func(d *Document) { d.ThemeColor = color }
}

// WithIcon replaces the icon set with a single icon. The MIME type is inferred
// from the extension.
func WithIcon(href string) AppOption {
	return func(d *Document) {
		d.Icons = []Icon{{Href: href}}
	}
}

// WithIcons replaces the icon set.
func WithIcons(icons ...Icon) AppOption {
	return func(d *Document) {
		d.Icons = append([]Icon(nil), icons...)
	}
}

// WithStyles replaces the stylesheet list.
func WithStyles(hrefs ...string) AppOption {
	return func(d *Document) {
		d.Styles = append([]string(nil), hrefs...)
	}
}

// WithImage sets og:image. Relative paths are resolved against the request.
// An empty value suppresses the image.
func WithImage(src string) AppOption {
	return func(d *Document) {
		d.Image = src
		d.NoImage = src == ""
	}
}

// WithNoImage suppresses og:image, including the framework default card.
func WithNoImage() AppOption {
	return func(d *Document) {
		d.Image = ""
		d.NoImage = true
	}
}

// WithExtraHead appends trusted markup to the head.
func WithExtraHead(markup string) AppOption {
	return func(d *Document) { d.ExtraHead = markup }
}

// WriteDocument writes a full HTML document around body. The document is
// resolved against r for absolute URLs and defaults.
func WriteDocument(w io.Writer, r *http.Request, doc Document, body Component) error {
	doc = resolveDocument(doc, r)

	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	fmt.Fprintf(&b, "<html lang=\"%s\">\n", escape(doc.Lang))
	b.WriteString("<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", escape(doc.Title))
	if doc.Description != "" {
		fmt.Fprintf(&b, "<meta name=\"description\" content=\"%s\">\n", escape(doc.Description))
	}
	if doc.ThemeColor != "" {
		fmt.Fprintf(&b, "<meta name=\"theme-color\" content=\"%s\">\n", escape(doc.ThemeColor))
	}
	for _, icon := range doc.Icons {
		writeIcon(&b, icon)
	}
	for _, href := range doc.Styles {
		fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s\">\n", escape(href))
	}
	fmt.Fprintf(&b, "<meta property=\"og:type\" content=\"%s\">\n", escape(doc.Type))
	fmt.Fprintf(&b, "<meta property=\"og:title\" content=\"%s\">\n", escape(doc.Title))
	if doc.Description != "" {
		fmt.Fprintf(&b, "<meta property=\"og:description\" content=\"%s\">\n", escape(doc.Description))
	}
	if doc.SiteName != "" {
		fmt.Fprintf(&b, "<meta property=\"og:site_name\" content=\"%s\">\n", escape(doc.SiteName))
	}
	if doc.URL != "" {
		fmt.Fprintf(&b, "<meta property=\"og:url\" content=\"%s\">\n", escape(doc.URL))
	}
	if doc.Image != "" {
		fmt.Fprintf(&b, "<meta property=\"og:image\" content=\"%s\">\n", escape(doc.Image))
	}
	if doc.Image != "" {
		b.WriteString("<meta name=\"twitter:card\" content=\"summary_large_image\">\n")
	} else {
		b.WriteString("<meta name=\"twitter:card\" content=\"summary\">\n")
	}
	if doc.ExtraHead != "" {
		b.WriteString(doc.ExtraHead)
		if !strings.HasSuffix(doc.ExtraHead, "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("</head>\n<body>\n")

	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}
	if body != nil {
		if err := body.Render(r.Context(), w); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n</body>\n</html>\n")
	return err
}

func writeIcon(b *strings.Builder, icon Icon) {
	if icon.Href == "" {
		return
	}
	rel := icon.Rel
	if rel == "" {
		rel = "icon"
	}
	iconType := icon.Type
	if iconType == "" {
		iconType = iconMIME(icon.Href)
	}
	fmt.Fprintf(b, "<link rel=\"%s\" href=\"%s\"", escape(rel), escape(icon.Href))
	if iconType != "" {
		fmt.Fprintf(b, " type=\"%s\"", escape(iconType))
	}
	if icon.Sizes != "" {
		fmt.Fprintf(b, " sizes=\"%s\"", escape(icon.Sizes))
	}
	b.WriteString(">\n")
}

func iconMIME(href string) string {
	switch strings.ToLower(path.Ext(href)) {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return ""
	}
}

// resolveDocument applies the framework defaults, per-request overrides, and
// absolute URLs.
func resolveDocument(doc Document, r *http.Request) Document {
	if doc.docFunc != nil {
		doc = mergeDocument(doc, doc.docFunc(r))
	}
	if doc.Lang == "" {
		doc.Lang = DefaultLang
	}
	if doc.Title == "" {
		doc.Title = DefaultTitle
	}
	if doc.Type == "" {
		doc.Type = DefaultType
	}
	if doc.SiteName == "" {
		doc.SiteName = doc.Title
	}
	if doc.Icons == nil {
		doc.Icons = defaultIcons()
	}
	if doc.NoImage {
		doc.Image = ""
	} else if doc.Image == "" {
		doc.Image = DefaultImagePath
	}
	if r != nil {
		if doc.Image != "" {
			doc.Image = absoluteURL(r, doc.Image)
		}
		canonical := r.URL.Path
		if canonical == "" {
			canonical = "/"
		}
		if doc.URL == "" {
			doc.URL = requestBaseURL(r) + canonical
		} else {
			doc.URL = absoluteURL(r, doc.URL)
		}
	}
	return doc
}

func mergeDocument(base, over Document) Document {
	if over.Lang != "" {
		base.Lang = over.Lang
	}
	if over.Title != "" {
		base.Title = over.Title
	}
	if over.Description != "" {
		base.Description = over.Description
	}
	if over.SiteName != "" {
		base.SiteName = over.SiteName
	}
	if over.Type != "" {
		base.Type = over.Type
	}
	if over.ThemeColor != "" {
		base.ThemeColor = over.ThemeColor
	}
	if over.URL != "" {
		base.URL = over.URL
	}
	if over.Icons != nil {
		base.Icons = over.Icons
	}
	if over.Styles != nil {
		base.Styles = over.Styles
	}
	if over.ExtraHead != "" {
		base.ExtraHead = over.ExtraHead
	}
	if over.NoImage {
		base.NoImage = true
		base.Image = ""
	} else if over.Image != "" {
		base.NoImage = false
		base.Image = over.Image
	}
	return base
}

func absoluteURL(r *http.Request, ref string) string {
	if ref == "" || strings.HasPrefix(ref, "data:") {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if u.IsAbs() {
		return ref
	}
	base, err := url.Parse(requestBaseURL(r) + "/")
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := headerValue(r, "X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	host := r.Host
	if forwarded := headerValue(r, "X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host
}

func headerValue(r *http.Request, key string) string {
	value := r.Header.Get(key)
	if i := strings.IndexByte(value, ','); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

func escape(value string) string {
	return html.EscapeString(value)
}
