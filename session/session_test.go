package session_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/metruzanca/gothic/session"
	_ "modernc.org/sqlite"
)

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func get(t *testing.T, client *http.Client, url string) string {
	t.Helper()
	res, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestSessionRoundTrip(t *testing.T) {
	store := session.NewMemoryStore()
	manager := session.NewManager(store, session.WithSecret([]byte("test-secret")))
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.Get(r)
		if value := r.URL.Query().Get("set"); value != "" {
			sess.Set("name", value)
		}
		_, _ = io.WriteString(w, sess.GetString("name"))
	}))
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := newClient()

	if body := get(t, client, srv.URL); body != "" {
		t.Fatalf("anonymous = %q", body)
	}
	if body := get(t, client, srv.URL+"/?set=ada"); body != "ada" {
		t.Fatalf("set = %q", body)
	}
	if body := get(t, client, srv.URL); body != "ada" {
		t.Fatalf("read = %q", body)
	}
	if store.Len() != 1 {
		t.Fatalf("store len = %d", store.Len())
	}
}

func TestSessionTamperedCookieIsIgnored(t *testing.T) {
	store := session.NewMemoryStore()
	manager := session.NewManager(store, session.WithSecret([]byte("test-secret")))
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.Get(r)
		if sess.IsNew {
			_, _ = io.WriteString(w, "new")
			return
		}
		_, _ = io.WriteString(w, "existing")
	}))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: "forged.signature"})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if string(body) != "new" {
		t.Fatalf("body = %q, want new session", body)
	}
}

func TestSessionRegenerateAndDestroy(t *testing.T) {
	store := session.NewMemoryStore()
	manager := session.NewManager(store, session.WithSecret([]byte("test-secret")))
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.Get(r)
		switch r.URL.Query().Get("do") {
		case "set":
			sess.Set("name", "ada")
		case "regen":
			sess.Regenerate()
			sess.Set("name", "ada")
		case "destroy":
			sess.Destroy()
		}
		_, _ = io.WriteString(w, sess.ID)
	}))
	srv := httptest.NewServer(handler)
	defer srv.Close()
	client := newClient()

	first := get(t, client, srv.URL+"?do=set")
	second := get(t, client, srv.URL+"?do=regen")
	if first == second {
		t.Fatalf("session id did not change: %s", first)
	}
	if store.Len() != 1 {
		t.Fatalf("store len after regenerate = %d, want 1", store.Len())
	}
	get(t, client, srv.URL+"?do=destroy")
	if store.Len() != 0 {
		t.Fatalf("store len after destroy = %d, want 0", store.Len())
	}
	if id := get(t, client, srv.URL); id == second {
		t.Fatalf("destroyed session id reused: %s", id)
	}
}

func TestSQLStore(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	store, err := session.NewSQLStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	expires := time.Now().Add(time.Hour)
	if err := store.Save(ctx, "abc", []byte(`{"name":"ada"}`), expires); err != nil {
		t.Fatal(err)
	}
	data, ok, err := store.Load(ctx, "abc")
	if err != nil || !ok || string(data) != `{"name":"ada"}` {
		t.Fatalf("load = %q %v %v", data, ok, err)
	}
	if err := store.Save(ctx, "abc", []byte(`{"name":"grace"}`), expires); err != nil {
		t.Fatal(err)
	}
	data, _, err = store.Load(ctx, "abc")
	if err != nil || string(data) != `{"name":"grace"}` {
		t.Fatalf("update = %q %v", data, err)
	}
	if err := store.Delete(ctx, "abc"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Load(ctx, "abc"); ok {
		t.Fatal("session survived delete")
	}
	if err := store.Save(ctx, "old", []byte(`{}`), time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if removed, err := store.Cleanup(ctx, time.Now()); err != nil || removed != 1 {
		t.Fatalf("cleanup = %d %v", removed, err)
	}
}
