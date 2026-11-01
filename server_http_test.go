package gothic_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	. "github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/gothicest"
)

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestUseMiddlewareCoversFrameworkPaths(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	var runtime, module, rpc atomic.Int32
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == RuntimePath:
				runtime.Add(1)
			case strings.HasPrefix(r.URL.Path, ModulesPrefix):
				module.Add(1)
			case strings.HasPrefix(r.URL.Path, RPCPrefix):
				rpc.Add(1)
			}
			next.ServeHTTP(w, r)
		})
	})
	RegisterModule(ModulesPrefix+"mw.js", []byte("export const x = 1;\n"))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := gothicest.NewTestServer(t, s)

	ts.Body(ts.Get(RuntimePath))
	ts.Body(ts.Get(ModulesPrefix + "mw.js"))
	ts.Body(ts.PostJSON(RPCPrefix+"Ping", map[string]any{"args": []any{}}))

	if runtime.Load() == 0 || module.Load() == 0 || rpc.Load() == 0 {
		t.Fatalf("middleware misses: runtime=%d module=%d rpc=%d", runtime.Load(), module.Load(), rpc.Load())
	}
}

func TestRPCRequiresJSONContentType(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := gothicest.NewTestServer(t, s)

	res, err := ts.Client().Post(ts.URL+RPCPrefix+"Ping", "text/plain", strings.NewReader(`{"args":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	body := ts.Body(res)
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body %s", res.StatusCode, body)
	}
	if !strings.Contains(body, "unsupported_media_type") {
		t.Fatalf("body = %s", body)
	}
}

func TestRPCCrossOriginRejected(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := gothicest.NewTestServer(t, s)

	req := mustRequest(t, http.MethodPost, ts.URL+RPCPrefix+"Ping", map[string]string{
		"Content-Type": "application/json",
		"Origin":       "https://evil.example",
	}, strings.NewReader(`{"args":[]}`))
	res := ts.Do(req)
	body := ts.Body(res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin status = %d, want 403; body %s", res.StatusCode, body)
	}

	origin, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	req = mustRequest(t, http.MethodPost, ts.URL+RPCPrefix+"Ping", map[string]string{
		"Content-Type": "application/json",
		"Origin":       origin.Scheme + "://" + origin.Host,
	}, strings.NewReader(`{"args":[]}`))
	res = ts.Do(req)
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("same-origin status = %d, want 200; body %s", res.StatusCode, body)
	}
}

func TestRPCBodyLimit(t *testing.T) {
	s := NewServer(WithLogger(testLogger()), WithMaxBodyBytes(8))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := gothicest.NewTestServer(t, s)

	res := ts.PostJSON(RPCPrefix+"Ping", map[string]any{"args": []any{}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body %s", res.StatusCode, body)
	}
}

func TestTypedProcedureError(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.RegisterProcedure("Create", func(ctx context.Context, body []byte) (any, error) {
		return nil, Invalid("validation_failed", "check the form", map[string]string{"title": "required"})
	})
	s.RegisterProcedure("Boom", func(ctx context.Context, body []byte) (any, error) {
		return nil, io.ErrUnexpectedEOF
	})
	ts := gothicest.NewTestServer(t, s)

	_, err := ts.Call("Create")
	var gothicErr *Error
	if !asTestError(err, &gothicErr) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if gothicErr.Status != http.StatusBadRequest || gothicErr.Code != "validation_failed" {
		t.Fatalf("error = %#v", gothicErr)
	}
	if gothicErr.Fields["title"] != "required" {
		t.Fatalf("fields = %#v", gothicErr.Fields)
	}

	_, err = ts.Call("Boom")
	if !asTestError(err, &gothicErr) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if gothicErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", gothicErr.Status)
	}
}

func asTestError(err error, target **Error) bool {
	return errors.As(err, target)
}

func TestRouterVerbsAndParams(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Router().Route("/api/items", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, chi.URLParam(r, "id"))
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	})
	ts := gothicest.NewTestServer(t, s)

	if body := ts.Body(ts.Get("/api/items/42")); body != "42" {
		t.Fatalf("GET = %q", body)
	}
	req := mustRequest(t, http.MethodDelete, ts.URL+"/api/items/42", nil, nil)
	res := ts.Do(req)
	ts.Body(res)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE = %d", res.StatusCode)
	}
}

func TestAsset(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Asset("/style.css", []byte("body{color:red}"))
	ts := gothicest.NewTestServer(t, s)

	res := ts.Get("/style.css")
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "body{color:red}" {
		t.Fatalf("GET = %d %q", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("content type = %q", got)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}
	if got := res.Header.Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("cache control = %q", got)
	}

	req := mustRequest(t, http.MethodGet, ts.URL+"/style.css", map[string]string{"If-None-Match": etag}, nil)
	res = ts.Do(req)
	ts.Body(res)
	if res.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional GET = %d, want 304", res.StatusCode)
	}

	req = mustRequest(t, http.MethodHead, ts.URL+"/style.css", nil, nil)
	res = ts.Do(req)
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "" {
		t.Fatalf("HEAD = %d %q", res.StatusCode, body)
	}
}

func TestServeShutdown(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve(ln) }()

	res, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("GET = %d %q", res.StatusCode, body)
	}

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("serve returned %v, want http.ErrServerClosed", err)
	}
}

func TestPanicRecovery(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Get("/boom", func(w http.ResponseWriter, r *http.Request) {
		panic("kaboom")
	})
	s.RegisterProcedure("Boom", func(ctx context.Context, body []byte) (any, error) {
		panic("kaboom")
	})
	ts := gothicest.NewTestServer(t, s)

	res := ts.Get("/boom")
	body := ts.Body(res)
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("route status = %d, want 500; body %s", res.StatusCode, body)
	}

	res = ts.PostJSON(RPCPrefix+"Boom", map[string]any{"args": []any{}})
	body = ts.Body(res)
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("rpc status = %d, want 500; body %s", res.StatusCode, body)
	}
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("rpc body = %s", body)
	}
}

func TestUseAfterFirstRequest(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	ts := gothicest.NewTestServer(t, s)
	ts.Body(ts.Get("/"))

	var hits atomic.Int32
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			next.ServeHTTP(w, r)
		})
	})
	ts.Body(ts.Get("/"))
	if hits.Load() != 1 {
		t.Fatalf("late middleware hits = %d, want 1", hits.Load())
	}
}

func TestRPCTrustedProxyOrigin(t *testing.T) {
	s := NewServer(WithLogger(testLogger()), WithTrustedProxies("127.0.0.1/32", "::1/128"))
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := gothicest.NewTestServer(t, s)

	req := mustRequest(t, http.MethodPost, ts.URL+RPCPrefix+"Ping", map[string]string{
		"Content-Type":      "application/json",
		"Origin":            "https://public.example",
		"X-Forwarded-Host":  "public.example",
		"X-Forwarded-Proto": "https",
	}, strings.NewReader(`{"args":[]}`))
	res := ts.Do(req)
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", res.StatusCode, body)
	}
}
