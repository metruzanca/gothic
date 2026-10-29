package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
)

// Parse parses a .gothic file from src. The name is used in error messages.
func Parse(name string, src []byte) (*File, error) {
	p := &parser{name: name, src: src}
	return p.parseFile()
}

type parser struct {
	name string
	src  []byte
	pos  int
}

func (p *parser) parseFile() (*File, error) {
	f := &File{}
	for {
		p.skipSpace()
		if p.eof() {
			break
		}
		switch {
		case p.consumeKeyword("package"):
			p.skipSpace()
			f.Package = p.readIdent()
			if f.Package == "" {
				return nil, p.errf("expected package name")
			}
		case p.consumeKeyword("view"):
			t, err := p.parseTemplate()
			if err != nil {
				return nil, err
			}
			f.Templates = append(f.Templates, t)
		case p.consumeKeyword("rpc"):
			proc, err := p.parseProcedure()
			if err != nil {
				return nil, err
			}
			f.Procedures = append(f.Procedures, proc)
		default:
			return nil, p.errf("expected package, view, or rpc")
		}
	}
	if f.Package == "" {
		return nil, p.errf("missing package declaration")
	}
	if len(f.Templates) == 0 {
		return nil, p.errf("no view declaration found")
	}
	return f, nil
}

func (p *parser) parseTemplate() (*Template, error) {
	p.skipSpace()
	t := &Template{Name: p.readIdent()}
	if t.Name == "" {
		return nil, p.errf("expected template name")
	}
	p.skipSpace()
	if p.peek() == '(' {
		t.Params = p.readBalanced('(', ')')
	}
	params, err := parseParams(t.Params)
	if err != nil {
		return nil, p.errf("invalid parameters for %s: %v", t.Name, err)
	}
	t.ParamList = params
	p.skipSpace()
	if !p.consume("{") {
		return nil, p.errf("expected { after template name")
	}

	p.skipSpace()
	for p.startsKeyword("state") {
		sd, err := p.parseState()
		if err != nil {
			return nil, err
		}
		t.State = append(t.State, sd)
		p.skipSpace()
	}

	nodes, err := p.parseNodesUntil('}')
	if err != nil {
		return nil, err
	}
	if !p.consume("}") {
		return nil, p.errf("expected } to close template")
	}
	t.Children = nodes
	return t, nil
}

func (p *parser) parseState() (StateDecl, error) {
	var sd StateDecl
	if !p.consumeKeyword("state") {
		return sd, p.errf("expected state")
	}
	p.skipSpace()
	sd.Name = p.readIdent()
	if sd.Name == "" {
		return sd, p.errf("expected state name")
	}
	p.skipSpace()
	if !p.consume("=") {
		return sd, p.errf("expected = in state declaration")
	}
	p.skipSpace()
	start := p.pos
	for !p.eof() && p.peek() != '\n' && p.peek() != '\r' {
		p.pos++
	}
	text := strings.TrimSpace(string(p.src[start:p.pos]))
	if text == "" {
		return sd, p.errf("expected Go expression for state %s", sd.Name)
	}
	expr, err := goparser.ParseExpr(text)
	if err != nil {
		return sd, p.errf("invalid state expression for %s: %v", sd.Name, err)
	}
	sd.Init = expr
	return sd, nil
}

func (p *parser) parseProcedure() (Procedure, error) {
	var proc Procedure
	p.skipSpace()
	proc.Name = p.readIdent()
	if proc.Name == "" {
		return proc, p.errf("expected procedure name")
	}
	p.skipSpace()
	params := p.readBalanced('(', ')')
	params = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(params, "("), ")"))
	if params != "" {
		return proc, p.errf("iteration 1 only supports zero-argument procedures")
	}
	p.skipSpace()
	start := p.pos
	for !p.eof() && p.peek() != '\n' && p.peek() != '\r' && p.peek() != '}' {
		p.pos++
	}
	proc.ResultType = strings.TrimSpace(string(p.src[start:p.pos]))
	if proc.ResultType == "" {
		return proc, p.errf("expected procedure result type")
	}
	return proc, nil
}

func (p *parser) parseNodesUntil(stop byte) ([]Node, error) {
	var nodes []Node
	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file")
		}
		if p.peek() == stop {
			return nodes, nil
		}
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if n != nil {
			nodes = append(nodes, n)
		}
	}
}

