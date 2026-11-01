package gothic

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestDefaultHealthEndpoints(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	s.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	for _, path := range []string{DefaultHealthPath, DefaultReadyPath} {
		res := ts.Get(path)
		body := ts.Body(res)
		if res.StatusCode != http.StatusOK || !strings.Contains(body, `"status":"ok"`) {
			t.Fatalf("GET %s = %d %q", path, res.StatusCode, body)
		}
		if got := res.Header.Get("Cache-Control"); got != "no-store" {
			t.Errorf("GET %s cache-control = %q", path, got)
		}
		if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("GET %s content type = %q", path, got)
		}

		res = ts.Do(mustRequest(t, http.MethodHead, ts.URL+path, nil, nil))
		body = ts.Body(res)
		if res.StatusCode != http.StatusOK || body != "" {
			t.Fatalf("HEAD %s = %d %q", path, res.StatusCode, body)
		}
	}

	res := ts.Do(mustRequest(t, http.MethodPost, ts.URL+DefaultHealthPath, nil, nil))
	ts.Body(res)
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST health = %d, want 405", res.StatusCode)
	}
	if got := res.Header.Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("allow = %q", got)
	}
}

func TestReadinessFailure(t *testing.T) {
	s := NewServer(
		WithLogger(testLogger()),
		WithReadiness("/readyz", func(ctx context.Context) error {
			return errors.New("database is down")
		}),
	)
	ts := NewTestServer(t, s)

	res := ts.Get("/readyz")
	body := ts.Body(res)
	if res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, "unavailable") {
		t.Fatalf("readyz = %d %q", res.StatusCode, body)
	}
	if res := ts.Get("/healthz"); res.StatusCode != http.StatusOK {
		ts.Body(res)
		t.Fatalf("healthz = %d, want 200", res.StatusCode)
	}
}

func TestHealthConfigurablePaths(t *testing.T) {
	s := NewServer(
		WithLogger(testLogger()),
		WithHealth("/health"),
		WithReadiness("/ready", nil),
	)
	ts := NewTestServer(t, s)

	if res := ts.Get("/health"); res.StatusCode != http.StatusOK {
		ts.Body(res)
		t.Fatalf("custom health = %d", res.StatusCode)
	}
	if res := ts.Get("/ready"); res.StatusCode != http.StatusOK {
		ts.Body(res)
		t.Fatalf("custom ready = %d", res.StatusCode)
	}
	if res := ts.Get(DefaultHealthPath); res.StatusCode != http.StatusNotFound {
		ts.Body(res)
		t.Fatalf("default health = %d, want 404", res.StatusCode)
	}

	disabled := NewServer(WithLogger(testLogger()), WithoutHealth())
	disabledServer := NewTestServer(t, disabled)
	if res := disabledServer.Get(DefaultHealthPath); res.StatusCode != http.StatusNotFound {
		disabledServer.Body(res)
		t.Fatalf("disabled health = %d, want 404", res.StatusCode)
	}
}

func TestHealthBypassesMiddleware(t *testing.T) {
	s := NewServer(WithLogger(testLogger()))
	var hits int
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			WriteError(w, r, Unauthorized("sign in"))
		})
	})
	s.MountApp("/", NewApp(baseModel{}))
	ts := NewTestServer(t, s)

	if res := ts.Get(DefaultHealthPath); res.StatusCode != http.StatusOK {
		ts.Body(res)
		t.Fatalf("health = %d, want 200", res.StatusCode)
	}
	if hits != 0 {
		t.Fatalf("middleware ran %d times for health", hits)
	}
	if res := ts.Get("/"); res.StatusCode != http.StatusUnauthorized {
		ts.Body(res)
		t.Fatalf("app = %d, want 401", res.StatusCode)
	}
}

type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (h *captureHandler) messages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.records))
	for i, record := range h.records {
		out[i] = record.Message
	}
	return out
}

func TestHealthDoesNotLogRequests(t *testing.T) {
	capture := &captureHandler{}
	s := NewServer(WithLogger(slog.New(capture)))
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	ts := NewTestServer(t, s)

	ts.Body(ts.Get(DefaultHealthPath))
	ts.Body(ts.Get(DevToolsWorkspacePath))
	ts.Body(ts.Get("/"))

	requests := 0
	for _, message := range capture.messages() {
		if message == "request" {
			requests++
		}
	}
	if requests != 1 {
		t.Fatalf("request logs = %d, want 1 (health and devtools should be silent)", requests)
	}
}

func TestReadinessPanicIsContained(t *testing.T) {
	s := NewServer(
		WithLogger(testLogger()),
		WithReadiness("/readyz", func(context.Context) error {
			panic("boom")
		}),
	)
	ts := NewTestServer(t, s)
	res := ts.Get("/readyz")
	body := ts.Body(res)
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body %s", res.StatusCode, body)
	}
}
