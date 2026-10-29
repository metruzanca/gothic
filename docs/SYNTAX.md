# gothic syntax

gothic is currently a proof of concept. The application model follows
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and The Elm
Architecture. A `.gth` file contains ordinary Go followed by one or more
views. One view, `App`, is the root. The others are pure components.
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
- `View` is generated from the `.gth` view.

## Example

`pingpong.gth` holds the Go prelude, the view, and the procedure contract:

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

A `.gth` app file has two sections:

1. A Go prelude: package, imports, model, messages, `Init`, `Update`,
   commands, and procedure interfaces.
2. One or more views, starting with a line that begins with `view`.

Everything before the first top-level `view` line is parsed as Go. Everything
from that line onward is parsed as gothic markup.

The app view is named `App`:

```gothic
view App(model Model) {
	...
}
```

Any other view is a component:

```gothic
view Badge(text string) {
	<span class="badge">{ text }</span>
}
```

Components can live in the same app file or in a separate `.gth` file in the
same directory. A separate component file contains only a package clause and
views. It has no model and no `Init` or `Update`.

## View language

### View signature

```gothic
view App(model Model) {
	...
}
```

The app view is a pure function of the model. The model parameter name and type
are used to generate the Go `View` method and the JavaScript renderer.

### Components

A component is a pure view function with named props:

```gothic
view Badge(text string) {
	<span class="badge">{ text }</span>
}
```

Invoke a component with a capitalized tag. Props use string literals,
expressions, or boolean shorthand:

```gothic
<Badge text={ model.Status } />
<Badge text="ready" />
```

A bare prop passes `true`, for example `<Card hidden />` when `Card` declares a
`hidden bool` parameter.

The compiler expands components into the app before generating Go and
JavaScript. Props are substituted directly into the component body. Components
can call other components, including components from other files in the same
package.

Rules:

- `App` is the only root and the only view with a model parameter.
- Components are stateless. They cannot declare `state`.
- Components render exactly one root element.
- Components should reference only their props and package-level declarations.
- Component children and slots are not supported yet.
- Components cannot be called from Go code yet.
- Capitalized tags are always components, so camelCase SVG tags are not
  supported yet.

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
	<Item key={ item.ID } item={ item } />
}
```

The range source is a Go expression, normally a model field. Add `key={ expr }`
to the root of the loop body, or to a component call, to enable keyed list
reconciliation. Keyed lists reuse nodes by key, update their text, and reorder
or remove them as the model changes. Lists without a key append and remove from
the end only. `key` is reserved and is not passed to components as a prop.

The server renders the initial children. The client adopts them and hydrates
their events.

### Events

An event attribute is a Go expression that produces a message:

```gothic
<button onclick={ PingClicked{} }>Ping</button>
<button onclick={ DeleteClicked{ID: item.ID} }>Delete</button>
```

The generated JavaScript dispatches the message value. There is no imperative
handler block in the application model.

Event handlers receive the DOM event as `event`. This is useful for inputs:

```gothic
<input oninput={ DraftChanged{Text: event.target.value} } />
```

`onenter` and `onescape` are shortcuts for keydown handlers:

```gothic
<input onenter={ SaveEdit{ID: todo.ID} } onescape={ CancelEdit{ID: todo.ID} } />
```

### Attributes and classes

Expression attributes update a DOM property or attribute:

```gothic
<input value={ model.Draft } checked={ todo.Completed } />
```

`class:name={ expr }` adds or removes a class when the expression changes:

```gothic
<li class:completed={ todo.Completed } class:editing={ editing } class:hidden={ hidden }>
```

`focus={ expr }` focuses the element when the expression becomes true.

### Routing

If the app defines a `HashChanged` message with a `Hash` field, the client
dispatches it on load and on every hashchange:

```go
type HashChanged struct{ Hash string }
```

Use it to map `#/`, `#/active`, and `#/completed` to a model field.

### Procedures

Declare client-callable procedures as a Go interface that embeds
`gothic.RPCService`:

```go
type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
	AddTodo(ctx context.Context, text string) error
	ToggleTodo(ctx context.Context, id int64) error
	ListTodos(ctx context.Context) ([]Todo, error)
}
```

The interface name is arbitrary. The compiler looks for the embedded
`gothic.RPCService`, then generates the registrar and the client stubs from the
interface methods.

The generator emits:

- a registrar `RegisterProcedures(s, impl)`,
- a client-side Go stub used to compile commands,
- a JavaScript RPC function.

The first parameter must be `context.Context`. Remaining parameters are
arguments sent by the client as JSON. The result must be `(T, error)` or
`error`. The client stub drops the context.

Commands call procedures directly:

```go
func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}

func callAdd(text string) gothic.Msg {
	if err := AddTodo(text); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return TodosLoaded{}
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

A command can also be a closure, which is useful when it needs values from the
message or model:

```go
return m, func() gothic.Msg {
	if err := AddTodo(m.Draft); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return loadTodos()
}
```

The runtime renders the new model first, then runs the command. When the
command returns a message, the runtime dispatches it. This keeps optimistic
updates visible before the asynchronous work completes.

## Rendering

- The server renders the initial model and serializes it into the page.
- Components are expanded into the app before either target is generated.
- Interpolated text is wrapped in comment markers. The client updates the text
  node in place.
- The client reads the model, adopts the server-rendered list nodes, hydrates
  their events, and binds event attributes.
- On dispatch, the runtime calls `Update`, replaces the model, updates text and
  lists, and runs the returned command.
- Rendering is direct DOM updates. There is no virtual DOM.

## Current limits

The proof of concept supports:

- one application model per package,
- one `App` view with a single model parameter,
- pure components with multiple named props,
- components in the same file or separate files in the same package,
- nested pure components and components inside lists,
- message structs with zero or simple fields,
- `Update` with a type switch,
- `append` and assignments,
- `if err != nil`,
- commands with procedure calls and command closures,
- procedure arguments,
- lists over a model field and keyed list reconciliation,
- reactive interpolated text and message-valued event bindings,
- event values such as `event.target.value`,
- dynamic attributes, class toggles, and focus bindings,
- hash-based route changes through a `HashChanged` message,
- procedure contracts through an interface embedding `gothic.RPCService`.

Not supported yet:

- stateful child components, child models, and message mapping,
- multiple app models,
- component children and slots,
- Go-callable component functions,
- conditionals in views,
- multiple `App` parameters,
- `Batch` and `Sequence`,
- typed procedure argument validation,
- message serialization,
- `go/packages` type information.

The earlier `state`, `rpc`, and imperative handler syntax still exists in the
compiler as a legacy path, but it is not the direction of the proof of concept.
