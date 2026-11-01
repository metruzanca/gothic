# 04 Routing and documents

Gothic apps are server-rendered and then hydrated. That means a URL has to work
twice: once on the server for the first request, and once in the browser after
the app takes over.

This chapter covers the single URL message, route links, path parameters,
request-scoped models, and the HTML document around the app.

## LocationChanged

Define one message for URL changes:

```go
type LocationChanged struct {
	Path  string
	Query string
	Hash  string
}
```

The client dispatches it on load and on `popstate`:

- `Path` is `window.location.pathname`.
- `Query` is `window.location.search`, including the leading `?`.
- `Hash` is `window.location.hash`, including `#`.

Handle it in `Update` like any other message:

```go
func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case LocationChanged:
		m.Path = msg.Path
		m.Page = "notfound"
		switch msg.Path {
		case "/":
			m.Page = "home"
		case "/about":
			m.Page = "about"
		}
	}
	return m, nil
}
```

If the app defines `LocationChanged`, the client also intercepts plain left
clicks on `route` links.

## Route links

Mark internal anchors with `route`:

```templ
<a href="/about" route>About</a>
```

A plain left click calls `history.pushState`, then dispatches
`LocationChanged`. Modifier clicks, `target` values other than `_self`,
`download` links, and external origins are left to the browser.

`route` is only supported on `<a>` elements and only in the App view.

## Path parameters

The `route` package matches patterns and captures segments:

```go
import "github.com/metruzanca/gothic/route"

params := route.Params("/todos/:id", msg.Path)
if params != nil {
	m.ID = params["id"]
}
```

`route.Params` returns `map[string]string`, or `nil` when the path does not
match.

Patterns are slash-separated. A segment that starts with `:` captures one
non-empty segment. The pattern and path must have the same number of segments.

```go
route.Params("/users/:name/posts/:post", "/users/ada/posts/7")
// map[string]string{"name": "ada", "post": "7"}
```

The same call works on the server and in `Update`. The generated JavaScript uses
an equivalent runtime helper.

Limits:

- no wildcard or optional segments,
- no trailing-slash normalization,
- duplicate parameter names keep the last segment,
- query and hash are not part of the pattern.

## Request-scoped models

A deep link needs the server to render the right state before hydration. Use
`NewAppFunc` to build the model from the request:

```go
s.MountApp("/", gothic.NewAppFunc(func(r *http.Request) Model {
	return NewModel(r.URL.Path)
}))
```

`MountApp` also registers a catch-all below the pattern, so paths below the
mount reach the app. On load, the client dispatches `LocationChanged` and
updates the model again.

A typical app matches paths in both places:

```go
func NewModel(path string) Model {
	m := Model{Path: path, Page: "notfound"}
	switch path {
	case "/":
		m.Page = "home"
	case "/about":
		m.Page = "about"
	}
	params := route.Params("/hello/:name", path)
	if params != nil {
		m.Page = "hello"
		m.Name = params["name"]
	}
	return m
}
```

Keep the matching logic inline in `Update`. The client compiler lowers the
statements in `Update` itself; it does not compile a helper function you call
from there. The router example inlines the `switch` and the `route.Params`
call for that reason.

## Programmatic navigation

The `browser` module covers history changes:

```go
import "github.com/metruzanca/gothic/browser"

browser.Navigate("/about")
browser.Replace("/about")
browser.Back()
```

These are client-only and valid in `Init`, `Update`, or commands. They require a
`LocationChanged` message. See [Browser interop](05-browser.md).

## Documents

`MountApp` renders a full HTML document: doctype, `lang`, charset, viewport,
title, description, icons, stylesheets, and OpenGraph tags.

The zero `Document` gets a title of `Gothic App`, no icons, and no image. Set
server-wide defaults with `SetDocument`:

```go
s := gothic.NewServer()
s.SetDocument(gothic.Document{
	SiteName: "Gothic",
	Styles:   []string{"/style.css"},
})
```

Override them for one mount with `WithDocument`:

```go
s.MountApp("/", gothic.NewApp(NewModel()),
	gothic.WithDocument(gothic.Document{
		Title:       "Gothicbook",
		Description: "A tiny design system demo.",
		Image:       "/og.png",
		Icons:       []gothic.Icon{{Href: "/favicon.svg"}},
		Styles:      []string{"/gothic-ui.css", "/style.css"},
	}),
)
```

- `WithDocument` replaces the server document for that mount. Zero fields fall
  back to the built-in defaults.
- Leave `Image` empty to omit `og:image`. Leave `Icons` nil to omit icons.
- Relative `og:image` and `og:url` values become absolute using the request host
  and forwarding headers.
- `ExtraHead` is inserted verbatim. Treat it as trusted markup.

To render the same shell around a component without a Model, use `WriteDocument`:

```go
gothic.WriteDocument(w, r, doc, component)
```

The `/_gothic/` prefix is reserved for the runtime and generated modules. The
framework ships no default icon or social image.

## What next

You have URL state and a document shell. The next chapter covers the browser
module and the small set of DOM APIs gothic exposes:

- [Browser interop](05-browser.md)
- [Syntax reference: routing](../SYNTAX.md#routing)
