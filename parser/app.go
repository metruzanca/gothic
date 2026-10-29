package parser

import (
	"fmt"
	goparser "go/parser"
	"go/token"
	"regexp"
)

var viewLine = regexp.MustCompile(`(?m)^view[ \t]+[A-Za-z_]\w*[ \t]*\(`)

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
		return nil, fmt.Errorf("%s: invalid Go prelude: %w", name, err)
	}
	pkg := f.Name.Name
	synthetic := append([]byte("package "+pkg+"\n"), viewSource...)
	view, err := Parse(name, synthetic)
	if err != nil {
		return nil, err
	}
	return &AppFile{
		Package:  pkg,
		GoSource: goSource,
		View:     view,
	}, nil
}
