package main

import (
	"context"
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
		`<meta property="og:description" content="A tiny design system demo for the installable gothic UI kit.">`,
		`<meta property="og:image" content="` + ts.URL + `/og.png?v=`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<link rel="icon" href="/_gothic/icon.svg" type="image/svg+xml">`,
		`<link rel="apple-touch-icon" href="/_gothic/apple-touch-icon.png" type="image/png" sizes="180x180">`,
		`<link rel="stylesheet" href="/gothic-ui.css?v=`,
		`<link rel="stylesheet" href="/style.css?v=`,
		`data-theme-toggle`,
		`prefers-color-scheme: dark`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("document missing %q\n\n%s", want, html)
		}
	}
}

func TestGothicbookNavigation(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	overview := getPage(t, ts.URL+"/")
	for _, want := range []string{
		`class="brand"`,
		`<span class="brand-book">book</span>`,
		`class="nav-group"`,
		`A small design system for gothic`,
		`href="/button"`,
		`href="/virtual-list"`,
		`class="tile"`,
	} {
		if !strings.Contains(overview, want) {
			t.Errorf("overview missing %q\n\n%s", want, overview)
		}
	}

	pages := []struct {
		path string
		want string
	}{
		{"/button", `<h1>Button</h1>`},
		{"/card", `<h1>Card</h1>`},
		{"/field", `<h1>Field</h1>`},
		{"/tabs", `<h1>Tabs</h1>`},
		{"/accordion", `<h1>Accordion</h1>`},
		{"/disclosure", `<h1>Disclosure</h1>`},
		{"/virtual-list", `<h1>Virtual list</h1>`},
		{"/conditionals", `<h1>Conditionals</h1>`},
		{"/stateful-lists", `<h1>Stateful lists</h1>`},
		{"/server-state", `<h1>Server state</h1>`},
	}
	for _, page := range pages {
		html := getPage(t, ts.URL+page.path)
		if !strings.Contains(html, page.want) {
			t.Errorf("GET %s missing %q\n\n%s", page.path, page.want, html)
		}
		if !strings.Contains(html, `class="sidebar"`) {
			t.Errorf("GET %s missing the sidebar", page.path)
		}
	}

	button := getPage(t, ts.URL+"/button")
	if !strings.Contains(button, `class="ui-button`) {
		t.Errorf("button page missing the button demo\n\n%s", button)
	}
	field := getPage(t, ts.URL+"/field")
	if !strings.Contains(field, "Your name") || !strings.Contains(field, "slot-input") {
		t.Errorf("field page missing the field demo\n\n%s", field)
	}
	tabs := getPage(t, ts.URL+"/tabs")
	if !strings.Contains(tabs, `class="ui-tabs`) || !strings.Contains(tabs, `{"Active":1}`) {
		t.Errorf("tabs page missing the tabs demo\n\n%s", tabs)
	}
	if !strings.Contains(tabs, "nav-link active") {
		t.Errorf("tabs nav item is not active\n\n%s", tabs)
	}
	if got := strings.Count(tabs, " disabled "); got != 1 {
		t.Errorf("tabs page has %d disabled buttons, want 1\n\n%s", got, tabs)
	}
	lists := getPage(t, ts.URL+"/stateful-lists")
	for _, want := range []string{
		`data-gothic-key="1"`,
		`Showing 1-10 of 20`,
		`Design tokens`,
		`Accessibility`,
	} {
		if !strings.Contains(lists, want) {
			t.Errorf("stateful lists page missing %q\n\n%s", want, lists)
		}
	}

	virtual := getPage(t, ts.URL+"/virtual-list")
	if got := strings.Count(virtual, `class="ui-virtual-row"`); got != 12 {
		t.Errorf("SSR rendered %d virtual rows, want 12", got)
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

	for _, path := range []string{"/gothic-ui.css", "/style.css", "/gothic-mark.svg", "/gothic-wordmark.svg", "/og.png"} {
		if getPage(t, ts.URL+path) == "" {
			t.Fatalf("%s is empty", path)
		}
	}

	js := getPage(t, ts.URL+"/_gothic/modules/app_gothic.js")
	for _, want := range []string{
		`from "/_gothic/modules/ui_gothic.js"`,
		`mountTabs`,
		`mountAccordion`,
		`mountCounter`,
		`mountDisclosure`,
		`createDisclosure`,
		`mountuiVirtualList`,
		`model.Visible`,
		`SlotTextChanged`,
		`IncrementClicked`,
		`ButtonClicked`,
		`window.alert("Hello from Gothicbook")`,
		`LocationChanged`,
		`dispatch`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("app JS missing %q\n\n%s", want, js)
		}
	}

	uiJS := getPage(t, ts.URL+"/_gothic/modules/ui_gothic.js")
	for _, want := range []string{
		`export function mountTabs`,
		`export function mountAccordion`,
		`export function mountCounter`,
		`export function mountDisclosure`,
		`export function createDisclosure`,
	} {
		if !strings.Contains(uiJS, want) {
			t.Errorf("UI JS missing %q", want)
		}
	}

	compJS := getPage(t, ts.URL+"/_gothic/modules/app_components_gothic.js")
	for _, want := range []string{
		`export function mountuiVirtualList`,
		`export function createuiVirtualList`,
		`virtualWindow(`,
	} {
		if !strings.Contains(compJS, want) {
			t.Errorf("component JS missing %q", want)
		}
	}
}

func TestGothicbookServerState(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	postRPC(t, ts.URL, "Increment", `[]`)
	page := getPage(t, ts.URL+"/server-state")
	stripped := regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(page, "")
	if !strings.Contains(stripped, `ui-counter-value">1</strong>`) {
		t.Fatalf("server count did not update\n\n%s", stripped)
	}
}

func TestGothicbookPaginationSSR(t *testing.T) {
	model := NewModelAt("/stateful-lists", 0)
	model.ListPage = 1
	model.Visible = model.PageTwo
	model.PageLabel = "Showing 11-20 of 20"

	var page strings.Builder
	if err := App(model).Render(context.Background(), &page); err != nil {
		t.Fatal(err)
	}
	html := page.String()
	markup := regexp.MustCompile(`(?s)<script type="application/json".*?</script>`).ReplaceAllString(html, "")

	for _, want := range []string{
		`data-gothic-key="11"`,
		`data-gothic-key="20"`,
		`Tables`,
		`Accessibility`,
		`Showing 11-20 of 20`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("page two SSR missing %q\n\n%s", want, markup)
		}
	}
	if strings.Contains(markup, "Design tokens") || strings.Contains(markup, "Validation") {
		t.Errorf("page two SSR contains page one items\n\n%s", markup)
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
