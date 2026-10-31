// Command gothic generates Go and JavaScript from .gth source files.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
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

type libraryConfig struct {
	Kind       string   `json:"kind"`
	Package    string   `json:"package"`
	Alias      string   `json:"alias"`
	ModuleFile string   `json:"module_file"`
	ModuleURL  string   `json:"module_url"`
	GoFile     string   `json:"go_file"`
	Manifest   string   `json:"manifest"`
	Inline     []string `json:"inline"`
}

type libraryManifest struct {
	Package    string             `json:"package"`
	Alias      string             `json:"alias"`
	Module     string             `json:"module"`
	Components []libraryComponent `json:"components"`
}

type libraryComponent struct {
	Name      string        `json:"name"`
	Go        string        `json:"go"`
	Mount     string        `json:"mount"`
	Create    string        `json:"create"`
	PropsType string        `json:"props_type,omitempty"`
	Props     []libraryProp `json:"props"`
	Listable  bool          `json:"listable"`
	Inline    bool          `json:"inline,omitempty"`
}

type libraryProp struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func generate(root string, out io.Writer) error {
	byDir := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gth") {
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
		fmt.Fprintf(out, "gothic: no .gth files found under %s\n", root)
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
	src     []byte
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
		base: strings.TrimSuffix(filepath.Base(path), ".gth"),
		src:  src,
	}
	appFile, appErr := viewparser.ParseApp(path, src)
	if appErr == nil {
		s.appFile = appFile
		s.view = appFile.View
		if app, err := goapp.ParseSource(path, appFile.GoSource); err == nil {
			s.app = app
		} else if hasAppView(s.view) {
			return nil, err
		}
		return s, nil
	}
	f, legacyErr := viewparser.Parse(path, src)
	if legacyErr != nil {
		var pe *viewparser.Error
		if errors.As(appErr, &pe) {
			return nil, appErr
		}
		return nil, legacyErr
	}
	s.view = f
	return s, nil
}

func propsTypesFromMessages(messages map[string][]goapp.Field) viewparser.PropsTypes {
	out := viewparser.PropsTypes{}
	for name, fields := range messages {
		params := make([]viewparser.Param, len(fields))
		for i, field := range fields {
			params[i] = viewparser.Param{Name: field.Name, Type: field.Type}
		}
		out[name] = viewparser.PropsType{Name: name, Fields: params}
	}
	return out
}

func usedPropsTypes(registry viewparser.ComponentSet, stateful []*generator.Stateful) map[string]bool {
	out := map[string]bool{}
	for _, comp := range stateful {
		if comp.PropsType != "" {
			out[comp.PropsType] = true
		}
	}
	for _, t := range registry {
		if len(t.ParamList) == 1 && t.ParamList[0].Name == "props" {
			out[t.ParamList[0].Type] = true
		}
	}
	return out
}

func hasAppView(f *viewparser.File) bool {
	for _, t := range f.Templates {
		if t.Name == "App" {
			return true
		}
	}
	return false
}

func generateDir(dir string, paths []string, out io.Writer) (int, int, error) {
	if cfg, ok, err := readLibraryConfig(dir); err != nil {
		return 0, 0, err
	} else if ok {
		return generateLibraryDir(dir, paths, cfg, out)
	}
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
	statefulViews, err := addAppComponents(registry, appSource)
	if err != nil {
		return 0, 0, wrapFile(appSource.path, err)
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
			return 0, 0, wrapFile(s.path, err)
		}
	}

	external, externalPure, externalProps, err := loadExternalComponents(appSource, registry)
	if err != nil {
		return 0, 0, wrapFile(appSource.path, err)
	}
	for _, ext := range external {
		registry[ext.Name] = &viewparser.Template{Name: ext.Name, Stateful: true, ParamList: ext.Props}
	}
	for name, t := range externalPure {
		if _, exists := registry[name]; exists {
			return 0, 0, fmt.Errorf("duplicate component %q", name)
		}
		registry[name] = t
	}
	if err := validateRegistry(registry); err != nil {
		return 0, 0, err
	}
	propsTypes := propsTypesFromMessages(appSource.app.Messages)
	for name, spec := range externalProps {
		propsTypes[name] = spec
	}
	expanded, err := viewparser.ExpandAppWithProps(appSource.view, registry, propsTypes)
	if err != nil {
		return 0, 0, wrapFile(appSource.path, err)
	}
	if err := validateApp(expanded); err != nil {
		return 0, 0, wrapFile(appSource.path, err)
	}
	stateful, err := buildStateful(appSource, statefulViews, registry)
	if err != nil {
		return 0, 0, wrapFile(appSource.path, err)
	}
	stateful = append(stateful, external...)
	propsTypeNames := usedPropsTypes(registry, stateful)
	n, err := generateApp(appSource, expanded, stateful, propsTypeNames, out)
	if err != nil {
		return 0, 0, err
	}
	written += n
	return written, len(sources), nil
}

