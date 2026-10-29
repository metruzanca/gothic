# gothic syntax

gothic is currently a proof of concept. The application model follows
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and The Elm
Architecture. Views are `.gothic` files. Application logic is ordinary Go.

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

`app.go`:

```go
package pingpong

type Model struct {
	Messages []string
}

type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

func NewModel() Model {
	return Model{Messages: []string{"ready"}}
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
```

`view.gothic`:

```gothic
package pingpong

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

`main.go`:

```go
app := gothic.NewApp(NewModel())
s := gothic.NewServer()
RegisterView(s, procedures{})
s.MountApp("/", app)
s.ListenAndServe(":8080")
```

## View language

A `.gothic` file contains a package declaration, one view function, and
optional `rpc` declarations.

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

Declare client-callable procedures in the view file:

```gothic
rpc Ping() string
```

The generator emits:

- a Go interface and registrar for the server implementation,
- a client-side Go stub used to compile commands,
- a JavaScript RPC function.

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
- message-valued event bindings.

Not supported yet:

- multiple app models or stateful child components,
- keyed lists,
- conditionals in views,
- expression-valued attributes,
- multiple view parameters,
- `Batch` and `Sequence`,
- message serialization,
- `go/packages` type information.

The earlier `state` and imperative handler syntax still exists in the compiler
as a legacy path, but it is not the direction of the proof of concept.
