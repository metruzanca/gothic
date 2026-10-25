package jsgen_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator/jsgen"
	"github.com/metruzanca/gothic/parser"
)

const demo = `package views

templ PingPong() {
	state messages = []string{"ready"}

	<div class="app">
		<button onclick={ func() {
			messages = append(messages, "ping")
			pong, err := api.Ping()
			if err != nil {
				messages = append(messages, err.Error())
				return
			}
			messages = append(messages, pong)
		} }>Ping</button>
		<ul>
			for _, m := range messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

rpc Ping() string
`

func TestGenerate(t *testing.T) {
	f, err := parser.Parse("pingpong.gothic", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	out, err := jsgen.Generate(f, jsgen.Options{
		FileName:   "pingpong.gothic",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`import { readState, signal, each, on, rpc } from "/_gothic/runtime.js";`,
		`const messages = signal(["ready"]);`,
		`Ping: (...args) => rpc("Ping", ...args)`,
		`on(root, "0", "click", async () => {`,
		`messages.push("ping");`,
		`let pong, err;`,
		`try {`,
		`pong = await api.Ping();`,
		`} catch (__gothicErr1) {`,
		`err = __gothicErr1;`,
		`if (err != null) {`,
		`messages.push(err.message);`,
		`return;`,
		`messages.push(pong);`,
		`each(root, "0", messages, pingpongRenderEach0);`,
		`function pingpongRenderEach0(_index, m)`,
		`document.createElement("li")`,
		`String(m)`,
		`hydrate();`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, "await await") {
		t.Errorf("generated JS contains a duplicated await\n\n%s", src)
	}
}

func TestGenerateIsStableAcrossPaths(t *testing.T) {
	f, err := parser.Parse("pingpong.gothic", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	a, err := jsgen.Generate(f, jsgen.Options{FileName: "examples/pingpong/views/pingpong.gothic"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := jsgen.Generate(f, jsgen.Options{FileName: "views/pingpong.gothic"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("generated JS depends on the source path\n\nA:\n%s\n\nB:\n%s", a, b)
	}
}
