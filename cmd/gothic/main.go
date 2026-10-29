// Command gothic generates Go and JavaScript from .gothic source files.
package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/metruzanca/gothic/generator"
	"github.com/metruzanca/gothic/generator/appjs"
	"github.com/metruzanca/gothic/generator/jsgen"
	"github.com/metruzanca/gothic/goapp"
	viewparser "github.com/metruzanca/gothic/parser"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "generate" {
		fmt.Fprintln(os.Stderr, "usage: gothic generate [path]")
		os.Exit(2)
	}
	root := "."
	if len(os.Args) > 2 {
		root = os.Args[2]
	}
	if err := generate(root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "gothic:", err)
		os.Exit(1)
	}
}

type result struct {
	sources int
	written int
}

func generate(root string, out io.Writer) error {
	byDir := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gothic") {
			return nil
		}
		dir := filepath.Dir(path)
		byDir[dir] = append(byDir[dir], path)
		return nil
	})
	if err != nil {
		return err
	}

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	res := result{}
	for _, dir := range dirs {
		written, sources, err := generateDir(dir, byDir[dir], out)
		if err != nil {
			return err
		}
		res.written += written
		res.sources += sources
	}
	switch {
	case res.sources == 0:
		fmt.Fprintf(out, "gothic: no .gothic files found under %s\n", root)
	case res.written == 0:
		fmt.Fprintf(out, "gothic: %d source file(s) up to date\n", res.sources)
	default:
		fmt.Fprintf(out, "gothic: wrote %d artifact(s) from %d source file(s)\n", res.written, res.sources)
	}
	return nil
}

type source struct {
	path    string
	dir     string
	base    string
	appFile *viewparser.AppFile
	app     *goapp.App
	view    *viewparser.File
}

func loadSource(path string) (*source, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := &source{
		path: path,
		dir:  filepath.Dir(path),
		base: strings.TrimSuffix(filepath.Base(path), ".gothic"),
	}
	if appFile, err := viewparser.ParseApp(path, src); err == nil {
		s.appFile = appFile
		s.view = appFile.View
		if app, err := goapp.ParseSource(path, appFile.GoSource); err == nil {
			s.app = app
		}
		return s, nil
	}
	f, err := viewparser.Parse(path, src)
	if err != nil {
		return nil, err
	}
	s.view = f
	return s, nil
}

func generateDir(dir string, paths []string, out io.Writer) (int, int, error) {
	sort.Strings(paths)
	sources := make([]*source, 0, len(paths))
	var apps []*source
	for _, path := range paths {
		s, err := loadSource(path)
		if err != nil {
			return 0, 0, err
		}
		sources = append(sources, s)
		if s.app != nil {
			apps = append(apps, s)
		}
	}
	if len(apps) > 1 {
		return 0, 0, fmt.Errorf("%s: multiple app files found in one directory", dir)
	}
	if len(apps) == 0 {
		written := 0
		for _, s := range sources {
			n, err := generateLegacy(s, out)
			if err != nil {
				return 0, 0, err
			}
			written += n
		}
		return written, len(sources), nil
	}

	appSource := apps[0]
	registry := viewparser.ComponentSet{}
	if err := addAppComponents(registry, appSource); err != nil {
		return 0, 0, fmt.Errorf("%s: %w", appSource.path, err)
	}

	written := 0
	for _, s := range sources {
		if s == appSource {
			continue
		}
		if s.appFile == nil {
			return 0, 0, fmt.Errorf("%s: cannot mix legacy views and an app in one directory", s.path)
		}
		if s.appFile.Package != appSource.appFile.Package {
			return 0, 0, fmt.Errorf("%s: package %q does not match app package %q", s.path, s.appFile.Package, appSource.appFile.Package)
		}
		if hasLegacySyntax(s.view) {
			n, err := generateLegacy(s, out)
			if err != nil {
				return 0, 0, err
			}
			written += n
			continue
		}
		if err := packageOnly(s.appFile.GoSource); err != nil {
			return 0, 0, fmt.Errorf("%s: component files may only contain a package clause and views: %w", s.path, err)
		}
		if err := addComponents(registry, s.view); err != nil {
			return 0, 0, fmt.Errorf("%s: %w", s.path, err)
		}
	}

	if err := validateRegistry(registry); err != nil {
		return 0, 0, err
	}
	expanded, err := viewparser.ExpandApp(appSource.view, registry)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", appSource.path, err)
	}
	if err := validateApp(expanded); err != nil {
		return 0, 0, fmt.Errorf("%s: %w", appSource.path, err)
	}
	n, err := generateApp(appSource, expanded, out)
	if err != nil {
		return 0, 0, err
	}
	written += n
	return written, len(sources), nil
}

func addAppComponents(registry viewparser.ComponentSet, s *source) error {
	appCount := 0
	for _, t := range s.view.Templates {
		if t.Name == "App" {
			appCount++
			continue
		}
		if err := addComponent(registry, t); err != nil {
			return err
		}
	}
	if appCount == 0 {
		return fmt.Errorf("no App view found")
	}
	if appCount > 1 {
		return fmt.Errorf("duplicate App view")
	}
	return nil
}

