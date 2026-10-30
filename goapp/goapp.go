// Package goapp extracts the Bubble Tea-style app declarations from a Go file.
package goapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// App is the parsed application model, messages, Update method, commands, and
// procedure interface.
type App struct {
	Package   string
	ModelType string
	Messages  map[string][]Field
	Init      *ast.FuncDecl
	Update    *ast.FuncDecl
	Commands  map[string]*ast.FuncDecl
	// ReferencedCommands are the command functions returned by Init or Update.
	ReferencedCommands map[string]bool
	// ProceduresType is the interface that embeds gothic.RPCService.
	ProceduresType string
	Procedures     []Procedure
	// Components are non-app models that declare Init and Update.
	Components map[string]*Component
	// Source is the Go prelude, used to generate the app Go file.
	Source []byte
	// Fset and Name locate declarations in Source for error messages.
	Fset *token.FileSet
	Name string
}

// Component is a stateful component model and its update loop.
type Component struct {
	ModelType          string
	Init               *ast.FuncDecl
	Update             *ast.FuncDecl
	Commands           map[string]*ast.FuncDecl
	ReferencedCommands map[string]bool
}

// Field is a struct field.
type Field struct {
	Name string
	Type string
}

// Procedure is a method declared on the procedure interface.
type Procedure struct {
	Name       string
	Params     []Param
	ResultType string
}

// Param is a procedure argument, excluding the leading context.Context.
type Param struct {
	Name string
	Type string
}

// Parse reads a Go app file and extracts the Bubble Tea-style declarations.
func Parse(path string) (*App, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSource(path, src)
}

// ParseSource extracts the declarations from a Go app source.
func ParseSource(name string, src []byte) (*App, error) {
	return parseSource(name, src, true)
}

// ParseComponents extracts component declarations from a library source.
func ParseComponents(name string, src []byte) (*App, error) {
	return parseSource(name, src, false)
}

func parseSource(name string, src []byte, requireApp bool) (*App, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	app := &App{
		Package:            f.Name.Name,
		Messages:           map[string][]Field{},
		Commands:           map[string]*ast.FuncDecl{},
		ReferencedCommands: map[string]bool{},
		Components:         map[string]*Component{},
		Source:             src,
		Fset:               fset,
		Name:               name,
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				switch t := ts.Type.(type) {
				case *ast.StructType:
					if ts.Name.Name == "Model" {
						app.ModelType = ts.Name.Name
						continue
					}
					app.Messages[ts.Name.Name] = fieldsOf(t)
				case *ast.InterfaceType:
					if !embedsRPCService(t) {
						continue
					}
					procedures, err := proceduresOf(t)
					if err != nil {
						return nil, fmt.Errorf("%s: %w", name, err)
					}
					app.ProceduresType = ts.Name.Name
					app.Procedures = procedures
				}
			}
		case *ast.FuncDecl:
			if d.Recv != nil {
				recv := receiverType(d.Recv)
				if recv == "Model" {
					switch d.Name.Name {
					case "Init":
						app.Init = d
					case "Update":
						app.Update = d
					}
					continue
				}
				if recv == "" {
					continue
				}
				comp := app.Components[recv]
				if comp == nil {
					comp = &Component{
						ModelType:          recv,
						Commands:           map[string]*ast.FuncDecl{},
						ReferencedCommands: map[string]bool{},
					}
					app.Components[recv] = comp
				}
				switch d.Name.Name {
				case "Init":
					comp.Init = d
				case "Update":
					comp.Update = d
				}
				continue
			}
			app.Commands[d.Name.Name] = d
		}
	}
	if requireApp {
		if app.ModelType == "" {
			return nil, fmt.Errorf("%s: Model type not found; declare type Model struct{...}", name)
		}
		if app.Init == nil {
			return nil, fmt.Errorf("%s: Model.Init not found; add func (m Model) Init() gothic.Cmd", name)
		}
		if app.Update == nil {
			return nil, fmt.Errorf("%s: Model.Update not found; add func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd)", name)
		}
	}
	app.collectCommands()
	for _, comp := range app.Components {
		if comp.Init == nil || comp.Update == nil {
			continue
		}
		collectCommandsFor(comp.Init, comp.Update, app.Commands, comp.ReferencedCommands)
	}
	return app, nil
}

