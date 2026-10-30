package generator_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator"
	"github.com/metruzanca/gothic/goapp"
	"github.com/metruzanca/gothic/parser"
)

const demo = `package views

view PingPong() {
	state messages = []string{"ready"}

	<div class="app">
		<button onclick={ func() {
			messages = append(messages, "ping")
			pong, err := Ping()
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
	f, err := parser.Parse("pingpong.gth", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.Generate(f, generator.Options{
		FileName:   "pingpong.gth",
		ModuleBase: "pingpong",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`package views`,
		`func PingPong() gothic.Component`,
		`messages := []string{"ready"}`,
		`data-gothic-root=\"pingpong\"`,
		`data-gothic-state=\"pingpong\"`,
		`data-gothic-on=\"0\"`,
		`<!--g:each:0:open-->`,
		`<!--g:each:0:close-->`,
		`for _, m := range messages {`,
		`html.EscapeString(fmt.Sprint(m))`,
		`//go:embed pingpong_gothic.js`,
		`type PingPongProcedures interface`,
		`func RegisterPingPong(s *gothic.Server, impl PingPongProcedures)`,
		`/_gothic/modules/pingpong_gothic.js`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, "await") {
		t.Errorf("generated Go must not contain await\n\n%s", src)
	}
}

const appView = `package app

view App(model Model) {
	<div>
		for _, m := range model.Messages {
			<li>{ m }</li>
		}
	</div>
}
`

const appSource = `package app

import (
	"context"

	"github.com/metruzanca/gothic"
)

type Model struct{ Messages []string; Status string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	return m, nil
}
`