func addComponents(registry viewparser.ComponentSet, f *viewparser.File) error {
	for _, t := range f.Templates {
		if t.Name == "App" {
			return fmt.Errorf("App view is only allowed in the app file")
		}
		if err := addComponent(registry, t); err != nil {
			return err
		}
	}
	return nil
}

func addComponent(registry viewparser.ComponentSet, t *viewparser.Template) error {
	if _, exists := registry[t.Name]; exists {
		return fmt.Errorf("duplicate component %q", t.Name)
	}
	if len(t.State) > 0 {
		return fmt.Errorf("component %s cannot declare state", t.Name)
	}
	if t.Name == "" || t.Name[0] < 'A' || t.Name[0] > 'Z' {
		return fmt.Errorf("component name %q must start with an uppercase letter", t.Name)
	}
	seen := map[string]bool{}
	for _, p := range t.ParamList {
		if !token.IsIdentifier(p.Name) {
			return fmt.Errorf("component %s: invalid prop name %q", t.Name, p.Name)
		}
		if seen[p.Name] {
			return fmt.Errorf("component %s: duplicate prop %q", t.Name, p.Name)
		}
		seen[p.Name] = true
	}
	registry[t.Name] = t
	return nil
}

func validateRegistry(registry viewparser.ComponentSet) error {
	for name, t := range registry {
		if len(t.Children) != 1 {
			return fmt.Errorf("component %s must have exactly one root element", name)
		}
		switch t.Children[0].(type) {
		case *viewparser.Element, *viewparser.ComponentCall:
		default:
			return fmt.Errorf("component %s root must be an element", name)
		}
	}
	return nil
}

func validateApp(f *viewparser.File) error {
	if len(f.Templates) != 1 || f.Templates[0].Name != "App" {
		return fmt.Errorf("app must contain one App view")
	}
	app := f.Templates[0]
	if len(app.ParamList) != 1 {
		return fmt.Errorf("App must have exactly one model parameter")
	}
	if len(app.State) > 0 {
		return fmt.Errorf("App cannot declare state")
	}
	if len(app.Children) != 1 {
		return fmt.Errorf("App must have exactly one root element")
	}
	if _, ok := app.Children[0].(*viewparser.Element); !ok {
		return fmt.Errorf("App root must be an element")
	}
	return nil
}

func generateApp(s *source, expanded *viewparser.File, out io.Writer) (int, error) {
	goSrc, err := generator.GenerateSingle(expanded, s.app, generator.Options{
		FileName:   s.path,
		ModuleBase: s.base,
		App:        s.app,
	})
	if err != nil {
		return 0, err
	}
	jsSrc, err := appjs.Generate(expanded, s.app, appjs.Options{
		FileName:   s.path,
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		return 0, err
	}
	legacy := filepath.Join(s.dir, s.base+"_app_gothic.go")
	if _, err := os.Stat(legacy); err == nil {
		_ = os.Remove(legacy)
	}
	return writeArtifacts(out, s.path, []artifact{
		{filepath.Join(s.dir, s.base+"_gothic.go"), goSrc},
		{filepath.Join(s.dir, s.base+"_gothic.js"), jsSrc},
	})
}

func generateLegacy(s *source, out io.Writer) (int, error) {
	goSrc, err := generator.Generate(s.view, generator.Options{
		FileName:   s.path,
		ModuleBase: s.base,
	})
	if err != nil {
		return 0, err
	}
	jsSrc, err := jsgen.Generate(s.view, jsgen.Options{
		FileName:   s.path,
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		return 0, err
	}
	return writeArtifacts(out, s.path, []artifact{
		{filepath.Join(s.dir, s.base+"_gothic.go"), goSrc},
		{filepath.Join(s.dir, s.base+"_gothic.js"), jsSrc},
	})
}

func hasLegacySyntax(f *viewparser.File) bool {
	if len(f.Procedures) > 0 {
		return true
	}
	for _, t := range f.Templates {
		if len(t.State) > 0 {
			return true
		}
	}
	return false
}

func packageOnly(src []byte) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "prelude.go", src, 0)
	if err != nil {
		return err
	}
	if len(f.Imports) > 0 || len(f.Decls) > 0 {
		return fmt.Errorf("found declarations before the first view")
	}
	return nil
}

type artifact struct {
	name string
	data []byte
}

func writeArtifacts(out io.Writer, source string, artifacts []artifact) (int, error) {
	fmt.Fprintf(out, "gothic: %s\n", source)
	written := 0
	for _, a := range artifacts {
		changed, err := writeIfChanged(a.name, a.data)
		if err != nil {
			return written, err
		}
		if changed {
			written++
			fmt.Fprintf(out, "  -> %s\n", a.name)
		} else {
			fmt.Fprintf(out, "  = %s (unchanged)\n", a.name)
		}
	}
	return written, nil
}

func writeIfChanged(path string, data []byte) (bool, error) {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return false, nil
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
