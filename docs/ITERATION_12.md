> Historical note: component props later moved to props structs in
> [Iteration 13](ITERATION_13.md). Component `Init` can now also receive props.

# Iteration 12: component commands and destroy lifecycle

Status: complete. Stateful components can run commands from `Init` and `Update`,
and unmounted components are destroyed so in-flight work cannot update them.

## Goal

Finish the component update loop without moving server procedures into
components.

- components can run their own commands,
- command results dispatch back to the component,
- procedures stay App-owned,
- unmounting a component cancels its future dispatches.

## Component commands

Component `Init` and `Update` can return `gothic.Cmd` like the App:

```go
type BoxModel struct {
	Status string
}

type Load struct{}
type Loaded struct{}

func (m BoxModel) Init() gothic.Cmd {
	return boxReady
}

func boxReady() gothic.Msg {
	js.SetTitle("ready")
	return Ready{}
}

func (m BoxModel) Update(msg gothic.Msg) (BoxModel, gothic.Cmd) {
	switch msg.(type) {
	case Load:
		return m, func() gothic.Msg {
			return Loaded{}
		}
	case Loaded:
		m.Status = "loaded"
	}
	return m, nil
}
```

The generated mount:

- compiles every referenced command into the component module,
- runs `init()` after the first render,
- runs `result[1]` after `Update`,
- dispatches the command result back to the component.

The component command loop is separate from the App loop. Component commands
never dispatch to the App directly. Use `emit` in a view event when a component
needs to send an action upward.

## Procedures remain App-owned

Components cannot declare procedures, and component code cannot call App
procedures.

The CLI passes the App procedure names to the component compiler. A call such as
`Increment()` in a component `Update`, `Init`, or command fails generation:

```
app.gth:14:3: Increment is an App procedure and cannot be called from a component
  hint: procedures are App-owned; emit an action to the App
```

The App handles the action in its own `Update` and runs the procedure there.

## Destroy lifecycle

Every component mount now exposes `destroy()` and keeps an `alive` flag.

- `dispatch` and `updateProps` return immediately after destroy.
- `run` ignores a command result that resolves after destroy.
- Each mount is destroyed before its DOM is removed.

The runtime destroys component instances when it removes:

- a conditional branch,
- a keyed list item,
- an appended list item that is no longer needed,
- a signal list item in the legacy `each` helper.

`destroyComponentTree` walks the wrapper and its descendants, calls `destroy` on
every stored instance, and clears `__gothicInstances`.

This prevents an async command from updating a component after its branch or
list page disappeared.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Browser checks with a test component:

- `Init` returned a command that set the document title and dispatched a message
  back to the component,
- an `Update` command dispatched a message and changed component state,
- a command that resolved while mounted ran normally,
- starting a command and unmounting in the same task left the command result
  ignored, with no storage write and no console errors.

## Limits

- components cannot declare procedures,
- component commands cannot call App procedures,
- no props in component `Init`,
- no prop-seeded initial state,
- no nested stateful components,
- no timer helpers yet,
- command errors are not surfaced beyond the existing update path,
- there is no command cancellation beyond the mounted guard.

## Follow-ups

- Props in `Init` through a prop-aware model constructor used by SSR and client.
- Nested stateful components.
- Path parameters and route matching helpers.
- The virtualized list example.
