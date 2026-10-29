package appjs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator/appjs"
	"github.com/metruzanca/gothic/goapp"
	"github.com/metruzanca/gothic/parser"
)

const appSrc = `package app

type Model struct{ Messages []string }

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case PingClicked:
		m.Messages = append(m.Messages, "ping")
		return m, callPing
	case PongReceived:
		m.Messages = append(m.Messages, msg.Text)
	case PingFailed:
		m.Messages = append(m.Messages, msg.Message)
	}
	return m, nil
}

func callPing() Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}
`

const viewSrc = `package app

view App(model Model) {
	<div>
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}
`

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "app.go")
	if err := os.WriteFile(appPath, []byte(appSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	app, err := goapp.Parse(appPath)
	if err != nil {
		t.Fatal(err)
	}
	view, err := parser.Parse("view.gothic", []byte(viewSrc))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "view.gothic",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`import { list, on, readState, rpc, text } from "/_gothic/runtime.js";`,
		`const Ping = (...args) => rpc("Ping", ...args);`,
		`on(root, "0", "click", () => dispatch({tag: "PingClicked"}));`,
		`const renderList0 = list(root, "0", (m) => {`,
		`text(e0, "0", m);`,
		`}, (node, m) => {`,
		`text(node, "0", m);`,
		`function render(model) {`,
		`renderList0(model.Messages);`,
		`function update(m, msg) {`,
		`switch (msg.tag) {`,
		`case "PingClicked":`,
		`m.Messages.push("ping");`,
		`return [m, callPing];`,
		`case "PongReceived":`,
		`m.Messages.push(msg.Text);`,
		`case "PingFailed":`,
		`m.Messages.push(msg.Message);`,
		`async function callPing() {`,
		`pong = await Ping();`,
		`if (err != null) {`,
		`return {tag: "PingFailed", Message: err.message};`,
		`return {tag: "PongReceived", Text: pong};`,
		`render(model);`,
		`run(init());`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}
