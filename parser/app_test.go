package parser_test

import (
	"errors"
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

func TestParseAppViewPositions(t *testing.T) {
	src := `package app

type Model struct{}

view App(model Model) {
	<div>
		<p>hi</p>
	</div>
}
`
	f, err := parser.ParseApp("app.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.View.Templates[0].Children[0].(*parser.Element)
	if file, line, col := root.Position(); file != "app.gth" || line != 6 || col != 2 {
		t.Fatalf("root position = %s:%d:%d", file, line, col)
	}
}

func TestParseAppErrorUsesOriginalLines(t *testing.T) {
	src := `package app

type Model struct{}

view App(model Model) {
	<div>
		if model.Show {
			<p>hi</p>
	</div>
}
`
	_, err := parser.ParseApp("app.gth", []byte(src))
	if err == nil {
		t.Fatal("expected an error")
	}
	var pe *parser.Error
	if !errors.As(err, &pe) {
		t.Fatalf("error type = %T", err)
	}
	if pe.Line != 9 {
		t.Fatalf("line = %d, want 9", pe.Line)
	}
	if !strings.Contains(pe.Hint, "line 7") {
		t.Fatalf("hint = %q", pe.Hint)
	}
}

func TestParseAppGoPreludeError(t *testing.T) {
	src := `package app

func broken( {

}

view App(model Model) {
	<div></div>
}
`
	_, err := parser.ParseApp("app.gth", []byte(src))
	if err == nil {
		t.Fatal("expected an error")
	}
	var pe *parser.Error
	if !errors.As(err, &pe) {
		t.Fatalf("error type = %T", err)
	}
	if pe.Line != 3 || !strings.Contains(pe.Source, "func broken") {
		t.Fatalf("position = %d source = %q", pe.Line, pe.Source)
	}
	if !strings.Contains(pe.Error(), "invalid Go prelude") || !strings.Contains(pe.Error(), "^") {
		t.Fatalf("error = %s", pe.Error())
	}
}
