# 01 Quickstart

Welcome to the gothic quickstart! This chapter gets a working app on screen and
walks through the whole loop: a model, a view, messages, and a server call.

Gothic is a full-stack Go framework. You write ordinary Go for your application
and a `.gth` view for your HTML. One build produces a Go binary that renders the
first page and serves the JavaScript that takes over in the browser. There is no
npm, no bundler, and no WebAssembly.

## Create a project

```sh
mkdir hello && cd hello
go mod init example.com/hello
go get github.com/metruzanca/gothic
go install github.com/metruzanca/gothic/cmd/gothic@latest
```

A gothic app is two source files:

- `app.gth` contains your model, messages, `Update`, and views.
- `main.go` creates the server and mounts the app.

The generator writes two files next to `app.gth`:

- `app_gothic.go`: the Go prelude plus server rendering, the generated `View`
  method, procedure registration, and client stubs.
- `app_gothic.js`: the compiled `Update`, commands, and DOM bindings.

Commit both generated files. The generated Go embeds its sibling JavaScript with
`//go:embed`, so it must exist at build time.

## Your first app

Start with a counter. In `app.gth`:

```templ
package main

import "github.com/metruzanca/gothic"

type Model struct {
	Count   int
	History []string
}

type Increment struct{}

func (m Model) Init() gothic.Cmd {
	return nil
}

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case Increment:
		m.Count++
		m.History = append(m.History, "click")
	}
	return m, nil
}

view App(model Model) {
	<main class="app">
		<h1>Hello, gothic</h1>
		<button onclick={ Increment{} }>Count: { model.Count }</button>
	</main>
}
```

In `main.go`:

```go
package main

import (
	"log"

	"github.com/metruzanca/gothic"
)

func main() {
	s := gothic.NewServer()
	s.MountApp("/", gothic.NewApp(Model{}))
	log.Fatal(s.ListenAndServe(":8080"))
}
```

Generate, build, and run:

```sh
gothic generate .
go run .
```

Open <http://localhost:8080> and click the button. The count updates without a
page reload.

### Health checks

Health checks are opt-in. Add them when you deploy:

```go
s := gothic.NewServer(
	gothic.WithHealth(gothic.HealthPath),
	gothic.WithReadiness(gothic.ReadyPath, func(ctx context.Context) error {
		return nil
	}),
)
```

The readiness check receives a request-scoped context with a five second
deadline. Leave the options out if you do not want the endpoints.

## What just happened

Every gothic app follows the same three-part loop:

- A **model** is the entire state of your app.
- A **view** is a pure function of that model. `view App(model Model)` becomes
  the model's `View` method.
- A **message** describes something that happened. `Update` receives a message,
  returns the next model, and optionally returns a **command**.

An event attribute such as `onclick={ Increment{} }` produces a message. The
client calls `Update`, renders the new model, then runs any command.

Messages are ordinary Go values, usually structs. Use a type switch in `Update`:

```go
switch msg := msg.(type) {
case Increment:
	m.Count++
case Reset:
	m.Count = 0
}
```

## Add a list

Views can branch and loop with Go syntax:

```templ
type Model struct {
	Count   int
	History []string
}

view App(model Model) {
	<main class="app">
		<button onclick={ Increment{} }>Count: { model.Count }</button>
		<ul>
			for _, item := range model.History {
				<li>{ item }</li>
			}
		</ul>
	</main>
}
```

`for` ranges over a Go expression, normally a model field. Add `key={ expr }` to
the root of the loop body when the list can reorder or remove items:

```templ
for _, item := range model.Items {
	<li key={ item.ID }>{ item.Text }</li>
}
```

Without a key, the client can only append and remove from the end. With a key,
it reuses nodes by identity. Keys are also how stateful components inside lists
keep their state.

`if`, `else if`, and `else` work the same way:

```templ
if model.Count == 0 {
	<p>No clicks yet.</p>
} else {
	<p>{ model.Count } clicks.</p>
}
```

## Talk to the server

A command is a function that returns a message. Commands are where you do
asynchronous work, including calling server procedures.

Declare a procedure contract in the prelude with an interface that embeds
`gothic.RPCService`:

```go
import "context"

type Procedures interface {
	gothic.RPCService
	Ping(ctx context.Context) (string, error)
}
```

Add a `Status` field to the model, then add messages and a command to `Update`:

```diff
 type Model struct {
 	Count   int
 	History []string
+	Status  string
 }
```

```go
type PingClicked struct{}
type PongReceived struct{ Text string }
type PingFailed struct{ Message string }

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case Increment:
		m.Count++
	case PingClicked:
		return m, func() gothic.Msg {
			text, err := Ping()
			if err != nil {
				return PingFailed{Message: err.Error()}
			}
			return PongReceived{Text: text}
		}
	case PongReceived:
		m.Status = msg.Text
	case PingFailed:
		m.Status = msg.Message
	}
	return m, nil
}
```

Render the status and add a button:

```templ
<p>{ model.Status }</p>
<button onclick={ PingClicked{} }>Ping</button>
```

Implement the procedure in `main.go` and register it:

```go
type procedures struct{}

func (procedures) Ping(ctx context.Context) (string, error) {
	return "pong", nil
}

func main() {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
	s.MountApp("/", gothic.NewApp(Model{}))
	log.Fatal(s.ListenAndServe(":8080"))
}
```

`RegisterProcedures` is generated from the interface. There is no route to
write, no JSON to decode, and no client stub to maintain. The command calls
`Ping()` directly and receives a typed result or an error.

> **Note**: procedures run on the server. `Update`, commands, and views compile
> to both Go and JavaScript, but a procedure call only exists on the client. The
> generated Go stub panics if it runs on the server.

## What next

You have seen the core loop: model, view, message, command, and procedure. The
rest of the guide fills in the details:

- [State and commands](02-state.md) goes deeper on messages, commands, and the
  client Go subset.
- [Components](03-components.md) shows how to split views into reusable pieces
  and give them local state.
- [Routing and documents](04-routing.md) covers URL state and the HTML shell.
- [Architecture](07-architecture.md) explains what the compiler emits.
