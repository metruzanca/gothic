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