func (p *parser) parseNode() (Node, error) {
	switch p.peek() {
	case '<':
		return p.parseElement()
	case '{':
		return p.parseInterpolation()
	default:
		if p.startsKeyword("for") {
			return p.parseFor()
		}
		return p.parseText(), nil
	}
}

func (p *parser) parseText() Node {
	start := p.pos
	for !p.eof() {
		switch p.peek() {
		case '<', '{':
			s := strings.TrimSpace(string(p.src[start:p.pos]))
			if s == "" {
				return nil
			}
			return &Text{Value: s}
		default:
			p.pos++
		}
	}
	s := strings.TrimSpace(string(p.src[start:p.pos]))
	if s == "" {
		return nil
	}
	return &Text{Value: s}
}

func (p *parser) parseInterpolation() (Node, error) {
	inner, err := p.readBalancedBraces()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(inner) == "" {
		return nil, p.errf("empty expression")
	}
	expr, err := goparser.ParseExpr(inner)
	if err != nil {
		return nil, p.errf("invalid expression %q: %v", strings.TrimSpace(inner), err)
	}
	return &Interpolation{Expr: expr}, nil
}

func (p *parser) parseFor() (Node, error) {
	if !p.consumeKeyword("for") {
		return nil, p.errf("expected for")
	}
	start := p.pos
	for !p.eof() && p.peek() != '{' {
		p.pos++
	}
	if p.eof() {
		return nil, p.errf("expected { after for header")
	}
	header := strings.TrimSpace(string(p.src[start:p.pos]))
	if header == "" {
		return nil, p.errf("empty for header")
	}

	rs, err := parseRangeHeader(header)
	if err != nil {
		return nil, p.errf("invalid for header %q: %v", header, err)
	}

	if !p.consume("{") {
		return nil, p.errf("expected { after for header")
	}
	children, err := p.parseNodesUntil('}')
	if err != nil {
		return nil, err
	}
	if !p.consume("}") {
		return nil, p.errf("expected } to close for")
	}

	key, value := "", ""
	if rs.Key != nil {
		if id, ok := rs.Key.(*ast.Ident); ok {
			key = id.Name
		}
	}
	if rs.Value != nil {
		if id, ok := rs.Value.(*ast.Ident); ok {
			value = id.Name
		}
	}
	return &For{Key: key, Value: value, X: rs.X, Children: children}, nil
}

func parseParams(raw string) ([]Param, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	src := "package p\nfunc _" + raw + " {}\n"
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "params.gothic", src, 0)
	if err != nil {
		return nil, err
	}
	if len(file.Decls) != 1 {
		return nil, fmt.Errorf("unexpected declaration shape")
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Type.Params == nil {
		return nil, fmt.Errorf("unexpected parameter shape")
	}
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return nil, fmt.Errorf("views cannot declare results")
	}
	var params []Param
	for _, field := range fn.Type.Params.List {
		if _, ok := field.Type.(*ast.Ellipsis); ok {
			return nil, fmt.Errorf("variadic parameters are not supported")
		}
		if len(field.Names) == 0 {
			return nil, fmt.Errorf("parameters must be named")
		}
		typ := exprString(field.Type)
		for _, name := range field.Names {
			params = append(params, Param{Name: name.Name, Type: typ})
		}
	}
	return params, nil
}

func exprString(e ast.Expr) string {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), e); err != nil {
		return ""
	}
	return b.String()
}

func parseRangeHeader(header string) (*ast.RangeStmt, error) {
	src := "package p\nfunc _() {\nfor " + header + " {\n}\n}\n"
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "for.gothic", src, 0)
	if err != nil {
		return nil, err
	}
	if len(file.Decls) != 1 {
		return nil, fmt.Errorf("unexpected declaration shape")
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Body == nil || len(fn.Body.List) != 1 {
		return nil, fmt.Errorf("unexpected for shape")
	}
	rs, ok := fn.Body.List[0].(*ast.RangeStmt)
	if !ok {
		return nil, fmt.Errorf("expected range loop")
	}
	return rs, nil
}

