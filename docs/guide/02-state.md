# 02 State and commands

Gothic uses the Model-View-Update loop from Bubble Tea and the Elm
Architecture. This chapter covers the pieces you will use every day: messages,
commands, and the Go that compiles to the browser.

## The model

The model owns all application state. It is a Go type with `Init` and `Update`
methods:

```go
type Message struct {
	Text string
}

type Model struct {
	Draft    string
	Messages []Message
}

func (m Model) Init() gothic.Cmd {
	return nil
}
```

`Init` runs once when the app mounts on the client. Return `nil` when there is
nothing to do, or return a command to load data.

The generated `View` method comes from the `App` view in your `.gth` file:

```templ
view App(model Model) {
	<div class="app">
		<input value={ model.Draft } oninput={ DraftChanged{Text: event.target.value} } />
		<ul>
			for i, message := range model.Messages {
				<li key={ i }>{ message.Text }</li>
			}
		</ul>
	</div>
}
```

There is one model per app and one `App` view. Everything in the view is a pure
function of the model.

## Messages

A message is any Go value. Most are small structs named for what happened:

```go
type DraftChanged struct{ Text string }
type PostClicked struct{}
type Posted struct{ Message Message }
type PostFailed struct{ Message string }
```

`Update` receives a message and returns the next model plus an optional command:

```go
func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case DraftChanged:
		m.Draft = msg.Text
	case PostClicked:
		text := m.Draft
		if text == "" {
			return m, nil
		}
		m.Draft = ""
		return m, post(text)
	case Posted:
		m.Messages = append(m.Messages, msg.Message)
	case PostFailed:
		m.Messages = append(m.Messages, Message{Text: msg.Message})
	}
	return m, nil
}
```

Prefer names that describe what happened (`PostClicked`, `TodosLoaded`,
`SaveFailed`) over names that describe what to do. The message set then reads
like a list of everything that can affect the app.

`Update` returns the model by value. Mutating a field on the value receiver is
how you build the next model:

```go
m.Count++
m.Items = append(m.Items, item)
```

## Commands

A command does work and returns a message:

```go
func post(text string) gothic.Cmd {
	return func() gothic.Msg {
		if err := AddTodo(text); err != nil {
			return PostFailed{Message: err.Error()}
		}
		return Posted{}
	}
}
```

A command can also be a named function with no parameters:

```go
func loadTodos() gothic.Msg {
	todos, err := ListTodos()
	if err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return TodosLoaded{Data: todos}
}
```

Return it from `Update` with `return m, loadTodos`.

The runtime renders the new model first, then runs the command. When the
command returns a message, the runtime dispatches it. That keeps optimistic
updates visible while the asynchronous work runs.

Commands are client-only. The Go stub exists so `Init` and `Update` compile, but
the server never runs them.

> **Note**: commands cannot take parameters. Close over the value instead:
> `func() gothic.Msg { return loadTodos() }`.

## Procedures

Procedures are server functions with a typed client stub. Declare the contract
as an interface that embeds `gothic.RPCService`:

```go
type Procedures interface {
	gothic.RPCService
	AddTodo(ctx context.Context, title string) error
	ListTodos(ctx context.Context) ([]Todo, error)
}
```

The first parameter must be `context.Context`. The result is either `(T, error)`
or `error`. The generator emits the registrar and the client stubs:

```go
func callAdd(text string) gothic.Msg {
	if err := AddTodo(text); err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return TodosLoaded{}
}
```

On the server, implement the interface and register it:

```go
RegisterProcedures(s, procedures{})
```

Procedures are App-owned. Components can emit an action to the App and let the
App run the procedure; they cannot declare their own.

## Batch and Sequence

Compose commands with `gothic.Batch` or `gothic.Sequence`:

```go
return m, gothic.Batch(loadTodos(), loadStats())

return m, gothic.Sequence(
	func() gothic.Msg { return FirstDone{} },
	func() gothic.Msg { return SecondDone{} },
)
```

- `Batch` starts commands concurrently.
- `Sequence` runs them in order, stopping if the component unmounts.
- Nil commands are ignored.
- Every non-nil result is dispatched.

Both are client-only, with Go stubs so `Init` and `Update` compile.

## The client Go subset

`Init`, `Update`, commands, and component views compile a growing subset
of Go. It covers the expressions you normally need in a view:

```go
item := m.Items[0]
m.Items[0] = item
page := m.Items[1:3]
tail := m.Items[1:]
m.Counts = map[string]int{"open": 1, "done": 2}
m.Total = len(m.Counts)
delete(m.Counts, "done")
m.Total += 1
m.Count--
```

Supported:

- field access, indexing, and two-index slicing,
- map literals, `len`, `delete`, and map indexing,
- `append`, assignments, and compound assignments,
- `++` and `--`,
- `if` and `if err != nil`,
- range loops over slices and maps with `break` and `continue`,
- non-type switches, including tagless switches and init statements,
- procedure calls and command closures.

Not supported:

- three-index slices,
- `&^=`,
- labeled branches,
- `goto`,
- `fallthrough`,
- declaring variables in view expressions.

Keep these differences from server Go in mind:

- `dom.Px(n)` formats pixel lengths for style attributes.
- `browser.ClientHeight(selector)` reads an element height for measurement
  commands.
- A missing map key reads as `undefined`, not the zero value.
- Map iteration uses JavaScript insertion order, and map keys arrive as strings.
- `/=` is floating division.
- Slices copy instead of sharing a backing array.

## What next

You now know the whole update loop. The next chapter splits views into
components and gives them local state:

- [Components](03-components.md)
- [Syntax reference: client Go subset](../SYNTAX.md#client-go-subset)
