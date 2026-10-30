package parser

import (
	"fmt"
	"strings"
)

// Error is a parse or expansion error with source context.
type Error struct {
	File    string
	Line    int
	Col     int
	Message string
	Source  string
	Hint    string
}

func (e *Error) Error() string {
	var b strings.Builder
	if e.File != "" {
		fmt.Fprintf(&b, "%s:%d:%d: %s", e.File, e.Line, e.Col, e.Message)
	} else {
		b.WriteString(e.Message)
	}
	if e.Source != "" {
		expanded, col := expandSourceLine(e.Source, e.Col)
		gutter := fmt.Sprintf("  %d | ", e.Line)
		b.WriteString("\n")
		b.WriteString(gutter)
		b.WriteString(expanded)
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", len(gutter)+col-1))
		b.WriteString("^")
	}
	if e.Hint != "" {
		b.WriteString("\n  hint: ")
		b.WriteString(e.Hint)
	}
	return b.String()
}

// PosError returns an error with the source context of pos.
func PosError(pos NodePosition, hint, format string, args ...any) error {
	return &Error{
		File:    pos.File,
		Line:    pos.Line,
		Col:     pos.Col,
		Message: fmt.Sprintf(format, args...),
		Source:  pos.Source,
		Hint:    hint,
	}
}

// NodeError returns an error with the source context of a view node.
func NodeError(n Node, hint, format string, args ...any) error {
	return PosError(n.SourcePos(), hint, format, args...)
}

func expandSourceLine(line string, col int) (string, int) {
	var b strings.Builder
	expandedCol := 1
	for i, r := range line {
		if r == '\t' {
			b.WriteString("    ")
			if i < col-1 {
				expandedCol += 4
			}
			continue
		}
		b.WriteRune(r)
		if i < col-1 {
			expandedCol++
		}
	}
	return strings.TrimRight(b.String(), "\r\n"), expandedCol
}