func addAppComponents(registry viewparser.ComponentSet, s *source) ([]*viewparser.Template, error) {
	appCount := 0
	var stateful []*viewparser.Template
	for _, t := range s.view.Templates {
		if t.Name == "App" {
			appCount++
			continue
		}
		if len(t.ParamList) > 0 {
			comp := s.app.Components[t.ParamList[0].Type]
			if comp != nil && comp.Init != nil && comp.Update != nil {
				t.Stateful = true
				if err := addComponent(registry, t); err != nil {
					return nil, err
				}
				stateful = append(stateful, t)
				continue
			}
		}
		if err := addComponent(registry, t); err != nil {
			return nil, err
		}
	}
	if appCount == 0 {
		return nil, fmt.Errorf("no App view found")
	}
	if appCount > 1 {
		return nil, fmt.Errorf("duplicate App view")
	}
	return stateful, nil
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
		if p.Name == "emit" {
			return fmt.Errorf("component %s: prop %q is reserved", t.Name, p.Name)
		}
		if seen[p.Name] {
			return fmt.Errorf("component %s: duplicate prop %q", t.Name, p.Name)
		}
		seen[p.Name] = true
	}
	registry[t.Name] = t
	return nil
}

func wrapFile(file string, err error) error {
	var pe *viewparser.Error
	if errors.As(err, &pe) {
		return err
	}
	return fmt.Errorf("%s: %w", file, err)
}

