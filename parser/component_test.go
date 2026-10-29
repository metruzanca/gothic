package parser_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/parser"
)

const componentDemo = `package views

view Status(text string) {
	<p class="status">{ text }</p>
}

view App(model Model) {
	<div>
		<Status text={ model.Status } />
	</div>
}
`

func TestParseComponentCall(t *testing.T) {
	f, err := parser.Parse("component.gothic", []byte(componentDemo))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Templates) != 2 {
		t.Fatalf("views = %d", len(f.Templates))
	}
	status := f.Templates[0]
	if len(status.ParamList) != 1 || status.ParamList[0].Name != "text" || status.ParamList[0].Type != "string" {
		t.Fatalf("params = %#v", status.ParamList)
	}
	root := f.Templates[1].Children[0].(*parser.Element)
	call, ok := root.Children[0].(*parser.ComponentCall)
	if !ok {
		t.Fatalf("child = %T", root.Children[0])
	}
	if call.Name != "Status" || len(call.Props) != 1 || call.Props[0].Name != "text" {
		t.Fatalf("call = %#v", call)
	}
	if got := exprText(t, call.Props[0].Value); got != "model.Status" {
		t.Fatalf("prop = %q", got)
	}
}

func TestParseComponentBooleanAndEmptyChildren(t *testing.T) {
	src := `package views
view Card(text string) {
	<p>{ text }</p>
}
view App(model Model) {
	<div><Card text="x" active></Card></div>
}
`
	f, err := parser.Parse("component.gothic", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.Templates[1].Children[0].(*parser.Element)
	call := root.Children[0].(*parser.ComponentCall)
	if len(call.Props) != 2 {
		t.Fatalf("props = %#v", call.Props)
	}
	if got := exprText(t, call.Props[1].Value); got != "true" {
		t.Fatalf("boolean prop = %q", got)
	}
}

func TestRejectComponentChildren(t *testing.T) {
	src := `package views
view Card(text string) {
	<p>{ text }</p>
}
view App(model Model) {
	<div><Card text="x"><span>hi</span></Card></div>
}
`
	if _, err := parser.Parse("component.gothic", []byte(src)); err == nil || !strings.Contains(err.Error(), "cannot have children") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandAppSubstitutesProps(t *testing.T) {
	f, err := parser.Parse("component.gothic", []byte(componentDemo))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	p := root.Children[0].(*parser.Element)
	if p.Name != "p" {
		t.Fatalf("component root = %q", p.Name)
	}
	interp := p.Children[0].(*parser.Interpolation)
	if got := exprText(t, interp.Expr); got != "(model.Status)" {
		t.Fatalf("interpolation = %q", got)
	}
}

func TestExpandAppNestedComponents(t *testing.T) {
	src := `package views
view Label(value string) {
	<span>{ value }</span>
}
view Card(item Item) {
	<section><Label value={ item.Name } /></section>
}
view App(model Model) {
	<div><Card item={ model.Item } /></div>
}
`
	f, err := parser.Parse("component.gothic", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	section := root.Children[0].(*parser.Element)
	span := section.Children[0].(*parser.Element)
	interp := span.Children[0].(*parser.Interpolation)
	if got := exprText(t, interp.Expr); !strings.Contains(got, "model.Item") || !strings.Contains(got, "Name") {
		t.Fatalf("interpolation = %q", got)
	}
}

func TestExpandAppComponentRootComponent(t *testing.T) {
	src := `package views
view Badge(text string) {
	<span>{ text }</span>
}
view Wrapper(text string) {
	<Badge text={ text } />
}
view App(model Model) {
	<div><Wrapper text="hello" /></div>
}
`
	f, err := parser.Parse("component.gothic", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	span := root.Children[0].(*parser.Element)
	if span.Name != "span" {
		t.Fatalf("root child = %q", span.Name)
	}
}

func TestExpandAppLoopVariableShadowsProp(t *testing.T) {
	src := `package views
view Item(item string) {
	<span>{ item }</span>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item item={ item } />
		}
	</div>
}
`
	f, err := parser.Parse("component.gothic", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	loop := root.Children[0].(*parser.For)
	span := loop.Children[0].(*parser.Element)
	interp := span.Children[0].(*parser.Interpolation)
	if got := exprText(t, interp.Expr); got != "(item)" {
		t.Fatalf("interpolation = %q", got)
	}
}

func TestExpandAppErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing prop",
			src: `package views
view Card(text string) { <p>{ text }</p> }
view App(model Model) { <div><Card /></div> }
`,
			want: "missing prop",
		},
		{
			name: "unknown prop",
			src: `package views
view Card(text string) { <p>{ text }</p> }
view App(model Model) { <div><Card text="x" other="y" /></div> }
`,
			want: "unknown prop",
		},
		{
			name: "duplicate prop",
			src: `package views
view Card(text string) { <p>{ text }</p> }
view App(model Model) { <div><Card text="x" text="y" /></div> }
`,
			want: "duplicate prop",
		},
		{
			name: "unknown component",
			src: `package views
view App(model Model) { <div><Missing /></div> }
`,
			want: "unknown component",
		},
		{
			name: "cycle",
			src: `package views
view A(model Model) { <div><B model={ model } /></div> }
view B(model Model) { <div><A model={ model } /></div> }
view App(model Model) { <div><A model={ model } /></div> }
`,
			want: "component cycle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := parser.Parse("component.gothic", []byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			_, err = parser.ExpandApp(f, componentsOf(f))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func componentsOf(f *parser.File) parser.ComponentSet {
	set := parser.ComponentSet{}
	for _, t := range f.Templates {
		if t.Name != "App" {
			set[t.Name] = t
		}
	}
	return set
}

func exprText(t *testing.T, e ast.Expr) string {
	t.Helper()
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), e); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
