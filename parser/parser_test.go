package parser_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/parser"
)

const demo = `package views

view PingPong() {
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

func TestParseDemo(t *testing.T) {
	f, err := parser.Parse("demo.gth", []byte(demo))
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
	sd := tpl.State[0]
	if sd.Name != "messages" {
		t.Fatalf("state name = %q", sd.Name)
	}
	if _, ok := sd.Init.(*ast.CompositeLit); !ok {
		t.Fatalf("state init = %T", sd.Init)
	}

	root, ok := tpl.Children[0].(*parser.Element)
	if !ok || root.Name != "div" {
		t.Fatalf("root = %#v", tpl.Children[0])
	}

	button := findElement(root, "button")
	if button == nil {
		t.Fatal("button not found")
	}
	ev, ok := button.Attributes[0].(*parser.EventAttr)
	if !ok || ev.Name != "onclick" {
		t.Fatalf("event attr = %#v", button.Attributes[0])
	}
	fn, ok := ev.Expr.(*ast.FuncLit)
	if !ok {
		t.Fatalf("event expression = %T", ev.Expr)
	}
	if len(fn.Body.List) != 4 {
		t.Fatalf("handler statements = %d", len(fn.Body.List))
	}
	if as, ok := fn.Body.List[0].(*ast.AssignStmt); !ok || as.Tok.String() != "=" {
		t.Fatalf("first statement = %T", fn.Body.List[0])
	}
	if _, ok := fn.Body.List[2].(*ast.IfStmt); !ok {
		t.Fatalf("third statement = %T", fn.Body.List[2])
	}

	ul := findElement(root, "ul")
	if ul == nil {
		t.Fatal("ul not found")
	}
	if len(ul.Children) != 1 {
		t.Fatalf("ul children = %d", len(ul.Children))
	}
	loop, ok := ul.Children[0].(*parser.For)
	if !ok {
		t.Fatalf("ul child = %T", ul.Children[0])
	}
	if loop.Key != "_" || loop.Value != "m" {
		t.Fatalf("for vars = %q %q", loop.Key, loop.Value)
	}
	if id, ok := loop.X.(*ast.Ident); !ok || id.Name != "messages" {
		t.Fatalf("for X = %T", loop.X)
	}
	li := findElementIn(loop.Children, "li")
	if li == nil {
		t.Fatal("li not found")
	}
	interp, ok := li.Children[0].(*parser.Interpolation)
	if !ok {
		t.Fatalf("li child = %T", li.Children[0])
	}
	if id, ok := interp.Expr.(*ast.Ident); !ok || id.Name != "m" {
		t.Fatalf("interpolation = %T", interp.Expr)
	}

	if len(f.Procedures) != 1 {
		t.Fatalf("procedures = %d", len(f.Procedures))
	}
	if p := f.Procedures[0]; p.Name != "Ping" || p.ResultType != "string" {
		t.Fatalf("procedure = %#v", p)
	}
}

func TestRejectsAwait(t *testing.T) {
	src := `package views
view Bad() {
	<button onclick={ func() {
		pong := await Ping()
		_ = pong
	} }>x</button>
}
`
	if _, err := parser.Parse("bad.gth", []byte(src)); err == nil {
		t.Fatal("expected a parse error for await")
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

const conditionalView = `package views

view App(model Model) {
	<div>
		if len(model.Items) > 0 {
			<ul>
				for _, item := range model.Items {
					<li>{ item }</li>
				}
			</ul>
		} else if model.Loading {
			<p>loading</p>
		} else {
			<p>empty</p>
		}
	</div>
}
`

func TestParseConditionals(t *testing.T) {
	f, err := parser.Parse("cond.gth", []byte(conditionalView))
	if err != nil {
		t.Fatal(err)
	}
	root, ok := f.Templates[0].Children[0].(*parser.Element)
	if !ok {
		t.Fatalf("root = %T", f.Templates[0].Children[0])
	}
	if len(root.Children) != 1 {
		t.Fatalf("root children = %d", len(root.Children))
	}
	cond, ok := root.Children[0].(*parser.If)
	if !ok {
		t.Fatalf("root child = %T", root.Children[0])
	}
	if _, ok := cond.Cond.(*ast.BinaryExpr); !ok {
		t.Fatalf("condition = %T", cond.Cond)
	}
	if len(cond.Then) != 1 {
		t.Fatalf("then nodes = %d", len(cond.Then))
	}
	if findElementIn(cond.Then, "ul") == nil {
		t.Fatal("then branch should contain ul")
	}
	if len(cond.Else) != 1 {
		t.Fatalf("else nodes = %d", len(cond.Else))
	}
	nested, ok := cond.Else[0].(*parser.If)
	if !ok {
		t.Fatalf("else branch = %T", cond.Else[0])
	}
	if len(nested.Else) != 1 {
		t.Fatalf("nested else nodes = %d", len(nested.Else))
	}
	if findElementIn(nested.Else, "p") == nil {
		t.Fatal("final else should contain p")
	}
}

func TestParseIfRequiresCondition(t *testing.T) {
	src := `package views

view App(model Model) {
	<div>
		if {
			<p>x</p>
		}
	</div>
}
`
	if _, err := parser.Parse("bad.gth", []byte(src)); err == nil {
		t.Fatal("expected an error for an empty if condition")
	}
}

func TestParseRouteAttribute(t *testing.T) {
	src := `package views

view App(model Model) {
	<div><a href="/about" route>About</a></div>
}
`
	f, err := parser.Parse("route.gth", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	root := f.Templates[0].Children[0].(*parser.Element)
	link := root.Children[0].(*parser.Element)
	found := false
	for _, attr := range link.Attributes {
		if _, ok := attr.(*parser.RouteAttr); ok {
			found = true
		}
	}
	if !found {
		t.Fatalf("route attribute not parsed: %#v", link.Attributes)
	}
}

func TestRejectRouteValue(t *testing.T) {
	src := `package views

view App(model Model) {
	<div><a href="/about" route="yes">About</a></div>
}
`
	if _, err := parser.Parse("route.gth", []byte(src)); err == nil || !strings.Contains(err.Error(), "route does not take a value") {
		t.Fatalf("error = %v", err)
	}
}
