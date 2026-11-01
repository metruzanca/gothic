# 08 The server

Gothic ships an HTTP server built on chi. You get procedures for your app, and
you also get ordinary routes, middleware, static files, and graceful shutdown
without adding a framework on top.

## Creating a server

```go
s := gothic.NewServer()
s.MountApp("/", gothic.NewApp(NewModel()))
log.Fatal(s.ListenAndServe(":8080"))
```

Options are applied in order before any route is registered:

```go
s := gothic.NewServer(
	gothic.WithAddr(":3000"),
	gothic.WithLogger(logger),
	gothic.WithMaxBodyBytes(4 << 20),
	gothic.WithRequestTimeout(10*time.Second),
	gothic.WithTrustedProxies("10.0.0.0/8"),
)
```

`WithAddr` is the address `Run` uses and the fallback for `ListenAndServe("")`.

## Lifecycle

`ListenAndServe` keeps the simple path. `Run` adds signal handling and a
drained shutdown:

```go
if err := s.Run(context.Background()); err != nil {
	log.Fatal(err)
}
```

`Run` listens on the configured address, serves, and waits for SIGINT or
SIGTERM. It then calls `Shutdown` with `WithShutdownTimeout`, which defaults to
15 seconds. For manual control use `Serve(ln)`, `ServeTLS(ln, cert, key)`,
`Shutdown(ctx)`, and `Close()`.

The underlying `http.Server` has real timeouts. Defaults are 5s to read
headers, 30s to read the request, 30s to write the response, and 120s idle. Each
one can be changed with an option. Long-lived streams should raise the write
timeout or bypass it.

## Middleware

`Use` adds middleware to the global chain. It covers the app routes, the client
runtime, generated modules, and procedures:

```go
s.Use(func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			gothic.WriteError(w, r, gothic.Unauthorized("sign in first"))
			return
		}
		next.ServeHTTP(w, r)
	})
})
```

Unlike chi, `Use` can be called after routes are registered, and it applies to
framework paths. Middleware added through `s.Router()` covers only routes
registered through that router, so prefer `Use` for auth and logging.

Defaults are a request ID, a recoverer, and an access log through `slog`.
Disable them with `WithoutDefaultMiddleware`, or add your own with
`WithMiddleware`. `WithRequestTimeout` bounds the request context, so handlers
must honor the context to observe it. `WithTrustedProxies` is required before
`X-Forwarded-For` and `X-Forwarded-Host` are believed.

## Health checks

Liveness and readiness endpoints are on by default:

| Path | Purpose |
|---|---|
| `/healthz` | Liveness. Returns `{"status":"ok"}` with no dependencies. |
| `/readyz` | Readiness. Returns 200, or 503 when the check fails. |

```go
s := gothic.NewServer(
	gothic.WithReadiness("/readyz", func(ctx context.Context) error {
		return db.PingContext(ctx)
	}),
)
```

The readiness check receives a request-scoped context with a five second
deadline. Failure returns `{"status":"unavailable"}` and logs the cause. Both
endpoints support `HEAD`, send `Cache-Control: no-store`, and are handled before
the middleware chain, so auth, CSRF, and access logging do not apply.

Change the paths with `WithHealth("/health")` and
`WithReadiness("/ready", check)`. An empty path disables one endpoint, and
`WithoutHealth()` disables both.

## Dev mode

`WithDevMode` enables development-only behavior and records the project
directory:

```go
s := gothic.NewServer(gothic.WithDevMode("."))
```

In dev mode, Chrome DevTools workspace discovery is answered at
`/.well-known/appspecific/com.chrome.devtools.json` with the absolute project
root and a stable UUID. DevTools can then map the served site to local files.

When dev mode is off, the same path returns a JSON 404 instead of falling
through to the app catch-all, so the probe does not receive HTML. The response
is handled before middleware and is not access logged.

Do not enable dev mode in production: the response contains an absolute
filesystem path.

## Routes

`Server` wraps the common chi methods:

