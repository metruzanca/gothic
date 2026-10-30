# gothic

**One language. Server-rendered HTML. Surgical client-side DOM updates. No JavaScript tax.**

## The gap

HTMX made server-driven UI mainstream again. You render HTML on the server, swap fragments, and skip most of the SPA stack.

Then the interface needs a counter, a dropdown, a form that updates state without a round trip, or a progress indicator driven by a request. HTMX stops short. You reach for JavaScript, then Alpine.js, then a build chain, and suddenly the "one language" server app has a second language and a second toolchain.

gothic closes that gap. Client interactivity becomes part of the component language, compiled from the same source as your server render.

## What gothic is

gothic is a fullstack Go framework inspired by [templ](https://github.com/a-h/templ) and [Bubble Tea](https://github.com/charmbracelet/bubbletea). It compiles a Go application and a `.gth` view to two targets:

- **Go** that renders HTML on the server, and
- **JavaScript** that dispatches messages, runs `Update`, and performs fine-grained DOM updates in the browser.

No virtual DOM. No hand-written client modules. No npm on the happy path.

```go
// pingpong.gth, Go prelude
type Model struct{ Messages []string }

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
// pingpong.gth, view
view App(model Model) {
	<div>
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}
```

Application logic is ordinary Go built on Bubble Tea's Model-View-Update pattern. Events dispatch messages. `Update` returns the next model and an optional command. The view is a pure function of the model. Procedures are declared as Go interfaces that embed `gothic.RPCService`, so there is no `rpc` keyword. Clicking Ping appends `ping` immediately, runs the command, receives `pong` or an error message, and updates the DOM. There is no `await`, no JavaScript array syntax, and no separate client dialect.

## The pipeline

```
pingpong.gth + main.go
    |
gothic generate
    |-- pingpong_gothic.go   prelude + server render + View + procedure registrar
    |-- pingpong_gothic.js   compiled Init/Update/commands, render, dispatch
    |
go build / go run
    |-- serves HTML and assets
    |-- embeds the JS runtime with go:embed
```

One binary. One build. The generated JavaScript ships from your Go server, embedded, with no separate deploy step.

## Why no virtual DOM

Generated code creates static DOM once and binds only the dynamic slots. State changes patch the exact text node, attribute, or list that depends on them. There is no tree diff to read around, no reconciliation to debug. The output is small, readable, and close to the DOM you already know.

The client runtime stays tiny: signals, effects, list adoption, event binding, and a JSON transport for procedures.

## How it fits together

- **Go-first syntax.** State initializers, loop headers, interpolations, and event handler bodies are parsed with Go's own parser. Embedded code reads like Go, not JavaScript.
- **Shared AST, two walkers.** The Go backend and the JavaScript backend consume the same parse tree, so server and client output cannot drift apart.
- **Signals with auto-tracking.** Component state compiles to signals. Reads register dependencies. Writes schedule surgical updates. `append` on state lowers to a reactive push.
- **Full hydration.** The server emits markers. The client adopts the existing DOM instead of rebuilding it, so first paint is server HTML and interactivity attaches without a flash.
- **Typed procedures.** Server functions become generated Go registrars and typed client calls. Same-origin JSON by default, with a no-JS form fallback planned.
- **`net/http` at the core.** chi v5 is the blessed router; `http.Handler` compatibility means no lock-in.
- **External API is separate.** huma generates OpenAPI 3.1 for integrators, and internal procedures are never exposed by accident.

## What already works

The ping/pong proof of concept runs end to end.

- `gothic generate` emits Go and JavaScript.
- The application is Bubble Tea-style MVU: `Model`, `Msg`, `Cmd`, `Init`, `Update`, and a generated `View`.
- Messages are ordinary Go values. Commands are `func() Msg`.
- The Go server renders the initial model with hydration markers and serialized state.
- The client adopts the server list, hydrates list-item events, binds message-valued events, runs `Update`, and patches text and lists.
- Pure components compose views with typed props, in the app file or separate files.
- Stateful components own their model, Init, Update, and view, and work in static positions, conditional branches, and keyed lists.
- Components unmount when their branch or list item leaves, so state resets on remount.
- The separately versioned UI kit ships pure Button/Card components and stateful Tabs/Accordion/Counter/Disclosure components.
- Components can emit action messages to the parent app; the app owns server procedures.
- Gothicbook is a small design system demo for the UI kit, including a server-backed action and a scrollable, paginated list of expandable disclosures.
- Procedures accept JSON arguments, and keyed lists update in place.
- Views support `if`, `else if`, and `else`; the server renders one branch and the client swaps it in place.
- The SQLite TodoMVC example supports filters, conditional empty and error states, inline editing, mark-all, and clear-completed.
- Clicking Ping produces `ready`, `ping`, `pong` with no reload.
- Failure handling produces `ready`, `ping`, `ping failed`.
- No hand-written JavaScript. No wasm. No npm.
- Unit, integration, and live browser tests pass.

gothic grows its own bootstrap compiler and selectively vendors templ code. It does not hard fork templ.

## What comes next

1. Component children and slots.
2. Browser JavaScript interop and first-class URL state.
3. Actionable diagnostics with source excerpts and hints.
4. Component commands, props in component Init/Update, and nested stateful components.
5. `Batch` and `Sequence` commands.
6. Typed procedure argument validation and the no-JS form fallback.
7. `go/packages` type information for stronger compiler checks.
8. Manifest, content hashing, formatter, LSP, and source maps.
9. A virtualized list example app.

## Why this can work

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) proves Model-View-Update is idiomatic and production-ready in Go.
- [templ](https://templ.guide) proves server-rendered components with excellent tooling can win.
- Svelte, Solid, and Gea prove compile-time surgical DOM updates beat virtual DOM diffing for size and speed.
- Alpine and HTMX prove developers want reactivity without adopting a frontend framework.
- `go:embed` and modern ES modules remove the need for a bundler in the default path.

gothic is the missing combination: Bubble Tea's state model, templ-style views, compiled DOM updates, and a single Go binary.

## Status

Early, honest, and working. The pipeline is real, the demo runs, and the design decisions are documented. gothic grows its own compiler and selectively vendors templ code. It does not hard fork templ.

If you want a fullstack Go framework where the client is not a second codebase, this is that bet.
