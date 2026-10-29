# Gothic

Gothic is a batteries-included Go framework for building full-stack apps.
It prioritizes simplicity and productivity.

## Goals

- **Fullstack Go.** Go is the only language you write for the backend and the
  frontend.
- **No WebAssembly.** Go on the server, plain JavaScript on the client.
- **Idiomatic Go, modern views.** Application logic reads like normal Go.
  Views stay concise and declarative.

## A taste

The syntax is under active development and subject to change. See the
[ping/pong app](examples/pingpong), the
[component demo](examples/components), and the
[TodoMVC app](examples/todo) for working examples.

```templ
//app.gth
type Model struct{}

type HelloClicked struct{}
type HelloReceived struct{ Message string }

// Procedures declares the RPC calls available to the client.
// The embedded gothic.RPCService marks it for code generation.
type Procedures interface {
	gothic.RPCService
	Hello(ctx context.Context) (string, error)
}

func (m Model) Init() gothic.Cmd { return nil }

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case HelloClicked:
		return m, callHello
	case HelloReceived:
		window.alert(msg.Message)
	}
	return m, nil
}

func callHello() gothic.Msg {
	message, err := Hello()
	if err != nil {
		return HelloReceived{Message: err.Error()}
	}
	return HelloReceived{Message: message}
}

view Badge(text string) {
	<span class="badge">{ text }</span>
}

view App(model Model) {
	<button onclick={ HelloClicked{} }>
		Say hello
		<Badge text="from Go" />
	</button>
}
```

```go
//main.go
type procedures struct{}

func (procedures) Hello(ctx context.Context) (string, error) {
	return "Hello from the server!", nil
}

func main() {
	s := gothic.NewServer()
	RegisterProcedures(s, procedures{})
	s.MountApp("/", gothic.NewApp(Model{}))
	s.ListenAndServe(":8080")
}
```

## Status

Early, working proof of concept. The ping/pong example runs end to end with
server rendering, hydration, a Bubble Tea-style update loop, a typed server
call, and composable pure view components.

## Learn more

- [Syntax](docs/SYNTAX.md)
- [Roadmap](docs/PLAN.md)
- [Proof of concept](docs/ITERATION_1.md)
- [Components](docs/ITERATION_2.md)
- [TodoMVC on SQLite](docs/ITERATION_3.md)
- [Pitch](docs/PITCH.md)
