package goapp_test

import (
	"testing"

	"github.com/metruzanca/gothic/goapp"
)

const appSrc = `package app

type Model struct{ Messages []string }

type PingClicked struct{}
type PongReceived struct{ Text string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) {
	switch msg.(type) {
	case PingClicked:
		return m, callPing
	case PongReceived:
	}
	return m, nil
}

func callPing() Msg { return PongReceived{Text: "pong"} }
`

func TestParseProcedureParams(t *testing.T) {
	src := `package app

type Model struct{}
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
	app, err := goapp.ParseSource("app.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(app.Procedures) != 3 {
		t.Fatalf("procedures = %#v", app.Procedures)
	}
	add := app.Procedures[0]
	if add.Name != "AddTodo" || len(add.Params) != 1 || add.Params[0].Name != "text" || add.Params[0].Type != "string" || add.ResultType != "" {
		t.Fatalf("AddTodo = %#v", add)
	}
	toggle := app.Procedures[1]
	if toggle.Name != "ToggleTodo" || len(toggle.Params) != 1 || toggle.Params[0].Type != "int64" {
		t.Fatalf("ToggleTodo = %#v", toggle)
	}
	list := app.Procedures[2]
	if list.Name != "ListTodos" || len(list.Params) != 0 || list.ResultType != "[]Todo" {
		t.Fatalf("ListTodos = %#v", list)
	}
}

func TestParseSource(t *testing.T) {
	app, err := goapp.ParseSource("app.go", []byte(appSrc))
	if err != nil {
		t.Fatal(err)
	}
	if app.ModelType != "Model" {
		t.Fatalf("model type = %q", app.ModelType)
	}
	if _, ok := app.Messages["PingClicked"]; !ok {
		t.Fatal("PingClicked message not found")
	}
	if _, ok := app.Messages["PongReceived"]; !ok {
		t.Fatal("PongReceived message not found")
	}
	if app.Init == nil || app.Update == nil {
		t.Fatal("Init or Update not found")
	}
	if !app.ReferencedCommands["callPing"] {
		t.Fatal("callPing was not collected as a referenced command")
	}
	if app.ProceduresType != "Procedures" {
		t.Fatalf("procedures type = %q", app.ProceduresType)
	}
	if len(app.Procedures) != 1 || app.Procedures[0].Name != "Ping" || app.Procedures[0].ResultType != "string" {
		t.Fatalf("procedures = %#v", app.Procedures)
	}
}
