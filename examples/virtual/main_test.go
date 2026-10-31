package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := newServer()
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return ts
}

func TestVirtualListSSR(t *testing.T) {
	ts := testServer(t)
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{
		`data-gothic-root="app"`,
		`data-gothic-component="VirtualList"`,
		`class="ui-virtual-list"`,
		`style="height:432px"`,
		`style="height:0px"`,
		`style="height:179568px"`,
		`Row 1`,
		`Row 12`,
		`5,000 rows`,
		`/gothic-ui.css`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}
	if got := strings.Count(html, `class="ui-virtual-row"`); got != 12 {
		t.Errorf("SSR rendered %d rows, want 12", got)
	}
}

func TestVirtualListModules(t *testing.T) {
	ts := testServer(t)

	res, err := http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	appJS := string(body)
	for _, want := range []string{
		`from "/_gothic/modules/ui_gothic.js"`,
		`from "/_gothic/modules/app_components_gothic.js"`,
		`mountuiVirtualList`,
		`createuiVirtualList`,
	} {
		if !strings.Contains(appJS, want) {
			t.Errorf("app JS missing %q\n\n%s", want, appJS)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_components_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	compJS := string(body)
	for _, want := range []string{
		`export function mountuiVirtualList`,
		`export function createuiVirtualList`,
		`virtualWindow(`,
	} {
		if !strings.Contains(compJS, want) {
			t.Errorf("component JS missing %q\n\n%s", want, compJS)
		}
	}
}