func (p *parser) parseElement() (Node, error) {
	if !p.consume("<") {
		return nil, p.errf("expected <")
	}
	if p.peek() == '/' {
		return nil, p.errf("unexpected closing tag")
	}
	name := p.readTagName()
	if name == "" {
		return nil, p.errf("expected element name")
	}
	if isComponentName(name) {
		return p.parseComponent(name)
	}
	el := &Element{Name: name}

	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file in element")
		}
		if p.consume("/>") {
			return el, nil
		}
		if p.consume(">") {
			break
		}
		attr, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		el.Attributes = append(el.Attributes, attr)
	}

	if isVoidElement(name) {
		return el, nil
	}

	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file in <%s>", name)
		}
		if p.startsWith("</") {
			break
		}
		child, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if child != nil {
			el.Children = append(el.Children, child)
		}
	}

	if !p.consume("</") {
		return nil, p.errf("expected closing tag for <%s>", name)
	}
	closing := p.readTagName()
	if closing != name {
		return nil, p.errf("expected </%s>, got </%s>", name, closing)
	}
	p.skipSpace()
	if !p.consume(">") {
		return nil, p.errf("expected > to close </%s>", name)
	}
	return el, nil
}

func (p *parser) parseComponent(name string) (Node, error) {
	call := &ComponentCall{Name: name}
	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file in <%s>", name)
		}
		if p.consume("/>") {
			return call, nil
		}
		if p.consume(">") {
			break
		}
		prop, err := p.parseProp()
		if err != nil {
			return nil, err
		}
		call.Props = append(call.Props, prop)
	}

	var children []Node
	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file in <%s>", name)
		}
		if p.startsWith("</") {
			break
		}
		child, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if child != nil {
			children = append(children, child)
		}
	}
	if len(children) > 0 {
		return nil, p.errf("component <%s> cannot have children yet", name)
	}
	if !p.consume("</") {
		return nil, p.errf("expected closing tag for <%s>", name)
	}
	closing := p.readTagName()
	if closing != name {
		return nil, p.errf("expected </%s>, got </%s>", name, closing)
	}
	p.skipSpace()
	if !p.consume(">") {
		return nil, p.errf("expected > to close </%s>", name)
	}
	return call, nil
}

func (p *parser) parseProp() (Prop, error) {
	var prop Prop
	prop.Name = p.readAttrName()
	if prop.Name == "" {
		return prop, p.errf("expected prop name")
	}
	if !token.IsIdentifier(prop.Name) {
		return prop, p.errf("invalid prop name %q", prop.Name)
	}
	p.skipSpace()
	if !p.consume("=") {
		prop.Value = ast.NewIdent("true")
		return prop, nil
	}
	p.skipSpace()
	if p.peek() == '{' {
		inner, err := p.readBalancedBraces()
		if err != nil {
			return prop, err
		}
		if strings.TrimSpace(inner) == "" {
			return prop, p.errf("empty prop expression")
		}
		expr, err := goparser.ParseExpr(inner)
		if err != nil {
			return prop, p.errf("invalid prop expression for %s: %v", prop.Name, err)
		}
		prop.Value = expr
		return prop, nil
	}
	if p.peek() != '"' {
		return prop, p.errf("expected quoted value or expression for prop %s", prop.Name)
	}
	v, err := p.readQuoted()
	if err != nil {
		return prop, err
	}
	prop.Value = &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(v)}
	return prop, nil
}

func (p *parser) parseAttribute() (Attribute, error) {
	name := p.readAttrName()
	if name == "" {
		return nil, p.errf("expected attribute name")
	}
	p.skipSpace()
	if !p.consume("=") {
		return &ConstantAttr{Name: name}, nil
	}
	p.skipSpace()
	if p.peek() == '{' {
		if !isEventAttribute(name) {
			return nil, p.errf("expression attributes are not supported in iteration 1")
		}
		inner, err := p.readBalancedBraces()
		if err != nil {
			return nil, err
		}
		expr, err := goparser.ParseExpr(inner)
		if err != nil {
			return nil, p.errf("invalid event expression for %s: %v", name, err)
		}
		return &EventAttr{Name: name, Expr: expr}, nil
	}
	if p.peek() != '"' {
		return nil, p.errf("expected quoted value for attribute %s", name)
	}
	v, err := p.readQuoted()
	if err != nil {
		return nil, err
	}
	return &ConstantAttr{Name: name, Value: v}, nil
}

