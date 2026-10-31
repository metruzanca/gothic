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
		`class="viewport"`,
		`data-gothic-on="0"`,
		`style="height:0px"`,
		`style="height:179568px"`,
		`class="range"`,
		`5000`,
		`Row 1`,
		`Row 12`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}
	if got := strings.Count(html, `class="row"`); got != visibleCount {
		t.Errorf("SSR rendered %d rows, want %d", got, visibleCount)
	}
}

func TestVirtualListModule(t *testing.T) {
	ts := testServer(t)
	res, err := http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	js := string(body)
	for _, want := range []string{
		`on(root, "0", "scroll", (event) => dispatch({tag: "Scrolled", Top: event.target.scrollTop}));`,
		`attr(root, "0", "height:" + px(model.Window.Top));`,
		`m.Window = virtualWindow(msg.Top, m.RowHeight, m.Height, len(m.Items));`,
		`renderList0(((model.Items) ?? []).slice(model.Window.Start, model.Window.Start + model.Window.Count));`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("module JS missing %q\n\n%s", want, js)
		}
	}
}
