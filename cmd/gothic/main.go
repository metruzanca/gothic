// Command gothic generates Go and JavaScript from .gothic source files.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/metruzanca/gothic/generator"
	"github.com/metruzanca/gothic/generator/appjs"
	"github.com/metruzanca/gothic/generator/jsgen"
	"github.com/metruzanca/gothic/goapp"
	"github.com/metruzanca/gothic/parser"
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
	var res result
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".gothic") {
			return nil
		}
		written, err := generateFile(path, out)
		if err != nil {
			return err
		}
		res.sources++
		res.written += written
		return nil
	})
	if err != nil {
		return err
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

func generateFile(path string, out io.Writer) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	base := strings.TrimSuffix(filepath.Base(path), ".gothic")
	dir := filepath.Dir(path)

	if appFile, err := parser.ParseApp(path, src); err == nil {
		if app, err := goapp.ParseSource(path, appFile.GoSource); err == nil {
			return generateAppFile(path, dir, base, appFile, app, out)
		}
	}

	f, err := parser.Parse(path, src)
	if err != nil {
		return 0, err
	}
	goSrc, err := generator.Generate(f, generator.Options{
		FileName:   path,
		ModuleBase: base,
	})
	if err != nil {
		return 0, err
	}
	jsSrc, err := jsgen.Generate(f, jsgen.Options{
		FileName:   path,
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		return 0, err
	}
	return writeArtifacts(out, path, []artifact{
		{filepath.Join(dir, base+"_gothic.go"), goSrc},
		{filepath.Join(dir, base+"_gothic.js"), jsSrc},
	})
}

func generateAppFile(path, dir, base string, appFile *parser.AppFile, app *goapp.App, out io.Writer) (int, error) {
	goSrc, err := generator.GenerateSingle(appFile.View, app, generator.Options{
		FileName:   path,
		ModuleBase: base,
		App:        app,
	})
	if err != nil {
		return 0, err
	}
	jsSrc, err := appjs.Generate(appFile.View, app, appjs.Options{
		FileName:   path,
		RuntimeURL: "/_gothic/runtime.js",
	})
	if err != nil {
		return 0, err
	}
	// Remove the old split app file if it exists.
	legacy := filepath.Join(dir, base+"_app_gothic.go")
	if _, err := os.Stat(legacy); err == nil {
		_ = os.Remove(legacy)
	}
	return writeArtifacts(out, path, []artifact{
		{filepath.Join(dir, base+"_gothic.go"), goSrc},
		{filepath.Join(dir, base+"_gothic.js"), jsSrc},
	})
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
