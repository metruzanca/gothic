package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func TestComponentsApp(t *testing.T) {
	s := gothic.NewServer()
	s.MountApp("/", gothic.NewApp(NewModel()))
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
		`data-gothic-root="app"`,
		`"Count":0`,
		`"History":["ready"]`,
		`<p class="count">`,
		`<!--g:text:0:open-->0<!--g:text:0:close-->`,
		`<li class="history">`,
		`data-gothic-on="1"`,
		`/_gothic/modules/app_gothic.js`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	js, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`text(root, "0", (model.Count));`,
		`on(node, "1", "click", (event) => dispatch({tag: "RepeatClicked", Text: (item)}));`,
		`(m.History = m.History || []).push(msg.Text);`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("generated JS missing %q\n\n%s", want, js)
		}
	}
}
