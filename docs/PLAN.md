# gothic: a fullstack Go framework for Bubble Tea-style web apps

> Status: proof of concept. The current model is Model-View-Update inspired by
> [Bubble Tea](https://github.com/charmbracelet/bubbletea). Application logic is
> ordinary Go. Views are `.gth` files and can be composed from pure and
> stateful components. An optional UI kit ships separately. The phase roadmap
> will be rewritten after the proof of concept stabilizes.

## Overview

gothic turns a Go application and a `.gth` view into:

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
- The `.gth` view is the source of `View()`. The generator emits the method.

The application is split into ordinary Go and a view:

```go
// pingpong.gth (Go prelude)
type Model struct {
	Messages []string
	Status   string
}

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

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
// pingpong.gth (views)
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

Components are pure views. The compiler expands them into `App` before it
generates Go and JavaScript. A component can live in the app file or in a
separate `.gth` file in the same package.

```go
// main.go
s := gothic.NewServer()
RegisterProcedures(s, procedures{})
s.MountApp("/", gothic.NewApp(NewModel()))
```

See [SYNTAX.md](SYNTAX.md) for the current grammar and limits.

## Pipeline

```
pingpong.gth + main.go
    |
gothic generate
    |-- pingpong_gothic.go   prelude + SSR + View method + registrar + client stubs
    |-- pingpong_gothic.js   compiled Init/Update/commands + render + dispatch
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
| `parser/` | Splits the Go prelude from the views, parses markup and Go expressions, and expands components. |
| `goapp/` | Extracts `Model`, messages, `Init`, `Update`, commands, and the procedure interface from the Go prelude. |
| `generator/` | Emits Go SSR code, the generated `View` method, registrars, and client stubs. |
| `generator/appjs/` | Compiles the supported Go app subset and the view into a JavaScript module. |
| `generator/jsgen/` | Legacy state-view JavaScript backend. Not the current direction. |
| `client/runtime/` | Embedded runtime helpers: `list`, `ifBlock`, `text`, `attr`, `classToggle`, `focus`, `on`, `readState`, `rpc`. |
| `cmd/gothic/` | The `gothic generate` CLI. |
| `gothic.go` | `RPCService`, `Msg`, `Cmd`, `Model`, `App`, `Component`, and module registry. |
| `server.go` | chi server, assets, procedures, and `MountApp`. |
| `examples/pingpong/` | The proof-of-concept app. |
| `examples/components/` | The pure-component demo. |
| `examples/todo/` | The SQLite todo demo. |
| `ui/` | Precompiled, independently versioned UI kit module. |
| `examples/gothicbook/` | The UI kit design system demo. |

The CLI detects an app file when the `.gth` file has a valid Go prelude with
a `Model` type and a view whose start line begins with `view`.

## Proof-of-concept scope

Supported now:

- one application model per package,
- one `App` view with a single model parameter,
- pure components with props, in the app file or separate files,
- compile-time component expansion and nested pure components,
- self-contained stateful components with their own model, Init, Update, and view,
- stateful components in static positions and keyed lists,
- precompiled external components from installable packages,
- the independently versioned gothic UI kit,
- `Msg = any` with messages as ordinary Go structs,
- `Update` with a type switch,
- `append`, field access, assignments, `if err != nil`, and returns,
- commands as functions or closures returning messages,
- procedure calls inside commands, awaited in JavaScript,
- procedure arguments decoded from JSON,
- direct procedure calls without an `api.` namespace,
- lists over a model field with server-side adoption,
- keyed list reconciliation by `key={ expr }`,
- `if`, `else if`, and `else` in views, including inside loops and pure components,
- `len` in conditions with null-safe client lowering,
- dynamic attributes, class toggles, and focus bindings,
- message-valued event bindings and event values,
- hash routing through a `HashChanged` message,
- component-to-parent action messages through `emit`,
- App-owned procedures passed to components as action props,
- SSR, model serialization, hydration, and client-side dispatch,
- error handling through messages,
- procedure contracts through an interface embedding `gothic.RPCService`,
- a generated client-only Go stub for each procedure.

Not supported yet:

- component commands and component-declared procedures,
- controlled components, props in component Init/Update, and function-valued props,
- stateful components nested inside other stateful components,
- list-created stateful components whose views contain loops,
- multiple app models,
- component children and slots,
- Go-callable component functions,
- stateful components inside conditionals,
- `for` loops as the direct child of an `if` branch,
- `switch` in views,
- multiple `App` parameters,
- `Batch` and `Sequence`,
- message serialization or logging,
- `go/packages` type information,
- typed procedure argument validation,
- CSRF, auth, and the no-JS form fallback,
- `gothic:json` manifests and content hashing,
- formatter, LSP, and source maps,
- browser JavaScript interop such as `window.alert`, `window.location`, or third-party browser globals.

The earlier `state` and imperative `func()` handler syntax still exists in the
compiler as a legacy path. It is not the direction of the project.

## Next steps

These will be planned in detail after the proof of concept is evaluated:

1. Component commands, component-declared procedures, and props in component Init/Update.
2. Controlled components, callback props, and nested stateful components.
3. Stateful components and top-level loops inside conditional branches.
4. More Go statements and expressions in `Update` and commands.
5. `Batch` and `Sequence` commands.
6. Typed procedure argument validation and the no-JS fallback.
7. `go/packages` for type information and stronger validation.
8. Manifest, content hashing, and production serving.
9. Formatter, LSP, and source maps.
10. Selective vendoring of templ's HTML and whitespace handling with tests.
11. A designed browser JavaScript interop bridge, such as typed bindings or a
    client-only declaration syntax, for `window.alert` and similar APIs.

## Risks

- The Go-to-JavaScript compiler subset grows with every app feature.
- Client-only Go stubs for procedures are a prototype shortcut.
- `Msg = any` gives up compile-time exhaustiveness. Bubble Tea accepts this; we
  can add optional validation later.
- Pure components are compile-time expanded, so reuse duplicates generated
  markup and components are not callable from Go.
- Stateful component support is still narrow: no commands, no parent events,
  no nested stateful components, and no internal loops for list-created
  instances.
- The external UI manifest and generated UI module must stay in sync.
- Conditional branches rebuild their DOM on swap. State held only in the DOM,
  such as caret position, is lost. Stateful components inside branches are not
  supported yet, and a `for` loop cannot be a direct branch child.
- Arbitrary browser globals such as `window.alert` have no designed bridge.
  The JavaScript backend can emit selector calls, but the Go prelude must still
  compile, so browser-only symbols need a future client-side declaration or
  typed binding.
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
