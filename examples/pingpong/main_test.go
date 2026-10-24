package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPingPongSlice(t *testing.T) {
	ts := httptest.NewServer(newServer())
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{
		`data-gothic-root="pingpong"`,
		`data-gothic-state="pingpong"`,
		`"messages":["ready"]`,
		`data-gothic-on="0"`,
		`<!--g:each:0:open-->`,
		`<li>ready</li>`,
		`<!--g:each:0:close-->`,
		`/_gothic/modules/pingpong_gothic.js`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/pingpong_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	js, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`messages.push("ping");`,
		`const pong = await api.Ping();`,
		`messages.push(pong);`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("generated JS missing %q", want)
		}
	}

	res, err = http.Post(ts.URL+"/_gothic/rpc/Ping", "application/json", strings.NewReader(`{"args":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	rpcBody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(rpcBody), `"pong"`) {
		t.Fatalf("rpc response = %q", rpcBody)
	}
}
