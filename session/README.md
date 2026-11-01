# gothic/session

Server sessions with a signed cookie identifier. Values live on the server; the
browser only receives the signed id. This is an optional module, separate from
core gothic.

```sh
go get github.com/metruzanca/gothic/session
```

## Use

```go
import "github.com/metruzanca/gothic/session"

store := session.NewMemoryStore() // lost on restart
sessions := session.NewManager(store,
	session.WithSecret(secret),
	session.WithSecure(true),
	session.WithSameSite(http.SameSiteLaxMode),
)
s.Use(sessions.Middleware)
```

```go
sess := session.Get(r)
sess.Set("theme", "dark")
theme, ok := session.Value[string](sess, "theme")
sess.Regenerate() // after a privilege change
sess.Destroy()    // at logout
```

Values are JSON round-tripped. Prefer strings, booleans, and plain structs.
Numbers decode as `float64`.

## Stores

- `session.NewMemoryStore()` keeps sessions in process memory. Lost on restart.
- `session.NewSQLStore(db)` creates a `gothic_sessions` table in an existing
  `*sql.DB`.

A custom store implements `session.Store`:

```go
type Store interface {
	Load(ctx context.Context, id string) ([]byte, bool, error)
	Save(ctx context.Context, id string, data []byte, expires time.Time) error
	Delete(ctx context.Context, id string) error
}
```

## Options

- `WithSecret([]byte)`: signing secret. Rotating it invalidates every cookie.
- `WithCookieName(string)`: cookie name, default `gothic_session`.
- `WithPath(string)`: cookie path, default `/`.
- `WithDomain(string)`: cookie domain.
- `WithMaxAge(time.Duration)`: lifetime, default seven days.
- `WithSecure(bool)`: sets the `Secure` cookie flag.
- `WithSameSite(http.SameSite)`: default `SameSite=Lax`.

The cookie is `HttpOnly` by default. Sessions are written before the first
response byte, so cookies survive redirects and early writes.

## Limits

- stored sessions have a fixed lifetime and are not extended on each request,
- server-side storage means a shared store is required for multiple processes,
- the module does not ship CSRF protection. Use core `gothic.WithCSRF` or your
  own middleware.
