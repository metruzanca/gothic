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

type StatusProps struct {
	Text string
}

view Status(props StatusProps) {
	<p class="status">{ props.Text }</p>
}

view App(model Model) {
	<div>
		<Status text={ model.Status } />
	</div>
}
`

func statusProps() parser.PropsTypes {
	return parser.PropsTypes{"StatusProps": {Name: "StatusProps", Fields: []parser.Param{
		{Name: "Text", Type: "string"},
	}}}
}

func TestParseComponentCall(t *testing.T) {
	f := parseViews(t, "component.gth", componentDemo)
	if len(f.Templates) != 2 {
		t.Fatalf("views = %d", len(f.Templates))
	}
	status := f.Templates[0]
	if len(status.ParamList) != 1 || status.ParamList[0].Name != "props" || status.ParamList[0].Type != "StatusProps" {
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
type CardProps struct {
	Text   string
	Active bool
}
view Card(props CardProps) {
	<p>{ props.Text }</p>
}
view App(model Model) {
	<div><Card text="x" active></Card></div>
}
`
	f := parseViews(t, "component.gth", src)
	root := f.Templates[1].Children[0].(*parser.Element)
	call := root.Children[0].(*parser.ComponentCall)
	if len(call.Props) != 2 {
		t.Fatalf("props = %#v", call.Props)
	}
	if got := exprText(t, call.Props[1].Value); got != "true" {
		t.Fatalf("boolean prop = %q", got)
	}
}

func TestExpandAppSubstitutesProps(t *testing.T) {
	f := parseViews(t, "component.gth", componentDemo)
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), statusProps())
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	p := root.Children[0].(*parser.Element)
	if p.Name != "p" {
		t.Fatalf("component root = %q", p.Name)
	}
	interp := p.Children[0].(*parser.Interpolation)
	got := exprText(t, interp.Expr)
	if !strings.Contains(got, "StatusProps{Text: model.Status}") || !strings.HasSuffix(got, ".Text") {
		t.Fatalf("interpolation = %q", got)
	}
}

func TestExpandAppNestedComponents(t *testing.T) {
	src := `package views
type LabelProps struct{ Value string }
type CardProps struct{ Item Item }
view Label(props LabelProps) {
	<span>{ props.Value }</span>
}
view Card(props CardProps) {
	<section><Label value={ props.Item.Name } /></section>
}
view App(model Model) {
	<div><Card item={ model.Item } /></div>
}
`
	f := parseViews(t, "component.gth", src)
	props := parser.PropsTypes{
		"LabelProps": {Name: "LabelProps", Fields: []parser.Param{{Name: "Value", Type: "string"}}},
		"CardProps":  {Name: "CardProps", Fields: []parser.Param{{Name: "Item", Type: "Item"}}},
	}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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
type BadgeProps struct{ Text string }
type WrapperProps struct{ Text string }
view Badge(props BadgeProps) {
	<span>{ props.Text }</span>
}
view Wrapper(props WrapperProps) {
	<Badge text={ props.Text } />
}
view App(model Model) {
	<div><Wrapper text="hello" /></div>
}
`
	f := parseViews(t, "component.gth", src)
	props := parser.PropsTypes{
		"BadgeProps":   {Name: "BadgeProps", Fields: []parser.Param{{Name: "Text", Type: "string"}}},
		"WrapperProps": {Name: "WrapperProps", Fields: []parser.Param{{Name: "Text", Type: "string"}}},
	}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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
type ItemProps struct{ Value string }
view Item(props ItemProps) {
	<span>{ props.Value }</span>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item value={ item } />
		}
	</div>
}
`
	f := parseViews(t, "component.gth", src)
	props := parser.PropsTypes{"ItemProps": {Name: "ItemProps", Fields: []parser.Param{{Name: "Value", Type: "string"}}}}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	loop := root.Children[0].(*parser.For)
	span := loop.Children[0].(*parser.Element)
	interp := span.Children[0].(*parser.Interpolation)
	if got := exprText(t, interp.Expr); !strings.Contains(got, "Value: item") {
		t.Fatalf("interpolation = %q", got)
	}
}