func TestGenerateAppView(t *testing.T) {
	f, err := parser.Parse("app.gth", []byte(appView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(appSource))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "app.gth",
		ModuleBase: "app",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`type Model struct`,
		`type Procedures interface`,
		`func (m Model) Update`,
		`func App(model Model) gothic.Component`,
		`func (m Model) View() gothic.Component`,
		`stateJSON, err := json.Marshal(model)`,
		`for _, m := range model.Messages`,
		`func RegisterProcedures(s *gothic.Server, impl Procedures)`,
		`func Ping() (string, error)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
	for _, path := range []string{`"context"`, `"github.com/metruzanca/gothic"`} {
		if got := strings.Count(src, path); got != 1 {
			t.Errorf("import %s appears %d times\n\n%s", path, got, src)
		}
	}
}

const procedureAppSource = `package app

import (
	"context"

	"github.com/metruzanca/gothic"
)

type Model struct{ Messages []string }
type Todo struct{ ID int64 }

type Procedures interface {
	gothic.RPCService
	AddTodo(ctx context.Context, text string) error
	ToggleTodo(ctx context.Context, id int64) error
	ListTodos(ctx context.Context) ([]Todo, error)
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`

func TestGenerateProcedureArgs(t *testing.T) {
	f, err := parser.Parse("app.gth", []byte(appView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(procedureAppSource))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "app.gth",
		ModuleBase: "app",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`var arg0 string`,
		`if err := gothic.UnmarshalArgs(body, &arg0); err != nil`,
		`if err := impl.AddTodo(ctx, arg0); err != nil`,
		`var arg0 int64`,
		`if err := gothic.UnmarshalArgs(body, &arg0); err != nil`,
		`func AddTodo(text string) error`,
		`func ToggleTodo(id int64) error`,
		`func ListTodos() ([]Todo, error)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
}

const componentView = `package app

view Status(text string) {
	<p class="status">{ text }</p>
}

view App(model Model) {
	<div>
		<Status text={ model.Status } />
	</div>
}
`

func TestGenerateAppExpandsComponents(t *testing.T) {
	f, err := parser.Parse("app.gth", []byte(componentView))
	if err != nil {
		t.Fatal(err)
	}
	components := parser.ComponentSet{}
	for _, t := range f.Templates {
		if t.Name != "App" {
			components[t.Name] = t
		}
	}
	expanded, err := parser.ExpandApp(f, components)
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(appSource))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(expanded, app, generator.Options{
		FileName:   "app.gth",
		ModuleBase: "app",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`func App(model Model) gothic.Component`,
		`write("<p")`,
		`write(" class=\"status")`,
		`<!--g:text:0:open-->`,
		`(model.Status)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, "func Status(") {
		t.Errorf("component should be expanded, not generated as a function\n\n%s", src)
	}
	if got := strings.Count(src, `data-gothic-root`); got != 1 {
		t.Errorf("root marker count = %d\n\n%s", got, src)
	}
}

func TestGenerateSingleRejectsAliasedGothic(t *testing.T) {
	src := `package app

import g "github.com/metruzanca/gothic"

type Model struct{}

func (m Model) Init() g.Cmd { return nil }

func (m Model) Update(msg g.Msg) (g.Model, g.Cmd) { return m, nil }
`
	f, err := parser.Parse("app.gth", []byte(appView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := generator.GenerateSingle(f, app, generator.Options{FileName: "app.gth", ModuleBase: "app", App: app}); err == nil {
		t.Fatal("expected an alias error")
	}
}

func TestGenerateRejectsMultipleRoots(t *testing.T) {
	src := `package views
view Bad() {
	<div></div>
	<div></div>
}
`
	f, err := parser.Parse("bad.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := generator.Generate(f, generator.Options{FileName: "bad.gth", ModuleBase: "bad"}); err == nil {
		t.Fatal("expected an error for multiple root elements")
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
		} else if model.Loading {
			<p>loading</p>
		} else {
			<p>empty</p>
		}
	</div>
}
`

func TestGenerateConditionals(t *testing.T) {
	f, err := parser.Parse("cond.gth", []byte(conditionalView))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.Generate(f, generator.Options{
		FileName:   "cond.gth",
		ModuleBase: "cond",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`<!--g:if:0:open-->`,
		`if len(model.Items) > 0 {`,
		`<!--g:if:0:then-->`,
		`<!--g:if:0:else-->`,
		`<!--g:if:0:close-->`,
		`<!--g:if:1:open-->`,
		`if model.Loading {`,
		`write("loading")`,
		`write("empty")`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateConditionalStatefulComponent(t *testing.T) {
	f, err := parser.Parse("cond.gth", []byte(`package app

view App(model Model) {
	<div>
		if model.Show {
			<Tabs />
		}
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Show bool }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "cond.gth",
		ModuleBase: "cond",
		App:        app,
		Stateful: []*generator.Stateful{{
			Name:        "Tabs",
			GoName:      "Tabs",
			External:    true,
			ImportAlias: "ui",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`<!--g:if:0:then-->`,
		`if model.Show {`,
		`ui.Tabs("c0", "").Render(ctx, w)`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateComponentSlot(t *testing.T) {
	f, err := parser.Parse("slot.gth", []byte(`package app

view Card(title string) {
	<section class="card">
		<h2>{ title }</h2>
		<slot />
	</section>
}

view App(model Model) {
	<div>
		<Card title="Hello">
			<p class="body">Slotted body</p>
		</Card>
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	components := parser.ComponentSet{}
	for _, tpl := range f.Templates {
		if tpl.Name != "App" {
			components[tpl.Name] = tpl
		}
	}
	expanded, err := parser.ExpandApp(f, components)
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{ Title string }

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(expanded, app, generator.Options{
		FileName:   "slot.gth",
		ModuleBase: "slot",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`write("<section")`,
		`write(" class=\"card")`,
		`write("Slotted body")`,
		`write("<p")`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateRouteLink(t *testing.T) {
	f, err := parser.Parse("route.gth", []byte(`package app

view App(model Model) {
	<div>
		<a href="/about" route>About</a>
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
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "route.gth",
		ModuleBase: "route",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	if !strings.Contains(src, `data-gothic-on=\"0\"`) {
		t.Errorf("route link missing click marker\n\n%s", src)
	}
	if strings.Contains(src, `href=\"/about\" route`) {
		t.Errorf("route attribute should not render\n\n%s", src)
	}
}

func TestRejectRouteOnNonAnchor(t *testing.T) {
	f, err := parser.Parse("route.gth", []byte(`package app

view App(model Model) {
	<div route>No</div>
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
	if _, err := generator.GenerateSingle(f, app, generator.Options{FileName: "route.gth", ModuleBase: "route", App: app}); err == nil || !strings.Contains(err.Error(), "route is only supported on <a>") {
		t.Fatalf("error = %v", err)
	}
}

func TestRouteErrorHasSourceContext(t *testing.T) {
	f, err := parser.Parse("route.gth", []byte(`package app

view App(model Model) {
	<div route>No</div>
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
	_, err = generator.GenerateSingle(f, app, generator.Options{FileName: "route.gth", ModuleBase: "route", App: app})
	if err == nil {
		t.Fatal("expected an error")
	}
	var pe *parser.Error
	if !errors.As(err, &pe) {
		t.Fatalf("error type = %T: %v", err, err)
	}
	if pe.File != "route.gth" || pe.Line != 4 {
		t.Fatalf("position = %s:%d:%d", pe.File, pe.Line, pe.Col)
	}
	if !strings.Contains(pe.Hint, "client-side navigation") {
		t.Fatalf("hint = %q", pe.Hint)
	}
	msg := err.Error()
	if !strings.Contains(msg, "route.gth:4:") || !strings.Contains(msg, "  4 | ") || !strings.Contains(msg, "^") {
		t.Fatalf("error lacks source context:\n%s", msg)
	}
}
