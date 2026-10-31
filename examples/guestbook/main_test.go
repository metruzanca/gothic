package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func newTestServer(t *testing.T) (*httptest.Server, *guestbook) {
	t.Helper()
	g := &guestbook{}
	s := gothic.NewServer()
	RegisterProcedures(s, g)
	s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
		return NewModel(g.Messages())
	}))
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return ts, g
}

func get(t *testing.T, url string) string {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return string(body)
}

func post(t *testing.T, url, args string) (int, string) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(`{"args":`+args+`}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return res.StatusCode, string(body)
}

func TestGuestbookEmptyState(t *testing.T) {
	ts, _ := newTestServer(t)
	html := get(t, ts.URL+"/")
	for _, want := range []string{
		`data-gothic-root="app"`,
		`"Messages":null`,
		`<!--g:if:0:then-->`,
		`<p>No messages yet.</p>`,
		`data-gothic-on="0 1"`,
		`data-gothic-on="2"`,
		`/_gothic/modules/app_gothic.js`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	js := get(t, ts.URL+"/_gothic/modules/app_gothic.js")
	for _, want := range []string{
		`dispatch({tag: "DraftChanged", Text: event.target.value})`,
		`if (event.key === "Enter") dispatch({tag: "PostClicked"});`,
		`posted = await Post(text);`,
		`(m.Messages = m.Messages || []).push(msg.Message);`,
		`ifBlock(root, "0",`,
		`len(model.Messages) == 0,`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("generated JS missing %q", want)
		}
	}
}

func TestGuestbookPost(t *testing.T) {
	ts, g := newTestServer(t)
	status, body := post(t, ts.URL+"/_gothic/rpc/Post", `["  hello  "]`)
	if status != http.StatusOK {
		t.Fatalf("rpc status = %d, body = %s", status, body)
	}
	if !strings.Contains(body, `"result":{"Text":"hello"}`) {
		t.Fatalf("rpc response = %s", body)
	}
	if len(g.Messages()) != 1 {
		t.Fatalf("stored messages = %v", g.Messages())
	}

	html := get(t, ts.URL+"/")
	for _, want := range []string{
		`"Messages":[{"Text":"hello"}]`,
		`<li data-gothic-key="0">`,
		`<!--g:text:0:open-->hello<!--g:text:0:close-->`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}
}

func TestGuestbookRejectsEmptyMessage(t *testing.T) {
	ts, g := newTestServer(t)
	status, body := post(t, ts.URL+"/_gothic/rpc/Post", `["   "]`)
	if status != http.StatusInternalServerError {
		t.Fatalf("rpc status = %d, body = %s", status, body)
	}
	if !strings.Contains(body, "write something first") {
		t.Fatalf("rpc response = %s", body)
	}
	if len(g.Messages()) != 0 {
		t.Fatalf("stored messages = %v", g.Messages())
	}
}
