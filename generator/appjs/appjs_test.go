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
		`import { attr, classToggle, entries, focus, ifBlock, len, list, on, readState, rpc, text } from "/_gothic/runtime.js";`,
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
		`len(model.Items) > 0`,
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

view Disclosure(model DisclosureModel, props DisclosureProps) {
	<div class="ui-disclosure">
		<button onclick={ DisclosureToggled{} }>{ props.Title }</button>
		if model.Expanded {
			<ul>
				for _, detail := range props.Details {
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

type DisclosureProps struct{ Title string; Details []string }
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
		PropsType: "DisclosureProps",
		Props: []parser.Param{
			{Name: "Title", Type: "string"},
			{Name: "Details", Type: "[]string"},
		},
		MountFunc:  "mountDisclosure",
		CreateFunc: "createDisclosure",
	}}, appjs.Options{FileName: "disclosure.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`export function createDisclosure(id, props)`,
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

func TestGenerateComponentPropsInUpdate(t *testing.T) {
	view, err := parser.Parse("tabs.gth", []byte(`package ui

view Tabs(model TabsModel, props TabsProps) {
	<div>{ model.Active }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("tabs.go", []byte(`package ui

type TabsProps struct{ Items []Tab }
type TabsModel struct{ Active int }
type Tab struct{ Label string }
type TabClicked struct{ Index int }

func (m TabsModel) Init() Cmd { return nil }

func (m TabsModel) Update(msg Msg, props TabsProps) (TabsModel, Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	case gothic.PropsChanged:
		if m.Active >= len(props.Items) {
			m.Active = len(props.Items) - 1
		}
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("TabsModel")
	if component == nil {
		t.Fatal("component app not found")
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name:       "Tabs",
		View:       view.Templates[0],
		App:        component,
		ModelJSON:  `{"Active":0}`,
		PropsType:  "TabsProps",
		Props:      []parser.Param{{Name: "Items", Type: "[]Tab"}},
		MountFunc:  "mountTabs",
		CreateFunc: "createTabs",
	}}, appjs.Options{FileName: "tabs.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`function update(m, msg, props) {`,
		`const result = update(model, msg, props);`,
		`case "PropsChanged":`,
		`if (!Object.is(props.Items, nextProps.Items))`,
		`dispatch({tag: "PropsChanged"});`,
		`updateProps(nextProps)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectComponentUpdatePropMismatch(t *testing.T) {
	view, err := parser.Parse("tabs.gth", []byte(`package ui

view Tabs(model TabsModel, props TabsProps) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("tabs.go", []byte(`package ui

type TabsProps struct{ Items []Tab }
type TabsModel struct{}
type Tab struct{}

func (m TabsModel) Init() Cmd { return nil }
func (m TabsModel) Update(msg Msg, props string) (TabsModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("TabsModel")
	_, err = appjs.GenerateComponents([]*generator.Stateful{{
		Name:       "Tabs",
		View:       view.Templates[0],
		App:        component,
		ModelJSON:  `{}`,
		PropsType:  "TabsProps",
		Props:      []parser.Param{{Name: "Items", Type: "[]Tab"}},
		MountFunc:  "mountTabs",
		CreateFunc: "createTabs",
	}}, appjs.Options{FileName: "tabs.gth"})
	if err == nil || !strings.Contains(err.Error(), "Update prop props has type string, want TabsProps") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "tabs.go:") {
		t.Fatalf("error lacks a source position: %v", err)
	}
}

func componentInitFixture(initSource string) (*parser.Template, *goapp.App) {
	view, _ := parser.Parse("counter.gth", []byte(`package ui

view Counter(model CounterModel) {
	<div>{ model.Count }</div>
}
`))
	app, _ := goapp.ParseComponents("counter.go", []byte(`package ui

type CounterModel struct{ Count int }

`+initSource+`

func (m CounterModel) Update(msg Msg) (CounterModel, Cmd) { return m, nil }
`))
	return view.Templates[0], app
}

func TestGenerateComponentInit(t *testing.T) {
	tpl, app := componentInitFixture(`func (m CounterModel) Init() Cmd {
	js.SetTitle("ready")
	return nil
}`)
	component := app.ComponentApp("CounterModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: tpl, App: component, ModelJSON: `{"Count":0}`,
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`function init() {`,
		`(document.title = String("ready"));`,
		`run(init());`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateComponentInitCommand(t *testing.T) {
	tpl, app := componentInitFixture(`func (m CounterModel) Init() Cmd {
	return doThing
}

func doThing() Msg {
	js.SetTitle("done")
	return nil
}`)
	component := app.ComponentApp("CounterModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: tpl, App: component, ModelJSON: `{"Count":0}`,
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`async function doThing() {`,
		`(document.title = String("done"));`,
		`run(init());`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectComponentProcedureCall(t *testing.T) {
	tpl, app := componentInitFixture(`func (m CounterModel) Init() Cmd {
	return load
}

func load() Msg {
	Ping()
	return nil
}`)
	component := app.ComponentApp("CounterModel")
	_, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: tpl, App: component, ModelJSON: `{"Count":0}`,
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth", ProcedureNames: map[string]bool{"Ping": true}})
	if err == nil || !strings.Contains(err.Error(), "Ping is an App procedure") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "emit an action") {
		t.Fatalf("error lacks a hint: %v", err)
	}
}

func TestRejectComponentInitProps(t *testing.T) {
	tpl, app := componentInitFixture(`func (m CounterModel) Init(label string) Cmd {
	return nil
}`)
	component := app.ComponentApp("CounterModel")
	_, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: tpl, App: component, ModelJSON: `{"Count":0}`,
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err == nil || !strings.Contains(err.Error(), "Init cannot take props without a props type") {
		t.Fatalf("error = %v", err)
	}
}

func TestComponentDestroyLifecycle(t *testing.T) {
	tpl, app := componentInitFixture(`func (m CounterModel) Init() Cmd { return nil }`)
	component := app.ComponentApp("CounterModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: tpl, App: component, ModelJSON: `{"Count":0}`,
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`let alive = true;`,
		`if (!alive) return;`,
		`destroy() { alive = false; }`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateOptionalComponentProps(t *testing.T) {
	appView, err := parser.Parse("box.gth", []byte(`package app

view App(model Model) {
	<div><Box /></div>
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
	box := &generator.Stateful{
		Name: "Box", PropsType: "BoxProps",
		Props:     []parser.Param{{Name: "Hint", Type: "*string"}},
		MountFunc: "mountBox", CreateFunc: "createBox",
		ModuleURL: "/_gothic/modules/box.js",
	}
	out, err := appjs.Generate(appView, app, appjs.Options{FileName: "box.gth", Stateful: []*generator.Stateful{box}})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	if !strings.Contains(src, `{Hint: null}`) {
		t.Errorf("optional prop should default to null\n\n%s", src)
	}
}

func TestGenerateProvidedOptionalComponentProp(t *testing.T) {
	appView, err := parser.Parse("box.gth", []byte(`package app

view App(model Model) {
	<div><Box hint={ model.Hint } /></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Hint *string }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	box := &generator.Stateful{
		Name: "Box", PropsType: "BoxProps",
		Props:     []parser.Param{{Name: "Hint", Type: "*string"}},
		MountFunc: "mountBox", CreateFunc: "createBox",
		ModuleURL: "/_gothic/modules/box.js",
	}
	out, err := appjs.Generate(appView, app, appjs.Options{FileName: "box.gth", Stateful: []*generator.Stateful{box}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `{Hint: model.Hint}`) {
		t.Errorf("provided optional prop missing\n\n%s", out)
	}
}

func TestGeneratePointerDeref(t *testing.T) {
	view, err := parser.Parse("box.gth", []byte(`package ui

view Box(model BoxModel, props BoxProps) {
	<div>
		if props.Hint != nil {
			<p>{ *props.Hint }</p>
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("box.go", []byte(`package ui

type BoxProps struct{ Hint *string }
type BoxModel struct{}

func (m BoxModel) Init() Cmd { return nil }
func (m BoxModel) Update(msg Msg) (BoxModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("BoxModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Box", View: view.Templates[0], App: component, ModelJSON: `{}`,
		PropsType: "BoxProps", Props: []parser.Param{{Name: "Hint", Type: "*string"}},
		MountFunc: "mountBox", CreateFunc: "createBox",
	}}, appjs.Options{FileName: "box.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`props.Hint != null`,
		`String(props.Hint)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectAddressOfProp(t *testing.T) {
	appView, err := parser.Parse("box.gth", []byte(`package app

view App(model Model) {
	<div><Box hint={ &model.Text } /></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Text string }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	box := &generator.Stateful{
		Name: "Box", PropsType: "BoxProps",
		Props:     []parser.Param{{Name: "Hint", Type: "*string"}},
		MountFunc: "mountBox", CreateFunc: "createBox",
		ModuleURL: "/_gothic/modules/box.js",
	}
	_, err = appjs.Generate(appView, app, appjs.Options{FileName: "box.gth", Stateful: []*generator.Stateful{box}})
	if err == nil || !strings.Contains(err.Error(), "address-of expressions") {
		t.Fatalf("error = %v", err)
	}
}

func TestGeneratePurePropsObject(t *testing.T) {
	view, err := parser.Parse("props.gth", []byte(`package app

view App(model Model) {
	<div>{ (CardProps{Title: "x"}).Title }</div>
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
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "props.gth",
		PropsTypes: map[string]bool{"CardProps": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	if !strings.Contains(src, `({Title: "x"}).Title`) {
		t.Errorf("props struct literal should be a plain object\n\n%s", src)
	}
	if strings.Contains(src, `tag: "CardProps"`) {
		t.Errorf("props struct literal should not be tagged as a message\n\n%s", src)
	}
}

func TestGenerateComponentInitProps(t *testing.T) {
	view, err := parser.Parse("counter.gth", []byte(`package ui

view Counter(model CounterModel, props CounterProps) {
	<div>{ model.Count }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("counter.go", []byte(`package ui

type CounterProps struct{ Label string }
type CounterModel struct{ Count int }

func (m CounterModel) Init(props CounterProps) Cmd {
	js.SetTitle(props.Label)
	return nil
}

func (m CounterModel) Update(msg Msg) (CounterModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("CounterModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: view.Templates[0], App: component, ModelJSON: `{"Count":0}`,
		PropsType: "CounterProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`function init(props) {`,
		`(document.title = String(props.Label));`,
		`run(init(props));`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectComponentInitWrongPropsType(t *testing.T) {
	view, err := parser.Parse("counter.gth", []byte(`package ui

view Counter(model CounterModel, props CounterProps) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("counter.go", []byte(`package ui

type CounterProps struct{ Label string }
type CounterModel struct{}

func (m CounterModel) Init(label string) Cmd { return nil }
func (m CounterModel) Update(msg Msg) (CounterModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("CounterModel")
	_, err = appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Counter", View: view.Templates[0], App: component, ModelJSON: `{}`,
		PropsType: "CounterProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountCounter", CreateFunc: "createCounter",
	}}, appjs.Options{FileName: "counter.gth"})
	if err == nil || !strings.Contains(err.Error(), "Init parameter has type string, want CounterProps") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateComponentConstructor(t *testing.T) {
	view, err := parser.Parse("tabs.gth", []byte(`package ui

view Tabs(model TabsModel, props TabsProps) {
	<div>{ model.Active }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("tabs.go", []byte(`package ui

type TabsProps struct{ Active *int }
type TabsModel struct{ Active int }

func NewTabsModel(props TabsProps) TabsModel {
	m := TabsModel{}
	if props.Active != nil {
		m.Active = *props.Active
	}
	return m
}

func (m TabsModel) Init() Cmd { return nil }
func (m TabsModel) Update(msg Msg) (TabsModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("TabsModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Tabs", View: view.Templates[0], App: component, ModelJSON: `{"Active":0}`, ModelType: "TabsModel",
		Constructor: "NewTabsModel", ConstructorTakesProps: true,
		PropsType: "TabsProps", Props: []parser.Param{{Name: "Active", Type: "*int"}},
		MountFunc: "mountTabs", CreateFunc: "createTabs",
	}}, appjs.Options{FileName: "tabs.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`function newTabs(props) {`,
		`const model = newTabs(props);`,
		`readComponentState(root) || newTabs(props)`,
		`if (props.Active != null)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectComponentConstructorWrongResult(t *testing.T) {
	view, err := parser.Parse("tabs.gth", []byte(`package ui

view Tabs(model TabsModel, props TabsProps) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("tabs.go", []byte(`package ui

type TabsProps struct{}
type TabsModel struct{}

func NewTabsModel(props TabsProps) string { return "" }
func (m TabsModel) Init() Cmd { return nil }
func (m TabsModel) Update(msg Msg) (TabsModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("TabsModel")
	_, err = appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Tabs", View: view.Templates[0], App: component, ModelJSON: `{}`, ModelType: "TabsModel",
		Constructor: "NewTabsModel", ConstructorTakesProps: true,
		PropsType: "TabsProps",
		MountFunc: "mountTabs", CreateFunc: "createTabs",
	}}, appjs.Options{FileName: "tabs.gth"})
	if err == nil || !strings.Contains(err.Error(), "constructor returns string, want TabsModel") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateCollectionExpressions(t *testing.T) {
	view, err := parser.Parse("coll.gth", []byte(`package app

view App(model Model) {
	<div>{ model.Items[0] }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Items  []string
	Counts map[string]int
	Index  int
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Items[0] = m.Items[1]
	m.Items = m.Items[:2]
	m.Items = m.Items[1:]
	m.Counts = map[string]int{"a": 1, "b": 2}
	m.Counts["c"] = len(m.Items)
	delete(m.Counts, "a")
	m.Index += 1
	m.Index *= 2
	m.Index++
	m.Index--
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "coll.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`m.Items[0] = m.Items[1];`,
		`m.Items = ((m.Items) ?? []).slice(0, 2);`,
		`m.Items = ((m.Items) ?? []).slice(1);`,
		`m.Counts = {["a"]: 1, ["b"]: 2};`,
		`m.Counts["c"] = len(m.Items);`,
		`delete (m.Counts)["a"];`,
		`m.Index += 1;`,
		`m.Index *= 2;`,
		`m.Index++;`,
		`m.Index--;`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectThreeIndexSlice(t *testing.T) {
	view, err := parser.Parse("coll.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Items []string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Items = m.Items[0:1:2]
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "coll.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "three-index slices are not supported") {
		t.Fatalf("error = %v", err)
	}
}

func TestRejectUnsupportedCompoundAssign(t *testing.T) {
	view, err := parser.Parse("coll.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Flags int }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Flags &^= 1
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "coll.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported assignment operator") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRangeLoop(t *testing.T) {
	view, err := parser.Parse("range.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Items  []string
	Counts map[string]int
	Total  int
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	for i, item := range m.Items {
		if item == "" {
			continue
		}
		m.Items[i] = item
		if i > 10 {
			break
		}
	}
	for _, count := range m.Counts {
		m.Total += count
	}
	for key := range m.Counts {
		_ = key
	}
	for range m.Items {
		m.Total++
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "range.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`for (const [i, item] of entries(m.Items)) {`,
		`continue;`,
		`break;`,
		`for (const [_, count] of entries(m.Counts)) {`,
		`for (const [key] of entries(m.Counts)) {`,
		`for (const __gothicItem of entries(m.Items)) {`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateSwitch(t *testing.T) {
	view, err := parser.Parse("switch.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Filter string
	Count  int
	Label  string
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch f := m.Filter; f {
	case "all", "active":
		m.Count++
	case "completed":
		m.Count--
	default:
		m.Label = "none"
	}
	switch {
	case m.Count > 0:
		m.Label = "positive"
	case m.Count < 0:
		m.Label = "negative"
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "switch.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const f = m.Filter;`,
		`switch (f) {`,
		`case "all":`,
		`case "active":`,
		`case "completed":`,
		`default:`,
		`switch (true) {`,
		`case m.Count > 0:`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectSwitchFallthrough(t *testing.T) {
	view, err := parser.Parse("switch.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Step int }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch m.Step {
	case 1:
		m.Step++
		fallthrough
	case 2:
		m.Step = 0
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "switch.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "fallthrough is not supported") {
		t.Fatalf("error = %v", err)
	}
}