// readBalancedBraces consumes the { at the current position and returns the
// text inside it. It understands Go strings, runes, raw strings, and comments.
func (p *parser) readBalancedBraces() (string, error) {
	if !p.consume("{") {
		return "", p.errf("expected {")
	}
	start := p.pos
	depth := 1
	for !p.eof() {
		c := p.peek()
		switch c {
		case '"', '\'':
			if err := p.skipQuoted(c); err != nil {
				return "", err
			}
		case '`':
			if err := p.skipRawString(); err != nil {
				return "", err
			}
		case '/':
			if p.pos+1 < len(p.src) {
				switch p.src[p.pos+1] {
				case '/':
					for !p.eof() && p.peek() != '\n' {
						p.pos++
					}
					continue
				case '*':
					p.pos += 2
					for !p.eof() {
						if p.peek() == '*' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
							p.pos += 2
							break
						}
						p.pos++
					}
					continue
				}
			}
			p.pos++
		case '{':
			depth++
			p.pos++
		case '}':
			depth--
			p.pos++
			if depth == 0 {
				return string(p.src[start : p.pos-1]), nil
			}
		default:
			p.pos++
		}
	}
	return "", p.errf("unterminated {")
}

func (p *parser) skipQuoted(quote byte) error {
	p.pos++
	for !p.eof() {
		c := p.peek()
		if c == '\\' {
			p.pos += 2
			continue
		}
		p.pos++
		if c == quote {
			return nil
		}
	}
	return p.errf("unterminated quoted literal")
}

func (p *parser) skipRawString() error {
	p.pos++
	for !p.eof() {
		c := p.peek()
		p.pos++
		if c == '`' {
			return nil
		}
	}
	return p.errf("unterminated raw string")
}

func (p *parser) readQuoted() (string, error) {
	if !p.consume("\"") {
		return "", p.errf("expected quoted string")
	}
	var b strings.Builder
	for !p.eof() {
		c := p.peek()
		p.pos++
		switch c {
		case '"':
			return b.String(), nil
		case '\\':
			if p.eof() {
				return "", p.errf("unterminated escape")
			}
			e := p.peek()
			p.pos++
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", p.errf("unterminated string")
}

func (p *parser) readBalanced(open, close byte) string {
	if p.eof() || p.peek() != open {
		return ""
	}
	start := p.pos
	depth := 0
	for !p.eof() {
		c := p.peek()
		p.pos++
		switch c {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return string(p.src[start:p.pos])
			}
		}
	}
	return string(p.src[start:p.pos])
}

func (p *parser) readIdent() string {
	start := p.pos
	for !p.eof() {
		c := p.peek()
		if isIdentStart(c) || (p.pos > start && isIdentPart(c)) {
			p.pos++
			continue
		}
		break
	}
	return string(p.src[start:p.pos])
}

func (p *parser) readTagName() string {
	start := p.pos
	for !p.eof() {
		c := p.peek()
		if isIdentPart(c) || c == '-' {
			p.pos++
			continue
		}
		break
	}
	return string(p.src[start:p.pos])
}

func (p *parser) readAttrName() string {
	start := p.pos
	for !p.eof() {
		c := p.peek()
		if isIdentPart(c) || c == '-' || c == ':' || c == '@' {
			p.pos++
			continue
		}
		break
	}
	return string(p.src[start:p.pos])
}

func (p *parser) skipSpace() {
	for !p.eof() {
		c := p.peek()
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
			continue
		}
		if c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
			for !p.eof() && p.peek() != '\n' {
				p.pos++
			}
			continue
		}
		if c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '*' {
			p.pos += 2
			for !p.eof() {
				if p.peek() == '*' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
					p.pos += 2
					break
				}
				p.pos++
			}
			continue
		}
		break
	}
}

func (p *parser) consume(s string) bool {
	if p.startsWith(s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *parser) startsWith(s string) bool {
	if p.pos+len(s) > len(p.src) {
		return false
	}
	return string(p.src[p.pos:p.pos+len(s)]) == s
}

func (p *parser) startsKeyword(s string) bool {
	if !p.startsWith(s) {
		return false
	}
	next := p.pos + len(s)
	if next >= len(p.src) {
		return true
	}
	return !isIdentPart(p.src[next])
}

func (p *parser) consumeKeyword(s string) bool {
	if p.startsKeyword(s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) eof() bool {
	return p.pos >= len(p.src)
}

func (p *parser) errf(format string, args ...any) error {
	line, col := 1, 1
	for i := 0; i < p.pos && i < len(p.src); i++ {
		if p.src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return fmt.Errorf("%s:%d:%d: %s", p.name, line, col, fmt.Sprintf(format, args...))
}

func isComponentName(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isEventAttribute(name string) bool {
	return strings.HasPrefix(name, "on") && len(name) > 2
}

func isVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}
