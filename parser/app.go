package parser

import (
	"fmt"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"regexp"
	"strings"
)

var viewLine = regexp.MustCompile(`(?m)^view[ \t]+[A-Za-z_]\w*[ \t]*\(`)

func preludeError(name string, src []byte, err error) error {
	if list, ok := err.(scanner.ErrorList); ok && len(list) > 0 {
		first := list[0]
		return &Error{
			File:    name,
			Line:    first.Pos.Line,
			Col:     first.Pos.Column,
			Message: "invalid Go prelude: " + first.Msg,
			Source:  sourceLineAt(src, first.Pos.Line),
		}
	}
	return &Error{File: name, Message: "invalid Go prelude: " + err.Error()}
}

func sourceLineAt(src []byte, line int) string {
	lines := strings.Split(string(src), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return strings.TrimRight(lines[line-1], "\r")
}

// AppFile is a .gth file split into its Go prelude and its view.
type AppFile struct {
	Package  string
	GoSource []byte
	View     *File
}

// ParseApp parses a .gth file that starts with a Go prelude and then
// declares a view.
func ParseApp(name string, src []byte) (*AppFile, error) {
	loc := viewLine.FindIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("%s: no view declaration found", name)
	}
	goSource := src[:loc[0]]
	viewSource := src[loc[0]:]

	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, name, goSource, 0)
	if err != nil {
		return nil, preludeError(name, goSource, err)
	}
	pkg := f.Name.Name
	synthetic := append([]byte("package "+pkg+"\n"), viewSource...)
	startLine := 1 + strings.Count(string(src[:loc[0]]), "\n")
	view, err := parseWithOffset(name, synthetic, src, startLine-2)
	if err != nil {
		return nil, err
	}
	return &AppFile{
		Package:  pkg,
		GoSource: goSource,
		View:     view,
	}, nil
}