func validateRegistry(registry viewparser.ComponentSet) error {
	for name, t := range registry {
		if t.Stateful && len(t.Children) == 0 {
			continue
		}
		if len(t.Children) != 1 {
			return viewparser.PosError(t.NodePosition, "wrap the component in one root element", "component %s must have exactly one root element", name)
		}
		switch t.Children[0].(type) {
		case *viewparser.Element, *viewparser.ComponentCall:
		default:
			return viewparser.PosError(t.NodePosition, "use an element as the component root", "component %s root must be an element", name)
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
		return viewparser.PosError(app.NodePosition, "declare exactly one model parameter: view App(model Model)", "App must have exactly one model parameter")
	}
	if len(app.State) > 0 {
		return viewparser.PosError(app.NodePosition, "move state into the model", "App cannot declare state")
	}
	if len(app.Children) != 1 {
		return viewparser.PosError(app.NodePosition, "wrap the App view in one root element", "App must have exactly one root element")
	}
	if _, ok := app.Children[0].(*viewparser.Element); !ok {
		return viewparser.PosError(app.NodePosition, "use an element as the App root", "App root must be an element")
	}
	return nil
}

func statefulProps(t *viewparser.Template, messages map[string][]goapp.Field) (string, []viewparser.Param, error) {
	if len(t.ParamList) == 1 {
		return "", nil, nil
	}
	if len(t.ParamList) != 2 {
		return "", nil, viewparser.PosError(t.NodePosition, "declare only a model and an optional props struct", "stateful component %s must take a model and one props struct", t.Name)
	}
	p := t.ParamList[1]
	if p.Name != "props" {
		return "", nil, viewparser.PosError(t.NodePosition, "name the props parameter props", "stateful component %s: props parameter must be named props, got %s", t.Name, p.Name)
	}
	fields, ok := messages[p.Type]
	if !ok {
		return "", nil, viewparser.PosError(t.NodePosition, "define type "+p.Type+" struct{...} above the component", "stateful component %s: props type %s not found", t.Name, p.Type)
	}
	props := make([]viewparser.Param, len(fields))
	for i, f := range fields {
		props[i] = viewparser.Param{Name: f.Name, Type: f.Type}
		if strings.EqualFold(f.Name, "key") {
			return "", nil, viewparser.PosError(t.NodePosition, "rename the Key field", "stateful component %s: props field Key is reserved for list keys", t.Name)
		}
		for j := 0; j < i; j++ {
			if strings.EqualFold(props[j].Name, props[i].Name) {
				return "", nil, viewparser.PosError(t.NodePosition, "rename one of the ambiguous fields", "stateful component %s: props fields %s and %s differ only by case", t.Name, props[j].Name, props[i].Name)
			}
		}
	}
	return p.Type, props, nil
}

func componentConstructor(commands map[string]*ast.FuncDecl, modelType string) (string, bool) {
	name := "New" + modelType
	fn, ok := commands[name]
	if !ok {
		return "", false
	}
	takesProps := fn.Type.Params != nil && len(fn.Type.Params.List) > 0
	return name, takesProps
}

func buildStateful(s *source, views []*viewparser.Template, registry viewparser.ComponentSet) ([]*generator.Stateful, error) {
	components := make([]*generator.Stateful, 0, len(views))
	for _, t := range views {
		if len(t.ParamList) == 0 {
			return nil, viewparser.PosError(t.NodePosition, "declare the model as the first parameter", "stateful component %s must have a model parameter", t.Name)
		}
		modelType := t.ParamList[0].Type
		app := s.app.ComponentApp(modelType)
		if app == nil {
			return nil, viewparser.PosError(t.NodePosition, "add Init and Update methods to the model", "stateful component %s: model %s has no Init/Update", t.Name, modelType)
		}
		expanded, err := viewparser.ExpandTemplateWithProps(t, registry, propsTypesFromMessages(s.app.Messages))
		if err != nil {
			return nil, fmt.Errorf("stateful component %s: %w", t.Name, err)
		}
		if len(expanded.Children) != 1 {
			return nil, viewparser.PosError(t.NodePosition, "wrap the component in one root element", "stateful component %s must have one root element", t.Name)
		}
		if _, ok := expanded.Children[0].(*viewparser.Element); !ok {
			return nil, viewparser.PosError(t.NodePosition, "use an element as the component root", "stateful component %s root must be an element", t.Name)
		}
		modelJSON, err := zeroModelJSON(s.app.Messages[modelType])
		if err != nil {
			return nil, fmt.Errorf("stateful component %s: %w", t.Name, err)
		}
		propsType, props, err := statefulProps(t, s.app.Messages)
		if err != nil {
			return nil, err
		}
		constructor, constructorTakesProps := componentConstructor(s.app.Commands, modelType)
		name := t.Name
		components = append(components, &generator.Stateful{
			Name:                  name,
			GoName:                "__gothic" + viewparserName(name) + "Component",
			ViewName:              "__gothic" + viewparserName(name) + "View",
			MountFunc:             "mount" + viewparserName(name),
			CreateFunc:            "create" + viewparserName(name),
			ModelType:             modelType,
			ModelJSON:             modelJSON,
			Constructor:           constructor,
			ConstructorTakesProps: constructorTakesProps,
			PropsType:             propsType,
			Props:                 props,
			View:                  expanded,
			App:                   app,
			Listable:              true,
			ModuleURL:             generator.ComponentsModuleURL(s.base),
		})
	}
	return components, nil
}

func paramsString(params []viewparser.Param) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Name + " " + p.Type
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func viewparserName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func zeroModelJSON(fields []goapp.Field) (string, error) {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Name == "" {
			return "", fmt.Errorf("embedded fields are not supported in component models")
		}
		var zero string
		switch {
		case f.Type == "string":
			zero = `""`
		case f.Type == "bool":
			zero = "false"
		case strings.HasPrefix(f.Type, "[]"):
			zero = "[]"
		case strings.HasPrefix(f.Type, "map["):
			zero = "{}"
		case strings.HasPrefix(f.Type, "*"):
			zero = "null"
		case f.Type == "int" || f.Type == "int8" || f.Type == "int16" || f.Type == "int32" || f.Type == "int64" ||
			f.Type == "uint" || f.Type == "uint8" || f.Type == "uint16" || f.Type == "uint32" || f.Type == "uint64" ||
			f.Type == "float32" || f.Type == "float64":
			zero = "0"
		default:
			return "", fmt.Errorf("unsupported model field type %s for %s", f.Type, f.Name)
		}
		parts = append(parts, fmt.Sprintf("%q:%s", f.Name, zero))
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func readLibraryConfig(dir string) (libraryConfig, bool, error) {
	var cfg libraryConfig
	data, err := os.ReadFile(filepath.Join(dir, "gothic.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil
		}
		return cfg, false, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, false, fmt.Errorf("%s: %w", filepath.Join(dir, "gothic.json"), err)
	}
	if cfg.Kind != "library" {
		return cfg, false, nil
	}
	return cfg, true, nil
}

func inlineComponents(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func generateLibraryDir(dir string, paths []string, cfg libraryConfig, out io.Writer) (int, int, error) {
	sort.Strings(paths)
	type libFile struct {
		path    string
		prelude []byte
		views   []*viewparser.Template
		app     *goapp.App
	}
	files := make([]libFile, 0, len(paths))
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return 0, 0, err
		}
		appFile, err := viewparser.ParseApp(path, src)
		if err != nil {
			return 0, 0, err
		}
		app, err := goapp.ParseComponents(path, appFile.GoSource)
		if err != nil {
			return 0, 0, err
		}
		files = append(files, libFile{path: path, prelude: appFile.GoSource, views: appFile.View.Templates, app: app})
	}

	propsTypes := viewparser.PropsTypes{}
	for _, f := range files {
		for name, fields := range f.app.Messages {
			params := make([]viewparser.Param, len(fields))
			for i, field := range fields {
				params[i] = viewparser.Param{Name: field.Name, Type: field.Type}
			}
			propsTypes[name] = viewparser.PropsType{Name: name, Fields: params}
		}
	}

	pure := viewparser.ComponentSet{}
	type statefulView struct {
		t    *viewparser.Template
		app  *goapp.App
		file string
	}
	var statefulViews []statefulView
	for _, f := range files {
		for _, t := range f.views {
			var app *goapp.App
			if len(t.ParamList) > 0 {
				for _, candidate := range files {
					if candidate.app.Components[t.ParamList[0].Type] != nil {
						app = candidate.app
						break
					}
				}
			}
			if app == nil {
				if _, exists := pure[t.Name]; exists {
					return 0, 0, fmt.Errorf("duplicate library component %q", t.Name)
				}
				if len(t.Children) != 1 {
					return 0, 0, fmt.Errorf("library component %s must have one root element", t.Name)
				}
				pure[t.Name] = t
				continue
			}
			statefulViews = append(statefulViews, statefulView{t: t, app: app, file: f.path})
		}
	}

	stateful := make([]*generator.Stateful, 0, len(statefulViews))
	inlineSet := inlineComponents(cfg.Inline)
	seenStateful := map[string]bool{}
	for _, sv := range statefulViews {
		t := sv.t
		if seenStateful[t.Name] {
			return 0, 0, fmt.Errorf("duplicate library component %q", t.Name)
		}
		seenStateful[t.Name] = true
		if _, exists := pure[t.Name]; exists {
			return 0, 0, fmt.Errorf("library component %q is both pure and stateful", t.Name)
		}
		comp := sv.app.Components[t.ParamList[0].Type]
		if comp.Init == nil || comp.Update == nil {
			return 0, 0, fmt.Errorf("library component %s: model %s needs Init and Update", t.Name, t.ParamList[0].Type)
		}
		modelJSON, err := zeroModelJSON(sv.app.Messages[t.ParamList[0].Type])
		if err != nil {
			return 0, 0, fmt.Errorf("library component %s: %w", t.Name, err)
		}
		propsType, props, err := statefulProps(t, sv.app.Messages)
		if err != nil {
			return 0, 0, err
		}
		expanded := t
		if !inlineSet[t.Name] {
			expanded, err = viewparser.ExpandTemplateWithProps(t, pure, propsTypes)
			if err != nil {
				return 0, 0, fmt.Errorf("library component %s: %w", t.Name, err)
			}
		}
		if len(expanded.Children) != 1 {
			return 0, 0, fmt.Errorf("library component %s must have one root element", t.Name)
		}
		if _, ok := expanded.Children[0].(*viewparser.Element); !ok {
			return 0, 0, fmt.Errorf("library component %s root must be an element", t.Name)
		}
		name := t.Name
		constructor, constructorTakesProps := componentConstructor(sv.app.Commands, t.ParamList[0].Type)
		stateful = append(stateful, &generator.Stateful{
			Name:                  name,
			GoName:                name,
			ViewName:              "__gothic" + viewparserName(name) + "View",
			MountFunc:             "mount" + viewparserName(name),
			CreateFunc:            "create" + viewparserName(name),
			ModelType:             t.ParamList[0].Type,
			ModelJSON:             modelJSON,
			Constructor:           constructor,
			ConstructorTakesProps: constructorTakesProps,
			PropsType:             propsType,
			Props:                 props,
			View:                  expanded,
			App:                   sv.app.ComponentApp(t.ParamList[0].Type),
			Listable:              true,
			Inline:                inlineSet[name],
			ModuleURL:             cfg.ModuleURL,
		})
	}

	var preludes [][]byte
	for _, f := range files {
		preludes = append(preludes, f.prelude)
	}

	var precompiled []*generator.Stateful
	for _, comp := range stateful {
		if !comp.Inline {
			precompiled = append(precompiled, comp)
		}
	}
	goSrc, err := generator.GenerateLibrary(generator.LibraryOptions{
		FileName:   filepath.Join(dir, "library.gth"),
		Package:    "",
		ModuleFile: cfg.ModuleFile,
		ModuleURL:  cfg.ModuleURL,
		Preludes:   preludes,
		Stateful:   precompiled,
	})
	if err != nil {
		return 0, 0, err
	}
	jsSrc, err := appjs.GenerateComponents(precompiled, appjs.Options{
		FileName:    filepath.Join(dir, "library.gth"),
		RuntimeURL:  "/_gothic/runtime.js",
		AllStateful: stateful,
		PropsTypes:  usedPropsTypes(pure, stateful),
	})
	if err != nil {
		return 0, 0, err
	}
	manifest := libraryManifest{Package: cfg.Package, Alias: cfg.Alias, Module: cfg.ModuleURL}
	for _, comp := range stateful {
		props := make([]libraryProp, len(comp.Props))
		for i, p := range comp.Props {
			props[i] = libraryProp{Name: p.Name, Type: p.Type}
		}
		manifest.Components = append(manifest.Components, libraryComponent{
			Name: comp.Name, Go: comp.GoName, Mount: comp.MountFunc, Create: comp.CreateFunc,
			PropsType: comp.PropsType, Props: props, Listable: comp.Listable, Inline: comp.Inline,
		})
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return 0, 0, err
	}
	manifestBytes = append(manifestBytes, '\n')
	n, err := writeArtifacts(out, filepath.Join(dir, "library.gth"), []artifact{
		{filepath.Join(dir, cfg.GoFile), goSrc},
		{filepath.Join(dir, cfg.ModuleFile), jsSrc},
		{filepath.Join(dir, cfg.Manifest), manifestBytes},
	})
	if err != nil {
		return 0, 0, err
	}
	return n, len(paths), nil
}

func appUsesComponent(f *viewparser.File, name string) bool {
	for _, t := range f.Templates {
		if t.Name != "App" {
			continue
		}
		for _, call := range collectComponentCalls(t.Children) {
			if call.Name == name {
				return true
			}
		}
	}
	return false
}

func findRowSlot(nodes []viewparser.Node) *viewparser.ComponentCall {
	for _, node := range nodes {
		switch n := node.(type) {
		case *viewparser.ComponentCall:
			if n.Name == "RowSlot" {
				return n
			}
		case *viewparser.Element:
			if c := findRowSlot(n.Children); c != nil {
				return c
			}
		case *viewparser.If:
			if c := findRowSlot(n.Then); c != nil {
				return c
			}
			if c := findRowSlot(n.Else); c != nil {
				return c
			}
		case *viewparser.For:
			if c := findRowSlot(n.Children); c != nil {
				return c
			}
		}
	}
	return nil
}

func walkComponentCalls(nodes []viewparser.Node, fn func(*viewparser.ComponentCall)) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *viewparser.ComponentCall:
			fn(n)
		case *viewparser.Element:
			walkComponentCalls(n.Children, fn)
		case *viewparser.If:
			walkComponentCalls(n.Then, fn)
			walkComponentCalls(n.Else, fn)
		case *viewparser.For:
			walkComponentCalls(n.Children, fn)
		}
	}
}

