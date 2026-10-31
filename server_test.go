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
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("runtime.js cache-control = %q, want no-store", got)
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
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("module cache-control = %q, want no-store", got)
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

type pathModel struct{ path string }

func (m pathModel) Init() Cmd               { return nil }
func (m pathModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m pathModel) View() Component {
	return ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, m.path)
		return err
	})
}

func TestMountAppCatchAll(t *testing.T) {
	s := NewServer()
	s.MountApp("/", NewAppFunc(func(r *http.Request) Model {
		return pathModel{path: r.URL.Path}
	}))
	ts := httptest.NewServer(s)
	defer ts.Close()

	for _, path := range []string{"/", "/about", "/items/42"} {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "<body>\n"+path+"\n</body>") {
			t.Fatalf("GET %s = %d %q", path, res.StatusCode, body)
		}
	}

	app := NewAppFunc(func(r *http.Request) Model { return pathModel{} })
	if err := app.RenderTo(context.Background(), io.Discard); err == nil {
		t.Fatal("RenderTo should fail for a request-scoped app")
	}
}

func TestServerServesDefaultAssets(t *testing.T) {
	s := NewServer()
	ts := httptest.NewServer(s)
	defer ts.Close()

	for _, tc := range []struct {
		path        string
		contentType string
		prefix      string
	}{
		{DefaultIconPath, "image/svg+xml", "<svg"},
		{DefaultIconPNGPath, "image/png", "\x89PNG"},
		{DefaultTouchPath, "image/png", "\x89PNG"},
		{DefaultImagePath, "image/png", "\x89PNG"},
	} {
		res, err := http.Get(ts.URL + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d", tc.path, res.StatusCode)
		}
		if got := res.Header.Get("Content-Type"); got != tc.contentType {
			t.Errorf("GET %s content type = %q, want %q", tc.path, got, tc.contentType)
		}
		if got := res.Header.Get("Cache-Control"); got == "" {
			t.Errorf("GET %s missing Cache-Control", tc.path)
		}
		if !strings.HasPrefix(string(body), tc.prefix) {
			t.Errorf("GET %s body does not start with %q", tc.path, tc.prefix)
		}
	}
}

func TestMountAppDocumentOptions(t *testing.T) {
	s := NewServer()
	s.SetDocument(Document{SiteName: "Gothic"})
	s.MountApp("/", NewApp(pathModel{}),
		WithTitle("Gothicbook"),
		WithDescription("A design system demo."),
		WithStyles("/style.css"),
	)
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	out := string(body)
	for _, want := range []string{
		"<!doctype html>",
		`<title>Gothicbook</title>`,
		`<meta name="description" content="A design system demo.">`,
		`<meta property="og:site_name" content="Gothic">`,
		`<link rel="stylesheet" href="/style.css">`,
		`<meta property="og:image" content="` + ts.URL + `/_gothic/og.png">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document missing %q\n\n%s", want, out)
		}
	}
	// Server-wide styles are replaced for this mount only.
	if strings.Contains(out, "gothic-ui.css") {
		t.Errorf("unexpected server styles\n\n%s", out)
	}
}

func TestMountAppRenderError(t *testing.T) {
	s := NewServer()
	s.MountApp("/", NewApp(failingModel{}))
	ts := httptest.NewServer(s)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.StatusCode)
	}
	if strings.Contains(string(body), "<!doctype html>") {
		t.Fatalf("expected no partial document, got %q", body)
	}
}

type failingModel struct{}

func (failingModel) Init() Cmd               { return nil }
func (failingModel) Update(Msg) (Model, Cmd) { return failingModel{}, nil }
func (failingModel) View() Component {
	return ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return io.ErrUnexpectedEOF
	})
}
