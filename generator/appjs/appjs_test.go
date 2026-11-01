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
	view, err := parser.ParseViews("view.gth", []byte(viewSrc))
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
		`import { attr, classToggle, entries, focus, ifBlock, len, list, on, px, readState, rpc, routeParams, text, virtualRowsWindow, virtualVariableWindow, virtualWindow } from "/_gothic/runtime.js";`,
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
	view, err := parser.ParseViews("view.gth", []byte(commandClosureView))
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
	view, err := parser.ParseViews("cond.gth", []byte(conditionalView))
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
		`import { mountTabs, createTabs } from "./tabs.js";`,
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
	view, err := parser.ParseViews("branch.gth", []byte(`package app

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
	view, err := parser.ParseViews("disclosure.gth", []byte(`package ui

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
	view, err := parser.ParseViews("switch.gth", []byte(`package app

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
	view, err := parser.ParseViews("browser.gth", []byte(`package app

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
	browser.SetTitle("start")
	return nil
}

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case SaveClicked:
		browser.SetItem("title", msg.Text)
		m.Stored = browser.GetItem("title")
		browser.SetTitle(m.Stored)
		browser.Log("saved", msg.Text)
	case ClearClicked:
		browser.RemoveItem("title")
		browser.Alert("cleared")
		return m, copyTitle
	}
	return m, nil
}

func copyTitle() Msg {
	browser.Copy("title")
	return nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "browser.gth"})
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
	view, err := parser.ParseViews("browser.gth", []byte(`package app

view App(model Model) {
	<div>{ browser.GetItem("title") }</div>
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
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "browser.gth"})
	if err == nil || !strings.Contains(err.Error(), "only be used in Init, Update, or commands") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateLocationState(t *testing.T) {
	view, err := parser.ParseViews("routes.gth", []byte(`package app

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
		browser.Navigate("/about")
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
	view, err := parser.ParseViews("routes.gth", []byte(`package app

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
	view, err := parser.ParseViews("routes.gth", []byte(`package app

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
		browser.Navigate("/about")
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "routes.gth"})
	if err == nil || !strings.Contains(err.Error(), "browser.Navigate requires a LocationChanged message") {
		t.Fatalf("error = %v", err)
	}
}

func TestJSErrorHasSourceContext(t *testing.T) {
	view, err := parser.ParseViews("browser.gth", []byte(`package app

view App(model Model) {
	<div>
		<p>{ browser.GetItem("name") }</p>
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
	_, err = appjs.Generate(view, app, appjs.Options{FileName: "browser.gth"})
	if err == nil {
		t.Fatal("expected an error")
	}
	var pe *parser.Error
	if !errors.As(err, &pe) {
		t.Fatalf("error type = %T: %v", err, err)
	}
	if pe.File != "browser.gth" || pe.Line != 5 {
		t.Fatalf("position = %s:%d:%d", pe.File, pe.Line, pe.Col)
	}
	if !strings.Contains(pe.Hint, "Update") {
		t.Fatalf("hint = %q", pe.Hint)
	}
}

func TestGenerateComponentPropsInUpdate(t *testing.T) {
	view, err := parser.ParseViews("tabs.gth", []byte(`package ui

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
	view, err := parser.ParseViews("tabs.gth", []byte(`package ui

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
	view, _ := parser.ParseViews("counter.gth", []byte(`package ui

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
	browser.SetTitle("ready")
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
	browser.SetTitle("done")
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
	if err == nil || !strings.Contains(err.Error(), "Init takes no parameters") {
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
	appView, err := parser.ParseViews("box.gth", []byte(`package app

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
	appView, err := parser.ParseViews("box.gth", []byte(`package app

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
	view, err := parser.ParseViews("box.gth", []byte(`package ui

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
	appView, err := parser.ParseViews("box.gth", []byte(`package app

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
	view, err := parser.ParseViews("props.gth", []byte(`package app

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

func TestGenerateComponentSeedsFromProps(t *testing.T) {
	view, err := parser.ParseViews("tabs.gth", []byte(`package ui

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

func (m TabsModel) Init() Cmd { return nil }
func (m TabsModel) Update(msg Msg, props TabsProps) (TabsModel, Cmd) {
	switch msg := msg.(type) {
	case gothic.PropsChanged:
		if msg.Initial && props.Active != nil {
			m.Active = *props.Active
		}
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	component := app.ComponentApp("TabsModel")
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Tabs", View: view.Templates[0], App: component, ModelJSON: `{"Active":0}`, ModelType: "TabsModel",
		PropsType: "TabsProps", Props: []parser.Param{{Name: "Active", Type: "*int"}},
		MountFunc: "mountTabs", CreateFunc: "createTabs",
	}}, appjs.Options{FileName: "tabs.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const saved = readComponentState(root);`,
		`if (!saved) model = update(model, {tag: "PropsChanged", Initial: true}, props)[0];`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, "newTabs") {
		t.Errorf("generated JS should not emit a constructor\n\n%s", src)
	}
}

func TestGenerateCollectionExpressions(t *testing.T) {
	view, err := parser.ParseViews("coll.gth", []byte(`package app

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
	view, err := parser.ParseViews("coll.gth", []byte(`package app

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
	view, err := parser.ParseViews("coll.gth", []byte(`package app

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
	view, err := parser.ParseViews("range.gth", []byte(`package app

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
	view, err := parser.ParseViews("switch.gth", []byte(`package app

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
	view, err := parser.ParseViews("switch.gth", []byte(`package app

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

func TestGenerateRouteParams(t *testing.T) {
	view, err := parser.ParseViews("routes.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Page string
	ID   string
}

type LocationChanged struct{ Path string; Query string; Hash string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case LocationChanged:
		params := route.Params("/todos/:id", msg.Path)
		if params != nil {
			m.ID = params["id"]
		}
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "routes.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const params = routeParams("/todos/:id", msg.Path);`,
		`if (params != null) {`,
		`m.ID = params["id"];`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectUnknownRouteFunction(t *testing.T) {
	view, err := parser.ParseViews("routes.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Path string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Path = route.Build("/todos/:id", msg.Path)
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "routes.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown route function Build") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateIfInit(t *testing.T) {
	view, err := parser.ParseViews("ifinit.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Path string; ID string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	if params := route.Params("/todos/:id", m.Path); params != nil {
		m.ID = params["id"]
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "ifinit.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const params = routeParams("/todos/:id", m.Path);`,
		`if (params != null) {`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func nestedComponentFixtures(t *testing.T) (*parser.Template, *parser.Template, *goapp.App) {
	t.Helper()
	childView, err := parser.ParseViews("child.gth", []byte(`package app

view Child(model ChildModel, props ChildProps) {
	<span>{ props.Label }</span>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	parentView, err := parser.ParseViews("parent.gth", []byte(`package app

view Parent(model ParentModel) {
	<div>
		<Child label={ "hi" } />
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("components.go", []byte(`package app

type ChildProps struct{ Label string }
type ChildModel struct{ Count int }
type ParentModel struct {
	Show   bool
	Labels []string
}

func (m ChildModel) Init() Cmd { return nil }
func (m ChildModel) Update(msg Msg, props ChildProps) (ChildModel, Cmd) { return m, nil }

func (m ParentModel) Init() Cmd { return nil }
func (m ParentModel) Update(msg Msg) (ParentModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	return childView.Templates[0], parentView.Templates[0], app
}

func TestGenerateNestedStatefulComponent(t *testing.T) {
	child, parent, app := nestedComponentFixtures(t)
	childComp := &generator.Stateful{
		Name: "Child", View: child, App: app.ComponentApp("ChildModel"), ModelJSON: `{"Count":0}`, ModelType: "ChildModel",
		PropsType: "ChildProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountChild", CreateFunc: "createChild",
	}
	parentComp := &generator.Stateful{
		Name: "Parent", View: parent, App: app.ComponentApp("ParentModel"), ModelJSON: `{"Show":false}`, ModelType: "ParentModel",
		MountFunc: "mountParent", CreateFunc: "createParent",
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{childComp, parentComp}, appjs.Options{
		FileName:    "components.gth",
		AllStateful: []*generator.Stateful{childComp, parentComp},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const c0Node = root.querySelector('[data-gothic-component-id="c0"]');`,
		`const c0 = mountChild(c0Node, {Label: "hi"}, emit);`,
		`if (c0Node) c0Node.__gothicInstances = { "c0": c0 };`,
		`c0.updateProps({Label: "hi"});`,
		`createChild("c0", {Label: "hi"});`,
		`export function mountChild(root, props, __gothicParentDispatch) {`,
		`export function createChild(id, props) {`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateNestedExternalComponentImport(t *testing.T) {
	child, parent, app := nestedComponentFixtures(t)
	childComp := &generator.Stateful{
		Name: "Child", View: child, App: app.ComponentApp("ChildModel"), ModelJSON: `{"Count":0}`, ModelType: "ChildModel",
		PropsType: "ChildProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountCounter", CreateFunc: "createCounter",
		External: true, ModuleURL: "/_gothic/modules/ui.js",
	}
	parentComp := &generator.Stateful{
		Name: "Parent", View: parent, App: app.ComponentApp("ParentModel"), ModelJSON: `{"Show":false}`, ModelType: "ParentModel",
		MountFunc: "mountParent", CreateFunc: "createParent", ModuleURL: "/_gothic/modules/app.js",
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{parentComp}, appjs.Options{
		FileName:    "components.gth",
		AllStateful: []*generator.Stateful{childComp, parentComp},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`import { createCounter, mountCounter } from "./ui.js";`,
		`const c0 = mountCounter(c0Node, {Label: "hi"}, emit);`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateNestedComponentInComponentConditional(t *testing.T) {
	child, _, app := nestedComponentFixtures(t)
	parentView, err := parser.ParseViews("parent.gth", []byte(`package app

view Parent(model ParentModel) {
	<div>
		if model.Show {
			<Child label={ "hi" } />
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	childComp := &generator.Stateful{
		Name: "Child", View: child, App: app.ComponentApp("ChildModel"), ModelJSON: `{"Count":0}`, ModelType: "ChildModel",
		PropsType: "ChildProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountChild", CreateFunc: "createChild",
	}
	parentComp := &generator.Stateful{
		Name: "Parent", View: parentView.Templates[0], App: app.ComponentApp("ParentModel"), ModelJSON: `{"Show":false}`, ModelType: "ParentModel",
		MountFunc: "mountParent", CreateFunc: "createParent",
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{childComp, parentComp}, appjs.Options{
		FileName:    "components.gth",
		AllStateful: []*generator.Stateful{childComp, parentComp},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`ifBlockLocal(root, "0",`,
		`createChild("c0", {Label: "hi"})`,
		`mountChild(`,
		`, emit);`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateNestedComponentInComponentList(t *testing.T) {
	child, _, app := nestedComponentFixtures(t)
	parentView, err := parser.ParseViews("parent.gth", []byte(`package app

view Parent(model ParentModel) {
	<div>
		for _, label := range model.Labels {
			<Child label={ label } />
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	childComp := &generator.Stateful{
		Name: "Child", View: child, App: app.ComponentApp("ChildModel"), ModelJSON: `{"Count":0}`, ModelType: "ChildModel",
		PropsType: "ChildProps", Props: []parser.Param{{Name: "Label", Type: "string"}},
		MountFunc: "mountChild", CreateFunc: "createChild",
	}
	parentComp := &generator.Stateful{
		Name: "Parent", View: parentView.Templates[0], App: app.ComponentApp("ParentModel"), ModelJSON: `{"Show":false}`, ModelType: "ParentModel",
		MountFunc: "mountParent", CreateFunc: "createParent",
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{childComp, parentComp}, appjs.Options{
		FileName:    "components.gth",
		AllStateful: []*generator.Stateful{childComp, parentComp},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`listLocal(root, "0",`,
		`createChild("c0", {Label: label})`,
		`mountChild(node, {Label: label}, emit);`,
		`inst.updateProps({Label: label})`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateDomPx(t *testing.T) {
	view, err := parser.ParseViews("px.gth", []byte(`package app

view App(model Model) {
	<div class="spacer" style={ "height:" + dom.Px(model.Top) }></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Top int }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "px.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`"height:" + px(model.Top)`,
		`import { attr, classToggle, entries, focus, ifBlock, len, list, on, px, readState, rpc, routeParams, text, virtualRowsWindow, virtualVariableWindow, virtualWindow }`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectUnknownDomFunction(t *testing.T) {
	view, err := parser.ParseViews("px.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Top int }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Top = dom.Rem(3)
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "px.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown dom function Rem") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateVirtualWindow(t *testing.T) {
	view, err := parser.ParseViews("virtual.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Top int }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) {
	win := virtual.WindowFor(m.Top, 36, 432, 5000)
	m.Top = win.Top
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "virtual.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const win = virtualWindow(m.Top, 36, 432, 5000);`,
		`m.Top = win.Top;`,
		`virtualWindow } from "/_gothic/runtime.js"`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectUnknownVirtualFunction(t *testing.T) {
	view, err := parser.ParseViews("virtual.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Top int }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) {
	m.Top = virtual.Page(1)
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "virtual.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown virtual function Page") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateBatchAndSequence(t *testing.T) {
	view, err := parser.ParseViews("batch.gth", []byte(`package app

view App(model Model) {
	<div>{ model.Log }</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Log string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	return m, gothic.Batch(
		gothic.Sequence(func() Msg { return nil }),
		func() Msg { return nil },
	)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "batch.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`{ __gothicSequence: [async () => {`,
		`{ __gothicBatch: [{ __gothicSequence: [async () => {`,
		`if (cmd.__gothicBatch) {`,
		`await Promise.all(cmd.__gothicBatch.map((sub) => run(sub)));`,
		`if (cmd.__gothicSequence) {`,
		`for (const sub of cmd.__gothicSequence) {`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateComponentBatchAndSequence(t *testing.T) {
	view, err := parser.ParseViews("box.gth", []byte(`package ui

view Box(model BoxModel) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("box.go", []byte(`package ui

type BoxModel struct{ Log string }

func (m BoxModel) Init() Cmd { return nil }

func (m BoxModel) Update(msg Msg) (BoxModel, Cmd) {
	return m, gothic.Batch(
		func() Msg { return nil },
		func() Msg { return nil },
	)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Box", View: view.Templates[0], App: app.ComponentApp("BoxModel"), ModelJSON: `{"Log":""}`, ModelType: "BoxModel",
		MountFunc: "mountBox", CreateFunc: "createBox",
	}}, appjs.Options{FileName: "box.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`if (cmd.__gothicBatch) {`,
		`await Promise.all(cmd.__gothicBatch.map((sub) => run(sub)));`,
		`if (!alive) return;`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated component JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateVariableWindow(t *testing.T) {
	view, err := parser.ParseViews("virtual.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Top     int
	Heights map[int]int
}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) {
	win := virtual.VariableWindow(m.Heights, 40, m.Top, 432, 5000)
	m.Top = win.Top
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "virtual.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const win = virtualVariableWindow(m.Heights, 40, m.Top, 432, 5000);`,
		`virtualVariableWindow, virtualWindow } from "/_gothic/runtime.js"`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateVirtualRowsWindow(t *testing.T) {
	view, err := parser.ParseViews("virtual.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Top     int
	Heights map[int]int
}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) {
	win := virtual.WindowForRows(m.Heights, 0, 40, m.Top, 432, 5000)
	m.Top = win.Top
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "virtual.gth",
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const win = virtualRowsWindow(m.Heights, 0, 40, m.Top, 432, 5000);`,
		`virtualRowsWindow, virtualVariableWindow, virtualWindow } from "/_gothic/runtime.js"`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateClientHeight(t *testing.T) {
	view, err := parser.ParseViews("height.gth", []byte(`package app

view App(model Model) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Height int }

func (m Model) Init() Cmd {
	m.Height = browser.ClientHeight("#rows")
	return nil
}
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{FileName: "height.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	if !strings.Contains(src, `m.Height = (document.querySelector("#rows")?.clientHeight ?? 0);`) {
		t.Errorf("generated JS missing ClientHeight lowering\n\n%s", src)
	}
}

func TestGenerateTypeSwitchOnField(t *testing.T) {
	view, err := parser.ParseViews("box.gth", []byte(`package ui

view Box(model BoxModel) {
	<div></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseComponents("box.go", []byte(`package ui

type BoxModel struct{ Label string }

func (m BoxModel) Init() Cmd { return nil }
func (m BoxModel) Update(msg Msg) (BoxModel, Cmd) {
	inner := msg
	switch child := inner.(type) {
	case Picked:
		m.Label = child.Label
	}
	return m, nil
}
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.GenerateComponents([]*generator.Stateful{{
		Name: "Box", View: view.Templates[0], App: app.ComponentApp("BoxModel"), ModelJSON: `{"Label":""}`, ModelType: "BoxModel",
		MountFunc: "mountBox", CreateFunc: "createBox",
	}}, appjs.Options{FileName: "box.gth"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const child = inner;`,
		`switch (child.tag) {`,
		`m.Label = child.Label;`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateStatefulSlot(t *testing.T) {
	view, err := parser.ParseViews("app.gth", []byte(`package app

view App(model Model) {
	<main>
		<Panel title={ model.Title }>
			<p>{ model.Name }</p>
			<button onclick={ Clicked{} }>Click</button>
		</Panel>
	</main>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct {
	Title string
	Name  string
}
type Clicked struct{}
func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := appjs.Generate(view, app, appjs.Options{
		FileName:   "app.gth",
		RuntimeURL: "/_gothic/runtime.js",
		Stateful: []*generator.Stateful{{
			Name: "Panel", GoName: "Panel", ViewName: "__gothicPanelView", ModelType: "PanelModel", ModelJSON: `{}`,
			PropsType: "PanelProps", Props: []parser.Param{{Name: "Title", Type: "string"}},
			MountFunc: "mountPanel", CreateFunc: "createPanel", Slot: true, ModuleURL: "/_gothic/modules/panel.js",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`const slotc0 = root.querySelector('[data-gothic-component-id="c0"] [data-gothic-slot="default"]');`,
		`slotc0.appendChild((() => { const frag = document.createDocumentFragment();`,
		`document.createElement("p")`,
		`text(slotc0, "0", model.Name);`,
		`on(slotc0, "0", "click", (event) => dispatch({tag: "Clicked"}));`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated JS missing %q\n\n%s", want, src)
		}
	}
}

func TestRejectStatefulChildrenWithoutSlot(t *testing.T) {
	view, err := parser.ParseViews("app.gth", []byte(`package app

view App(model Model) {
	<main>
		<Panel>
			<p>Hello</p>
		</Panel>
	</main>
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
	_, err = appjs.Generate(view, app, appjs.Options{
		FileName:   "app.gth",
		RuntimeURL: "/_gothic/runtime.js",
		Stateful: []*generator.Stateful{{
			Name: "Panel", GoName: "Panel", ViewName: "__gothicPanelView", ModelType: "PanelModel", ModelJSON: `{}`,
			MountFunc: "mountPanel", CreateFunc: "createPanel", ModuleURL: "/_gothic/modules/panel.js",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "does not accept children") {
		t.Fatalf("error = %v", err)
	}
}
