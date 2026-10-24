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
	"github.com/metruzanca/gothic/generator/jsgen"
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
	f, err := parser.Parse(path, src)
	if err != nil {
		return 0, err
	}
	base := strings.TrimSuffix(filepath.Base(path), ".gothic")
	dir := filepath.Dir(path)

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

	fmt.Fprintf(out, "gothic: %s\n", path)
	written := 0
	for _, artifact := range []struct {
		name string
		data []byte
	}{
		{filepath.Join(dir, base+"_gothic.go"), goSrc},
		{filepath.Join(dir, base+"_gothic.js"), jsSrc},
	} {
		changed, err := writeIfChanged(artifact.name, artifact.data)
		if err != nil {
			return written, err
		}
		if changed {
			written++
			fmt.Fprintf(out, "  -> %s\n", artifact.name)
		} else {
			fmt.Fprintf(out, "  = %s (unchanged)\n", artifact.name)
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
