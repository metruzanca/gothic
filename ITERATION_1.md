# Iteration 1: the ping/pong vertical slice

Status: complete and verified. The syntax is Go-first; see [SYNTAX.md](SYNTAX.md)
for the full grammar.

## Goal

Prove that one `.gothic` source file can produce a server-rendered page whose
client interactivity is compiled from the same source, with no hand-written
JavaScript and no WebAssembly. The demo is deliberately small: a Ping button
that adds `ping` to the DOM, calls a Go procedure, receives `pong`, and adds
`pong` to the DOM.

## What the demo does

1. The Go server renders the page with the initial state, `ready`.
2. The browser hydrates the list, adopting the server-rendered `<li>ready</li>`.
3. Clicking Ping runs generated JavaScript that appends `ping` immediately.
4. The generated client calls `POST /_gothic/rpc/Ping`.
5. The Go procedure returns `pong`, and the client appends it.
6. No page reload, no hand-written `.js` source, no npm, no wasm.

## Run it

```sh
go generate ./examples/pingpong
go run ./examples/pingpong
```

`go generate` reports the artifacts it writes. `go run` prints the URL it is
listening on, for example `gothic: listening on http://localhost:8080`. Set
`PORT` to use a different port.

Open the printed URL and click Ping. The list becomes:

```
ready
ping
pong
```

## Implementation map

| Path | Purpose |
|---|---|
| `parser/` | Parses the iteration 1 `.gothic` subset into an AST |
| `generator/` | Emits Go SSR code with hydration markers and state serialization |
| `generator/jsgen/` | Emits the client ES module |
| `client/runtime/gothic.js` | Signals, effects, list adoption, event binding, RPC transport |
| `client/runtime/embed.go` | Embeds the runtime in the Go binary |
| `gothic.go` | Component interface, render helper, module registry |
| `server.go` | chi router wrapper, asset serving, procedure dispatch |
| `cmd/gothic/` | `gothic generate` CLI |
| `examples/pingpong/` | The demo, Go only plus the `.gothic` source |

## Language subset

The syntax is Go-first. Embedded code is parsed with Go's own parser, so there
is no `await`, no JavaScript array literal, and no custom statement dialect.

```gothic
package pingpong

templ PingPong() {
	state messages = []string{"ready"}

	<div class="app">
		<button onclick={ func() {
			messages = append(messages, "ping")
			pong, err := api.Ping()
			if err != nil {
				messages = append(messages, err.Error())
				return
			}
			messages = append(messages, pong)
		} }>Ping</button>
		<ul>
			for _, m := range messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}

rpc Ping() string
```

Supported:

- One root element per template.
- `state name = <Go expression>` declarations at the top of a template.
- Constant elements and constant attributes.
- Static text and `{ <Go expression> }` interpolations.
- `for ... range` over a state identifier, with one element child.
- `on<event>={ func() { ... } }` handlers with Go statements.
- Two-value procedure calls with Go-style error handling.
- Zero-argument `rpc Name() string` declarations.

The generated JavaScript awaits procedure calls automatically. It lowers
`append` on state to a signal push, and it lowers `err != nil` and `err.Error()`
to their JavaScript equivalents.

Not supported yet: template parameters, nested lists, events inside lists,
top-level dynamic expressions, reactive conditionals, keyed lists, state types
other than string slices, procedure arguments, and component composition.

## Generated contract

The Go generator emits SSR HTML with three kinds of markers:

- `data-gothic-root="pingpong"` identifies the hydration root.
- `<script type="application/json" data-gothic-state="pingpong">` carries the
  initial state.
- `<!--g:each:0:open-->` and `<!--g:each:0:close-->` delimit an adoptable list.
- `data-gothic-on="0"` identifies an element to wire to an event handler.

The JS generator emits a module that reads the state, creates signals, adopts
the existing list nodes, binds the button, and calls procedures through
`rpc(name, ...args)`. It imports the embedded runtime from `/_gothic/runtime.js`.

Generated artifacts per source file:

```
foo.gothic -> foo_gothic.go
           -> foo_gothic.js
```

The generated Go file embeds its sibling JS with `//go:embed` and registers it
with the server from `init`, so the binary serves its own generated JavaScript.

## Runtime behavior

`gothic.js` provides:

- `signal(initial)`: callable getter/setter with dependency tracking and an
  array `push` helper.
- `effect(fn)`: runs immediately, tracks reads, and is rescheduled in a
  microtask after a write.
- `each(root, marker, signal, render)`: finds the comment markers, adopts the
  server-rendered children, and appends or removes nodes as the list changes.
- `on(root, marker, event, handler)`: binds an event and removes the marker
  attribute.
- `rpc(name, ...args)`: POSTs JSON to `/_gothic/rpc/<name>` and unwraps the
  result or error.
- `readState(id)`: parses the serialized state script.

## Server and procedures

`gothic.NewServer()` wraps chi v5. It serves:

- `/_gothic/runtime.js` from the embedded runtime.
- `/_gothic/modules/*` from modules registered by generated code.
- `/_gothic/rpc/*` for procedure calls.
- Everything else through the chi router.

Generated code produces a typed Go interface and a registrar for each file:

```go
type PingPongProcedures interface {
	Ping(ctx context.Context) (string, error)
}

func RegisterPingPong(s *gothic.Server, impl PingPongProcedures)
```

The application implements the interface in Go, which keeps the server side
typed while the client calls the generated `api.Ping()`.

## Verification

Automated:

```sh
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/pingpong/views/pingpong_gothic.js
```

Tests cover parser structure, Go output markers and registrar, JS output
bindings, server asset and RPC routes, and the example's SSR and RPC.

Manual browser verification was also run against the live server:

- initial DOM contains `ready`,
- clicking Ping yields `ready`, `ping`, `pong`,
- the RPC endpoint returns `{"result":"pong"}`,
- the runtime and generated module are served as JavaScript,
- with `PING_FAIL=1`, clicking Ping yields `ready`, `ping`, `ping failed`,
  proving the two-value error path.

## Deviations from PLAN.md

Iteration 1 uses a clean-room bootstrap compiler (`parser/`, `generator/`,
`generator/jsgen/`) instead of completing the templ hard fork described in
PLAN.md. This was a deliberate scoping call:

- The slice needs only a tiny grammar, not templ's full parser and generator.
- A partially modified templ fork would add coupling and merge risk before the
  pipeline is proven.
- The bootstrap keeps the emitted contract stable: hydration markers, module
  URLs, runtime API, server routes, and procedure registrars.

The templ fork migration is iteration 2. It should replace the parser and both
generators while preserving the contract above, so the runtime, server, and
example keep working unchanged.

Other accepted iteration 1 limits:

- Static root id, one instance per page.
- Unkeyed list.
- String-slice state only.
- Zero-argument procedures only.
- One state declaration per line.
- Range source must be a state identifier.
- No events or nested lists inside lists.
- The component emits its own module script tag, so using it twice duplicates
  the tag.
- No CSRF, auth, or no-JS form fallback.
- No bundling or minification.

## Recommended next step

Replace the bootstrap parser/generator with the templ fork, keeping the
generated contract stable. After that, remove limits in this order:

1. Multiple instances and dynamic root ids.
2. Keyed lists and events inside lists.
3. Top-level reactive expressions.
4. Procedure arguments and richer state types.
5. No-JS form fallback and CSRF.
