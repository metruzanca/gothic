package gothic

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

// CSRF defaults.
const (
	DefaultCSRFCookieName = "gothic_csrf"
	DefaultCSRFHeader     = "X-Gothic-Csrf"
	DefaultCSRFFormField  = "csrf_token"
)

const csrfMaxAge = 24 * time.Hour

type csrfTokenKey struct{}

// CSRF implements signed double-submit CSRF protection. GET responses mint a
// signed token cookie and expose the token through the page config, the client
// runtime sends it as a header, and forms submit it as a hidden field.
type CSRF struct {
	secret    []byte
	cookie    string
	header    string
	formField string
	secure    bool
	sameSite  http.SameSite
	exempt    []string
}

// CSRFOption configures CSRF protection.
type CSRFOption func(*CSRF)

// WithCSRFSecret sets the signing key. When it is empty, a random key is
// generated at startup.
func WithCSRFSecret(secret []byte) CSRFOption {
	return func(c *CSRF) {
		if len(secret) > 0 {
			c.secret = append([]byte(nil), secret...)
		}
	}
}

// WithCSRFCookieName sets the cookie name.
func WithCSRFCookieName(name string) CSRFOption {
	return func(c *CSRF) {
		if name != "" {
			c.cookie = name
		}
	}
}

// WithCSRFHeader sets the request header name.
func WithCSRFHeader(name string) CSRFOption {
	return func(c *CSRF) {
		if name != "" {
			c.header = name
		}
	}
}

// WithCSRFSecure marks the cookie Secure.
func WithCSRFSecure(secure bool) CSRFOption {
	return func(c *CSRF) { c.secure = secure }
}

// WithCSRFSameSite sets the cookie SameSite policy.
func WithCSRFSameSite(sameSite http.SameSite) CSRFOption {
	return func(c *CSRF) { c.sameSite = sameSite }
}

// WithCSRFExempt skips the token check for paths below the given prefixes.
func WithCSRFExempt(prefixes ...string) CSRFOption {
	return func(c *CSRF) { c.exempt = append(c.exempt, prefixes...) }
}

// NewCSRF returns a CSRF helper with random signing material.
func NewCSRF(opts ...CSRFOption) *CSRF {
	c := &CSRF{
		secret:    make([]byte, 32),
		cookie:    DefaultCSRFCookieName,
		header:    DefaultCSRFHeader,
		formField: DefaultCSRFFormField,
		sameSite:  http.SameSiteLaxMode,
	}
	if _, err := rand.Read(c.secret); err != nil {
		panic("gothic: cannot read random bytes: " + err.Error())
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Middleware ensures a token exists and rejects unsafe requests that do not
// carry it in the configured header or form field.
func (c *CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := c.ensureToken(w, r)
		r = r.WithContext(context.WithValue(r.Context(), csrfTokenKey{}, token))
		if !safeMethod(r.Method) && !c.exempted(r.URL.Path) && !c.valid(r, token) {
			WriteError(w, r, Forbidden("invalid or missing CSRF token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Token returns the token for the request, or an empty string when the
// middleware is not installed.
func (c *CSRF) Token(r *http.Request) string { return CSRFToken(r) }

// Token returns the CSRF token for the request.
func CSRFToken(r *http.Request) string {
	echo, _ := r.Context().Value(csrfTokenKey{}).(string)
	return echo
}

func (c *CSRF) ensureToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(c.cookie); err == nil {
		if c.verify(cookie.Value) {
			return cookie.Value
		}
	}
	token := c.mint()
	http.SetCookie(w, &http.Cookie{
		Name:     c.cookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrfMaxAge / time.Second),
		Expires:  time.Now().Add(csrfMaxAge),
		Secure:   c.secure,
		HttpOnly: true,
		SameSite: c.sameSite,
	})
	return token
}

func (c *CSRF) mint() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("gothic: cannot read random bytes: " + err.Error())
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token + "." + c.sign(token)
}

func (c *CSRF) sign(token string) string {
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (c *CSRF) verify(value string) bool {
	token, signature, ok := strings.Cut(value, ".")
	if !ok || token == "" {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(c.sign(token)))
}

func (c *CSRF) valid(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	if header := r.Header.Get(c.header); header != "" {
		return subtle.ConstantTimeCompare([]byte(header), []byte(token)) == 1
	}
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err == nil {
			return subtle.ConstantTimeCompare([]byte(r.PostForm.Get(c.formField)), []byte(token)) == 1
		}
	}
	return false
}

func (c *CSRF) exempted(path string) bool {
	for _, prefix := range c.exempt {
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// WithCSRF enables signed double-submit CSRF protection. Browser requests must
// send the token from the page config as the configured header, and forms must
// include a hidden csrf_token field.
func WithCSRF(opts ...CSRFOption) Option {
	return func(s *Server) { s.csrf = NewCSRF(opts...) }
}
