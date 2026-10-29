// Package parser parses the gothic DSL and the Go fragments embedded in it.
package parser

import "go/ast"

// File is a parsed .gothic file.
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
}

// StateDecl declares component state. Init is a real Go expression.
type StateDecl struct {
	Name string
	Init ast.Expr
}

// Node is a template body node.
type Node interface {
	node()
}

// Element is an HTML element.
type Element struct {
	Name       string
	Key        ast.Expr
	Attributes []Attribute
	Children   []Node
}

func (*Element) node() {}

// Text is literal text.
type Text struct {
	Value string
}

func (*Text) node() {}

// Interpolation is an inline { expression } node.
type Interpolation struct {
	Expr ast.Expr
}

func (*Interpolation) node() {}

// For renders a list. X is a real Go expression, normally a state identifier.
type For struct {
	Key      string
	Value    string
	X        ast.Expr
	Children []Node
}

func (*For) node() {}

// ComponentCall invokes a component view with named props.
type ComponentCall struct {
	Name  string
	Key   ast.Expr
	Props []Prop
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
