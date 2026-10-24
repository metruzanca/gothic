package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReportsArtifacts(t *testing.T) {
	dir := t.TempDir()
	src := `package views

templ Hello() {
	<div>Hi</div>
}
`
	srcPath := filepath.Join(dir, "hello.gothic")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := generate(dir, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"gothic: " + srcPath,
		"hello_gothic.go",
		"hello_gothic.js",
		"wrote 2 artifact(s) from 1 source file(s)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n\n%s", want, got)
		}
	}

	out.Reset()
	if err := generate(dir, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1 source file(s) up to date") {
		t.Errorf("second run should report up to date\n\n%s", out.String())
	}
}

func TestGenerateReportsNoSources(t *testing.T) {
	var out bytes.Buffer
	if err := generate(t.TempDir(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no .gothic files found") {
		t.Fatalf("output = %q", out.String())
	}
}
