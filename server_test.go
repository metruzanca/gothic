package gothic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerServesRuntimeModuleAndRPC(t *testing.T) {
	s := NewServer()
	RegisterModule("/_gothic/modules/test.js", []byte("export const x = 1;\n"))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = Render(w, r, ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "hello")
			return err
		}))
	})

	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "hello" {
		t.Fatalf("GET / = %d %q", res.StatusCode, body)
	}

	res, err = http.Get(ts.URL + "/_gothic/runtime.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "export function signal") {
		t.Fatalf("runtime.js = %d %q", res.StatusCode, body)
	}

	res, err = http.Get(ts.URL + "/_gothic/modules/test.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "export const x = 1;\n" {
		t.Fatalf("module = %d %q", res.StatusCode, body)
	}

	res, err = http.Post(ts.URL+"/_gothic/rpc/Ping", "application/json", strings.NewReader(`{"args":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"pong"`) {
		t.Fatalf("rpc = %d %q", res.StatusCode, body)
	}

	res, err = http.Post(ts.URL+"/_gothic/rpc/Missing", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound || !strings.Contains(string(body), "unknown procedure") {
		t.Fatalf("missing rpc = %d %q", res.StatusCode, body)
	}
}