func anySliceExpr(e ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "gothic"},
			Sel: &ast.Ident{Name: "AnySlice"},
		},
		Args: []ast.Expr{e},
	}
}

func bakeRowNodes(nodes []viewparser.Node, rowName, rowParam, rowType string) (int, error) {
	found := 0
	for _, node := range nodes {
		switch n := node.(type) {
		case *viewparser.ComponentCall:
			if n.Name != "RowSlot" {
				continue
			}
			var itemExpr ast.Expr
			for _, prop := range n.Props {
				if prop.Name == "item" {
					itemExpr = prop.Value
				}
			}
			if itemExpr == nil {
				return 0, fmt.Errorf("row placeholder must have an item prop")
			}
			typeExpr, err := parser.ParseExpr(rowType)
			if err != nil {
				return 0, fmt.Errorf("row component parameter type %s: %w", rowType, err)
			}
			n.Name = rowName
			n.Props = []viewparser.Prop{{
				Name: rowParam,
				Value: &ast.TypeAssertExpr{
					X:    itemExpr,
					Type: typeExpr,
				},
			}}
			found++
		case *viewparser.Element:
			n2, err := bakeRowNodes(n.Children, rowName, rowParam, rowType)
			if err != nil {
				return 0, err
			}
			found += n2
		case *viewparser.If:
			n2, err := bakeRowNodes(n.Then, rowName, rowParam, rowType)
			if err != nil {
				return 0, err
			}
			found += n2
			n2, err = bakeRowNodes(n.Else, rowName, rowParam, rowType)
			if err != nil {
				return 0, err
			}
			found += n2
		case *viewparser.For:
			n2, err := bakeRowNodes(n.Children, rowName, rowParam, rowType)
			if err != nil {
				return 0, err
			}
			found += n2
		}
	}
	return found, nil
}

