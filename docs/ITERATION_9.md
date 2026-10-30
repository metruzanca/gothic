# Iteration 9: browser interop and URL state

Status: complete. A curated `js` package covers common browser APIs, and URL
state is first-class through `LocationChanged`, `route` links, and request-scoped
app models.

## Goal

Give apps a designed way out of Go for browser APIs, and make the URL a real
state container.

- one small interop surface instead of arbitrary `window` access,
- path, query, and hash state that survives reloads and bookmarks,
- pushState, replaceState, back, and forward,
- server rendering for deep links,
- a working router example.

## Browser interop

Package: `github.com/metruzanca/gothic/js`.

| Function | Browser API |
|---|---|
| `js.Alert(message string)` | `window.alert` |
| `js.Log(values ...any)` | `console.log` |
| `js.SetTitle(title string)` | `document.title` |
| `js.GetItem(key string) string` | `localStorage.getItem`, empty string when missing |
| `js.SetItem(key, value string)` | `localStorage.setItem` |
| `js.RemoveItem(key string)` | `localStorage.removeItem` |
| `js.Copy(text string)` | `navigator.clipboard.writeText` |
| `js.Focus(selector string)` | `document.querySelector(selector)?.focus()` |
| `js.Reload()` | `location.reload` |
| `js.Navigate(path string)` | pushState plus `LocationChanged` |
| `js.Replace(path string)` | replaceState plus `LocationChanged` |
| `js.Back()` | `history.back` |

Rules:

- calls are valid in `Init`, `Update`, and commands,
- calls in views are a compile error,
- the Go stubs panic if called on the server,
- there is no arbitrary `window` access and no `eval`,
- `Copy` is awaited inside async commands and fire-and-forget in `Update`.

## URL state

Apps define:

```go
type LocationChanged struct {
	Path  string
	Query string
	Hash  string
}
```

The client dispatches it on load and on `popstate`. The fields mirror the
browser location:

- `Path` is `window.location.pathname`,
- `Query` is `window.location.search`, including the leading `?`,
- `Hash` is `window.location.hash`, including `#`.

The generated app module also adds:

- `navigate(path)`, which pushes history and dispatches the message,
- `replaceLocation(path)`, which replaces history and dispatches,
- `routeClick`, which handles `route` anchors.

`LocationChanged` takes precedence over `HashChanged` when both are defined.
Hash routing remains available for hash-only apps.

## Route links

```templ
<a href="/about" route>About</a>
```

`route` is not rendered as an HTML attribute. It attaches a click handler that:

- ignores modified clicks and non-left buttons,
- ignores `target` values other than `_self`,
- ignores `download` links,
- ignores cross-origin URLs,
- prevents the default navigation for internal links,
- calls `navigate` with the resolved path, query, and hash.

The Go renderer allocates a click marker for the anchor, and the client binds it.
Route links are only supported in the App view and require `LocationChanged`.
A non-anchor element with `route` is a compile error.

## Request-scoped models and deep links

`NewAppFunc` builds the initial model from the request:

```go
s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
	return NewModel(r.URL.Path)
}))
```

`MountApp` registers the pattern and a catch-all below it. For `"/"` the
catch-all is `"/*"`. This lets the server render the right page for `/about`,
`/settings`, or an unknown path before hydration.

`RenderTo` still works for apps created with `NewApp`. It returns an error for
request-scoped apps because there is no request to build the model from.

## Router example

New `examples/router` module:

- `/`, `/about`, and `/settings` pages,
- unknown paths render a not-found page,
- route links for navigation,
- an "Open settings" button that returns a command calling `js.Navigate`,
- a "Back" button calling `js.Back`,
- `js.SetTitle` updates the document title,
- localStorage remembers the previous path,
- the server renders deep links through `NewAppFunc`.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
```

Modules:

```sh
go run ./cmd/gothic generate ./ui
(cd examples/gothicbook && go generate ./... && go test ./...)
(cd examples/router && go generate ./... && go test ./...)
node --check examples/router/app_gothic.js
```

Browser checks:

- opening `/about` directly renders About on the server and hydrates it,
- the document title changes after the location message,
- clicking Home and Settings updates the URL without a reload,
- the "Open settings" command navigates to `/settings`,
- Back returns to the previous path through `popstate`,
- localStorage shows the previous path,
- no console errors.

## Limits

- no arbitrary browser globals outside the `js` package,
- no path parameters, nested routers, or route matching helpers,
- `Query` keeps its leading `?`,
- no automatic scroll restoration,
- no no-JS navigation fallback,
- route links are not supported inside stateful components,
- there is no route-level data loading or code splitting,
- the interop surface grows only by design, not by request.

## Follow-ups

- Actionable diagnostics (D).
- Path parameters and route matching helpers.
- Scroll restoration and focus management on navigation.
- A no-JS fallback and server-side form handling.
