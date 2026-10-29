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

view Hello() {
	<div>Hi</div>
}
`
	srcPath := filepath.Join(dir, "hello.gth")
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

func TestGenerateAppWithComponents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.gth", `package main

type Model struct{ Status string }

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }

view App(model Model) {
	<div><Card text={ model.Status } /></div>
}
`)
	writeFile(t, dir, "card.gth", `package main

view Card(text string) {
	<p>{ text }</p>
}
`)

	var out bytes.Buffer
	if err := generate(dir, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "card_gothic.go")); !os.IsNotExist(err) {
		t.Fatalf("component file generated Go artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "card_gothic.js")); !os.IsNotExist(err) {
		t.Fatalf("component file generated JS artifact: %v", err)
	}
	goSrc, err := os.ReadFile(filepath.Join(dir, "app_gothic.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`func App(model Model)`, `<p`, `model.Status`} {
		if !strings.Contains(string(goSrc), want) {
			t.Errorf("generated Go missing %q\n\n%s", want, goSrc)
		}
	}
	jsSrc, err := os.ReadFile(filepath.Join(dir, "app_gothic.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`text(root, "0", (model.Status));`} {
		if !strings.Contains(string(jsSrc), want) {
			t.Errorf("generated JS missing %q\n\n%s", want, jsSrc)
		}
	}
	if !strings.Contains(out.String(), "wrote 2 artifact(s) from 2 source file(s)") {
		t.Fatalf("output = %q", out.String())
	}

	out.Reset()
	if err := generate(dir, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "2 source file(s) up to date") {
		t.Fatalf("second run output = %q", out.String())
	}
}

func TestGenerateRejectsDuplicateComponent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.gth", `package main

type Model struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }

view Card() { <p>one</p> }

view App(model Model) { <div><Card /></div> }
`)
	writeFile(t, dir, "other.gth", `package main

view Card() { <p>two</p> }
`)
	var out bytes.Buffer
	err := generate(dir, &out)
	if err == nil || !strings.Contains(err.Error(), "duplicate component") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRejectsUnknownComponent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.gth", `package main

type Model struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }

view App(model Model) { <div><Missing /></div> }
`)
	var out bytes.Buffer
	err := generate(dir, &out)
	if err == nil || !strings.Contains(err.Error(), "unknown component") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateRejectsComponentPrelude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.gth", `package main

type Model struct{}

func (m Model) Init() Cmd { return nil }

func (m Model) Update(msg Msg) (Model, Cmd) { return m, nil }

view App(model Model) { <div><Card /></div> }
`)
	writeFile(t, dir, "card.gth", `package main

var X = 1

view Card() { <p>card</p> }
`)
	var out bytes.Buffer
	err := generate(dir, &out)
	if err == nil || !strings.Contains(err.Error(), "component files may only contain") {
		t.Fatalf("error = %v", err)
	}
}

func writeFile(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateReportsNoSources(t *testing.T) {
	var out bytes.Buffer
	if err := generate(t.TempDir(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no .gth files found") {
		t.Fatalf("output = %q", out.String())
	}
}
