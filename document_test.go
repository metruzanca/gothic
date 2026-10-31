package gothic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func writeTestDocument(t *testing.T, r *http.Request, doc Document) string {
	t.Helper()
	var b strings.Builder
	body := ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<main>app</main>")
		return err
	})
	if err := WriteDocument(&b, r, doc, body); err != nil {
		t.Fatalf("WriteDocument: %v", err)
	}
	return b.String()
}

func TestDocumentDefaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://example.com/about?x=1", nil)
	out := writeTestDocument(t, r, Document{})
	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		`<meta charset="utf-8">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1">`,
		`<title>Gothic App</title>`,
		`<link rel="icon" href="/_gothic/icon.svg" type="image/svg+xml">`,
		`<link rel="icon" href="/_gothic/icon-512.png" type="image/png" sizes="512x512">`,
		`<link rel="apple-touch-icon" href="/_gothic/apple-touch-icon.png" type="image/png" sizes="180x180">`,
		`<meta property="og:type" content="website">`,
		`<meta property="og:title" content="Gothic App">`,
		`<meta property="og:site_name" content="Gothic App">`,
		`<meta property="og:url" content="https://example.com/about">`,
		`<meta property="og:image" content="https://example.com/_gothic/og.png">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		"<main>app</main>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
	if strings.Contains(out, "x=1") {
		t.Errorf("og:url should not include the query string\n\n%s", out)
	}
}

func TestDocumentOptions(t *testing.T) {
	doc := Document{}
	for _, opt := range []AppOption{
		WithTitle("Gothicbook"),
		WithDescription("A design system demo."),
		WithSiteName("Gothic"),
		WithLanguage("es"),
		WithThemeColor("#f7f7f5"),
		WithImage("/og.png"),
		WithStyles("/gothic-ui.css", "/style.css"),
		WithIcon("/favicon.ico"),
		WithExtraHead(`<script src="/extra.js"></script>`),
	} {
		opt(&doc)
	}

	r := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	out := writeTestDocument(t, r, doc)
	for _, want := range []string{
		`<html lang="es">`,
		`<title>Gothicbook</title>`,
		`<meta name="description" content="A design system demo.">`,
		`<meta property="og:description" content="A design system demo.">`,
		`<meta property="og:site_name" content="Gothic">`,
		`<meta name="theme-color" content="#f7f7f5">`,
		`<link rel="icon" href="/favicon.ico" type="image/x-icon">`,
		`<link rel="stylesheet" href="/gothic-ui.css">`,
		`<link rel="stylesheet" href="/style.css">`,
		`<meta property="og:image" content="https://example.com/og.png">`,
		`<script src="/extra.js"></script>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
	if strings.Contains(out, DefaultIconPath) {
		t.Errorf("WithIcon should replace the default icon set\n\n%s", out)
	}
}

func TestDocumentEscapesValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	out := writeTestDocument(t, r, Document{
		Title:       `A & B <"quoted">`,
		Description: `Tom & Jerry's "show"`,
		Styles:      []string{`/x.css?a=1&b="2"`},
	})
	for _, want := range []string{
		`<title>A &amp; B &lt;&#34;quoted&#34;&gt;</title>`,
		`<meta name="description" content="Tom &amp; Jerry&#39;s &#34;show&#34;">`,
		`<link rel="stylesheet" href="/x.css?a=1&amp;b=&#34;2&#34;">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing escaped %q\n\n%s", want, out)
		}
	}
}

func TestDocumentForwardedHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://internal:8080/page", nil)
	r.Host = "internal:8080"
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "gothic.example.com")
	out := writeTestDocument(t, r, Document{})
	for _, want := range []string{
		`<meta property="og:url" content="https://gothic.example.com/page">`,
		`<meta property="og:image" content="https://gothic.example.com/_gothic/og.png">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
}

func TestDocumentAbsoluteImagePassesThrough(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	out := writeTestDocument(t, r, Document{Image: "https://cdn.example.com/card.png"})
	if !strings.Contains(out, `content="https://cdn.example.com/card.png"`) {
		t.Errorf("absolute image should pass through\n\n%s", out)
	}
}

func TestDocumentNoImage(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	for name, doc := range map[string]Document{
		"flag":   {NoImage: true},
		"option": applyOptions(WithImage("")),
	} {
		out := writeTestDocument(t, r, doc)
		if strings.Contains(out, `property="og:image"`) {
			t.Errorf("%s: unexpected og:image\n\n%s", name, out)
		}
		if !strings.Contains(out, `<meta name="twitter:card" content="summary">`) {
			t.Errorf("%s: want summary twitter card\n\n%s", name, out)
		}
	}
}

func applyOptions(opts ...AppOption) Document {
	doc := Document{}
	for _, opt := range opts {
		opt(&doc)
	}
	return doc
}

func TestDocumentFuncMergesOverBase(t *testing.T) {
	doc := applyOptions(
		WithTitle("Base"),
		WithSiteName("Gothic"),
		WithStyles("/base.css"),
		WithDocumentFunc(func(r *http.Request) Document {
			return Document{Title: "About", Description: "About page"}
		}),
	)
	r := httptest.NewRequest(http.MethodGet, "https://example.com/about", nil)
	out := writeTestDocument(t, r, doc)
	for _, want := range []string{
		`<title>About</title>`,
		`<meta name="description" content="About page">`,
		`<meta property="og:site_name" content="Gothic">`,
		`<link rel="stylesheet" href="/base.css">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
}

func TestWriteDocumentPropagatesErrors(t *testing.T) {
	body := ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return errors.New("boom")
	})
	err := WriteDocument(io.Discard, httptest.NewRequest(http.MethodGet, "/", nil), Document{}, body)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want boom", err)
	}
}
