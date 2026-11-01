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
		`style="height:432px"`,
		`style="height:392px"`,
		`style="height:179568px"`,
		`style="height:10808px"`,
		`Row 1`,
		`Row 12`,
		`Item 1`,
		`Second list row 1`,
		`5,000 rows`,
		`/style.css`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}
	if got := strings.Count(html, `class="row"`); got != 12 {
		t.Errorf("SSR rendered %d fixed rows, want 12", got)
	}
	if got := strings.Count(html, `class="other-row"`); got != 7 {
		t.Errorf("SSR rendered %d second-list rows, want 7", got)
	}
	if strings.Contains(html, "gothic-ui.css") {
		t.Errorf("example should not load the UI kit stylesheet")
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
		`virtualWindow(`,
	} {
		if !strings.Contains(appJS, want) {
			t.Errorf("app JS missing %q\n\n%s", want, appJS)
		}
	}
}
