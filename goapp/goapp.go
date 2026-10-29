// Package goapp extracts the Bubble Tea-style app declarations from a Go file.
package goapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// App is the parsed application model, messages, Update method, and commands.
type App struct {
	Package   string
	ModelType string
	Messages  map[string][]Field
	Init      *ast.FuncDecl
	Update    *ast.FuncDecl
	Commands  map[string]*ast.FuncDecl
	// ReferencedCommands are the command functions returned by Init or Update.
	ReferencedCommands map[string]bool
}

// Field is a struct field.
type Field struct {
	Name string
	Type string
}

// Parse reads a Go app file and extracts the Bubble Tea-style declarations.
func Parse(path string) (*App, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	app := &App{
		Package:            f.Name.Name,
		Messages:           map[string][]Field{},
		Commands:           map[string]*ast.FuncDecl{},
		ReferencedCommands: map[string]bool{},
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
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if ts.Name.Name == "Model" {
					app.ModelType = ts.Name.Name
					continue
				}
				app.Messages[ts.Name.Name] = fieldsOf(st)
			}
		case *ast.FuncDecl:
			if d.Recv != nil {
				if receiverType(d.Recv) != "Model" {
					continue
				}
				switch d.Name.Name {
				case "Init":
					app.Init = d
				case "Update":
					app.Update = d
				}
				continue
			}
			app.Commands[d.Name.Name] = d
		}
	}
	if app.ModelType == "" {
		return nil, fmt.Errorf("%s: Model type not found", path)
	}
	if app.Update == nil {
		return nil, fmt.Errorf("%s: Model.Update not found", path)
	}
	app.collectCommands()
	return app, nil
}

func (a *App) collectCommands() {
	for _, fn := range []*ast.FuncDecl{a.Init, a.Update} {
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
					if _, exists := a.Commands[id.Name]; exists {
						a.ReferencedCommands[id.Name] = true
					}
				}
			}
			return true
		})
	}
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
