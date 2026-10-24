// Package parser parses the iteration 1 subset of the gothic DSL.
//
// The subset is intentionally small. It covers package declarations,
// components with a single string-slice state, constant elements and
// attributes, text, variable expressions, reactive each blocks, click
// handlers with the imperative mini block, and zero-argument procedures.
package parser

// File is a parsed .gothic file.
type File struct {
	Package    string
	Procedures []Procedure
	Templates  []*Template
}

// Procedure is a client-callable server procedure declared with proc.
type Procedure struct {
	Name       string
	ResultType string
}

// Template is a component declared with templ.
type Template struct {
	Name     string
	Params   string
	State    []StateDecl
	Children []Node
}

// StateDecl declares component state. Iteration 1 supports string slices only.
type StateDecl struct {
	Name string
	Type string
	Init []string
}

// Node is a template body node.
type Node interface {
	node()
}

// Element is an HTML element.
type Element struct {
	Name       string
	Attributes []Attribute
	Children   []Node
}

func (*Element) node() {}

// Text is literal text.
type Text struct {
	Value string
}

func (*Text) node() {}

// Expr is an inline { expression } node. Iteration 1 supports identifiers.
type Expr struct {
	Value string
}

func (*Expr) node() {}

// Each renders a reactive list over a state value.
type Each struct {
	List     string
	Var      string
	Children []Node
}

func (*Each) node() {}

// Attribute is an element attribute.
type Attribute interface {
	attr()
}

// ConstantAttr is a name or name="value" attribute.
type ConstantAttr struct {
	Name  string
	Value string
}

func (*ConstantAttr) attr() {}

// EventAttr is an on<event>={ ... } handler.
type EventAttr struct {
	Name string
	Body []HandlerStmt
}

func (*EventAttr) attr() {}

// HandlerStmt is a statement in an event handler block.
type HandlerStmt interface {
	stmt()
}

// HandlerCall is a dotted call such as messages.Push("ping").
type HandlerCall struct {
	Target string
	Args   []Arg
}

func (*HandlerCall) stmt() {}

// HandlerAwaitAssign is name := await dotted.Call(args).
type HandlerAwaitAssign struct {
	Name string
	Call HandlerCall
}

func (*HandlerAwaitAssign) stmt() {}

// Arg is a call argument. Iteration 1 supports string literals and identifiers.
type Arg struct {
	IsString bool
	Value    string
}