func bakeInlineRow(appView *viewparser.File, inlineName string, tmpl *viewparser.Template, registry viewparser.ComponentSet) (string, error) {
	var app *viewparser.Template
	for _, t := range appView.Templates {
		if t.Name == "App" {
			app = t
			break
		}
	}
	if app == nil {
		return "", fmt.Errorf("inline component %s: app has no App view", inlineName)
	}
	rowName := ""
	for _, call := range collectComponentCalls(app.Children) {
		if call.Name != inlineName {
			continue
		}
		kept := make([]viewparser.Prop, 0, len(call.Props))
		for _, prop := range call.Props {
			switch prop.Name {
			case "row":
				id, ok := prop.Value.(*ast.Ident)
				if !ok {
					return "", fmt.Errorf("component %s: row must be a view name", inlineName)
				}
				if rowName != "" && rowName != id.Name {
					return "", fmt.Errorf("component %s: use one row view per app", inlineName)
				}
				rowName = id.Name
			case "items":
				prop.Value = anySliceExpr(prop.Value)
				kept = append(kept, prop)
			default:
				kept = append(kept, prop)
			}
		}
		call.Props = kept
	}
	if rowName == "" {
		return "", fmt.Errorf("component %s: add row={ RowView }", inlineName)
	}
	row, ok := registry[rowName]
	if !ok {
		return "", fmt.Errorf("component %s: unknown row view %s", inlineName, rowName)
	}
	if row.Stateful {
		return "", fmt.Errorf("component %s: row view %s must be pure", inlineName, rowName)
	}
	if len(row.ParamList) != 1 {
		return "", fmt.Errorf("component %s: row view %s must take one parameter", inlineName, rowName)
	}
	found, err := bakeRowNodes(tmpl.Children, rowName, row.ParamList[0].Name, row.ParamList[0].Type)
	if err != nil {
		return "", fmt.Errorf("component %s: %w", inlineName, err)
	}
	if found == 0 {
		return "", fmt.Errorf("component %s: view has no row placeholder", inlineName)
	}
	return rowName, nil
}

