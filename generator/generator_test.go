package generator_test

import (
	"strings"
	"testing"

	"github.com/metruzanca/gothic/generator"
	"github.com/metruzanca/gothic/goapp"
	"github.com/metruzanca/gothic/parser"
)

const demo = `package views

templ PingPong() {
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
	f, err := parser.Parse("pingpong.gothic", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.Generate(f, generator.Options{
		FileName:   "pingpong.gothic",
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

templ View(model Model) {
	<div>
		for _, m := range model.Messages {
			<li>{ m }</li>
		}
	</div>
}
`

const appSource = `package app

type Model struct{ Messages []string }

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
	f, err := parser.Parse("app.gothic", []byte(appView))
	if err != nil {
		t.Fatal(err)
	}
	app, err := goapp.ParseSource("app.go", []byte(appSource))
	if err != nil {
		t.Fatal(err)
	}
	out, err := generator.Generate(f, generator.Options{
		FileName:   "app.gothic",
		ModuleBase: "app",
		App:        app,
	})
	if err != nil {
		t.Fatal(err)
	}
	src := string(out)
	for _, want := range []string{
		`func View(model Model) gothic.Component`,
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
}

func TestGenerateRejectsMultipleRoots(t *testing.T) {
	src := `package views
templ Bad() {
	<div></div>
	<div></div>
}
`
	f, err := parser.Parse("bad.gothic", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := generator.Generate(f, generator.Options{FileName: "bad.gothic", ModuleBase: "bad"}); err == nil {
		t.Fatal("expected an error for multiple root elements")
	}
}
