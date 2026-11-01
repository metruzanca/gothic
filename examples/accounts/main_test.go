package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/metruzanca/gothic/gothicest"
	_ "modernc.org/sqlite"
)

func newTestServer(t *testing.T) *gothicest.TestServer {
	t.Helper()
	db, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := newServer(db)
	if err != nil {
		t.Fatal(err)
	}
	return gothicest.NewTestServer(t, s)
}

var tokenPattern = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

func tokenFrom(t *testing.T, body string) string {
	t.Helper()
	match := tokenPattern.FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("no csrf token in\n%s", body)
	}
	return match[1]
}

func postForm(t *testing.T, ts *gothicest.TestServer, path string, values url.Values) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return ts.Do(req)
}

func TestLoginLogoutFlow(t *testing.T) {
	ts := newTestServer(t)

	loginPage := ts.Body(ts.Get("/login"))
	token := tokenFrom(t, loginPage)
	if !strings.Contains(loginPage, "Sign in") {
		t.Fatalf("login page missing heading\n%s", loginPage)
	}

	res := postForm(t, ts, "/login", url.Values{
		"email":      {demoEmail},
		"password":   {demoPassword},
		"csrf_token": {token},
	})
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, "Hello, Demo") {
		t.Fatalf("login = %d\n%s", res.StatusCode, body)
	}

	token = tokenFrom(t, body)
	res = postForm(t, ts, "/logout", url.Values{"csrf_token": {token}})
	body = ts.Body(res)
	if res.StatusCode != http.StatusOK || !strings.Contains(body, "Sign in") {
		t.Fatalf("logout = %d\n%s", res.StatusCode, body)
	}
}

func TestLoginRequiresCSRF(t *testing.T) {
	ts := newTestServer(t)
	ts.Body(ts.Get("/login"))
	res := postForm(t, ts, "/login", url.Values{
		"email":    {demoEmail},
		"password": {demoPassword},
	})
	body := ts.Body(res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403\n%s", res.StatusCode, body)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	ts := newTestServer(t)
	token := tokenFrom(t, ts.Body(ts.Get("/login")))
	res := postForm(t, ts, "/login", url.Values{
		"email":      {demoEmail},
		"password":   {"wrong"},
		"csrf_token": {token},
	})
	body := ts.Body(res)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401\n%s", res.StatusCode, body)
	}
	if !strings.Contains(body, "do not match") {
		t.Fatalf("missing error message\n%s", body)
	}
}

func TestProtectedPageRedirects(t *testing.T) {
	ts := newTestServer(t)
	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res := ts.Get("/")
	ts.Body(res)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", res.StatusCode)
	}
	if got := res.Header.Get("Location"); !strings.HasPrefix(got, "/login?next=") {
		t.Fatalf("location = %q", got)
	}
}
