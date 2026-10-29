# gothic: a fullstack Go framework for Bubble Tea-style web apps

> Status: proof of concept. The current model is Model-View-Update inspired by
> [Bubble Tea](https://github.com/charmbracelet/bubbletea). Application logic is
> ordinary Go. Views are `.gothic` files. The phase roadmap will be rewritten
> after the proof of concept stabilizes.

## Overview

gothic turns a Go application and a `.gothic` view into:

- Go server code that renders the initial HTML, and
- JavaScript that dispatches messages, runs `Update`, and patches the DOM
  directly.

The goals:

- Fill the biggest gap in HTMX-style workflows: client state, events, and
  asynchronous server calls without leaving Go.
- Keep server rendering and hydration.
- Avoid a virtual DOM.
- Avoid npm, bundlers, and WebAssembly on the happy path.

## Why not a hard fork

The original plan called for a hard fork of templ. That is no longer the chosen
path.

- gothic's application model is Bubble Tea-style MVU, not templ's component
  model.
- Embedded Go is parsed with `go/ast`, which is a better fit for a two-target
  compiler than raw Go strings.
- A hard fork of an active upstream means freezing or maintaining merges.
- The compiler needs full control of the Go and JavaScript walkers, hydration,
  and the client runtime.

gothic keeps a bootstrap compiler and selectively vendors specific templ code
under MIT where it saves real work, especially HTML escaping, whitespace, and
raw-text element handling. Revisit the fork only if templ source compatibility
or a full LSP is required sooner than gothic can build it.

## Application model

Following Bubble Tea:

```go
type Msg = any
type Cmd func() Msg

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}
```

- Messages are any Go values and are handled with a type switch.
- Commands perform work and return a message.
- The model owns `Update`. There is one transition method per model, not one per
  message.
- `Init` returns an optional command.
- The `.gothic` view is the source of `View()`. The generator emits the method.

The application is split into ordinary Go and a view:

```go
// app.go
type Model struct {
	Messages []string
}

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

func (m Model) Init() gothic.Cmd { return nil }

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case PingClicked:
		m.Messages = append(m.Messages, "ping")
		return m, callPing
	case PongReceived:
		m.Messages = append(m.Messages, msg.Text)
	case PingFailed:
		m.Messages = append(m.Messages, msg.Message)
	}
	return m, nil
}

func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}
```

```gothic
// view.gothic
package main

templ View(model Model) {
	<div class="app">
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

rpc Ping() string
```

```go
// main.go
app := gothic.NewApp(NewModel())
s := gothic.NewServer()
RegisterView(s, procedures{})
s.MountApp("/", app)
```

See [SYNTAX.md](SYNTAX.md) for the current grammar and limits.

## Pipeline

```
app.go + view.gothic
    |
gothic generate
    |-- view_gothic.go   SSR + View method + procedure registrar + client stubs
    |-- view_gothic.js   compiled Init/Update/commands + render + dispatch loop
    |
go build / go run
    |-- serves HTML, the runtime, generated modules, and RPC endpoints
    |-- embeds the runtime and generated modules with go:embed
```

No npm is required. The generated JavaScript ships as ES modules from the Go
binary.

## Runtime loop

Server:

1. Read the initial model.
2. Render `model.View()`.
3. Serialize the model into the page.

Client:

1. Read the serialized model.
2. Adopt the server-rendered DOM.
3. Bind events and lists.
4. Run `Init`.
5. On a message, call `Update(model, msg)`.
6. Replace the model, render directly to the DOM, then run the returned command.
7. When a command returns a message, dispatch it.

Rendering is compiled and direct. Static DOM is created once. Lists are adopted
from the server and reconciled. There is no virtual DOM.

## Current implementation

| Path | Purpose |
|---|---|
| `parser/` | Parses `.gothic` views, markup, Go expressions, and `rpc` declarations. |
| `goapp/` | Extracts `Model`, messages, `Init`, `Update`, and commands from Go app files. |
| `generator/` | Emits Go SSR code, the generated `View` method, registrars, and client stubs. |
| `generator/appjs/` | Compiles the supported Go app subset and the view into a JavaScript module. |
| `generator/jsgen/` | Legacy state-view JavaScript backend. Not the current direction. |
| `client/runtime/` | Embedded runtime helpers: `list`, `on`, `readState`, `rpc`. |
| `cmd/gothic/` | The `gothic generate` CLI. |
| `gothic.go` | `Msg`, `Cmd`, `Model`, `App`, `Component`, and module registry. |
| `server.go` | chi server, assets, procedures, and `MountApp`. |
| `examples/pingpong/` | The proof-of-concept app. |

The CLI detects an app view when a `.gothic` file has a template with a model
parameter, no legacy `state` declarations, and an `app.go` beside it.

## Proof-of-concept scope

Supported now:

- one application model per package,
- one view template with a single model parameter,
- `Msg = any` with messages as ordinary Go structs,
- `Update` with a type switch,
- `append`, field access, assignments, `if err != nil`, and returns,
- commands as functions returning messages,
- procedure calls inside commands, awaited in JavaScript,
- direct procedure calls without an `api.` namespace,
- lists over a model field with server-side adoption,
- message-valued event bindings,
- SSR, model serialization, hydration, and client-side dispatch,
- error handling through messages,
- a generated client-only Go stub for each procedure.

Not supported yet:

- keyed lists and list item updates,
- conditionals in views,
- nested stateful components and multiple app models,
- expression-valued attributes,
- multiple view parameters,
- `Batch` and `Sequence`,
- message serialization or logging,
- `go/packages` type information,
- procedure arguments and typed validation,
- CSRF, auth, and the no-JS form fallback,
- `gothic:json` manifests and content hashing,
- formatter, LSP, and source maps.

The earlier `state` and imperative `func()` handler syntax still exists in the
compiler as a legacy path. It is not the direction of the project.

## Next steps

These will be planned in detail after the proof of concept is evaluated:

1. Keyed list reconciliation and item updates.
2. Conditionals in views.
3. More Go statements and expressions in `Update` and commands.
4. Nested components and message mapping.
5. `Batch` and `Sequence` commands.
6. Procedure arguments, typed validation, and the no-JS fallback.
7. `go/packages` for type information and stronger validation.
8. Manifest, content hashing, and production serving.
9. Formatter, LSP, and source maps.
10. Selective vendoring of templ's HTML and whitespace handling with tests.

## Risks

- The Go-to-JavaScript compiler subset grows with every app feature.
- Client-only Go stubs for procedures are a prototype shortcut.
- `Msg = any` gives up compile-time exhaustiveness. Bubble Tea accepts this; we
  can add optional validation later.
- Keyed list reconciliation and conditional rendering are not yet designed.
- The model is serialized whole. Large models may need selective state later.
- Markup correctness is now ours. Selective vendoring and golden tests are the
  mitigation.
- The client runtime and generated JavaScript are not yet minimized or hashed.

## Prior art

- **Bubble Tea**: the direct model for `Msg`, `Cmd`, `Init`, `Update`, and
  model-owned transitions.
- **Lustre**: MVU in a non-JavaScript language, with managed effects. It uses a
  virtual DOM; gothic borrows the architecture, not the rendering.
- **templ**: the inspiration for the view language and the source of selective
  vendoring ideas.
- **Svelte**, **Solid**, **Gea**: compile-time direct DOM updates.
- **huma**, **Fuego**, **chi**: OpenAPI and routing on `net/http`.
