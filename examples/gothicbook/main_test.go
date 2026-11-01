package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func getPage(t *testing.T, url string) string {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d", url, res.StatusCode)
	}
	return string(body)
}

func TestGothicbookDocument(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	html := getPage(t, ts.URL+"/")
	for _, want := range []string{
		"<!doctype html>",
		`<title>Gothicbook</title>`,
		`<meta name="description" content="A tiny design system demo for the installable gothic UI kit.">`,
		`<meta property="og:title" content="Gothicbook">`,
		`<meta property="og:site_name" content="Gothic">`,
		`<meta property="og:image" content="` + ts.URL + `/og.png?v=`,
		`<link rel="icon" href="/gothic-mark.svg" type="image/svg+xml">`,
		`<link rel="stylesheet" href="/style.css?v=`,
		`data-theme-toggle`,
		`prefers-color-scheme: dark`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("document missing %q\n\n%s", want, html)
		}
	}
	if strings.Contains(html, "gothic-ui.css") {
		t.Errorf("document should not load the removed UI stylesheet")
	}
}

func TestGothicbookPages(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	overview := getPage(t, ts.URL+"/")
	for _, want := range []string{
		`class="brand"`,
		`<span class="brand-book">book</span>`,
		`A headless design system for gothic`,
		`href="/button"`,
		`href="/combobox"`,
		`class="tile"`,
	} {
		if !strings.Contains(overview, want) {
			t.Errorf("overview missing %q\n\n%s", want, overview)
		}
	}

	pages := []struct {
		path string
		want []string
	}{
		{"/button", []string{`<h1>Button</h1>`, `data-part="button"`, `Server count:`}},
		{"/field", []string{`<h1>Field</h1>`, `data-part="field"`, `data-part="checkbox"`, `data-part="switch"`}},
		{"/tabs", []string{`<h1>Tabs</h1>`, `role="tablist"`, `role="tab"`, `role="tabpanel"`}},
		{"/accordion", []string{`<h1>Accordion</h1>`, `data-part="accordion"`, `aria-expanded="true"`}},
		{"/combobox", []string{`<h1>Combobox</h1>`, `role="combobox"`, `aria-controls="fruit-listbox"`, `name="fruit"`}},
	}
	for _, page := range pages {
		html := getPage(t, ts.URL+page.path)
		for _, want := range page.want {
			if !strings.Contains(html, want) {
				t.Errorf("GET %s missing %q\n\n%s", page.path, want, html)
			}
		}
		if !strings.Contains(html, `class="sidebar"`) {
			t.Errorf("GET %s missing the sidebar", page.path)
		}
	}

	notFound := getPage(t, ts.URL+"/does-not-exist")
	if !strings.Contains(notFound, "Not found") || !strings.Contains(notFound, "No page matches") {
		t.Errorf("missing not found page\n\n%s", notFound)
	}
}

func TestGothicbookAssetsAndModules(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	for _, path := range []string{"/style.css", "/gothic-mark.svg", "/gothic-wordmark.svg", "/og.png"} {
		if getPage(t, ts.URL+path) == "" {
			t.Fatalf("%s is empty", path)
		}
	}

	appJS := getPage(t, ts.URL+"/_gothic/modules/app_gothic.js")
	for _, want := range []string{
		`from "./ui_gothic.js"`,
		`mountCombobox`,
		`createCombobox`,
		`ComboboxValueChanged`,
		`IncrementClicked`,
		`LocationChanged`,
		`dispatch`,
	} {
		if !strings.Contains(appJS, want) {
			t.Errorf("app JS missing %q\n\n%s", want, appJS)
		}
	}

	uiJS := getPage(t, ts.URL+"/_gothic/modules/ui_gothic.js")
	for _, want := range []string{
		`export function mountCombobox`,
		`export function createCombobox`,
	} {
		if !strings.Contains(uiJS, want) {
			t.Errorf("UI JS missing %q", want)
		}
	}
}

func TestGothicbookServerState(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	postRPC(t, ts.URL, "Increment", `[]`)
	page := getPage(t, ts.URL+"/button")
	stripped := regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(page, "")
	if !strings.Contains(stripped, "Server count: 1") {
		t.Fatalf("server count did not update\n\n%s", stripped)
	}
}

func postRPC(t *testing.T, base, name, args string) {
	t.Helper()
	body := fmt.Sprintf(`{"args":%s}`, args)
	res, err := http.Post(base+"/_gothic/rpc/"+name, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		t.Fatalf("%s returned %d: %s", name, res.StatusCode, data)
	}
}
