# 07 Architecture

Gothic is a compiler and a small runtime. This chapter explains the path from a
`.gth` file to a running app, so you know what is generated and what the client
does at runtime.

## The pipeline

```
app.gth + main.go
    |
gothic generate
    |-- app_gothic.go   prelude, SSR, View method, registrar, client stubs
    |-- app_gothic.js   compiled Init/Update/commands, render, dispatch
    |
go build
    |-- serves HTML and the generated modules from one binary
```

A `.gth` file has two sections. Everything before the first top-level line that
starts with `view` is parsed as Go. Everything from that line on is gothic
markup.

The compiler:

1. Splits the Go prelude from the views.
2. Parses the views with a small markup parser and `go/parser` for embedded
   expressions.
3. Expands pure components inline, before either target is generated.
4. Emits Go for server rendering.
5. Emits JavaScript for `Init`, `Update`, commands, and DOM bindings.

The generated Go embeds its sibling JavaScript with `//go:embed`. Nothing else
is needed at deploy time.

## Server rendering

The server renders the initial model to HTML. The app state is serialized into
the page in a script tag:

```html
<script type="application/json" data-gothic-state="app">...</script>
```

Stateful components get their own state script inside their wrapper:

```html
<div data-gothic-component="Tabs" data-gothic-component-id="c0">
	<script type="application/json" data-gothic-component-state>...</script>
	...
</div>
```

The client reads the state before the first render. That is what makes
hydration possible without shipping a second copy of the initial data.

## Hydration and the runtime loop

On load the client:

1. Reads the serialized state.
2. Adopts the server-rendered DOM instead of rebuilding it.
3. Binds events, attributes, classes, focus, and lists.
4. Runs `Init`.
5. On a message, calls `Update`, replaces the model, updates the DOM, and runs
   the returned command.
6. When a command returns a message, dispatches it.

Text updates are marker-driven. Lists are adopted from the server and
reconciled by key. Branches swap only when the condition changes.

There is no virtual DOM. The generated code patches the exact text node,
attribute, or list entry that changed. That keeps the runtime small and the
output close to the DOM you already know.

## Component expansion

Pure components disappear at compile time. A component call becomes its root
element with the props substituted:

```templ
view Badge(props BadgeProps) {
	<span class="badge">{ props.Text }</span>
}

view App(model Model) {
	<div><Badge text={ model.Status } /></div>
}
```

The compiler expands the call to a `<span>` with
`BadgeProps{Text: model.Status}` in place. The generated Go and JavaScript
never know the component existed.

Stateful components are separate. Each one gets a Go wrapper for SSR and a JS
module with its own model, `Init`, `Update`, render, and lifecycle. The parent
mounts it, passes props, and receives `emit` messages. Closing a branch or
removing a keyed list item destroys the instance.

## Diagnostics

Parse and compiler errors show the file, line, source line, caret, and a hint:

```
app.gth:7:3: expected }
  7 |   if model.Show {
    |   ^
hint: the if block opened on line 5 is missing a closing }
```

The parser tracks node positions through expansion, so an error inside a
component points at the component source, not the caller. Generator errors use
the same shape.

## Module layout

| Path | Purpose |
|---|---|
| `parser/` | Splits the Go prelude from views, parses markup and expressions, expands components |
| `goapp/` | Extracts `Model`, messages, `Init`, `Update`, commands, and procedures from the Go prelude |
| `generator/` | Emits Go SSR code, the `View` method, registrars, and client stubs |
| `generator/appjs/` | Compiles the client Go subset and views into a JavaScript module |
| `client/runtime/` | Embedded runtime helpers for text, lists, branches, attrs, events, and RPC |
| `cmd/gothic/` | The `gothic generate` CLI |
| `gothic.go` | `Model`, `Msg`, `Cmd`, `PropsChanged`, `Children`, and the app registry |
| `server.go` | chi server, procedures, the global middleware chain, and `MountApp` |
| `scope.go` | Scoped apps, base paths, and route prefixing |
| `middleware.go` | Request IDs, panic recovery, access logs, and trusted proxies |
| `serve.go` | `http.Server` timeouts, TLS, and graceful shutdown |
| `option.go` | Server options |
| `asset.go` | Embedded assets with ETags and range support |
| `error.go` | Typed HTTP errors with status, code, and fields |
| `http.go` | JSON helpers and error responses |
| `health.go` | Opt-in liveness and readiness endpoints |
| `csrf.go` | Signed double-submit CSRF protection |
| `session/` | Server sessions, shipped as its own module |
| `auth/` | argon2id passwords, session login, and user middleware, shipped as its own module |
| `gothicest/` | `NewTestServer` for app tests |
| `document.go` | HTML document, metadata, icons, stylesheets, and previews |
| `route/` | Path pattern matching shared by the server and client |
| `browser/` | Curated browser APIs, shipped as its own module |
| `virtual/` | Long-list window math, shipped as its own module |
| `ui/` | Precompiled UI kit, shipped as its own module |

## Why no virtual DOM

A virtual DOM diffs two trees and decides what to patch. Gothic already knows
what changed: the compiler saw which expression feeds which text node, which
condition owns which branch, and which key owns which list item. The generated
code updates those directly.

The trade-off is that markup correctness is the compiler's job. The parser,
expansion, and runtime helpers are covered by unit tests and example tests.

## Testing and verification

Before committing, the project runs:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/guestbook/app_gothic.js
node --check examples/router/app_gothic.js
node --check virtual/example/app_gothic.js
```

Nested modules have their own tests:

```sh
(cd ui && go test ./...)
(cd browser && go test ./...)
(cd virtual && go test ./...)
(cd virtual/example && go test ./...)
(cd session && go test ./...)
(cd auth && go test ./...)
(cd examples/accounts && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

## Current limits

- one app model per package,
- one `App` view with one model parameter,
- message serialization is JSON,
- no `go/packages` type information yet,
- no source maps or JavaScript minification,
- no formatter or LSP,
- client Go subset is smaller than server Go.

These are implementation limits, not design promises. The compiler is the place
to fix them.

## What next

You have finished the guide. Where to go next depends on what you are building:

- Read the [syntax reference](../SYNTAX.md) for the full grammar.
- Browse the [examples](../../examples) for working apps.
- Check the [roadmap](../PLAN.md) for what is planned.
- Read the [pitch](../PITCH.md) for the design goals.
