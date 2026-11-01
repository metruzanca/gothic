package gothic

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func csrfTestServer(t *testing.T, opts ...CSRFOption) (*TestServer, *url.URL) {
	t.Helper()
	s := NewServer(WithLogger(testLogger()), WithCSRF(opts...))
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = WriteDocument(w, r, Document{Title: "csrf"}, ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "ok")
			return err
		}))
	})
	s.Post("/submit", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "submitted")
	})
	s.Post("/hooks/github", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hook")
	})
	s.RegisterProcedure("Ping", func(ctx context.Context, body []byte) (any, error) {
		return "pong", nil
	})
	ts := NewTestServer(t, s)
	parsed, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	return ts, parsed
}

func csrfTokenFromJar(t *testing.T, ts *TestServer, u *url.URL) string {
	t.Helper()
	for _, cookie := range ts.Client().Jar.Cookies(u) {
		if cookie.Name == DefaultCSRFCookieName {
			return cookie.Value
		}
	}
	return ""
}

func TestCSRFTokenInDocument(t *testing.T) {
	ts, u := csrfTestServer(t, WithCSRFSecret([]byte("secret")))
	body := ts.Body(ts.Get("/"))

	token := csrfTokenFromJar(t, ts, u)
	if token == "" {
		t.Fatal("missing csrf cookie")
	}
	if !strings.Contains(token, ".") {
		t.Fatalf("unsigned token = %q", token)
	}
	if !strings.Contains(body, `window.__gothic=Object.assign(window.__gothic||{},{"csrf":"`+token+`"})`) {
		t.Fatalf("document missing token config\n%s", body)
	}
}

func TestCSRFRejectsUnsafeRequests(t *testing.T) {
	ts, u := csrfTestServer(t, WithCSRFSecret([]byte("secret")))
	ts.Body(ts.Get("/"))
	token := csrfTokenFromJar(t, ts, u)

	res := ts.PostJSON("/submit", map[string]any{})
	body := ts.Body(res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("missing token = %d %q", res.StatusCode, body)
	}

	req := mustRequest(t, http.MethodPost, ts.URL+"/submit", map[string]string{
		"Content-Type":    "application/json",
		DefaultCSRFHeader: token,
	}, strings.NewReader(`{}`))
	res = ts.Do(req)
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("header token = %d %q", res.StatusCode, body)
	}
}

func TestCSRFAcceptsFormToken(t *testing.T) {
	ts, u := csrfTestServer(t, WithCSRFSecret([]byte("secret")))
	ts.Body(ts.Get("/"))
	token := csrfTokenFromJar(t, ts, u)

	form := url.Values{DefaultCSRFFormField: {token}}
	req := mustRequest(t, http.MethodPost, ts.URL+"/submit", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, strings.NewReader(form.Encode()))
	res := ts.Do(req)
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "submitted" {
		t.Fatalf("form token = %d %q", res.StatusCode, body)
	}
}

func TestCSRFExemptPrefix(t *testing.T) {
	ts, _ := csrfTestServer(t, WithCSRFSecret([]byte("secret")), WithCSRFExempt("/hooks"))
	res := ts.PostJSON("/hooks/github", map[string]any{})
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "hook" {
		t.Fatalf("exempt = %d %q", res.StatusCode, body)
	}
}

func TestCSRFGuardsRPC(t *testing.T) {
	ts, u := csrfTestServer(t, WithCSRFSecret([]byte("secret")))
	ts.Body(ts.Get("/"))
	token := csrfTokenFromJar(t, ts, u)

	res := ts.PostJSON(RPCPrefix+"Ping", map[string]any{"args": []any{}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("rpc without token = %d %q", res.StatusCode, body)
	}

	req := mustRequest(t, http.MethodPost, ts.URL+RPCPrefix+"Ping", map[string]string{
		"Content-Type":    "application/json",
		DefaultCSRFHeader: token,
	}, strings.NewReader(`{"args":[]}`))
	res = ts.Do(req)
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, `"pong"`) {
		t.Fatalf("rpc with token = %d %q", res.StatusCode, body)
	}
}
