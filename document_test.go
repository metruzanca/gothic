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
		`<meta property="og:type" content="website">`,
		`<meta property="og:title" content="Gothic App">`,
		`<meta property="og:site_name" content="Gothic App">`,
		`<meta property="og:url" content="https://example.com/about">`,
		`<meta name="twitter:card" content="summary">`,
		"<main>app</main>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
	if strings.Contains(out, "x=1") {
		t.Errorf("og:url should not include the query string\n\n%s", out)
	}
	if strings.Contains(out, `property="og:image"`) {
		t.Errorf("document should not invent an og:image\n\n%s", out)
	}
	if strings.Contains(out, `<link rel="icon"`) {
		t.Errorf("document should not invent icons\n\n%s", out)
	}
}

func TestDocumentValues(t *testing.T) {
	doc := Document{
		Title:       "Gothicbook",
		Description: "A design system demo.",
		SiteName:    "Gothic",
		Lang:        "es",
		ThemeColor:  "#f7f7f5",
		Image:       "/og.png",
		Styles:      []string{"/gothic-ui.css", "/style.css"},
		Icons:       []Icon{{Href: "/favicon.ico"}},
		ExtraHead:   `<script src="/extra.js"></script>`,
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
	out := writeTestDocument(t, r, Document{})
	if strings.Contains(out, `property="og:image"`) {
		t.Errorf("unexpected og:image\n\n%s", out)
	}
	if !strings.Contains(out, `<meta name="twitter:card" content="summary">`) {
		t.Errorf("want summary twitter card\n\n%s", out)
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
