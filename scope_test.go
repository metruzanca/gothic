package gothic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type baseModel struct{}

func (baseModel) Init() Cmd               { return nil }
func (baseModel) Update(Msg) (Model, Cmd) { return baseModel{}, nil }
func (baseModel) View() Component {
	return ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "base="+BasePath(ctx))
		return err
	})
}

func TestScopeIsolatesProcedures(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "root", nil
	})
	admin := s.Scope("/admin")
	admin.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "admin", nil
	})
	ts := NewTestServer(t, s)

	root, err := ts.Call("Ping")
	if err != nil {
		t.Fatal(err)
	}
	if string(root) != `"root"` {
		t.Fatalf("root = %s", root)
	}

	res := ts.PostJSON("/admin"+RPCPrefix+"Ping", map[string]any{"args": []any{}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"admin"`) {
		t.Fatalf("admin = %d %s", res.StatusCode, body)
	}

	if body := ts.Body(ts.Get("/admin" + RuntimePath)); !strings.Contains(body, "export function") {
		t.Fatalf("scoped runtime missing")
	}
}

func TestScopeBasePathReachesViews(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	admin := s.Scope("/admin")
	admin.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	for _, path := range []string{"/admin/", "/admin/deep/link"} {
		body := ts.Body(ts.Get(path))
		if !strings.Contains(body, "base=/admin") {
			t.Fatalf("GET %s missing base path\n%s", path, body)
		}
	}
}

func TestScopeDocumentDefaults(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.SetDocument(Document{Title: "Root"})
	admin := s.Scope("/admin")
	admin.SetDocument(Document{Title: "Admin"})
	s.MountApp("/", NewApp(baseModel{}))
	admin.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	if body := ts.Body(ts.Get("/admin/")); !strings.Contains(body, "<title>Admin</title>") {
		t.Fatalf("admin document missing title\n%s", body)
	}
	if body := ts.Body(ts.Get("/")); !strings.Contains(body, "<title>Root</title>") {
		t.Fatalf("root document missing title\n%s", body)
	}
}

func TestScopeRoutesAndAssets(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	admin := s.Scope("/admin")
	admin.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "users")
	})
	admin.Asset("/app.css", []byte("body{}"))
	ts := NewTestServer(t, s)

	if body := ts.Body(ts.Get("/admin/users")); body != "users" {
		t.Fatalf("GET users = %q", body)
	}
	if body := ts.Body(ts.Get("/admin/app.css")); body != "body{}" {
		t.Fatalf("GET asset = %q", body)
	}
	if res := ts.Get("/users"); res.StatusCode != http.StatusNotFound {
		body := ts.Body(res)
		t.Fatalf("unscoped users = %d %q, want 404", res.StatusCode, body)
	}
}

func TestNestedScopes(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	api := s.Scope("/api")
	v1 := api.Scope("/v1")
	v1.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "v1", nil
	})
	ts := NewTestServer(t, s)

	res := ts.PostJSON("/api/v1"+RPCPrefix+"Ping", map[string]any{"args": []any{}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"v1"`) {
		t.Fatalf("nested rpc = %d %s", res.StatusCode, body)
	}
	if body := ts.Body(ts.Get("/api/v1" + RuntimePath)); !strings.Contains(body, "export function") {
		t.Fatalf("nested runtime missing")
	}
}

func TestScopeMiddlewareIsScoped(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "root")
	})
	admin := s.Scope("/admin")
	admin.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Scoped", "yes")
			next.ServeHTTP(w, r)
		})
	})
	admin.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin")
	})
	ts := NewTestServer(t, s)

	res := ts.Get("/admin/")
	ts.Body(res)
	if got := res.Header.Get("X-Scoped"); got != "yes" {
		t.Fatalf("admin header = %q", got)
	}

	res = ts.Get("/")
	ts.Body(res)
	if got := res.Header.Get("X-Scoped"); got != "" {
		t.Fatalf("root header = %q, want none", got)
	}
}
