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
	f, err := parser.Parse("component.gth", []byte(componentDemo))
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
	f, err := parser.Parse("component.gth", []byte(src))
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

func TestParseComponentChildren(t *testing.T) {
	src := `package views
view Card(text string) {
	<p>{ text }<slot /></p>
}
view App(model Model) {
	<div><Card text="x"><span>hi</span></Card></div>
}
`
	f, err := parser.Parse("component.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.Templates[1].Children[0].(*parser.Element)
	call := root.Children[0].(*parser.ComponentCall)
	if len(call.Children) != 1 {
		t.Fatalf("children = %d", len(call.Children))
	}
	if span, ok := call.Children[0].(*parser.Element); !ok || span.Name != "span" {
		t.Fatalf("child = %#v", call.Children[0])
	}
	p := f.Templates[0].Children[0].(*parser.Element)
	if _, ok := p.Children[1].(*parser.Slot); !ok {
		t.Fatalf("slot = %T", p.Children[1])
	}
}

func TestExpandAppSubstitutesProps(t *testing.T) {
	f, err := parser.Parse("component.gth", []byte(componentDemo))
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
	f, err := parser.Parse("component.gth", []byte(src))
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
	f, err := parser.Parse("component.gth", []byte(src))
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
	f, err := parser.Parse("component.gth", []byte(src))
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
			f, err := parser.Parse("component.gth", []byte(tt.src))
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

func TestParseListKey(t *testing.T) {
	src := `package views
view Item(item Model) {
	<li>{ item.Name }</li>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item key={ item.ID } item={ item } />
		}
	</div>
}
`
	f, err := parser.Parse("component.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.Templates[1].Children[0].(*parser.Element)
	loop := root.Children[0].(*parser.For)
	call := loop.Children[0].(*parser.ComponentCall)
	if got := exprText(t, call.Key); got != "item.ID" {
		t.Fatalf("key = %q", got)
	}
	if len(call.Props) != 1 || call.Props[0].Name != "item" {
		t.Fatalf("props = %#v", call.Props)
	}
}

func TestExpandComponentKey(t *testing.T) {
	src := `package views
view Item(item Model) {
	<li>{ item.Name }</li>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item key={ item.ID } item={ item } />
		}
	</div>
}
`
	f, err := parser.Parse("component.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	loop := root.Children[0].(*parser.For)
	li := loop.Children[0].(*parser.Element)
	if got := exprText(t, li.Key); got != "item.ID" {
		t.Fatalf("expanded key = %q", got)
	}
}

func TestParseDynamicAttributes(t *testing.T) {
	src := `package views
view App(model Model) {
	<div>
		<input value={ model.Draft } checked={ model.Done } focus={ model.Editing } />
		<li class:completed={ model.Done } class:hidden={ model.Hidden }></li>
	</div>
}
`
	f, err := parser.Parse("attrs.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.Templates[0].Children[0].(*parser.Element)
	input := root.Children[0].(*parser.Element)
	if len(input.Attributes) != 3 {
		t.Fatalf("input attrs = %#v", input.Attributes)
	}
	for i, name := range []string{"value", "checked", "focus"} {
		attr, ok := input.Attributes[i].(*parser.ExprAttr)
		if !ok || attr.Name != name {
			t.Fatalf("attr %d = %#v", i, input.Attributes[i])
		}
	}
	li := root.Children[1].(*parser.Element)
	if len(li.Attributes) != 2 {
		t.Fatalf("li attrs = %#v", li.Attributes)
	}
	for i, name := range []string{"completed", "hidden"} {
		attr, ok := li.Attributes[i].(*parser.ClassToggleAttr)
		if !ok || attr.Name != name {
			t.Fatalf("class attr %d = %#v", i, li.Attributes[i])
		}
	}
}

func TestExpandConditionalInsideComponent(t *testing.T) {
	src := `package views

view Badge(text string, active bool) {
	<span>
		if active {
			<b class="on">{ text }</b>
		} else {
			<i class="off">{ text }</i>
		}
	</span>
}

view App(model Model) {
	<div>
		<Badge text={ model.Text } active={ model.Active } />
	</div>
}
`
	f, err := parser.Parse("cond.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	components := parser.ComponentSet{}
	for _, tpl := range f.Templates {
		components[tpl.Name] = tpl
	}
	app, err := parser.ExpandApp(f, components)
	if err != nil {
		t.Fatal(err)
	}
	root := app.Templates[0].Children[0].(*parser.Element)
	wrapper := root.Children[0].(*parser.Element)
	cond, ok := wrapper.Children[0].(*parser.If)
	if !ok {
		t.Fatalf("expanded child = %T", wrapper.Children[0])
	}
	if got := exprText(t, cond.Cond); got != "(model.Active)" {
		t.Fatalf("condition = %q", got)
	}
	if len(cond.Then) != 1 || len(cond.Else) != 1 {
		t.Fatalf("branches = %d %d", len(cond.Then), len(cond.Else))
	}
}

func TestExpandComponentChildren(t *testing.T) {
	src := `package views

view Card(text string) {
	<section>
		<h2>{ text }</h2>
		<slot />
	</section>
}

view App(model Model) {
	<div>
		<Card text="Hello">
			<p class="body">Child</p>
		</Card>
	</div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	section := root.Children[0].(*parser.Element)
	if section.Name != "section" || len(section.Children) != 2 {
		t.Fatalf("section = %#v", section)
	}
	p, ok := section.Children[1].(*parser.Element)
	if !ok || p.Name != "p" {
		t.Fatalf("child = %#v", section.Children[1])
	}
}

func TestExpandNestedComponentChildren(t *testing.T) {
	src := `package views

view Box() {
	<div class="box"><slot /></div>
}

view App(model Model) {
	<div>
		<Box>
			<Box><p>nested</p></Box>
		</Box>
	</div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := parser.ExpandApp(f, componentsOf(f))
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	outer := root.Children[0].(*parser.Element)
	inner := outer.Children[0].(*parser.Element)
	if inner.Name != "div" {
		t.Fatalf("inner = %#v", inner)
	}
	if p, ok := inner.Children[0].(*parser.Element); !ok || p.Name != "p" {
		t.Fatalf("nested child = %#v", inner.Children[0])
	}
}

func TestExpandRejectsChildrenWithoutSlot(t *testing.T) {
	src := `package views

view Card(text string) {
	<p>{ text }</p>
}

view App(model Model) {
	<div><Card text="x"><span>hi</span></Card></div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ExpandApp(f, componentsOf(f)); err == nil || !strings.Contains(err.Error(), "does not accept children") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandRejectsMultipleSlots(t *testing.T) {
	src := `package views

view Box() {
	<div><slot /><slot /></div>
}

view App(model Model) {
	<div><Box><p>x</p></Box></div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ExpandApp(f, componentsOf(f)); err == nil || !strings.Contains(err.Error(), "must declare one <slot />") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandRejectsStatefulChildren(t *testing.T) {
	src := `package views

view Tabs() {
	<div></div>
}

view App(model Model) {
	<div><Tabs><p>x</p></Tabs></div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	components := componentsOf(f)
	components["Tabs"].Stateful = true
	if _, err := parser.ExpandApp(f, components); err == nil || !strings.Contains(err.Error(), "cannot accept children yet") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandRejectsAppSlot(t *testing.T) {
	src := `package views

view App(model Model) {
	<div><slot /></div>
}
`
	f, err := parser.Parse("slot.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ExpandApp(f, map[string]*parser.Template{}); err == nil || !strings.Contains(err.Error(), "slot can only be used") {
		t.Fatalf("error = %v", err)
	}
}
