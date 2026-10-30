package appjs_test

import (
	"errors"
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

func TestGenerateConditionalStatefulComponent(t *testing.T) {
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
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName: "view.gth",
		Stateful: []*generator.Stateful{{
			Name:       "Tabs",
			ModuleURL:  "/_gothic/modules/tabs.js",
			MountFunc:  "mountTabs",
			CreateFunc: "createTabs",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`import { mountTabs, createTabs } from "/_gothic/modules/tabs.js";`,
		`ifBlock(root, "0",`,
		`createTabs("c0")`,
		`frag.appendChild(e0);`,
		`querySelector('[data-gothic-component-id="c0"]')`,
		`mountTabs(gc1Node, dispatch)`,
		`__gothicInstances`,
		`updateProps()`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateLoopInsideConditionalBranch(t *testing.T) {
	view, err := parser.Parse("branch.gth", []byte(`package app

view App(model Model) {
	<div>
		if model.Show {
			for _, item := range model.Items {
				<li>{ item }</li>
			}
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Show bool; Items []string }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "branch.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`document.createComment("g:each:0:open")`,
		`document.createComment("g:each:0:close")`,
		`list(region, "0"`,
		`model.Items`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateComponentWithLoopsAndConditionals(t *testing.T) {
	view, err := parser.Parse("disclosure.gth", []byte(`package ui

view Disclosure(model DisclosureModel, title string, details []string) {
	<div class="ui-disclosure">
		<button onclick={ DisclosureToggled{} }>{ title }</button>
		if model.Expanded {
			<ul>
				for _, detail := range details {
					<li>{ detail }</li>
				}
			</ul>
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("ui.go", []byte(`package ui

type DisclosureModel struct{ Expanded bool }
type DisclosureToggled struct{}

func (m DisclosureModel) Init() Cmd { return nil }
func (m DisclosureModel) Update(msg Msg) (DisclosureModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	componentApp := app.ComponentApp("DisclosureModel")
	if componentApp == nil {
		t.Fatal("component app not found")
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name:      "Disclosure",
		View:      view.Templates[0],
		App:       componentApp,
		ModelJSON: `{"Expanded":false}`,
		Props: []parser.Param{
			{Name: "title", Type: "string"},
			{Name: "details", Type: "[]string"},
		},
		MountFunc:  "mountDisclosure",
		CreateFunc: "createDisclosure",
	}}, appjs.Options{FileName: "disclosure.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`export function createDisclosure(id, title, details)`,
		`document.createComment("g:if:0:open")`,
		`document.createComment("g:each:0:open")`,
		`data-gothic-component-id`,
		`wrapper.setAttribute('data-gothic-component', "Disclosure");`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, "cannot be created inside a list") {
		t.Errorf("CreateFunc should not throw\n\n%s", src)
	}
}

func TestGenerateTypeSwitchWithoutAssignment(t *testing.T) {
	view, err := parser.Parse("switch.gth", []byte(`package app

view App(model Model) {
	<div>{ model.On }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ On bool }
type Toggle struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg.(type) {
	case Toggle:
		m.On = !m.On
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "switch.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`switch (msg.tag) {`,
		`case "Toggle":`,
		`m.On = !m.On;`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateJSCalls(t *testing.T) {
	view, err := parser.Parse("js.gth", []byte(`package app

view App(model Model) {
	<div>{ model.Stored }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Stored string }
type SaveClicked struct{ Text string }
type ClearClicked struct{}

func (m Model) Init() Cmd {
	js.SetTitle("start")
	return nil
}

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case SaveClicked:
		js.SetItem("title", msg.Text)
		m.Stored = js.GetItem("title")
		js.SetTitle(m.Stored)
		js.Log("saved", msg.Text)
	case ClearClicked:
		js.RemoveItem("title")
		js.Alert("cleared")
		return m, copyTitle
	}
	return m, nil
}

func copyTitle() Msg {
	js.Copy("title")
	return nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "js.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`(document.title = String("start"));`,
		`localStorage.setItem("title", msg.Text);`,
		`m.Stored = (localStorage.getItem("title") ?? "");`,
		`(document.title = String(m.Stored));`,
		`console.log("saved", msg.Text);`,
		`localStorage.removeItem("title");`,
		`window.alert("cleared");`,
		`await navigator.clipboard.writeText("title");`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectJSCallsInViews(t *testing.T) {
	view, err := parser.Parse("js.gth", []byte(`package app

view App(model Model) {
	<div>{ js.GetItem("title") }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "js.gth"})
	if err == nil || !strings.Contains(err.Error(), "only be used in Init, Update, or commands") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateLocationState(t *testing.T) {
	view, err := parser.Parse("routes.gth", []byte(`package app

view App(model Model) {
	<div>
		<a href="/about" route>About</a>
		<button onclick={ GoAbout{} }>Go</button>
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}
type LocationChanged struct{ Path string; Query string; Hash string }
type GoAbout struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg.(type) {
	case LocationChanged:
	case GoAbout:
		js.Navigate("/about")
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "routes.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`on(root, "0", "click", routeClick);`,
		`navigate("/about");`,
		`function routeClick(event) {`,
		`window.addEventListener("popstate"`,
		`dispatch(locationMessage());`,
		`history.pushState(null, "", path);`,
		`Path: window.location.pathname`,
		`Query: window.location.search`,
		`Hash: window.location.hash`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectRouteWithoutLocationMessage(t *testing.T) {
	view, err := parser.Parse("routes.gth", []byte(`package app

view App(model Model) {
	<div><a href="/about" route>About</a></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "routes.gth"})
	if err == nil || !strings.Contains(err.Error(), "route links require a LocationChanged message") {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectNavigateWithoutLocationMessage(t *testing.T) {
	view, err := parser.Parse("routes.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}
type Go struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg.(type) {
	case Go:
		js.Navigate("/about")
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "routes.gth"})
	if err == nil || !strings.Contains(err.Error(), "js.Navigate requires a LocationChanged message") {
		t.Fatalf("error = %v", err)
	}
}

func TestJSErrorHasSourceContext(t *testing.T) {
	view, err := parser.Parse("js.gth", []byte(`package app

view App(model Model) {
	<div>
		<p>{ js.GetItem("name") }</p>
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "js.gth"})
	if err == nil {
		t.Fatal("expected an error")
	}
	var pe *parser.Error
	if !errors.As(err, &pe) {
		t.Fatalf("error type = %T: %v", err, err)
	}
	if pe.File != "js.gth" || pe.Line != 5 {
		t.Fatalf("position = %s:%d:%d", pe.File, pe.Line, pe.Col)
	}
	if !strings.Contains(pe.Hint, "Update") {
		t.Fatalf("hint = %q", pe.Hint)
	}
}
