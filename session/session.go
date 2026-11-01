// Package session provides cookie-backed server sessions.
//
// A session is a JSON map stored on the server and referenced by a signed
// cookie. The default store keeps sessions in memory. SQLStore keeps them in a
// table in an existing database.
package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultCookieName is the session cookie name.
	DefaultCookieName = "gothic_session"
	// DefaultMaxAge is the session lifetime.
	DefaultMaxAge = 7 * 24 * time.Hour
)

type sessionKey struct{}

// Session is the per-request session state.
type Session struct {
	// ID is the current session identifier.
	ID string
	// IsNew reports whether the session has not been persisted yet.
	IsNew bool

	values     map[string]any
	previousID string
	dirty      bool
	destroyed  bool
	persisted  bool
	expires    time.Time
}

// Get returns a value from the session.
func (s *Session) Get(key string) any {
	if s == nil {
		return nil
	}
	return s.values[key]
}

// GetString returns a string value from the session.
func (s *Session) GetString(key string) string {
	value, _ := s.Get(key).(string)
	return value
}

// Value returns a typed value from the session.
func Value[T any](s *Session, key string) (T, bool) {
	var zero T
	if s == nil {
		return zero, false
	}
	value, ok := s.values[key]
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	return typed, ok
}

// Set stores a value in the session.
func (s *Session) Set(key string, value any) {
	if s.values == nil {
		s.values = map[string]any{}
	}
	s.values[key] = value
	s.dirty = true
}

// Delete removes a value from the session.
func (s *Session) Delete(key string) {
	delete(s.values, key)
	s.dirty = true
}

// Clear removes every value from the session.
func (s *Session) Clear() {
	s.values = map[string]any{}
	s.dirty = true
}

// Regenerate replaces the session identifier and keeps the values. Use it
// after a privilege change to prevent session fixation.
func (s *Session) Regenerate() {
	if s.IsNew {
		return
	}
	s.previousID = s.ID
	s.ID = newID()
	s.IsNew = true
	s.dirty = true
}

// Destroy marks the session for deletion when the response is written.
func (s *Session) Destroy() {
	s.destroyed = true
	s.dirty = true
}

// Store persists sessions. Implementations must be safe for concurrent use.
type Store interface {
	// Load returns the encoded session and whether it exists.
	Load(ctx context.Context, id string) ([]byte, bool, error)
	// Save stores an encoded session.
	Save(ctx context.Context, id string, data []byte, expires time.Time) error
	// Delete removes a session.
	Delete(ctx context.Context, id string) error
}

// Manager loads and saves sessions around a handler.
type Manager struct {
	store    Store
	secret   []byte
	cookie   string
	path     string
	domain   string
	maxAge   time.Duration
	secure   bool
	sameSite http.SameSite
}

// Option configures a Manager.
type Option func(*Manager)

// WithCookieName sets the cookie name.
func WithCookieName(name string) Option {
	return func(m *Manager) {
		if name != "" {
			m.cookie = name
		}
	}
}

// WithSecret sets the HMAC key used to sign session identifiers. When it is
// empty, a random key is generated at startup and sessions do not survive a
// restart.
func WithSecret(secret []byte) Option {
	return func(m *Manager) {
		if len(secret) > 0 {
			m.secret = append([]byte(nil), secret...)
		}
	}
}

// WithPath sets the cookie path.
func WithPath(path string) Option {
	return func(m *Manager) {
		if path != "" {
			m.path = path
		}
	}
}

// WithDomain sets the cookie domain.
func WithDomain(domain string) Option {
	return func(m *Manager) { m.domain = domain }
}

// WithMaxAge sets the session lifetime.
func WithMaxAge(age time.Duration) Option {
	return func(m *Manager) {
		if age > 0 {
			m.maxAge = age
		}
	}
}

// WithSecure marks the cookie Secure.
func WithSecure(secure bool) Option {
	return func(m *Manager) { m.secure = secure }
}

// WithSameSite sets the cookie SameSite policy.
func WithSameSite(sameSite http.SameSite) Option {
	return func(m *Manager) { m.sameSite = sameSite }
}

