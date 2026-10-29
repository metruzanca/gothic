package generator_test

import (
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
