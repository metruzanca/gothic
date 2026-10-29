# gothic syntax

gothic is currently a proof of concept. The application model follows
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and The Elm
Architecture. A `.gothic` file contains ordinary Go followed by one view.
`main.go` remains the entrypoint.

## Model-View-Update

The application is a Go type that implements:

```go
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Component
}
```

Following Bubble Tea:

- `Msg` is `any`. Messages are ordinary Go values and are handled with a type
  switch.
- `Cmd` is `func() Msg`. A command performs work and returns a message.
- `Update` is a method on the model and returns the next model and an optional
  command.
- `Init` returns an optional command.
- `View` is generated from the `.gothic` view.

## Example

`pingpong.gothic` holds the Go prelude, the view, and the procedure contract:

```gothic
package pingpong

import (
	"context"

	"github.com/metruzanca/gothic"
)

type Model struct {
	Messages []string
}

func NewModel() Model {
	return Model{Messages: []string{"ready"}}
}

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}

func (m Model) Init() gothic.Cmd {
	return nil
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

func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}

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
```

`main.go` stays the entrypoint:

```go
type procedures struct{}

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

The generator emits:

- `pingpong_gothic.go`: the Go prelude, the view, the generated `View` method,
  the registrar, procedure client stubs, and the embedded module.
- `pingpong_gothic.js`: the compiled app and view module.

## File layout

A `.gothic` app file has two sections:

1. A Go prelude: package, imports, model, messages, `Init`, `Update`,
   commands, and procedure interfaces.
2. A view, starting with a line that begins with `templ`.

Everything before the first top-level `templ` line is parsed as Go. Everything
from that line onward is parsed as gothic markup.

## View language

### View signature

```gothic
templ View(model Model) {
	...
}
```

The view is a pure function of the model. The model parameter name and type are
used to generate the Go `View` method and the JavaScript renderer.

### Markup

Elements, constant attributes, text, and interpolation work as in templ:

```gothic
<div class="app">
	<p>Hello</p>
	<span>{ model.Count }</span>
</div>
```

Interpolation takes a Go expression. It is parsed with `go/parser`.

### Lists

```gothic
for _, item := range model.Items {
	<li>{ item }</li>
}
```

The range source is a Go expression, normally a model field. The server renders
the initial children. The client adopts them and appends or removes nodes as the
model changes.

### Events

An event attribute is a Go expression that produces a message:

```gothic
<button onclick={ PingClicked{} }>Ping</button>
<button onclick={ DeleteClicked{ID: item.ID} }>Delete</button>
```

The generated JavaScript dispatches the message value. There is no imperative
handler block in the application model.

### Procedures

Declare client-callable procedures as a Go interface that embeds
`gothic.RPCService`:

```go
type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}
```

The interface name is arbitrary. The compiler looks for the embedded
`gothic.RPCService`, then generates the registrar and the client stubs from the
interface methods.

The generator emits:

- a registrar `RegisterProcedures(s, impl)`,
- a client-side Go stub used to compile commands,
- a JavaScript RPC function.

The first parameter must be `context.Context`. The result must be `(T, error)`
or `error`. The client stub drops the context.

Commands call procedures directly:

```go
func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}
```

The Go stub panics if it is called on the server. The server never runs
commands, so this is safe for the proof of concept.

An interface without `gothic.RPCService` is an ordinary Go interface and is not
treated as a procedure contract.

## Commands

A command is a function that returns a message:

```go
func loadItems() gothic.Msg {
	items, err := ListItems()
	if err != nil {
		return LoadFailed{Message: err.Error()}
	}
	return Loaded{Items: items}
}
```

`Update` returns it as the second value:

```go
return m, loadItems
```

The runtime renders the new model first, then runs the command. When the
command returns a message, the runtime dispatches it. This keeps optimistic
updates visible before the asynchronous work completes.

## Rendering

- The server renders the initial model and serializes it into the page.
- The client reads the model, adopts the server-rendered list nodes, and binds
  event attributes.
- On dispatch, the runtime calls `Update`, replaces the model, renders again,
  and runs the returned command.
- Rendering is direct DOM updates. There is no virtual DOM.

## Current limits

The proof of concept supports:

- one application model per package,
- one view template with a single model parameter,
- message structs with zero or simple fields,
- `Update` with a type switch,
- `append` and assignments,
- `if err != nil`,
- commands with procedure calls,
- lists over a model field,
- message-valued event bindings,
- procedure contracts through an interface embedding `gothic.RPCService`.

Not supported yet:

- multiple app models or stateful child components,
- keyed lists,
- conditionals in views,
- expression-valued attributes,
- multiple view parameters,
- `Batch` and `Sequence`,
- procedure arguments and typed validation,
- message serialization,
- `go/packages` type information.

The earlier `state`, `rpc`, and imperative handler syntax still exists in the
compiler as a legacy path, but it is not the direction of the proof of concept.
