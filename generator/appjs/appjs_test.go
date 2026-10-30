package appjs_test

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator"
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
	view, err := parser.Parse("view.gth", []byte(viewSrc))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "view.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`import { attr, classToggle, focus, ifBlock, list, on, readState, rpc, text } from "/_gothic/runtime.js";`,
		`const Ping = (...args) => rpc("Ping", ...args);`,
		`on(root, "0", "click", (event) => dispatch({tag: "PingClicked"}));`,
		`const renderList0 = list(root, "0", (m, __i0) => {`,
		`text(e0, "0", m);`,
		`}, (node, m, __i0) => {`,
		`text(node, "0", m);`,
		`function render(model) {`,
		`renderList0(model.Messages);`,
		`function update(m, msg) {`,
		`switch (msg.tag) {`,
		`case "PingClicked":`,
		`(m.Messages = m.Messages || []).push("ping");`,
		`return [m, callPing];`,
		`case "PongReceived":`,
		`(m.Messages = m.Messages || []).push(msg.Text);`,
		`case "PingFailed":`,
		`(m.Messages = m.Messages || []).push(msg.Message);`,
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

const commandClosureApp = `package app

type Model struct{ Text string; Todos []string }
type AddClicked struct{}
type Added struct{ Todos []string }
type AddFailed struct{ Message string }

type Procedures interface {
	gothic.RPCService
	AddTodo(ctx context.Context, text string) error
	ListTodos(ctx context.Context) ([]string, error)
}

func (m Model) Init() Cmd { return loadTodos }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case AddClicked:
		text := m.Text
		return m, func() Msg {
			err := AddTodo(text)
			if err != nil {
				return AddFailed{Message: err.Error()}
			}
			return loadTodos()
		}
	}
	return m, nil
}

func loadTodos() Msg {
	todos, err := ListTodos()
	if err != nil {
		return AddFailed{Message: err.Error()}
	}
	return Added{Todos: todos}
}
`

const commandClosureView = `package app

view App(model Model) {
	<div>{ model.Text }</div>
}
`

func TestGenerateCommandClosure(t *testing.T) {
	view, err := parser.Parse("view.gth", []byte(commandClosureView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(commandClosureApp))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "view.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`return [m, async () => {`,
		`let err = null;`,
		`err = await AddTodo(text);`,
		`return await loadTodos();`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

const conditionalView = `package app

view App(model Model) {
	<div>
		if len(model.Items) > 0 {
			<ul>
				for _, item := range model.Items {
					<li>{ item }</li>
				}
			</ul>
		} else {
			<p>empty</p>
		}
	</div>
}
`

func TestGenerateConditionals(t *testing.T) {
	view, err := parser.Parse("cond.gth", []byte(conditionalView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Items []string; Loading bool }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "cond.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`ifBlock(root, "0",`,
		`((model.Items) ?? []).length > 0`,
		`document.createComment("g:each:0:open")`,
		`document.createDocumentFragment()`,
		`return frag;`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectStatefulComponentInsideConditional(t *testing.T) {
	view := &parser.File{Templates: []*parser.Template{{
		Name:   "App",
		Params: "(model Model)",
		ParamList: []parser.Param{
			{Name: "model", Type: "Model"},
		},
		Children: []parser.Node{&parser.Element{
			Name: "div",
			Children: []parser.Node{&parser.If{
				Cond: &ast.Ident{Name: "model"},
				Then: []parser.Node{&parser.ComponentCall{Name: "Tabs"}},
			}},
		}},
	}}}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName: "view.gth",
		Stateful: []*generator.Stateful{{
			Name:       "Tabs",
			ModuleURL:  "/_gothic/modules/tabs.js",
			MountFunc:  "mountTabs",
			CreateFunc: "createTabs",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "inside an if branch is not supported") {
		t.Fatalf("error = %v", err)
	}
}
