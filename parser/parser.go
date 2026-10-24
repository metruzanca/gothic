package parser

import (
	"fmt"
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
		case p.consumeKeyword("templ"):
			t, err := p.parseTemplate()
			if err != nil {
				return nil, err
			}
			f.Templates = append(f.Templates, t)
		case p.consumeKeyword("proc"):
			proc, err := p.parseProcedure()
			if err != nil {
				return nil, err
			}
			f.Procedures = append(f.Procedures, proc)
		default:
			return nil, p.errf("expected package, templ, or proc")
		}
	}
	if f.Package == "" {
		return nil, p.errf("missing package declaration")
	}
	if len(f.Templates) == 0 {
		return nil, p.errf("no templ declaration found")
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
	start := p.pos
	for !p.eof() && p.peek() != '=' {
		p.pos++
	}
	if p.eof() {
		return sd, p.errf("expected = in state declaration")
	}
	sd.Type = strings.TrimSpace(string(p.src[start:p.pos]))
	if !p.consume("=") {
		return sd, p.errf("expected = in state declaration")
	}
	p.skipSpace()
	if !p.consume("[") {
		return sd, p.errf("expected [ for state initializer")
	}
	for {
		p.skipSpace()
		if p.consume("]") {
			break
		}
		if p.peek() != '"' {
			return sd, p.errf("expected string in state initializer")
		}
		v, err := p.readQuoted()
		if err != nil {
			return sd, err
		}
		sd.Init = append(sd.Init, v)
		p.skipSpace()
		if p.consume(",") {
			continue
		}
		if p.consume("]") {
			break
		}
		return sd, p.errf("expected , or ] in state initializer")
	}
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
		return p.parseExpr()
	case '@':
		if p.startsKeyword("@each") {
			return p.parseEach()
		}
		return nil, p.errf("unknown @ directive")
	default:
		return p.parseText(), nil
	}
}

func (p *parser) parseText() Node {
	start := p.pos
	for !p.eof() {
		switch p.peek() {
		case '<', '{', '@':
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

func (p *parser) parseExpr() (Node, error) {
	if !p.consume("{") {
		return nil, p.errf("expected {")
	}
	start := p.pos
	for !p.eof() && p.peek() != '}' {
		p.pos++
	}
	if p.eof() {
		return nil, p.errf("expected } to close expression")
	}
	v := strings.TrimSpace(string(p.src[start:p.pos]))
	p.pos++
	if v == "" {
		return nil, p.errf("empty expression")
	}
	return &Expr{Value: v}, nil
}

func (p *parser) parseEach() (Node, error) {
	if !p.consume("@each") {
		return nil, p.errf("expected @each")
	}
	p.skipSpace()
	list := p.readIdent()
	if list == "" {
		return nil, p.errf("expected list identifier after @each")
	}
	p.skipSpace()
	if !p.consumeKeyword("as") {
		return nil, p.errf("expected as after @each list")
	}
	p.skipSpace()
	v := p.readIdent()
	if v == "" {
		return nil, p.errf("expected item identifier after as")
	}
	p.skipSpace()
	if !p.consume("{") {
		return nil, p.errf("expected { after @each header")
	}
	children, err := p.parseNodesUntil('}')
	if err != nil {
		return nil, err
	}
	if !p.consume("}") {
		return nil, p.errf("expected } to close @each")
	}
	return &Each{List: list, Var: v, Children: children}, nil
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
	if p.peek() == '{' && isEventAttribute(name) {
		body, err := p.parseHandlerBlock()
		if err != nil {
			return nil, err
		}
		return &EventAttr{Name: name, Body: body}, nil
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

func (p *parser) parseHandlerBlock() ([]HandlerStmt, error) {
	if !p.consume("{") {
		return nil, p.errf("expected { for event handler")
	}
	var stmts []HandlerStmt
	for {
		p.skipSpace()
		if p.eof() {
			return nil, p.errf("unexpected end of file in event handler")
		}
		if p.consume("}") {
			return stmts, nil
		}
		stmt, err := p.parseHandlerStmt()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
}

func (p *parser) parseHandlerStmt() (HandlerStmt, error) {
	first := p.readIdent()
	if first == "" {
		return nil, p.errf("expected statement in event handler")
	}
	p.skipSpace()
	if p.consume(":=") {
		p.skipSpace()
		p.consumeKeyword("await")
		p.skipSpace()
		target := p.readDottedIdent()
		if target == "" {
			return nil, p.errf("expected call target after :=")
		}
		args, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		return &HandlerAwaitAssign{
			Name: first,
			Call: HandlerCall{Target: target, Args: args},
		}, nil
	}

	target := first
	for p.peek() == '.' {
		p.pos++
		next := p.readIdent()
		if next == "" {
			return nil, p.errf("expected identifier after .")
		}
		target += "." + next
	}
	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	return &HandlerCall{Target: target, Args: args}, nil
}

func (p *parser) parseArgs() ([]Arg, error) {
	p.skipSpace()
	if !p.consume("(") {
		return nil, p.errf("expected ( for call arguments")
	}
	var args []Arg
	for {
		p.skipSpace()
		if p.consume(")") {
			return args, nil
		}
		if p.peek() == '"' {
			v, err := p.readQuoted()
			if err != nil {
				return nil, err
			}
			args = append(args, Arg{IsString: true, Value: v})
		} else {
			v := p.readDottedIdent()
			if v == "" {
				return nil, p.errf("expected argument")
			}
			args = append(args, Arg{Value: v})
		}
		p.skipSpace()
		if p.consume(",") {
			continue
		}
		if p.consume(")") {
			return args, nil
		}
		return nil, p.errf("expected , or ) in call arguments")
	}
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

func (p *parser) readDottedIdent() string {
	start := p.pos
	if p.readIdent() == "" {
		return ""
	}
	for p.peek() == '.' {
		p.pos++
		if p.readIdent() == "" {
			break
		}
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