```go
s.Get("/health", func(w http.ResponseWriter, r *http.Request) {
	_ = gothic.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
})

s.Route("/api/items", func(r chi.Router) {
	r.Get("/{id}", getItem)
	r.Delete("/{id}", deleteItem)
})
```

`Put`, `Patch`, `Delete`, `Head`, `Options`, `Method`, `Mount`, `Handle`,
`Route`, `With`, `Group`, `NotFound`, and `MethodNotAllowed` are available.
Path parameters come from `chi.URLParam`. Use `Router()` when you need the full
router.

A mounted app installs a catch-all below its pattern for deep links. Register
API routes under a more specific prefix so they are matched first.

## Procedures

Procedures are declared in the app file and registered by generated code:

```go
type Procedures interface {
	gothic.RPCService
	CreateTodo(ctx context.Context, title string) error
}
```

Clients call them over `POST /_gothic/rpc/<Name>` with `{"args":[...]}`. The
server enforces the transport rules:

- `POST` only,
- `Content-Type: application/json` (or a `+json` suffix),
- same-origin `Origin` and `Sec-Fetch-Site` when the browser sends them,
- a body limit that defaults to 1 MiB,
- one JSON error envelope for every failure.

Requests without an `Origin` header are allowed, so command-line clients and
server-to-server calls work.

## Errors

Return a `*gothic.Error` to choose the status, the machine code, and the field
messages:

```go
func CreateTodo(ctx context.Context, title string) error {
	if title == "" {
		return gothic.Invalid("validation_failed", "check the form", map[string]string{
			"title": "required",
		})
	}
	return nil
}
```

The response is:

```json
{"error":{"message":"check the form","code":"validation_failed","fields":{"title":"required"}}}
```

The client runtime throws an `Error` with `status`, `code`, and `fields`
attached, so `Update` can branch on the code or show field messages.

Plain errors become 500 responses with their message preserved. Wrap sensitive
errors with `gothic.Internal("internal server error", err)` so the cause is
logged and not sent.

Constructors: `BadRequest`, `Invalid`, `Unauthorized`, `Forbidden`, `NotFound`,
`Conflict`, `Internal`, and `Errorf`.

## JSON helpers

```go
func createItem(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := gothic.DecodeJSON(w, r, &input); err != nil {
		gothic.WriteError(w, r, err)
		return
	}
	_ = gothic.JSON(w, http.StatusCreated, input)
}
```

`DecodeJSON` requires a JSON content type, enforces the default limit, rejects
trailing data, and returns an `*Error`. `DecodeJSONLimit` takes a custom limit.
`WriteError` negotiates JSON or plain text from the `Accept` header.

## Static assets

`Asset` serves one in-memory file with a content type, an ETag, and conditional
requests:

```go
//go:embed style.css
var styleCSS []byte

s.Asset("/style.css", styleCSS)
```

`Static` serves a filesystem below a prefix, and `StaticDir` does the same for a
directory on disk:

```go
//go:embed public
var public embed.FS

assets, _ := fs.Sub(public, "public")
s.Static("/assets", assets)
```

Both set `Cache-Control: public, max-age=3600` by default. Use
`WithImmutableAssets()` for content-addressed files whose names change when the
content changes. Range requests and `HEAD` work, and files served through
`Static` also get `Last-Modified`.

## Testing

`NewTestServer` starts the server on a random port and closes it with the test:

```go
func TestCreate(t *testing.T) {
	s := gothic.NewServer(gothic.WithLogger(slog.New(slog.DiscardHandler)))
	s.RegisterProcedure("Create", create)
	ts := gothic.NewTestServer(t, s)

	body, err := ts.Call("Create", "write docs")
	if err != nil {
		t.Fatal(err)
	}
	_ = body
}
```

`Call` returns the decoded result or an `*Error` with the status, code, and
fields. `Get`, `PostJSON`, `Do`, and `Body` cover ordinary routes.

## Scoped apps

