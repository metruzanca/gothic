package parser

import (
	"fmt"
	"go/ast"
)

// ComponentSet maps component names to their declarations.
type ComponentSet map[string]*Template

// ExpandApp returns a copy of f with only the App view, with component calls
// expanded inline. Component declarations are resolved from components.
func ExpandApp(f *File, components ComponentSet) (*File, error) {
	var app *Template
	for _, t := range f.Templates {
		if t.Name != "App" {
			continue
		}
		if app != nil {
			return nil, fmt.Errorf("duplicate App view")
		}
		app = t
	}
	if app == nil {
		return nil, fmt.Errorf("App view not found")
	}
	expanded, err := expandTemplate(app, nil, nil, components)
	if err != nil {
		return nil, err
	}
	if err := rejectSlots(expanded.Children); err != nil {
		return nil, err
	}
	return &File{
		Package:    f.Package,
		Procedures: f.Procedures,
		Templates:  []*Template{expanded},
	}, nil
}

func expandTemplate(t *Template, env map[string]ast.Expr, stack []string, components ComponentSet) (*Template, error) {
	out := &Template{
		Name:      t.Name,
		Params:    t.Params,
		ParamList: append([]Param(nil), t.ParamList...),
		State:     append([]StateDecl(nil), t.State...),
	}
	for _, child := range t.Children {
		n, err := expandNode(child, env, stack, components)
		if err != nil {
			return nil, err
		}
		if n != nil {
			out.Children = append(out.Children, n)
		}
	}
	return out, nil
}

func expandNode(n Node, env map[string]ast.Expr, stack []string, components ComponentSet) (Node, error) {
	switch n := n.(type) {
	case *Text:
		return &Text{Value: n.Value}, nil
	case *Interpolation:
		return &Interpolation{Expr: substExpr(n.Expr, env)}, nil
	case *Element:
		el := &Element{Name: n.Name, Key: substExpr(n.Key, env)}
		for _, attr := range n.Attributes {
			switch a := attr.(type) {
			case *ConstantAttr:
				el.Attributes = append(el.Attributes, &ConstantAttr{Name: a.Name, Value: a.Value})
			case *EventAttr:
				el.Attributes = append(el.Attributes, &EventAttr{Name: a.Name, Expr: substExpr(a.Expr, env)})
			case *ExprAttr:
				el.Attributes = append(el.Attributes, &ExprAttr{Name: a.Name, Expr: substExpr(a.Expr, env)})
			case *ClassToggleAttr:
				el.Attributes = append(el.Attributes, &ClassToggleAttr{Name: a.Name, Expr: substExpr(a.Expr, env)})
			}
		}
		for _, child := range n.Children {
			expanded, err := expandNode(child, env, stack, components)
			if err != nil {
				return nil, err
			}
			if expanded != nil {
				el.Children = append(el.Children, expanded)
			}
		}
		return el, nil
	case *For:
		childEnv := cloneEnv(env)
		delete(childEnv, n.Key)
		delete(childEnv, n.Value)
		loop := &For{
			Key:   n.Key,
			Value: n.Value,
			X:     substExpr(n.X, env),
		}
		for _, child := range n.Children {
			expanded, err := expandNode(child, childEnv, stack, components)
			if err != nil {
				return nil, err
			}
			if expanded != nil {
				loop.Children = append(loop.Children, expanded)
			}
		}
		return loop, nil
	case *Slot:
		return &Slot{}, nil
	case *If:
		out := &If{Cond: substExpr(n.Cond, env)}
		for _, child := range n.Then {
			expanded, err := expandNode(child, env, stack, components)
			if err != nil {
				return nil, err
			}
			if expanded != nil {
				out.Then = append(out.Then, expanded)
			}
		}
		for _, child := range n.Else {
			expanded, err := expandNode(child, env, stack, components)
			if err != nil {
				return nil, err
			}
			if expanded != nil {
				out.Else = append(out.Else, expanded)
			}
		}
		return out, nil
	case *ComponentCall:
		return expandComponentCall(n, env, stack, components)
	default:
		return nil, fmt.Errorf("unsupported node type %T", n)
	}
}

func expandComponentCall(call *ComponentCall, env map[string]ast.Expr, stack []string, components ComponentSet) (Node, error) {
	component, ok := components[call.Name]
	if !ok {
		return nil, fmt.Errorf("unknown component %s", call.Name)
	}
	if component.Stateful {
		if len(call.Children) > 0 {
			return nil, fmt.Errorf("stateful component %s cannot accept children yet", call.Name)
		}
		props := make([]Prop, len(call.Props))
		for i, prop := range call.Props {
			props[i] = Prop{Name: prop.Name, Value: substExpr(prop.Value, env)}
		}
		return &ComponentCall{Name: call.Name, Key: substExpr(call.Key, env), Props: props}, nil
	}
	for _, name := range stack {
		if name == call.Name {
			return nil, fmt.Errorf("component cycle: %s", componentCycle(stack, call.Name))
		}
	}
	if len(component.Children) != 1 {
		return nil, fmt.Errorf("component %s must have exactly one root element", call.Name)
	}

	props := map[string]ast.Expr{}
	for _, prop := range call.Props {
		if _, exists := props[prop.Name]; exists {
			return nil, fmt.Errorf("component %s: duplicate prop %q", call.Name, prop.Name)
		}
		props[prop.Name] = substExpr(prop.Value, env)
	}
	childEnv := map[string]ast.Expr{}
	for _, param := range component.ParamList {
		value, ok := props[param.Name]
		if !ok {
			return nil, fmt.Errorf("component %s: missing prop %q", call.Name, param.Name)
		}
		childEnv[param.Name] = value
		delete(props, param.Name)
	}
	for name := range props {
		return nil, fmt.Errorf("component %s: unknown prop %q", call.Name, name)
	}

	children, err := expandNodes(call.Children, env, stack, components)
	if err != nil {
		return nil, err
	}
	if err := rejectSlots(children); err != nil {
		return nil, err
	}
	expanded, err := expandTemplate(component, childEnv, append(stack, call.Name), components)
	if err != nil {
		return nil, err
	}
	slots := 0
	nodes, err := insertSlots(expanded.Children, children, &slots)
	if err != nil {
		return nil, err
	}
	if slots > 1 {
		return nil, fmt.Errorf("component %s must declare one <slot />, found %d", call.Name, slots)
	}
	if slots == 0 && len(children) > 0 {
		return nil, fmt.Errorf("component %s does not accept children; add <slot /> to its view", call.Name)
	}
	expanded.Children = nodes
	if len(expanded.Children) != 1 {
		return nil, fmt.Errorf("component %s must expand to one root element", call.Name)
	}
	root, ok := expanded.Children[0].(*Element)
	if !ok {
		return nil, fmt.Errorf("component %s root must be an element", call.Name)
	}
	if call.Key != nil {
		root.Key = substExpr(call.Key, env)
	}
	return root, nil
}

