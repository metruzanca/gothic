# 08 The server

Gothic ships an HTTP server built on chi. You get procedures for your app, and
you also get ordinary routes, middleware, assets, and graceful shutdown without
adding a framework on top.

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
`SetDocument` sets the document defaults shared by mounted apps.

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
Disable them with `WithoutDefaultMiddleware`. `WithRequestTimeout` bounds the
request context, so handlers must honor the context to observe it.
`WithTrustedProxies` is required before `X-Forwarded-For` and
`X-Forwarded-Host` are believed.

## Health checks

Health checks are opt-in. Add the endpoints you want:

```go
s := gothic.NewServer(
	gothic.WithHealth(gothic.HealthPath),
	gothic.WithReadiness(gothic.ReadyPath, func(ctx context.Context) error {
		return db.PingContext(ctx)
	}),
)
```

| Path | Purpose |
|---|---|
| `/healthz` | Liveness. Returns `{"status":"ok"}` with no dependencies. |
| `/readyz` | Readiness. Returns 200, or 503 when the check fails. |

The readiness check receives a request-scoped context with a five second
deadline. Failure returns `{"status":"unavailable"}` and logs the cause. Both
endpoints support `HEAD`, send `Cache-Control: no-store`, and are handled before
the middleware chain, so auth, CSRF, and access logging do not apply.

Change the paths with `WithHealth` and `WithReadiness`. An empty path disables
one endpoint.

## Routes

`Get`, `Post`, and `Handle` cover the common cases:

```go
s.Get("/health", func(w http.ResponseWriter, r *http.Request) {
	_ = gothic.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
})

s.Handle("/hooks/", webhookHandler)
```

For anything else, use the chi router directly:

```go
s.Router().Route("/api/items", func(r chi.Router) {
	r.Get("/{id}", getItem)
	r.Delete("/{id}", deleteItem)
})

s.Router().NotFound(notFound)
```

Path parameters come from `chi.URLParam`. `Router` also exposes `Group`,
`With`, `Mount`, and the rest of chi. Middleware added through the router covers
only routes registered through it.

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

## Assets

`Asset` serves one in-memory file with a content type, an ETag, and conditional
requests:

```go
//go:embed style.css
var styleCSS []byte

s.Asset("/style.css", styleCSS)
```

Assets are cached for an hour by default and support `GET`, `HEAD`, and range
requests. For directory serving, use `http.FileServerFS` with `s.Handle` or
`s.Router()`.

## Testing

`gothicest.NewTestServer` starts the server on a random port and closes it with
the test:

```go
import "github.com/metruzanca/gothic/gothicest"

func TestCreate(t *testing.T) {
	s := gothic.NewServer(gothic.WithLogger(slog.New(slog.DiscardHandler)))
	s.RegisterProcedure("Create", create)
	ts := gothicest.NewTestServer(t, s)

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

CSRF is opt-in and independent of sessions, so it also protects a login form
before a session exists. Server-to-server clients need the token too, or an
exempt path.

## Optional modules

These ship as separate Go modules:

- [`session`](../../session/README.md): server sessions with signed cookies and
  memory or SQL stores.
- [`auth`](../../auth/README.md): argon2id passwords, session login, and user
  middleware.
- [`browser`](05-browser.md): curated browser APIs.
- [`virtual`](06-long-lists.md): long-list windowing.
- [`ui`](../../ui/README.md): the component kit.

Forms use standard `net/http` parsing for now. A dedicated form package is
planned.

## Not included yet

OAuth and OIDC, OpenAPI, protobuf, gRPC, mail, and background jobs. All of those
are additive to the surface above.
