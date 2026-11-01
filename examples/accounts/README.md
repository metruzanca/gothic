# Accounts example

A small session and login app with no client JavaScript. It shows the pieces
you need before building your own auth: server sessions, CSRF-protected forms,
argon2id password hashes, and user middleware.

## Run

```sh
go run ./examples/accounts
```

Open <http://localhost:8080>. The demo account is `demo@example.com` with
password `gothic`.

The example uses an in-memory SQLite database by default, so users and sessions
reset on every run. Set `DB_PATH=/path/to/accounts.db` to persist them. Set
`SESSION_SECRET` to a stable value, otherwise a random key is generated at
startup and old cookies stop validating after a restart.

## What it shows

- `gothic/session` with the SQL store, signed cookies, and `HttpOnly` plus
  `SameSite=Lax` defaults.
- `gothic.WithCSRF` with a hidden `csrf_token` field on every form.
- `gothic/auth` with argon2id hashes, `Login`, `Logout`, `RequireUserRedirect`,
  and session regeneration on sign in.
- `gothic.ParseForm` and field validation that re-renders the form with an
  error message.
- `gothic.WriteDocument` for plain server-rendered pages without a `.gth` app.

## Routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/login` | Sign-in form |
| `POST` | `/login` | Check credentials and start a session |
| `POST` | `/logout` | Destroy the session |
| `GET` | `/` | Account page, requires a session |

## Notes

CSRF protection is opt-in. When it is enabled, every state-changing request
needs the token: the client runtime sends it as `X-Gothic-Csrf`, and plain
forms submit it as `csrf_token`. Requests without a token get a 403. Use
`gothic.WithCSRFExempt` for webhook paths.

Sessions are server-side. Only a signed identifier is stored in the cookie, so
rotating `SESSION_SECRET` invalidates every session.
