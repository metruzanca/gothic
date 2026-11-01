package auth

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/session"
)

// DefaultSessionKey is the session value that holds the user ID.
const DefaultSessionKey = "gothic_user"

// Manager connects a user store to sessions.
type Manager struct {
	Sessions   *session.Manager
	Store      Store
	SessionKey string
}

// NewManager returns a manager using sessions and store.
func NewManager(sessions *session.Manager, store Store) *Manager {
	return &Manager{Sessions: sessions, Store: store, SessionKey: DefaultSessionKey}
}

type userKey struct{}

// Middleware loads the current user into the request context. Install it after
// the session middleware.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, ok := m.loadUser(r); ok {
			r = r.WithContext(context.WithValue(r.Context(), userKey{}, user))
		}
		next.ServeHTTP(w, r)
	})
}

// CurrentUser returns the authenticated user, or false when the request is
// anonymous.
func (m *Manager) CurrentUser(r *http.Request) (User, bool) {
	if user, ok := r.Context().Value(userKey{}).(User); ok {
		return user, true
	}
	return m.loadUser(r)
}

// Login checks the credentials, regenerates the session, and stores the user
// ID. The session middleware persists and sets the cookie.
func (m *Manager) Login(r *http.Request, email, password string) (User, error) {
	if m.Store == nil {
		return User{}, errors.New("auth: nil store")
	}
	user, hash, err := m.Store.UserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Run the same work for unknown users so the response time does
			// not reveal whether the email exists.
			CheckPassword(dummyHash(), password)
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}
	if !CheckPassword(hash, password) {
		return User{}, ErrInvalidCredentials
	}
	sess := m.Sessions.Get(r)
	if sess == nil {
		return User{}, errors.New("auth: session middleware is not installed")
	}
	sess.Regenerate()
	sess.Set(m.key(), user.ID)
	return user, nil
}

// Logout destroys the session.
func (m *Manager) Logout(r *http.Request) error {
	sess := m.Sessions.Get(r)
	if sess == nil {
		return errors.New("auth: session middleware is not installed")
	}
	sess.Destroy()
	return nil
}

// CreateUser hashes password and stores a new user.
func (m *Manager) CreateUser(ctx context.Context, email, name, password string) (User, error) {
	if m.Store == nil {
		return User{}, errors.New("auth: nil store")
	}
	email = NormalizeEmail(email)
	if email == "" {
		return User{}, errors.New("auth: email is empty")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	return m.Store.CreateUser(ctx, email, name, hash)
}

// RequireUser rejects anonymous requests with 401.
func (m *Manager) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := m.CurrentUser(r); !ok {
			gothic.WriteError(w, r, gothic.Unauthorized("sign in required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireUserRedirect sends anonymous requests to loginPath.
func (m *Manager) RequireUserRedirect(loginPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := m.CurrentUser(r); !ok {
				target := loginPath
				if target == "" {
					target = "/login"
				}
				if r.Method == http.MethodGet {
					target += "?next=" + url.QueryEscape(r.URL.RequestURI())
				}
				http.Redirect(w, r, target, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *Manager) loadUser(r *http.Request) (User, bool) {
	if m.Store == nil || m.Sessions == nil {
		return User{}, false
	}
	sess := m.Sessions.Get(r)
	if sess == nil {
		return User{}, false
	}
	id := strings.TrimSpace(sess.GetString(m.key()))
	if id == "" {
		return User{}, false
	}
	user, err := m.Store.UserByID(r.Context(), id)
	if err != nil {
		return User{}, false
	}
	return user, true
}

func (m *Manager) key() string {
	if m.SessionKey != "" {
		return m.SessionKey
	}
	return DefaultSessionKey
}

var dummy struct {
	once sync.Once
	hash string
}

func dummyHash() string {
	dummy.once.Do(func() {
		hash, err := HashPassword("gothic-dummy-password")
		if err == nil {
			dummy.hash = hash
		}
	})
	return dummy.hash
}
