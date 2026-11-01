# Modules

Gothic is split into small Go modules with independent versions. Core gothic
contains the compiler, runtime, views, components, routing, documents, and the
HTTP server. Everything else is optional.

| Module | Import path | Tag prefix | Depends on |
|---|---|---|---|
| Gothic | `github.com/metruzanca/gothic` | `vX.Y.Z` | chi, sqlite for tests |
| UI kit | `github.com/metruzanca/gothic/ui` | `ui/vX.Y.Z` | core |
| Browser | `github.com/metruzanca/gothic/browser` | `browser/vX.Y.Z` | standard library |
| Virtual | `github.com/metruzanca/gothic/virtual` | `virtual/vX.Y.Z` | standard library |
| Sessions | `github.com/metruzanca/gothic/session` | `session/vX.Y.Z` | sqlite for tests |
| Auth | `github.com/metruzanca/gothic/auth` | `auth/vX.Y.Z` | core, sessions |

Current releases:

| Module | Version |
|---|---|
| Core | `v0.5.0` |
| UI kit | `ui/v0.3.0` |
| Browser | `browser/v0.1.0` |
| Virtual | `virtual/v0.1.0` |
| Sessions | `session/v0.1.0` |
| Auth | `auth/v0.1.0` |

## Core

```sh
go get github.com/metruzanca/gothic
```

The compiler, the client runtime, the server, `Document`, `route`, `dom`, and
the `gothicest` test helpers. This is the only required module.

## UI kit

```sh
go get github.com/metruzanca/gothic/ui
```

Headless compound primitives: `Button`, `Field`, `Checkbox`, `Switch`,
`Tabs`, `Accordion`, and a stateful `Combobox`. The kit ships markup and
behavior, not CSS. Style parts with `data-part` and state with `data-*`.

Add the package to an app with:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

## Browser

```sh
go get github.com/metruzanca/gothic/browser
```

A curated, client-only surface for alerts, logging, titles, localStorage,
clipboard, focus, reload, navigation, and element measurement. Calls lower to
browser APIs in the generated JavaScript.

## Virtual

```sh
go get github.com/metruzanca/gothic/virtual
```

Headless window math for long lists. Use `WindowFor` for fixed rows and
`VariableWindow` or `WindowForRows` for measured rows. The view stays in your
app.

## Sessions

```sh
go get github.com/metruzanca/gothic/session
```

Server-side sessions with a signed `HttpOnly` cookie, memory and SQL stores,
regeneration, and destruction. This module does not depend on core gothic.

## Auth

```sh
go get github.com/metruzanca/gothic/auth
```

Users, argon2id password hashes, session-backed login and logout, and user
middleware. It depends on core `gothic` for the error envelope and on `session`.

## Test helpers

`gothicest` is a package in the core module, not a separate module:

```go
import "github.com/metruzanca/gothic/gothicest"

ts := gothicest.NewTestServer(t, s)
```

It wraps `httptest.Server`, keeps a cookie jar, and adds `Call` for procedures.

## Examples

The directories under `examples/` are development apps, not published modules.
Some have their own `go.mod` so they can require the released modules and keep
local `replace` directives. `virtual/example` is an example module too.

## Local development

Modules that depend on core or another module use local `replace` directives or
a `go.work` file while developing in this repository:

- `ui/go.work` points at the checkout.
- `auth/go.mod` replaces core and sessions with `..` and `../session`.
- Example modules replace core, UI, browser, virtual, sessions, and auth.

Release commits remove those local paths and pin released versions. The
follow-up commit restores them for development. See [RELEASING.md](RELEASING.md).

## Versioning

Each module has its own tag:

- core: `vX.Y.Z`
- UI: `ui/vX.Y.Z`
- browser: `browser/vX.Y.Z`
- virtual: `virtual/vX.Y.Z`
- sessions: `session/vX.Y.Z`
- auth: `auth/vX.Y.Z`

Tags are never moved or deleted. If a release is wrong, publish a new patch
version. Release core first when core changes, then the modules that depend on
it.
