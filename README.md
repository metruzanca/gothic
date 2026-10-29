# gothic

**Fullstack Go. One language. Server-rendered. No JavaScript to write.**

gothic lets you build modern web apps in Go. You write Go for the application
and declarative views for the UI. gothic compiles them to server-rendered HTML
and small, direct JavaScript DOM updates.

No WebAssembly. No hand-written client code. No npm.

## Goals

- **Fullstack Go.** Go is the only language you write for the backend and the
  frontend.
- **No WebAssembly.** Go on the server, plain JavaScript on the client.
- **Idiomatic Go, modern views.** Application logic reads like normal Go.
  Views stay concise and declarative.
- **Direct DOM updates.** No virtual DOM. Only the parts that change are touched.

## A taste

```go
type Model struct{}

type HelloClicked struct{}
type HelloReceived struct{ Message string }

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

templ View(model Model) {
	<button onclick={ HelloClicked{} }>Say hello</button>
}
```

`window.alert` is a browser-side call compiled to JavaScript. The syntax is
under active development and subject to change. See the
[examples directory](examples/pingpong) for the current working app.

## Status

Early, working proof of concept. The ping/pong example runs end to end with
server rendering, hydration, a Bubble Tea-style update loop, and a typed server
call.

## Learn more

- [Syntax](docs/SYNTAX.md)
- [Roadmap](docs/PLAN.md)
- [Proof of concept](docs/ITERATION_1.md)
- [Pitch](docs/PITCH.md)
