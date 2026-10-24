# gothic

**One language. Server-rendered HTML. Surgical client-side DOM updates. No JavaScript tax.**

## The gap

HTMX made server-driven UI mainstream again. You render HTML on the server, swap fragments, and skip most of the SPA stack.

Then the interface needs a counter, a dropdown, a form that updates state without a round trip, or a progress indicator driven by a request. HTMX stops short. You reach for JavaScript, then Alpine.js, then a build chain, and suddenly the "one language" server app has a second language and a second toolchain.

gothic closes that gap. Client interactivity becomes part of the component language, compiled from the same source as your server render.

## What gothic is

gothic is a fork of [templ](https://github.com/a-h/templ) that compiles one `.gothic` component file to two targets:

- **Go** that renders HTML on the server, and
- **JavaScript** that performs fine-grained DOM updates in the browser.

No virtual DOM. No hand-written client modules. No npm on the happy path.

```gothic
templ PingPong() {
	state messages []string = ["ready"]

	<div>
		<button onclick={
			messages.Push("ping")
			pong := await api.Ping()
			messages.Push(pong)
		}>Ping</button>
		<ul>
			@each messages as m {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

proc Ping() string
```

Clicking Ping appends `ping` to the DOM immediately, calls a typed Go procedure, receives `pong`, and appends that too. The server rendered the initial list. The client hydrated it in place.

## The pipeline

```
.gothic source
    |
gothic generate
    |-- _gothic.go    server render, hydration markers, serialized state
    |-- _gothic.js    DOM bindings and effects
    |-- manifest      roots, modules, procedure names
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

- **Shared AST, two walkers.** The Go backend and the JavaScript backend consume the same parse tree, so server and client output cannot drift apart.
- **Signals with auto-tracking.** Component state compiles to signals. Reads register dependencies. Writes schedule surgical updates.
- **Full hydration.** The server emits markers. The client adopts the existing DOM instead of rebuilding it, so first paint is server HTML and interactivity attaches without a flash.
- **Typed procedures.** Server functions become generated Go registrars and typed client calls. Same-origin JSON by default, with a no-JS form fallback planned.
- **`net/http` at the core.** chi v5 is the blessed router; `http.Handler` compatibility means no lock-in.
- **External API is separate.** huma generates OpenAPI 3.1 for integrators, and internal procedures are never exposed by accident.

## What already works

Iteration 1 is implemented and verified: the ping/pong slice above runs end to end.

- `gothic generate` emits Go and JavaScript.
- The Go server renders the initial list with hydration markers and serialized state.
- The client hydrates the existing list, binds the button, and calls the procedure.
- Clicking Ping produces `ready`, `ping`, `pong` in the DOM with no reload.
- No hand-written JavaScript. No wasm. No npm.
- Unit, integration, and live browser tests pass.

Iteration 1 uses a bootstrap compiler to prove the pipeline. Iteration 2 replaces it with the full templ fork while keeping the generated contract stable.

## What comes next

1. The templ fork, so the full component language is available.
2. Multiple component instances, keyed lists, and events inside lists.
3. General Go-expression transpilation with type information.
4. Richer state, procedure arguments, and client-callable functions.
5. No-JS form fallback, CSRF, and the progressive enhancement story in full.
6. Islands, async resources, and OpenAPI integration.

## Why this can work

- [ghtmx](https://ghtmx.dev) already proves a hard fork of templ can ship.
- [templ](https://templ.guide) proves server-rendered components with excellent tooling can win.
- Svelte, Solid, and Gea prove compile-time surgical DOM updates beat virtual DOM diffing for size and speed.
- Alpine and HTMX prove developers want reactivity without adopting a frontend framework.
- `go:embed` and modern ES modules remove the need for a bundler in the default path.

gothic is the missing combination: templ's server language, compiled reactivity, and a single Go binary.

## Status

Early, honest, and working. The pipeline is real, the demo runs, and the design decisions are documented. The next milestone is replacing the bootstrap compiler with the templ fork without changing the generated contract.

If you want a fullstack Go framework where the client is not a second codebase, this is that bet.
