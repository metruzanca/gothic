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
[component demo](examples/components), the
[TodoMVC app](examples/todo),
[Gothicbook](examples/gothicbook), and the
[router example](examples/router) for working examples.

```gothic
//app.gth
import (
	"context"

	"github.com/metruzanca/gothic"
	"github.com/metruzanca/gothic/js"
)

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
		js.Alert(msg.Message)
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

type BadgeProps struct {
	Text string
}

view Badge(props BadgeProps) {
	<span class="badge">{ props.Text }</span>
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

## UI kit

The optional UI kit is a separately versioned Go module:

```sh
go get github.com/metruzanca/gothic/ui
```

See [ui/README.md](ui/README.md) and the [Gothicbook demo](examples/gothicbook).

## Status

Early, working proof of concept. The ping/pong example runs end to end with
server rendering, hydration, a Bubble Tea-style update loop, a typed server
call, composable pure view components, and self-contained stateful components.
Views support `if` / `else if` / `else`, pure components accept children
through a default slot, and stateful components work inside branches and keyed
lists. Components use props structs with optional pointer fields, read props in `Init`
and `Update`, react to prop changes, and run their own commands with unmount
guards. TodoMVC uses conditionals for empty states and inline editing.
Gothicbook includes slotted form fields and a scrollable, paginated list of
expandable components. A small `js` package covers browser interop, and the
router example uses path-based URL state with back and forward support. Parse
and compiler errors include source excerpts, carets, and hints. An installable
UI kit is included.

## Learn more

- [Syntax](docs/SYNTAX.md)
- [Roadmap](docs/PLAN.md)
- [Proof of concept](docs/ITERATION_1.md)
- [Components](docs/ITERATION_2.md)
- [TodoMVC on SQLite](docs/ITERATION_3.md)
- [Stateful components and UI kit](docs/ITERATION_4.md)
- [App-owned actions](docs/ITERATION_5.md)
- [Conditional rendering](docs/ITERATION_6.md)
- [Stateful branches and lists](docs/ITERATION_7.md)
- [Component children and slots](docs/ITERATION_8.md)
- [Browser interop and URL state](docs/ITERATION_9.md)
- [Actionable diagnostics](docs/ITERATION_10.md)
- [Component props and Init](docs/ITERATION_11.md)
- [Component commands and destroy](docs/ITERATION_12.md)
- [Component props structs](docs/ITERATION_13.md)
- [Pitch](docs/PITCH.md)

Iteration notes are historical records of the state at that time. Current
syntax and limits live in [SYNTAX.md](docs/SYNTAX.md) and
[PLAN.md](docs/PLAN.md).