func collectComponentCalls(nodes []viewparser.Node) []*viewparser.ComponentCall {
	var calls []*viewparser.ComponentCall
	walkComponentCalls(nodes, func(call *viewparser.ComponentCall) {
		calls = append(calls, call)
	})
	return calls
}

func loadExternalComponents(s *source, appRegistry viewparser.ComponentSet) ([]*generator.Stateful, viewparser.ComponentSet, viewparser.PropsTypes, error) {
	paths := componentDirectives(s.src)
	var out []*generator.Stateful
	pure := viewparser.ComponentSet{}
	propsTypes := viewparser.PropsTypes{}
	for _, path := range paths {
		cmd := exec.Command("go", "list", "-f", "{{.Dir}}", path)
		cmd.Dir = s.dir
		data, err := cmd.Output()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("resolve component package %s: %w", path, err)
		}
		pkgDir := strings.TrimSpace(string(data))
		manifestData, err := os.ReadFile(filepath.Join(pkgDir, "gothic-components.json"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read component manifest for %s: %w", path, err)
		}
		var manifest libraryManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, nil, nil, fmt.Errorf("parse component manifest for %s: %w", path, err)
		}
		if manifest.Alias == "" {
			return nil, nil, nil, fmt.Errorf("component package %s: missing alias in manifest", path)
		}
		statefulNames := map[string]bool{}
		var inlineManifest []libraryComponent
		for _, comp := range manifest.Components {
			statefulNames[comp.Name] = true
			if comp.Inline {
				inlineManifest = append(inlineManifest, comp)
				continue
			}
			props := make([]viewparser.Param, len(comp.Props))
			for i, p := range comp.Props {
				props[i] = viewparser.Param{Name: p.Name, Type: p.Type}
			}
			out = append(out, &generator.Stateful{
				Name:        manifest.Alias + "." + comp.Name,
				GoName:      comp.Go,
				MountFunc:   comp.Mount,
				CreateFunc:  comp.Create,
				PropsType:   comp.PropsType,
				Props:       props,
				Listable:    comp.Listable,
				External:    true,
				ImportAlias: manifest.Alias,
				ImportPath:  manifest.Package,
				ModuleURL:   manifest.Module,
			})
		}

		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read component package %s: %w", path, err)
		}
		type pureFile struct {
			path    string
			appFile *viewparser.AppFile
			app     *goapp.App
		}
		var pureFiles []pureFile
		pkgProps := viewparser.PropsTypes{}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gth") {
				continue
			}
			file := filepath.Join(pkgDir, entry.Name())
			src, err := os.ReadFile(file)
			if err != nil {
				return nil, nil, nil, err
			}
			appFile, err := viewparser.ParseApp(file, src)
			if err != nil {
				return nil, nil, nil, err
			}
			app, err := goapp.ParseComponents(file, appFile.GoSource)
			if err != nil {
				return nil, nil, nil, err
			}
			pureFiles = append(pureFiles, pureFile{path: file, appFile: appFile, app: app})
			for name, fields := range app.Messages {
				params := make([]viewparser.Param, len(fields))
				for i, field := range fields {
					params[i] = viewparser.Param{Name: field.Name, Type: field.Type}
				}
				pkgProps[name] = viewparser.PropsType{Name: name, Fields: params}
			}
		}
		localPure := viewparser.ComponentSet{}
		for _, f := range pureFiles {
			for _, t := range f.appFile.View.Templates {
				if statefulNames[t.Name] {
					continue
				}
				localPure[t.Name] = t
			}
		}
		for _, comp := range inlineManifest {
			inlineName := manifest.Alias + "." + comp.Name
			if !appUsesComponent(s.view, inlineName) {
				continue
			}
			var tmpl *viewparser.Template
			var compApp *goapp.App
			for _, f := range pureFiles {
				for _, t := range f.appFile.View.Templates {
					if t.Name != comp.Name {
						continue
					}
					tmpl = t
					if len(t.ParamList) > 0 {
						compApp = f.app.ComponentApp(t.ParamList[0].Type)
					}
				}
			}
			if tmpl == nil {
				return nil, nil, nil, fmt.Errorf("component package %s: inline component %s has no view", path, comp.Name)
			}
			if len(tmpl.ParamList) == 0 || compApp == nil {
				return nil, nil, nil, fmt.Errorf("component package %s: inline component %s has no model Init and Update", path, comp.Name)
			}
			modelType := tmpl.ParamList[0].Type
			modelJSON, err := zeroModelJSON(compApp.Messages[modelType])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("inline component %s: %w", comp.Name, err)
			}
			if findRowSlot(tmpl.Children) != nil {
				rowName, err := bakeInlineRow(s.view, inlineName, tmpl, appRegistry)
				if err != nil {
					return nil, nil, nil, err
				}
				_ = rowName
			}
			inlineRegistry := viewparser.ComponentSet{}
			for name, t := range localPure {
				inlineRegistry[name] = t
			}
			for name, t := range appRegistry {
				inlineRegistry[name] = t
			}
			inlineProps := propsTypesFromMessages(s.app.Messages)
			for name, spec := range pkgProps {
				inlineProps[name] = spec
			}
			expanded, err := viewparser.ExpandTemplateWithProps(tmpl, inlineRegistry, inlineProps)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("inline component %s: %w", comp.Name, err)
			}
			for i := range expanded.ParamList {
				expanded.ParamList[i].Type = manifest.Alias + "." + expanded.ParamList[i].Type
			}
			expanded.Params = paramsString(expanded.ParamList)
			constructor, constructorTakesProps := componentConstructor(compApp.Commands, modelType)
			props := make([]viewparser.Param, len(comp.Props))
			for i, p := range comp.Props {
				props[i] = viewparser.Param{Name: p.Name, Type: p.Type}
			}
			prefix := viewparserName(manifest.Alias + comp.Name)
			out = append(out, &generator.Stateful{
				Name:                  manifest.Alias + "." + comp.Name,
				DisplayName:           comp.Name,
				GoName:                "__gothic" + prefix + "Component",
				ViewName:              "__gothic" + prefix + "View",
				MountFunc:             "mount" + prefix,
				CreateFunc:            "create" + prefix,
				ModelType:             modelType,
				ModelJSON:             modelJSON,
				Constructor:           constructor,
				ConstructorTakesProps: constructorTakesProps,
				PropsType:             comp.PropsType,
				Props:                 props,
				View:                  expanded,
				App:                   compApp,
				Listable:              comp.Listable,
				Inline:                true,
				ImportAlias:           manifest.Alias,
				ImportPath:            manifest.Package,
				ModuleURL:             generator.ComponentsModuleURL(s.base),
			})
		}

		for _, f := range pureFiles {
			for _, t := range f.appFile.View.Templates {
				if statefulNames[t.Name] {
					continue
				}
				for _, p := range t.ParamList {
					if p.Name == "emit" {
						return nil, nil, nil, fmt.Errorf("component %s: prop %q is reserved", t.Name, p.Name)
					}
				}
				if len(t.ParamList) == 1 && t.ParamList[0].Name == "props" {
					if spec, ok := pkgProps[t.ParamList[0].Type]; ok {
						qualified := manifest.Alias + "." + t.ParamList[0].Type
						t.ParamList[0].Type = qualified
						pkgProps[qualified] = viewparser.PropsType{Name: qualified, Fields: spec.Fields}
					}
				}
				name := manifest.Alias + "." + t.Name
				if _, exists := pure[name]; exists {
					return nil, nil, nil, fmt.Errorf("duplicate component %q", name)
				}
				pure[name] = t
			}
		}
		for name, spec := range pkgProps {
			propsTypes[name] = spec
		}
	}
	return out, pure, propsTypes, nil
}

