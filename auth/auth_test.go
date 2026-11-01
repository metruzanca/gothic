package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/auth"
	"github.com/metruzanca/gothic/session"
	_ "modernc.org/sqlite"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash = %q", hash)
	}
	if !auth.CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("valid password rejected")
	}
	if auth.CheckPassword(hash, "wrong password") {
		t.Fatal("invalid password accepted")
	}
	if auth.CheckPassword("not-a-hash", "anything") {
		t.Fatal("malformed hash accepted")
	}
	if _, err := auth.HashPassword(""); !errors.Is(err, auth.ErrEmptyPassword) {
		t.Fatalf("empty password error = %v", err)
	}
}

func authServer(t *testing.T) (*auth.Manager, *gothic.TestServer) {
	t.Helper()
	sessions := session.NewManager(session.NewMemoryStore(), session.WithSecret([]byte("test-secret")))
	store := auth.NewMemoryStore()
	manager := auth.NewManager(sessions, store)
	if _, err := manager.CreateUser(context.Background(), "ada@example.com", "Ada", "secret"); err != nil {
		t.Fatal(err)
	}

	s := gothic.NewServer(gothic.WithLogger(slog.New(slog.DiscardHandler)))
	s.Use(sessions.Middleware)
	s.Use(manager.Middleware)
	s.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		form, err := gothic.ParseForm(w, r)
		if err != nil {
			gothic.WriteError(w, r, err)
			return
		}
		user, err := manager.Login(r, form.String("email"), form.String("password"))
		if err != nil {
			gothic.WriteError(w, r, gothic.Unauthorized("invalid email or password"))
			return
		}
		_, _ = io.WriteString(w, "welcome "+user.Name)
	})
	s.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
		if err := manager.Logout(r); err != nil {
			gothic.WriteError(w, r, err)
			return
		}
		_, _ = io.WriteString(w, "bye")
	})
	me := manager.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := manager.CurrentUser(r)
		_, _ = io.WriteString(w, user.Name)
	}))
	s.Handle("/me", me)
	redirect := manager.RequireUserRedirect("/login")
	s.Handle("/private", redirect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "private")
	})))
	return manager, gothic.NewTestServer(t, s)
}

func postForm(t *testing.T, ts *gothic.TestServer, path string, values url.Values) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return ts.Do(req)
}

func TestLoginFlow(t *testing.T) {
	_, ts := authServer(t)

	res := postForm(t, ts, "/login", url.Values{"email": {"ada@example.com"}, "password": {"secret"}})
	body := ts.Body(res)
	if res.StatusCode != http.StatusOK || body != "welcome Ada" {
		t.Fatalf("login = %d %q", res.StatusCode, body)
	}
	if body := ts.Body(ts.Get("/me")); body != "Ada" {
		t.Fatalf("me = %q", body)
	}

	req := mustRequest(t, http.MethodPost, ts.URL+"/logout", nil, nil)
	res = ts.Do(req)
	ts.Body(res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("logout = %d", res.StatusCode)
	}
	if res := ts.Get("/me"); res.StatusCode != http.StatusUnauthorized {
		ts.Body(res)
		t.Fatalf("me after logout = %d, want 401", res.StatusCode)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	_, ts := authServer(t)

	res := postForm(t, ts, "/login", url.Values{"email": {"ada@example.com"}, "password": {"wrong"}})
	ts.Body(res)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", res.StatusCode)
	}

	res = postForm(t, ts, "/login", url.Values{"email": {"nobody@example.com"}, "password": {"secret"}})
	ts.Body(res)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown user status = %d", res.StatusCode)
	}
}

func TestRequireUserRedirect(t *testing.T) {
	_, ts := authServer(t)
	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res := ts.Get("/private")
	ts.Body(res)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if got := res.Header.Get("Location"); !strings.HasPrefix(got, "/login?next=") {
		t.Fatalf("location = %q", got)
	}
}

func TestSQLStore(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	store, err := auth.NewSQLStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser(ctx, "Ada@Example.com", "Ada", hash)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "ada@example.com" {
		t.Fatalf("email = %q", user.Email)
	}
	found, stored, err := store.UserByEmail(ctx, "ADA@example.com")
	if err != nil || found.ID != user.ID || stored != hash {
		t.Fatalf("by email = %#v %q %v", found, stored, err)
	}
	if _, err := store.CreateUser(ctx, "ada@example.com", "Other", hash); !errors.Is(err, auth.ErrEmailTaken) {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := store.UserByID(ctx, "missing"); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func mustRequest(t *testing.T, method, target string, headers map[string]string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req
}
