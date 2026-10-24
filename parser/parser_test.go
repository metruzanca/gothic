package parser_test

import (
	"testing"

	"github.com/metruzanca/gothic/parser"
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

func TestParseDemo(t *testing.T) {
	f, err := parser.Parse("demo.gothic", []byte(demo))
	if err != nil {
		t.Fatal(err)
	}
	if f.Package != "views" {
		t.Fatalf("package = %q", f.Package)
	}
	if len(f.Templates) != 1 {
		t.Fatalf("templates = %d", len(f.Templates))
	}
	tpl := f.Templates[0]
	if tpl.Name != "PingPong" {
		t.Fatalf("template name = %q", tpl.Name)
	}
	if len(tpl.State) != 1 {
		t.Fatalf("state count = %d", len(tpl.State))
	}
	if got := tpl.State[0]; got.Name != "messages" || got.Type != "[]string" || len(got.Init) != 1 || got.Init[0] != "ready" {
		t.Fatalf("state = %#v", got)
	}
	if len(tpl.Children) != 1 {
		t.Fatalf("root nodes = %d", len(tpl.Children))
	}
	root, ok := tpl.Children[0].(*parser.Element)
	if !ok || root.Name != "div" {
		t.Fatalf("root = %#v", tpl.Children[0])
	}
	if len(root.Attributes) != 1 {
		t.Fatalf("root attrs = %#v", root.Attributes)
	}
	if a, ok := root.Attributes[0].(*parser.ConstantAttr); !ok || a.Name != "class" || a.Value != "app" {
		t.Fatalf("class attr = %#v", root.Attributes[0])
	}

	button := findElement(root, "button")
	if button == nil {
		t.Fatal("button not found")
	}
	if len(button.Attributes) != 1 {
		t.Fatalf("button attrs = %#v", button.Attributes)
	}
	ev, ok := button.Attributes[0].(*parser.EventAttr)
	if !ok || ev.Name != "onclick" {
		t.Fatalf("button attr = %#v", button.Attributes[0])
	}
	if len(ev.Body) != 3 {
		t.Fatalf("handler statements = %d", len(ev.Body))
	}
	call, ok := ev.Body[0].(*parser.HandlerCall)
	if !ok || call.Target != "messages.Push" || len(call.Args) != 1 || !call.Args[0].IsString || call.Args[0].Value != "ping" {
		t.Fatalf("first statement = %#v", ev.Body[0])
	}
	await, ok := ev.Body[1].(*parser.HandlerAwaitAssign)
	if !ok || await.Name != "pong" || await.Call.Target != "api.Ping" {
		t.Fatalf("second statement = %#v", ev.Body[1])
	}

	ul := findElement(root, "ul")
	if ul == nil {
		t.Fatal("ul not found")
	}
	var each *parser.Each
	for _, child := range ul.Children {
		if e, ok := child.(*parser.Each); ok {
			each = e
		}
	}
	if each == nil || each.List != "messages" || each.Var != "m" {
		t.Fatalf("each = %#v", each)
	}
	li := findElementIn(each.Children, "li")
	if li == nil {
		t.Fatal("li not found")
	}
	expr, ok := li.Children[0].(*parser.Expr)
	if !ok || expr.Value != "m" {
		t.Fatalf("li expr = %#v", li.Children[0])
	}

	if len(f.Procedures) != 1 {
		t.Fatalf("procedures = %d", len(f.Procedures))
	}
	if p := f.Procedures[0]; p.Name != "Ping" || p.ResultType != "string" {
		t.Fatalf("procedure = %#v", p)
	}
}

func findElement(root *parser.Element, name string) *parser.Element {
	for _, child := range root.Children {
		if el, ok := child.(*parser.Element); ok && el.Name == name {
			return el
		}
	}
	return nil
}

func findElementIn(nodes []parser.Node, name string) *parser.Element {
	for _, child := range nodes {
		if el, ok := child.(*parser.Element); ok && el.Name == name {
			return el
		}
	}
	return nil
}
