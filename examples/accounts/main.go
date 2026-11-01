// Command accounts is a small session and login example. It uses SQLite for
// users and sessions, argon2id password hashes, CSRF-protected forms, and
// server-rendered pages with no client JavaScript.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"errors"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/auth"
	"github.com/metruzanca/gothic/session"
	_ "modernc.org/sqlite"
)

const (
	demoEmail    = "demo@example.com"
	demoPassword = "gothic"
)

//go:embed style.css
var styleCSS []byte

type app struct {
	auth *auth.Manager
}

func main() {
	db, err := openDB(dbPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s, err := newServer(db)
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func dbPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return ":memory:"
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// A single connection keeps an in-memory database shared.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func newServer(db *sql.DB) (*gothic.Server, error) {
	sessionStore, err := session.NewSQLStore(db)
	if err != nil {
		return nil, err
	}
	userStore, err := auth.NewSQLStore(db)
	if err != nil {
		return nil, err
	}

	secret := []byte(os.Getenv("SESSION_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		log.Printf("gothic: SESSION_SECRET is not set, sessions end when the server restarts")
	}

	sessions := session.NewManager(sessionStore, session.WithSecret(secret))
	manager := auth.NewManager(sessions, userStore)
	if err := seed(manager); err != nil {
		return nil, err
	}
	a := &app{auth: manager}

	s := gothic.NewServer(
		gothic.WithCSRF(gothic.WithCSRFSecret(secret)),
	)
	s.Asset("/style.css", styleCSS)
	s.Use(sessions.Middleware)
	s.Use(manager.Middleware)

	s.Get("/login", a.loginPage)
	s.Post("/login", a.login)
	s.Post("/logout", a.logout)
	s.Handle("/", manager.RequireUserRedirect("/login")(http.HandlerFunc(a.home)))
	return s, nil
}

func seed(manager *auth.Manager) error {
	_, _, err := manager.Store.UserByEmail(context.Background(), demoEmail)
	if err == nil {
		return nil
	}
	if !errors.Is(err, auth.ErrNotFound) {
		return err
	}
	if _, err := manager.CreateUser(context.Background(), demoEmail, "Demo", demoPassword); err != nil {
		return err
	}
	log.Printf("gothic: demo account %s / %s", demoEmail, demoPassword)
	return nil
}

func (a *app) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.auth.CurrentUser(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	next := safeNext(r.URL.Query().Get("next"))
	a.renderLogin(w, r, http.StatusOK, "", next, "")
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	form, err := gothic.ParseForm(w, r)
	if err != nil {
		gothic.WriteError(w, r, err)
		return
	}
	email := form.String("email")
	password := form.String("password")
	next := safeNext(form.String("next"))
	if email == "" {
		form.AddError("email", "Enter your email.")
	}
	if password == "" {
		form.AddError("password", "Enter your password.")
	}
	if err := form.Err(); err != nil {
		a.renderLogin(w, r, http.StatusBadRequest, email, next, "Enter your email and password.")
		return
	}
	user, err := a.auth.Login(r, email, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			a.renderLogin(w, r, http.StatusUnauthorized, email, next, "That email and password do not match.")
			return
		}
		gothic.WriteError(w, r, gothic.Internal("internal server error", err))
		return
	}
	log.Printf("gothic: %s signed in", user.Email)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.Logout(r); err != nil {
		gothic.WriteError(w, r, gothic.Internal("internal server error", err))
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	user, ok := a.auth.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var body strings.Builder
	body.WriteString(`<main class="card"><p class="eyebrow">Signed in</p><h1>Hello, `)
	body.WriteString(html.EscapeString(user.Name))
	body.WriteString(`</h1><p class="muted">`)
	body.WriteString(html.EscapeString(user.Email))
	body.WriteString(`</p><form method="post" action="/logout">`)
	writeCSRF(&body, r)
	body.WriteString(`<button type="submit">Sign out</button></form></main>`)
	writePage(w, r, http.StatusOK, "Account", body.String())
}

func (a *app) renderLogin(w http.ResponseWriter, r *http.Request, status int, email, next, message string) {
	var body strings.Builder
	body.WriteString(`<main class="card"><p class="eyebrow">Accounts</p><h1>Sign in</h1>`)
	if message != "" {
		body.WriteString(`<p class="error">`)
		body.WriteString(html.EscapeString(message))
		body.WriteString(`</p>`)
	}
	body.WriteString(`<form method="post" action="/login">`)
	writeCSRF(&body, r)
	body.WriteString(`<input type="hidden" name="next" value="`)
	body.WriteString(html.EscapeString(next))
	body.WriteString(`"><label>Email<input type="email" name="email" autocomplete="username" value="`)
	body.WriteString(html.EscapeString(email))
	body.WriteString(`"></label><label>Password<input type="password" name="password" autocomplete="current-password"></label>`)
	body.WriteString(`<button type="submit">Sign in</button></form>`)
	body.WriteString(`<p class="hint">Demo account: ` + demoEmail + ` / ` + demoPassword + `</p></main>`)
	writePage(w, r, status, "Sign in", body.String())
}

func writeCSRF(body *strings.Builder, r *http.Request) {
	body.WriteString(`<input type="hidden" name="csrf_token" value="`)
	body.WriteString(html.EscapeString(gothic.CSRFToken(r)))
	body.WriteString(`">`)
}

func writePage(w http.ResponseWriter, r *http.Request, status int, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	doc := gothic.Document{
		Title:       title,
		Description: "Session, CSRF, and password example for gothic.",
		Styles:      []string{"/style.css"},
	}
	_ = gothic.WriteDocument(w, r, doc, gothic.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, body)
		return err
	}))
}

// safeNext only allows same-origin absolute paths.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}
