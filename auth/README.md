# gothic/auth

Users, argon2id password hashes, session-backed login, and middleware for
gothic. This is an optional module. It depends on core gothic and
`github.com/metruzanca/gothic/session`.

```sh
go get github.com/metruzanca/gothic/auth
```

## Use

```go
import (
	"github.com/metruzanca/gothic/auth"
	"github.com/metruzanca/gothic/session"
)

store, err := auth.NewSQLStore(db)
sessions := session.NewManager(session.NewSQLStore(db), session.WithSecret(secret))
manager := auth.NewManager(sessions, store)

s.Use(sessions.Middleware)
s.Use(manager.Middleware)
```

```go
user, err := manager.CreateUser(ctx, "ada@example.com", "Ada", password)
user, err = manager.Login(r, "ada@example.com", password)
err = manager.Logout(r)
user, ok := manager.CurrentUser(r)
```

`Login` regenerates the session after a successful sign-in. Bad credentials
return `auth.ErrInvalidCredentials`, and unknown emails run the same password
check so response time does not reveal whether an account exists.

## Protecting routes

```go
s.Handle("/private", manager.RequireUser(private))
s.Handle("/account", manager.RequireUserRedirect("/login")(account))
```

`RequireUser` returns 401. `RequireUserRedirect` sends browsers to the login page
with a `next` query parameter.

## Store

- `auth.NewMemoryStore()` keeps users in process memory.
- `auth.NewSQLStore(db)` creates a `gothic_users` table.

A custom store implements `auth.Store`:

```go
type Store interface {
	UserByEmail(ctx context.Context, email string) (User, string, error)
	UserByID(ctx context.Context, id string) (User, error)
	CreateUser(ctx context.Context, email, name, passwordHash string) (User, error)
}
```

Passwords are hashed with argon2id by `auth.HashPassword` and checked with
`auth.CheckPassword`. The store returns the hash to the manager.

## Errors

- `auth.ErrNotFound`
- `auth.ErrEmailTaken`
- `auth.ErrInvalidCredentials`
- `auth.ErrEmptyPassword`

## Forms

This module does not ship form parsing or validation. Use `net/http`:

```go
if err := r.ParseForm(); err != nil {
	gothic.WriteError(w, r, gothic.BadRequest("invalid form data"))
	return
}
email := strings.TrimSpace(r.PostForm.Get("email"))
password := r.PostForm.Get("password")
```

A dedicated form package is planned.

See `examples/accounts` for a complete login flow with sessions, CSRF, and SQL
stores.
