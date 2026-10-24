package generator_test

import (
	"strings"
	"testing"

	"github.com/metru/gothic/generator"
	"github.com/metru/gothic/parser"
)

const demo = `package views

templ PingPong() {
	state messages []string = ["ready"]

	<div class="app">
		<button onclick={
			messages.Push("ping")
			pong := await api.Ping()
			messages.Push(pong)
		}>Ping</button>
		<ul>
			@each messages as m {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

proc Ping() string
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
		`data-gothic-root=\"pingpong\"`,
		`data-gothic-state=\"pingpong\"`,
		`data-gothic-on=\"0\"`,
		`<!--g:each:0:open-->`,
		`<!--g:each:0:close-->`,
		`[]string{"ready"}`,
		`//go:embed pingpong_gothic.js`,
		`type PingPongProcedures interface`,
		`func RegisterPingPong(s *gothic.Server, impl PingPongProcedures)`,
		`/_gothic/modules/pingpong_gothic.js`,
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
