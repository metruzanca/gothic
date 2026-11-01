package generator_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator"
	"github.com/metruzanca/gothic/goapp"
	"github.com/metruzanca/gothic/parser"
)

func testApp(t *testing.T) *goapp.App {
	t.Helper()
	app, err := goapp.ParseSource("app.gth", []byte(`package app

type Model struct {
	Messages []string
	Status   string
	Items    []string
	Loading  bool
	Done     bool
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	return app
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
	f, err := parser.ParseViews("app.gth", []byte(appView))
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
	f, err := parser.ParseViews("app.gth", []byte(appView))
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

view Status(props StatusProps) {
	<p class="status">{ props.Text }</p>
}

view App(model Model) {
	<div>
		<Status text={ model.Status } />
	</div>
}
`

func TestGenerateAppExpandsComponents(t *testing.T) {
	f, err := parser.ParseViews("app.gth", []byte(componentView))
	if err != nil {
		t.Fatal(err)
	}
	components := parser.ComponentSet{}
	for _, t := range f.Templates {
		if t.Name != "App" {
			components[t.Name] = t
		}
	}
	props := parser.PropsTypes{"StatusProps": {Name: "StatusProps", Fields: []parser.Param{{Name: "Text", Type: "string"}}}}
	expanded, err := parser.ExpandAppWithProps(f, components, props)
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
	f, err := parser.ParseViews("app.gth", []byte(appView))
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
	f, err := parser.ParseViews("bad.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	app := testApp(t)
	if _, err := generator.GenerateSingle(f, app, generator.Options{FileName: "bad.gth", ModuleBase: "bad", App: app}); err == nil {
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
	f, err := parser.ParseViews("cond.gth", []byte(conditionalView))
	if err != nil {
		t.Fatal(err)
	}
	app := testApp(t)
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "cond.gth",
		ModuleBase: "cond",
		App:        app,
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
	f, err := parser.ParseViews("cond.gth", []byte(`package app

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

func TestGenerateComponentChildren(t *testing.T) {
	f, err := parser.ParseViews("children.gth", []byte(`package app

view Card(props CardProps) {
	<section class="card">
		<h2>{ props.Title }</h2>
		{ props.Children }
	</section>
}

view App(model Model) {
	<div>
		<Card title="Hello">
			<p class="body">Child body</p>
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
	props := parser.PropsTypes{"CardProps": {Name: "CardProps", Fields: []parser.Param{
		{Name: "Children", Type: "gothic.Children"},
		{Name: "Title", Type: "string"},
	}}}
	expanded, err := parser.ExpandAppWithProps(f, components, props)
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

import "github.com/metruzanca/gothic"

type Model struct{ Title string }
type CardProps struct {
	gothic.Children
	Title string
}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(expanded, app, generator.Options{
		FileName:   "children.gth",
		ModuleBase: "children",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`write("<section")`,
		`write(" class=\"card")`,
		`write("Child body")`,
		`write("<p")`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
}

func TestGenerateRouteLink(t *testing.T) {
	f, err := parser.ParseViews("route.gth", []byte(`package app

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
	f, err := parser.ParseViews("route.gth", []byte(`package app

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
	f, err := parser.ParseViews("route.gth", []byte(`package app

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

func TestGenerateNestedStatefulComponentView(t *testing.T) {
	appFile, err := parser.ParseViews("app.gth", []byte(`package app

view App(model Model) {
	<main></main>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	parentFile, err := parser.ParseViews("parent.gth", []byte(`package app

view Parent(model ParentModel) {
	<div>
		<Badge />
	</div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(`package app

type Model struct{}
type ParentModel struct{}

func (m Model) Init() Cmd { return nil }
func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }
func (m ParentModel) Init() Cmd { return nil }
func (m ParentModel) Update(msg Msg) (ParentModel, Cmd) { return m, nil }
`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.GenerateSingle(appFile, app, generator.Options{
		FileName:   "app.gth",
		ModuleBase: "app",
		App:        app,
		Stateful: []*generator.Stateful{
			{Name: "Badge", GoName: "Badge", External: true, ImportAlias: "ui"},
			{Name: "Parent", GoName: "__gothicParentComponent", ViewName: "__gothicParentView", ModelType: "ParentModel", View: parentFile.Templates[0], ModelJSON: `{}`},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	if !strings.Contains(src, `ui.Badge("c0", "").Render(ctx, w)`) {
		t.Errorf("generated Go missing nested component call\n\n%s", src)
	}
}

func TestGenerateBooleanAttributes(t *testing.T) {
	src := `package views

view App(model Model) {
	<button disabled={ model.Done }>Save</button>
}
`
	f, err := parser.ParseViews("app.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	app := testApp(t)
	out, err := generator.GenerateSingle(f, app, generator.Options{
		FileName:   "app.gth",
		ModuleBase: "app",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	generated := string(out)
	for _, want := range []string{
		`if model.Done {`,
		` disabled`,
	} {
		if !strings.Contains(generated, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, generated)
		}
	}
	if strings.Contains(generated, `disabled=\"`) {
		t.Errorf("generated Go should not render disabled with a value\n\n%s", generated)
	}
}

func TestGenerateOmitsEmptyAriaAndData(t *testing.T) {
	f, err := parser.ParseViews("app.gth", []byte(`package app

view App(model Model) {
	<div aria-expanded={ model.Open } data-active={ model.Active } class={ model.Label }></div>
}
`))
	if err != nil {
		t.Fatal(err)
	}
	app := testApp(t)
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
		`attrVal0 := fmt.Sprint(model.Open)`,
		`if attrVal0 != "" {`,
		`attrVal1 := fmt.Sprint(model.Active)`,
		`if attrVal1 != "" {`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated Go missing %q\n\n%s", want, src)
		}
	}
	if strings.Contains(src, `if attrVal2 != ""`) {
		t.Errorf("class should not use the aria/data omit path\n\n%s", src)
	}
}