func TestExpandAppErrors(t *testing.T) {
	props := parser.PropsTypes{"CardProps": {Name: "CardProps", Fields: []parser.Param{{Name: "Text", Type: "string"}}}}
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing prop",
			src: `package views
type CardProps struct{ Text string }
view Card(props CardProps) { <p>{ props.Text }</p> }
view App(model Model) { <div><Card /></div> }
`,
			want: "missing prop",
		},
		{
			name: "unknown prop",
			src: `package views
type CardProps struct{ Text string }
view Card(props CardProps) { <p>{ props.Text }</p> }
view App(model Model) { <div><Card text="x" other="y" /></div> }
`,
			want: "unknown prop",
		},
		{
			name: "duplicate prop",
			src: `package views
type CardProps struct{ Text string }
view Card(props CardProps) { <p>{ props.Text }</p> }
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
			name: "individual props",
			src: `package views
view Card(text string) { <p>{ text }</p> }
view App(model Model) { <div><Card text="x" /></div> }
`,
			want: "must take one props struct",
		},
		{
			name: "cycle",
			src: `package views
type AProps struct{ Value string }
type BProps struct{ Value string }
view A(props AProps) { <div><B value={ props.Value } /></div> }
view B(props BProps) { <div><A value={ props.Value } /></div> }
view App(model Model) { <div><A value="x" /></div> }
`,
			want: "component cycle",
		},
	}
	cycleProps := parser.PropsTypes{
		"CardProps": {Name: "CardProps", Fields: []parser.Param{{Name: "Text", Type: "string"}}},
		"AProps":    {Name: "AProps", Fields: []parser.Param{{Name: "Value", Type: "string"}}},
		"BProps":    {Name: "BProps", Fields: []parser.Param{{Name: "Value", Type: "string"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parseViews(t, "component.gth", tt.src)
			if tt.name == "cycle" {
				props = cycleProps
			}
			_, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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
type ItemProps struct{ Item Model }
view Item(props ItemProps) {
	<li>{ props.Item.Name }</li>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item key={ item.ID } item={ item } />
		}
	</div>
}
`
	f := parseViews(t, "component.gth", src)
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
type ItemProps struct{ Item Model }
view Item(props ItemProps) {
	<li>{ props.Item.Name }</li>
}
view App(model Model) {
	<div>
		for _, item := range model.Items {
			<Item key={ item.ID } item={ item } />
		}
	</div>
}
`
	f := parseViews(t, "component.gth", src)
	props := parser.PropsTypes{"ItemProps": {Name: "ItemProps", Fields: []parser.Param{{Name: "Item", Type: "Model"}}}}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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
	f := parseViews(t, "attrs.gth", src)
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

type BadgeProps struct {
	Text   string
	Active bool
}
view Badge(props BadgeProps) {
	<span>
		if props.Active {
			<b class="on">{ props.Text }</b>
		} else {
			<i class="off">{ props.Text }</i>
		}
	</span>
}

view App(model Model) {
	<div>
		<Badge text={ model.Text } active={ model.Active } />
	</div>
}
`
	f := parseViews(t, "cond.gth", src)
	props := parser.PropsTypes{"BadgeProps": {Name: "BadgeProps", Fields: []parser.Param{
		{Name: "Text", Type: "string"},
		{Name: "Active", Type: "bool"},
	}}}
	app, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
	if err != nil {
		t.Fatal(err)
	}
	root := app.Templates[0].Children[0].(*parser.Element)
	wrapper := root.Children[0].(*parser.Element)
	cond, ok := wrapper.Children[0].(*parser.If)
	if !ok {
		t.Fatalf("expanded child = %T", wrapper.Children[0])
	}
	if got := exprText(t, cond.Cond); !strings.Contains(got, "model.Active") {
		t.Fatalf("condition = %q", got)
	}
	if len(cond.Then) != 1 || len(cond.Else) != 1 {
		t.Fatalf("branches = %d %d", len(cond.Then), len(cond.Else))
	}
}

func TestExpandComponentChildren(t *testing.T) {
	src := `package views

view Card(props CardProps) {
	<section>
		<h2>{ props.Text }</h2>
		{ props.Children }
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
	f := parseViews(t, "children.gth", src)
	props := parser.PropsTypes{"CardProps": {Name: "CardProps", Fields: []parser.Param{
		{Name: "Children", Type: "gothic.Children"},
		{Name: "Text", Type: "string"},
	}}}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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

view Box(props BoxProps) {
	<div class="box">{ props.Children }</div>
}

view App(model Model) {
	<div>
		<Box>
			<Box><p>nested</p></Box>
		</Box>
	</div>
}
`
	f := parseViews(t, "children.gth", src)
	props := parser.PropsTypes{"BoxProps": {Name: "BoxProps", Fields: []parser.Param{
		{Name: "Children", Type: "gothic.Children"},
	}}}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), props)
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

