package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func TestPingPongApp(t *testing.T) {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
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
		`"Messages":["ready"]`,
		`data-gothic-on="0"`,
		`<!--g:each:0:open-->`,
		`<li>`,
		`<!--g:text:0:open-->ready<!--g:text:0:close-->`,
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
		`m.Messages.push("ping");`,
		`return [m, callPing];`,
		`pong = await Ping();`,
		`return {tag: "PongReceived", Text: pong};`,
		`text(e0, "0", m);`,
		`text(node, "0", m);`,
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