// ComponentApp returns an App view of one stateful component model.
func (a *App) ComponentApp(modelType string) *App {
	comp := a.Components[modelType]
	if comp == nil {
		return nil
	}
	return &App{
		Package:            a.Package,
		ModelType:          comp.ModelType,
		Messages:           a.Messages,
		Init:               comp.Init,
		Update:             comp.Update,
		Commands:           a.Commands,
		ReferencedCommands: comp.ReferencedCommands,
		Source:             a.Source,
		Fset:               a.Fset,
		Name:               a.Name,
	}
}

func (a *App) collectCommands() {
	collectCommandsFor(a.Init, a.Update, a.Commands, a.ReferencedCommands)
}

func collectCommandsFor(init, update *ast.FuncDecl, commands map[string]*ast.FuncDecl, referenced map[string]bool) {
	for _, fn := range []*ast.FuncDecl{init, update} {
		if fn == nil || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			ret, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, res := range ret.Results {
				if id, ok := res.(*ast.Ident); ok {
					if _, exists := commands[id.Name]; exists {
						referenced[id.Name] = true
					}
				}
			}
			return true
		})
	}
}

func embedsRPCService(it *ast.InterfaceType) bool {
	if it.Methods == nil {
		return false
	}
	for _, m := range it.Methods.List {
		if len(m.Names) != 0 {
			continue
		}
		sel, ok := m.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		id, ok := sel.X.(*ast.Ident)
		if ok && id.Name == "gothic" && sel.Sel.Name == "RPCService" {
			return true
		}
	}
	return false
}

func proceduresOf(it *ast.InterfaceType) ([]Procedure, error) {
	if it.Methods == nil {
		return nil, nil
	}
	var procedures []Procedure
	for _, m := range it.Methods.List {
		if len(m.Names) != 1 {
			continue
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			continue
		}
		p := Procedure{Name: m.Names[0].Name}
		params, err := procedureParams(ft)
		if err != nil {
			return nil, fmt.Errorf("procedure %s: %w", p.Name, err)
		}
		p.Params = params
		if err := procedureResults(ft, &p); err != nil {
			return nil, fmt.Errorf("procedure %s: %w", p.Name, err)
		}
		procedures = append(procedures, p)
	}
	return procedures, nil
}

func procedureParams(ft *ast.FuncType) ([]Param, error) {
	if ft.Params == nil || len(ft.Params.List) == 0 {
		return nil, fmt.Errorf("must take context.Context as its first parameter")
	}
	first := ft.Params.List[0]
	if len(first.Names) > 1 || !isContextType(first.Type) {
		return nil, fmt.Errorf("must take context.Context as its first parameter")
	}
	var params []Param
	index := 0
	for _, field := range ft.Params.List[1:] {
		if _, ok := field.Type.(*ast.Ellipsis); ok {
			return nil, fmt.Errorf("variadic parameters are not supported")
		}
		typ := exprString(field.Type)
		if len(field.Names) == 0 {
			params = append(params, Param{Name: fmt.Sprintf("arg%d", index), Type: typ})
			index++
			continue
		}
		for _, name := range field.Names {
			params = append(params, Param{Name: name.Name, Type: typ})
			index++
		}
	}
	return params, nil
}

func procedureResults(ft *ast.FuncType, p *Procedure) error {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return fmt.Errorf("must return (T, error) or error")
	}
	results := ft.Results.List
	switch len(results) {
	case 1:
		if !isErrorType(results[0].Type) {
			return fmt.Errorf("must return (T, error) or error")
		}
		p.ResultType = ""
	case 2:
		if !isErrorType(results[1].Type) {
			return fmt.Errorf("second result must be error")
		}
		p.ResultType = exprString(results[0].Type)
	default:
		return fmt.Errorf("must return (T, error) or error")
	}
	return nil
}

func isContextType(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "context" && sel.Sel.Name == "Context"
}

func isErrorType(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "error"
}

func fieldsOf(st *ast.StructType) []Field {
	if st.Fields == nil {
		return nil
	}
	var fields []Field
	for _, f := range st.Fields.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			fields = append(fields, Field{Type: typ})
			continue
		}
		for _, name := range f.Names {
			fields = append(fields, Field{Name: name.Name, Type: typ})
		}
	}
	return fields
}

func receiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	}
	return "any"
}