func componentDirectives(src []byte) []string {
	var paths []string
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "//gothic:components") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			paths = append(paths, parts[1])
		}
	}
	return paths
}

func generateApp(s *source, expanded *viewparser.File, stateful []*generator.Stateful, propsTypes map[string]bool, out io.Writer) (int, error) {
	goSrc, err := generator.GenerateSingle(expanded, s.app, generator.Options{
		FileName:   s.path,
		ModuleBase: s.base,
		App:        s.app,
		Stateful:   stateful,
	})
	if err != nil {
		return 0, err
	}
	jsSrc, err := appjs.Generate(expanded, s.app, appjs.Options{
		FileName:            s.path,
		RuntimeURL:          "/_gothic/runtime.js",
		Stateful:            stateful,
		ComponentsModuleURL: generator.ComponentsModuleURL(s.base),
		PropsTypes:          propsTypes,
	})
	if err != nil {
		return 0, err
	}
	artifacts := []artifact{
		{filepath.Join(s.dir, s.base+"_gothic.go"), goSrc},
		{filepath.Join(s.dir, s.base+"_gothic.js"), jsSrc},
	}
	localStateful := make([]*generator.Stateful, 0, len(stateful))
	for _, comp := range stateful {
		if !comp.External {
			localStateful = append(localStateful, comp)
		}
	}
	if len(localStateful) > 0 {
		procedureNames := make(map[string]bool, len(s.app.Procedures))
		for _, p := range s.app.Procedures {
			procedureNames[p.Name] = true
		}
		compSrc, err := appjs.GenerateComponents(localStateful, appjs.Options{
			FileName:       s.path,
			RuntimeURL:     "/_gothic/runtime.js",
			Stateful:       stateful,
			AllStateful:    stateful,
			ProcedureNames: procedureNames,
			PropsTypes:     propsTypes,
		})
		if err != nil {
			return 0, err
		}
		artifacts = append(artifacts, artifact{filepath.Join(s.dir, generator.ComponentsModuleFileName(s.base)), compSrc})
	}
	legacy := filepath.Join(s.dir, s.base+"_app_gothic.go")
	if _, err := os.Stat(legacy); err == nil {
		_ = os.Remove(legacy)
	}
	return writeArtifacts(out, s.path, artifacts)
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
