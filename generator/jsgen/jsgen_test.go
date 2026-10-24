package jsgen_test

import (
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator/jsgen"
	"github.com/metruzanca/gothic/parser"
)

const demo = `package views

templ PingPong() {
	state messages []string = ["ready"]

	<div class="app">
		<button onclick={
			messages.Push("ping")
			pong := await api.Ping()
			messages.Push(pong)
		}>Ping</button>
		<ul>
			@each messages as m {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

proc Ping() string
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
		`const messages = signal(state["messages"] ?? []);`,
		`Ping: (...args) => rpc("Ping", ...args)`,
		`messages.push("ping");`,
		`const pong = await api.Ping();`,
		`messages.push(pong);`,
		`each(root, "0", messages, pingpongRenderEach0);`,
		`function pingpongRenderEach0(m)`,
		`document.createElement("li")`,
		`hydrate();`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}