// NewManager returns a session manager using store. A nil store panics.
func NewManager(store Store, opts ...Option) *Manager {
	if store == nil {
		panic("gothic/session: nil store")
	}
	m := &Manager{
		store:    store,
		cookie:   DefaultCookieName,
		path:     "/",
		maxAge:   DefaultMaxAge,
		sameSite: http.SameSiteLaxMode,
		secret:   randomBytes(32),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// CookieName returns the cookie name.
func (m *Manager) CookieName() string { return m.cookie }

// Middleware loads the session into the request context and persists it after
// the handler runs. Values set before the response is written are included.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := m.load(r)
		r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess))
		cw := &cookieWriter{ResponseWriter: w, persist: func() { m.persist(r, sess, w) }}
		next.ServeHTTP(cw, r)
		m.persist(r, sess, w)
	})
}

// Get returns the session for the request, or nil when the middleware is not
// installed.
func (m *Manager) Get(r *http.Request) *Session { return Get(r) }

// Get returns the session from the request context.
func Get(r *http.Request) *Session {
	sess, _ := r.Context().Value(sessionKey{}).(*Session)
	return sess
}

// FromContext returns the session from the context.
func FromContext(ctx context.Context) *Session {
	sess, _ := ctx.Value(sessionKey{}).(*Session)
	return sess
}

func (m *Manager) load(r *http.Request) *Session {
	if cookie, err := r.Cookie(m.cookie); err == nil {
		if id, ok := m.verify(cookie.Value); ok {
			data, found, err := m.store.Load(r.Context(), id)
			if err == nil && found {
				sess := &Session{ID: id, values: map[string]any{}}
				_ = json.Unmarshal(data, &sess.values)
				if sess.values == nil {
					sess.values = map[string]any{}
				}
				return sess
			}
		}
	}
	return &Session{ID: newID(), IsNew: true, values: map[string]any{}}
}

func (m *Manager) persist(r *http.Request, sess *Session, w http.ResponseWriter) {
	if sess.persisted {
		return
	}
	ctx := r.Context()
	if sess.destroyed {
		sess.persisted = true
		_ = m.store.Delete(ctx, sess.ID)
		if sess.previousID != "" {
			_ = m.store.Delete(ctx, sess.previousID)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     m.cookie,
			Value:    "",
			Path:     m.path,
			Domain:   m.domain,
			MaxAge:   -1,
			Secure:   m.secure,
			HttpOnly: true,
			SameSite: m.sameSite,
		})
		return
	}
	if !sess.dirty && !sess.IsNew {
		return
	}
	if sess.IsNew && !sess.dirty {
		return
	}
	data, err := json.Marshal(sess.values)
	if err != nil {
		return
	}
	sess.expires = time.Now().Add(m.maxAge)
	if err := m.store.Save(ctx, sess.ID, data, sess.expires); err != nil {
		return
	}
	if sess.previousID != "" && sess.previousID != sess.ID {
		_ = m.store.Delete(ctx, sess.previousID)
	}
	sess.persisted = true
	sess.IsNew = false
	sess.dirty = false
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookie,
		Value:    m.sign(sess.ID),
		Path:     m.path,
		Domain:   m.domain,
		MaxAge:   int(m.maxAge / time.Second),
		Expires:  sess.expires,
		Secure:   m.secure,
		HttpOnly: true,
		SameSite: m.sameSite,
	})
}

func (m *Manager) sign(id string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(id))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return id + "." + signature
}

func (m *Manager) verify(value string) (string, bool) {
	id, signature, ok := strings.Cut(value, ".")
	if !ok {
		return "", false
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(id))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(want)) {
		return "", false
	}
	return id, true
}

func newID() string {
	return hex.EncodeToString(randomBytes(16))
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("gothic/session: cannot read random bytes: " + err.Error())
	}
	return b
}

// cookieWriter persists the session before the first response byte so the
// Set-Cookie header is still writable.
type cookieWriter struct {
	http.ResponseWriter
	persist func()
	wrote   bool
}

func (w *cookieWriter) WriteHeader(status int) {
	if !w.wrote {
		w.wrote = true
		w.persist()
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *cookieWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
		w.persist()
	}
	return w.ResponseWriter.Write(p)
}

func (w *cookieWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *cookieWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
