# Iteration 2: pure view components

Status: working. This iteration adds composable, stateless components to the
Model-View-Update proof of concept. See [SYNTAX.md](SYNTAX.md) for the current
grammar.

## Goal

Prove that an app view can be built from small, reusable view functions without
leaving Go:

- components have named props,
- components can live in the app file or in separate files,
- components compose with other components,
- components work inside lists and server-rendered list items,
- prop-driven text updates without a reload,
- no component state, no virtual DOM, and no hand-written JavaScript.

Stateful child components are the next phase. The working demo is
[`examples/components`](../examples/components).

## Example

```templ
view Status(text string) {
	<p class="status">{ text }</p>
}

view App(model Model) {
	<div class="app">
		<Status text={ model.Status } />
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<MessageItem message={ m } />
			}
		</ul>
	</div>
}
```

`MessageItem` can live in its own file:

```templ
package main

view MessageItem(message string) {
	<li class="message">
		{ message }
		<button onclick={ RepeatClicked{Text: message} }>repeat</button>
	</li>
}
```

## Semantics

- `App` is the root. It is the only view with the model parameter.
- Every other view is a pure component.
- Props are matched by name against the component parameters.
- String literals, Go expressions, and boolean shorthand are supported:

```templ
<Status text={ model.Status } />
<Status text="ready" />
<Card hidden />
```

- Components render exactly one root element.
- Components cannot declare `state`, `Init`, or `Update`.
- Component children and slots are not supported yet.
- Component calls are expanded at generation time.

## File layout

A component can be declared in the app file, after the Go prelude:

```templ
package main

// Go prelude: Model, messages, Init, Update, commands, procedures.

view Status(text string) { ... }

view App(model Model) { ... }
```

Or in a separate file in the same directory:

```templ
package main

view Status(text string) { ... }
```

A separate component file may contain only a package clause and views. It is
not compiled to a Go file or a JavaScript module. Its views are expanded into
the app during `gothic generate`.

Directories may contain one app file. Extra `.gth` files are treated as
component libraries when an app file exists in the same directory.

## Pipeline

```
app.gth + status.gth
    |
parse and classify files
    |
build component registry
    |
expand App
    |
app_gothic.go   expanded App, SSR, text markers, View method
app_gothic.js   expanded App, Init/Update/commands, text and list bindings
```

Component declarations do not produce artifacts.

## Runtime

- SSR wraps every interpolation in text comment markers.
- The client runtime updates those text nodes directly.
- Lists adopt server-rendered children.
- List-item events are hydrated before the first render, so buttons inside
  server-rendered items work immediately.
- New list items are built and bound by the generated renderer.
- Lists remain unkeyed. Iteration 3 adds keyed lists.

## Limits

- Components are compile-time macros, not Go-callable functions.
- Prop types are declared but not independently type-checked before the app is
  built.
- Props may be evaluated more than once if used in multiple expressions.
- No component state, child models, or message mapping yet.
- No children or slots yet.
- No conditionals. Dynamic attributes and class toggles were added in iteration 3.
- One app per directory.

## Verification

```sh
go generate ./examples/pingpong
go build ./...
go vet ./...
go test -count=1 ./...
node --check examples/pingpong/pingpong_gothic.js
```

Browser checks:

- the status component renders the initial status,
- clicking Ping updates the status text and appends `ping` and `pong`,
- clicking a repeat button on a server-rendered list item appends its message,
  which verifies list-item hydration.
