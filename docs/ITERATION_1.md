# Iteration 1: the ping/pong proof of concept

Status: working proof of concept. The application model follows Bubble Tea and
The Elm Architecture. Iteration 2 adds pure components, see
[ITERATION_2.md](ITERATION_2.md). See [SYNTAX.md](SYNTAX.md) for the current
grammar.

## Goal

Prove that one `.gth` app file and one `main.go` can produce:

- a server-rendered page,
- a client that dispatches messages, runs `Update`, and patches the DOM directly,
- a typed server procedure call,
- with no hand-written JavaScript and no WebAssembly.

The demo is a Ping button. Clicking it:

1. appends `ping` to the DOM immediately,
2. runs a command that calls the `Ping` procedure,
3. dispatches `PongReceived` or `PingFailed`,
4. appends `pong` or the error message.

## Application files

```
examples/pingpong/
  pingpong.gth  Go prelude, view, procedure interface
  main.go          entrypoint and Ping implementation
```

`pingpong.gth` starts with ordinary Go:

```go
type Model struct {
	Messages []string
}

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

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
```

Then the view:

```templ
view App(model Model) {
	<div class="app">
		<button onclick={ PingClicked{} }>Ping</button>
		<ul>
			for _, m := range model.Messages {
				<li>{ m }</li>
			}
		</ul>
	</div>
}
```

`main.go` wires the app and implements the procedure interface:

```go
func (procedures) Ping(ctx context.Context) (string, error) {
	return "pong", nil
}

func main() {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
	s.MountApp("/", gothic.NewApp(NewModel()))
	s.ListenAndServe(":8080")
}
```

## Run it

```sh
go generate ./examples/pingpong
go run ./examples/pingpong
```

`go generate` reports the generated files. `go run` prints the URL, for example
`gothic: listening on http://localhost:8080`. Set `PORT` to use a different
port.

Open the printed URL and click Ping. The list becomes:

```
ready
ping
pong
```

With `PING_FAIL=1`, the list becomes:

```
ready
ping
ping failed
```

## Implementation map

| Path | Purpose |
|---|---|
| `parser/` | Parses `.gth` files, splits the Go prelude from the view, and parses markup and Go expressions. |
| `goapp/` | Extracts the model, messages, `Init`, `Update`, commands, and procedure interface from the Go prelude. |
| `generator/` | Emits the Go view, the `View` method, the registrar, client stubs, and the app Go file. |
| `generator/appjs/` | Compiles the supported Go app subset and the view to JavaScript. |
| `client/runtime/gothic.js` | `list`, `text`, `on`, `readState`, `rpc`. |
| `gothic.go` | `RPCService`, `Msg`, `Cmd`, `Model`, `App`, `Component`, and module registry. |
| `server.go` | chi server, assets, procedures, and `MountApp`. |
| `cmd/gothic/` | The `gothic generate` CLI. |

## Generated contract

The generator emits two files:

- `pingpong_gothic.go`: the Go prelude, the view function, the generated `View`
  method, the procedure registrar, client-only procedure stubs, and the embedded
  module.
- `pingpong_gothic.js`: the compiled app and view module.

The SSR output contains:

- `data-gothic-root="view"` for hydration,
- `<script type="application/json" data-gothic-state="view">` with the model,
- `<!--g:each:0:open-->` and `<!--g:each:0:close-->` around list items,
- `data-gothic-on="0"` for event binding,
- a module script tag for the generated JavaScript.

The JavaScript generator emits:

- `const Ping = (...args) => rpc("Ping", ...args)`,
- list adoption and rendering,
- message-valued event bindings,
- `render(model)`,
- `dispatch(msg)` and `run(cmd)`,
- compiled `init`, `update`, and commands.

## Runtime loop

1. The client reads the serialized model.
2. It adopts the server-rendered list nodes.
3. It binds the event marker.
4. It renders the initial model and runs `Init`.
5. On an event, it dispatches a message.
6. `Update` returns the next model and a command.
7. The runtime renders the model, then runs the command.
8. The command returns a message, which is dispatched again.

Rendering is direct DOM updates. There is no virtual DOM.

## Verification

Automated:

```sh
go build ./...
go vet ./...
go test -count=1 ./...
node --check examples/pingpong/pingpong_gothic.js
```

Tests cover the parser and prelude split, Go app extraction, procedure
interface extraction, JavaScript generation, SSR output, and the example
integration.

Browser verification:

- initial DOM contains `ready`,
- clicking Ping produces `ready`, `ping`, `pong`,
- with `PING_FAIL=1`, clicking Ping produces `ready`, `ping`, `ping failed`.

## Current limits

- One app model per package, with one view template and one model parameter.
- Messages are `any`. There is no compile-time exhaustiveness check.
- The JavaScript compiler supports a small Go subset: type switches, `append`,
  assignments, `if err != nil`, returns, message literals, and command calls.
- Lists are unkeyed and only append or remove from the end. Item updates and
  reordering are not supported.
- No conditionals, expression attributes, nested components, or multiple view
  parameters.
- No `Batch` or `Sequence`.
- Procedure methods must have no extra arguments yet. CSRF, the no-JS fallback,
  and typed validation are not implemented.
- Client-only procedure stubs panic if called on the server.
- No manifest, content hashing, formatter, LSP, or source maps.
- The legacy `state`, `rpc`, and `func()` syntax still exists but is not the
  direction.