func expandNodes(nodes []Node, env map[string]ast.Expr, stack []string, components ComponentSet) ([]Node, error) {
	out := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		expanded, err := expandNode(node, env, stack, components)
		if err != nil {
			return nil, err
		}
		if expanded != nil {
			out = append(out, expanded)
		}
	}
	return out, nil
}

func insertSlots(nodes []Node, children []Node, slots *int) ([]Node, error) {
	out := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		switch n := node.(type) {
		case *Slot:
			*slots = *slots + 1
			out = append(out, children...)
		case *Element:
			replaced, err := insertSlots(n.Children, children, slots)
			if err != nil {
				return nil, err
			}
			n.Children = replaced
			out = append(out, n)
		case *For:
			replaced, err := insertSlots(n.Children, children, slots)
			if err != nil {
				return nil, err
			}
			n.Children = replaced
			out = append(out, n)
		case *If:
			thenNodes, err := insertSlots(n.Then, children, slots)
			if err != nil {
				return nil, err
			}
			elseNodes, err := insertSlots(n.Else, children, slots)
			if err != nil {
				return nil, err
			}
			n.Then = thenNodes
			n.Else = elseNodes
			out = append(out, n)
		default:
			out = append(out, node)
		}
	}
	return out, nil
}

func rejectSlots(nodes []Node) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *Slot:
			return fmt.Errorf("slot can only be used inside a pure component view")
		case *Element:
			if err := rejectSlots(n.Children); err != nil {
				return err
			}
		case *For:
			if err := rejectSlots(n.Children); err != nil {
				return err
			}
		case *If:
			if err := rejectSlots(n.Then); err != nil {
				return err
			}
			if err := rejectSlots(n.Else); err != nil {
				return err
			}
		}
	}
	return nil
}

func componentCycle(stack []string, name string) string {
	out := ""
	for _, item := range stack {
		if out != "" {
			out += " -> "
		}
		out += item
	}
	if out != "" {
		out += " -> "
	}
	return out + name
}

func cloneEnv(env map[string]ast.Expr) map[string]ast.Expr {
	out := make(map[string]ast.Expr, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

func substExpr(e ast.Expr, env map[string]ast.Expr) ast.Expr {
	if e == nil {
		return nil
	}
	if len(env) == 0 {
		return e
	}
	switch e := e.(type) {
	case *ast.Ident:
		if value, ok := env[e.Name]; ok {
			return &ast.ParenExpr{X: value}
		}
		return &ast.Ident{Name: e.Name}
	case *ast.BasicLit:
		return &ast.BasicLit{Kind: e.Kind, Value: e.Value}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: substExpr(e.X, env), Sel: &ast.Ident{Name: e.Sel.Name}}
	case *ast.CallExpr:
		args := make([]ast.Expr, len(e.Args))
		for i, arg := range e.Args {
			args[i] = substExpr(arg, env)
		}
		return &ast.CallExpr{Fun: substExpr(e.Fun, env), Args: args, Ellipsis: e.Ellipsis}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{X: substExpr(e.X, env), Op: e.Op, Y: substExpr(e.Y, env)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: e.Op, X: substExpr(e.X, env)}
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: substExpr(e.X, env)}
	case *ast.StarExpr:
		return &ast.StarExpr{X: substExpr(e.X, env)}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: substExpr(e.X, env), Index: substExpr(e.Index, env)}
	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X:      substExpr(e.X, env),
			Low:    substExpr(e.Low, env),
			High:   substExpr(e.High, env),
			Max:    substExpr(e.Max, env),
			Slice3: e.Slice3,
		}
	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{X: substExpr(e.X, env), Type: e.Type}
	case *ast.CompositeLit:
		elts := make([]ast.Expr, len(e.Elts))
		for i, elt := range e.Elts {
			elts[i] = substExpr(elt, env)
		}
		return &ast.CompositeLit{Type: e.Type, Elts: elts, Incomplete: e.Incomplete}
	case *ast.KeyValueExpr:
		key := e.Key
		if _, ok := e.Key.(*ast.Ident); !ok {
			key = substExpr(e.Key, env)
		}
		return &ast.KeyValueExpr{Key: key, Value: substExpr(e.Value, env)}
	case *ast.FuncLit:
		return e
	default:
		return e
	}
}

// ExpandTemplate expands pure component calls in one template. Stateful calls
// are left intact.
func ExpandTemplate(t *Template, components ComponentSet) (*Template, error) {
	return expandTemplate(t, nil, nil, components)
}
