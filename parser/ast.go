// Package parser parses the gothic DSL and the Go fragments embedded in it.
package parser

import "go/ast"

// File is a parsed .gth file.
type File struct {
	Package    string
	Procedures []Procedure
	Templates  []*Template
}

// Procedure is a client-callable server procedure declared with rpc.
type Procedure struct {
	Name       string
	ResultType string
}

// Param is a named view parameter.
type Param struct {
	Name string
	Type string
}

// Template is a view declared with view.
type Template struct {
	Name      string
	Params    string
	ParamList []Param
	State     []StateDecl
	Children  []Node
	// Stateful is set by the compiler for component roots with their own model.
	Stateful bool
}

// StateDecl declares component state. Init is a real Go expression.
type StateDecl struct {
	Name string
	Init ast.Expr
}

// NodePosition is the source location of a view node.
type NodePosition struct {
	File string
	Line int
	Col  int
}

// Position returns the source location of the node.
func (p NodePosition) Position() (string, int, int) {
	return p.File, p.Line, p.Col
}

// Node is a template body node.
type Node interface {
	node()
	Position() (string, int, int)
}

// Element is an HTML element.
type Element struct {
	NodePosition
	Name       string
	Key        ast.Expr
	Attributes []Attribute
	Children   []Node
}

func (*Element) node() {}

// Text is literal text.
type Text struct {
	NodePosition
	Value string
}

func (*Text) node() {}

// Interpolation is an inline { expression } node.
type Interpolation struct {
	NodePosition
	Expr ast.Expr
}

func (*Interpolation) node() {}

// For renders a list. X is a real Go expression, normally a state identifier.
type For struct {
	NodePosition
	Key      string
	Value    string
	X        ast.Expr
	Children []Node
}

func (*For) node() {}

// If renders one branch based on a Go condition.
type If struct {
	NodePosition
	Cond ast.Expr
	Then []Node
	Else []Node
}

func (*If) node() {}

// Slot marks where a pure component's children are inserted.
type Slot struct {
	NodePosition
}

func (*Slot) node() {}

// ComponentCall invokes a component view with named props and optional children.
type ComponentCall struct {
	NodePosition
	Name     string
	Key      ast.Expr
	Props    []Prop
	Children []Node
}

func (*ComponentCall) node() {}

// Prop is a component prop.
type Prop struct {
	Name  string
	Value ast.Expr
}

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

// KeyAttr declares a list key for an element.
type KeyAttr struct {
	Expr ast.Expr
}

func (*KeyAttr) attr() {}

// EventAttr is an on<event>={ expression } attribute.
type EventAttr struct {
	Name string
	Expr ast.Expr
}

func (*EventAttr) attr() {}

// ExprAttr is a dynamic attribute such as value={ expr } or checked={ expr }.
type ExprAttr struct {
	Name string
	Expr ast.Expr
}

func (*ExprAttr) attr() {}

// ClassToggleAttr is a class:name={ expression } attribute.
type ClassToggleAttr struct {
	Name string
	Expr ast.Expr
}

func (*ClassToggleAttr) attr() {}

// RouteAttr marks an anchor for client-side navigation.
type RouteAttr struct{}

func (*RouteAttr) attr() {}
