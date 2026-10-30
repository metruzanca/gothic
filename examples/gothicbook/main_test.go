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

func TestGothicbook(t *testing.T) {
	s := gothic.NewServer()
	mountApp(s, &counterStore{})
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{
		"Gothicbook",
		`class="ui-button`,
		`class="ui-card`,
		`class="ui-tabs`,
		`class="ui-accordion`,
		`data-gothic-component="Tabs"`,
		`data-gothic-component="Counter"`,
		`data-gothic-component="Disclosure"`,
		`class="ui-disclosure-list"`,
		`data-gothic-key="1"`,
		`Showing 1-10 of 20`,
		`Design tokens`,
		`Accessibility`,
		`Increment on client`,
		`Server count:`,
		`/gothic-ui.css`,
		`/style.css`,
		`<pre><code>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	for _, path := range []string{"/gothic-ui.css", "/style.css"} {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		css, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if len(css) == 0 {
			t.Fatalf("%s is empty", path)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	js, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`from "/_gothic/modules/ui_gothic.js"`,
		`mountTabs`,
		`mountAccordion`,
		`mountCounter`,
		`mountDisclosure`,
		`createDisclosure`,
		`model.Visible`,
		`IncrementClicked`,
		`dispatch`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("app JS missing %q\n\n%s", want, js)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/ui_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	uiJS, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`export function mountTabs`,
		`export function mountAccordion`,
		`export function mountCounter`,
		`export function mountDisclosure`,
		`export function createDisclosure`,
	} {
		if !strings.Contains(string(uiJS), want) {
			t.Errorf("UI JS missing %q", want)
		}
	}

	postRPC(t, ts.URL, "Increment", `[]`)
	res, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	stripped := regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(string(body), "")
	if !strings.Contains(stripped, "Server count:1") {
		t.Fatalf("server count did not update\n\n%s", stripped)
	}
}

func TestGothicbookPaginationSSR(t *testing.T) {
	model := NewModel(0)
	model.Page = 1
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