func TestExpandRejectsChildrenWithoutChildrenProp(t *testing.T) {
	src := `package views

view Card(props CardProps) {
	<p>{ props.Text }</p>
}

view App(model Model) {
	<div><Card text="x"><span>hi</span></Card></div>
}
`
	f := parseViews(t, "children.gth", src)
	props := parser.PropsTypes{"CardProps": {Name: "CardProps", Fields: []parser.Param{
		{Name: "Text", Type: "string"},
	}}}
	if _, err := parser.ExpandAppWithProps(f, componentsOf(f), props); err == nil || !strings.Contains(err.Error(), "does not accept children") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandRejectsMultipleChildren(t *testing.T) {
	src := `package views

view Box(props BoxProps) {
	<div>{ props.Children }{ props.Children }</div>
}

view App(model Model) {
	<div><Box><p>x</p></Box></div>
}
`
	f := parseViews(t, "children.gth", src)
	props := parser.PropsTypes{"BoxProps": {Name: "BoxProps", Fields: []parser.Param{
		{Name: "Children", Type: "gothic.Children"},
	}}}
	if _, err := parser.ExpandAppWithProps(f, componentsOf(f), props); err == nil || !strings.Contains(err.Error(), "must render { props.Children } once") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandRejectsStatefulChildrenWithoutChildrenProp(t *testing.T) {
	src := `package views

view Tabs() {
	<div></div>
}

view App(model Model) {
	<div><Tabs><p>x</p></Tabs></div>
}
`
	f := parseViews(t, "children.gth", src)
	components := componentsOf(f)
	components["Tabs"].Stateful = true
	if _, err := parser.ExpandApp(f, components); err == nil || !strings.Contains(err.Error(), "stateful component Tabs cannot accept children") {
		t.Fatalf("error = %v", err)
	}
}

func TestExpandPurePropsStruct(t *testing.T) {
	src := `package views

type CardProps struct {
	Title string
	Body  *string
}

view Card(props CardProps) {
	<section>
		<h2>{ props.Title }</h2>
		if props.Body != nil {
			<p>{ *props.Body }</p>
		}
	</section>
}

view App(model Model) {
	<div>
		<Card title="Hello" />
	</div>
}
`
	appFile, err := parser.ParseApp("props.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	f := appFile.View
	propsTypes := parser.PropsTypes{
		"CardProps": {
			Name: "CardProps",
			Fields: []parser.Param{
				{Name: "Title", Type: "string"},
				{Name: "Body", Type: "*string"},
			},
		},
	}
	expanded, err := parser.ExpandAppWithProps(f, componentsOf(f), propsTypes)
	if err != nil {
		t.Fatal(err)
	}
	root := expanded.Templates[0].Children[0].(*parser.Element)
	section := root.Children[0].(*parser.Element)
	h2 := section.Children[0].(*parser.Element)
	interp := h2.Children[0].(*parser.Interpolation)
	if got := exprText(t, interp.Expr); !strings.Contains(got, `CardProps{Title: "Hello", Body: nil}`) {
		t.Fatalf("title expression = %q", got)
	}
	cond, ok := section.Children[1].(*parser.If)
	if !ok {
		t.Fatalf("conditional = %T", section.Children[1])
	}
	if got := exprText(t, cond.Cond); !strings.Contains(got, "Body: nil") {
		t.Fatalf("condition = %q", got)
	}
}

func TestExpandPurePropsStructRequiredField(t *testing.T) {
	src := `package views

type CardProps struct{ Title string }

view Card(props CardProps) {
	<section>{ props.Title }</section>
}

view App(model Model) {
	<div><Card /></div>
}
`
	appFile, err := parser.ParseApp("props.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	f := appFile.View
	propsTypes := parser.PropsTypes{"CardProps": {Name: "CardProps", Fields: []parser.Param{{Name: "Title", Type: "string"}}}}
	if _, err := parser.ExpandAppWithProps(f, componentsOf(f), propsTypes); err == nil || !strings.Contains(err.Error(), `missing prop "Title"`) {
		t.Fatalf("error = %v", err)
	}
}

func parseViews(t *testing.T, name, src string) *parser.File {
	t.Helper()
	appFile, err := parser.ParseApp(name, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return appFile.View
}
