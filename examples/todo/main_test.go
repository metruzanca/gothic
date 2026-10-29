package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
)

func TestTodoApp(t *testing.T) {
	ctx := context.Background()
	st, err := openStore(filepath.Join(t.TempDir(), "todo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.close() })

	if err := st.add(ctx, "write tests"); err != nil {
		t.Fatal(err)
	}
	todos, err := st.list(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Text != "write tests" || todos[0].Status != "open" {
		t.Fatalf("todos = %#v", todos)
	}

	s := gothic.NewServer()
	RegisterProcedures(s, procedures{store: st})
	mountApp(s, st)
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{
		`data-gothic-root="app"`,
		`"Todos":[{"ID":1,"Text":"write tests","Done":false,"Status":"open"}]`,
		`data-gothic-key="1"`,
		`data-gothic-on="0"`,
		`data-gothic-on="2"`,
		`data-gothic-on="3"`,
		`/_gothic/modules/app_gothic.js`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("SSR missing %q\n\n%s", want, html)
		}
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/app_gothic.js")
	if err != nil {
		t.Fatal(err)
	}
	js, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{
		`list(root, "0", (todo) => {`,
		`(todo) => String(todo.ID)`,
		`await AddTodo(text)`,
		`await ToggleTodo(id)`,
		`await DeleteTodo(id)`,
		`event.target.value`,
	} {
		if !strings.Contains(string(js), want) {
			t.Errorf("generated JS missing %q\n\n%s", want, js)
		}
	}

	postRPC(t, ts.URL, "AddTodo", `["buy milk"]`)
	todos, err = st.list(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 || todos[1].Text != "buy milk" {
		t.Fatalf("after add: %#v", todos)
	}

	postRPC(t, ts.URL, "ToggleTodo", `[1]`)
	todos, err = st.list(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !todos[0].Done || todos[0].Status != "done" {
		t.Fatalf("after toggle: %#v", todos[0])
	}

	postRPC(t, ts.URL, "DeleteTodo", `[1]`)
	todos, err = st.list(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Text != "buy milk" {
		t.Fatalf("after delete: %#v", todos)
	}

	// All users share the same store, so a second app sees the same list.
	res, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), "buy milk") {
		t.Fatalf("shared list missing added todo\n\n%s", body)
	}
}

func postRPC(t *testing.T, base, name, args string) {
	t.Helper()
	body := fmt.Sprintf(`{"args":%s}`, args)
	res, err := http.Post(base+"/_gothic/rpc/"+name, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		t.Fatalf("%s returned %d: %s", name, res.StatusCode, data)
	}
}
