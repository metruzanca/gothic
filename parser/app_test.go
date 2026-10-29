package parser_test

import (
	"strings"
	"testing"

	"github.com/metruzanca/gothic/parser"
)

func TestParseApp(t *testing.T) {
	src := `package app

type Model struct{ Messages []string }

view App(model Model) {
	<div></div>
}
`
	f, err := parser.ParseApp("app.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if f.Package != "app" {
		t.Fatalf("package = %q", f.Package)
	}
	if !strings.Contains(string(f.GoSource), "type Model struct") {
		t.Fatalf("prelude = %q", f.GoSource)
	}
	if len(f.View.Templates) != 1 || f.View.Templates[0].Name != "App" {
		t.Fatalf("view templates = %#v", f.View.Templates)
	}
}

func TestParseAppRequiresTempl(t *testing.T) {
	if _, err := parser.ParseApp("app.gth", []byte("package app\n")); err == nil {
		t.Fatal("expected an error when no view declaration exists")
	}
}