`Scope` mounts a second app, or a separate API surface, below a prefix. The
child shares the router, module registry, logger, and transport settings, and
owns its procedures, document defaults, and middleware:

```go
s := gothic.NewServer()
s.MountApp("/", publicApp)

admin := s.Scope("/admin")
admin.SetDocument(gothic.Document{Title: "Admin"})
RegisterAdminProcedures(admin, adminImpl)
admin.MountApp("/", adminApp)
```

Generated registrars keep working because a scoped server is a `*gothic.Server`.
Routes registered through the child are prefixed automatically:

```go
admin.Get("/users", listUsers) // served at /admin/users
```

Procedures are namespaced by scope. `POST /admin/_gothic/rpc/Ping` reaches the
admin implementation, and `POST /_gothic/rpc/Ping` reaches the root one. The
runtime, modules, and RPC base path all resolve under the prefix: generated
modules import the runtime and sibling modules relatively, and the runtime
derives its RPC base from its own URL. Generated code uses the prefix for its
module URL, and custom rendering can call `gothic.BasePath(ctx)` to do the same.

Scopes can nest. `s.Scope("/api").Scope("/v1")` serves under `/api/v1`, and the
longest matching prefix wins.

`Router`, `With`, `Group`, `NotFound`, and `MethodNotAllowed` operate on the
shared chi router, so patterns registered through them are absolute.

## Sessions

`gothic/session` keeps state on the server and stores only a signed identifier
in an `HttpOnly` cookie:

```go
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

The defaults are a seven day lifetime, `Path=/`, `HttpOnly`, and
`SameSite=Lax`. `session.NewSQLStore(db)` stores sessions in a
`gothic_sessions` table in an existing database. Sessions are written before
the first response byte, so cookies survive redirects and early writes.
Rotating the secret invalidates every cookie.

Values are JSON round-tripped, so prefer strings, booleans, and plain structs.
Numbers decode as `float64`. Stored sessions have a fixed lifetime and are not
extended on each request.

## CSRF

`WithCSRF` enables signed double-submit protection:

```go
s := gothic.NewServer(gothic.WithCSRF(gothic.WithCSRFSecret(secret)))
```

A GET mints a token cookie and injects the token into the page config. The
client runtime sends it as `X-Gothic-Csrf`, and plain forms submit it as a
`csrf_token` field. Unsafe requests without a valid token get a 403.

```go
s := gothic.NewServer(
	gothic.WithCSRF(
		gothic.WithCSRFSecret(secret),
		gothic.WithCSRFExempt("/webhooks/"),
	),
)
```

CSRF is opt-in and independent of sessions, so it also protects the login form
before a session exists. Server-to-server clients need the token too, or an
exempt path.

## Users and forms

`gothic/auth` connects users to sessions and hashes passwords with argon2id:

```go
users, err := auth.NewSQLStore(db)
manager := auth.NewManager(sessions, users)

s.Use(manager.Middleware)
s.Handle("/private", manager.RequireUser(private))

_, err = manager.CreateUser(ctx, "ada@example.com", "Ada", password)
user, err := manager.Login(r, email, password) // regenerates the session
err = manager.Logout(r)
```

`RequireUser` returns 401. `RequireUserRedirect("/login")` sends browsers to a
sign-in page with a `next` query parameter. `Login` returns
`auth.ErrInvalidCredentials` for bad credentials and performs the same work for
unknown emails, so response time does not reveal whether an account exists.

For no-JavaScript forms, `ParseForm` and field validation map to the same error
envelope as procedures:

```go
form, err := gothic.ParseForm(w, r)
if err != nil {
	gothic.WriteError(w, r, err)
	return
}
email := form.Required("email", "Enter your email.")
if err := form.Err(); err != nil {
	renderLogin(w, r, email, err) // err has Fields
	return
}
```

See [examples/accounts](../../examples/accounts) for a complete login flow.

## Not included yet

OAuth and OIDC, OpenAPI, protobuf, gRPC, mail, and background jobs. All of those
are additive to the surface above.
