# Iteration 5: App-owned actions for components

Status: working proof of concept. This iteration lets stateful components emit
action messages to the parent App while the App keeps ownership of server
procedures.

## Goal

Keep the current self-contained component model, but allow a component to send
a message upward:

- the App owns server procedures,
- the App passes an action into a component as a prop,
- the component emits that action when an event happens,
- the App handles the action in `Update` and runs its commands or procedures,
- the component receives new server data through props or the App model.

Tabs and Accordion remain purely client-side. No server mode was added to them.

## Component action props

A component can accept a message prop:

```gothic
view Button(model ButtonModel, label string, primary bool, onclick gothic.Msg) {
	<button
		class="ui-button"
		class:ui-button-primary={ primary }
		onclick={ emit(onclick) }
	>
		{ label }
	</button>
}
```

The App passes the message:

```gothic
<ui.Button label="Increment" onclick={ IncrementClicked{} } />
```

The App handles it and runs its procedure:

```go
case IncrementClicked:
	return m, func() gothic.Msg {
		count, err := Increment()
		if err != nil {
			return CountFailed{Message: err.Error()}
		}
		return CountUpdated{Count: count}
	}
```

The component does not know about the procedure, RPC transport, or server data.

## Runtime changes

- Component mount functions receive the parent App `dispatch` function.
- A component-local `emit(msg)` calls the parent dispatch.
- `emit` is reserved as a component prop name.
- Event expressions that are direct `emit(...)` calls bypass the component's
  local dispatch and go to the parent.
- App-level `emit(msg)` is available to pure components and App views.

## UI kit

`ui.Button` now accepts `onclick gothic.Msg`. A Button without an action can be
created with `onclick={ nil }`.

Its message is emitted to the App; it is not executed by the Button component.

## Gothicbook

Gothicbook adds a small server action section:

- an in-memory counter in the server,
- an `Increment` procedure,
- a `ui.Button` whose `onclick` is `IncrementClicked{}`,
- an App command that calls `Increment` and updates the model,
- a displayed server count.

Client Tabs and Accordion still make no RPC requests. The server Button does.

## Limits

- Components cannot run commands directly.
- Components cannot declare procedures.
- Component `Init`/`Update` cannot read props.
- Function-valued callback props are not supported.
- Component-to-parent emission is message-only.
- Components can emit messages but cannot control parent state directly.

Those limitations keep this phase small and preserve the current app model.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
```

UI module:

```sh
cd ui
go test ./...
```

Gothicbook:

```sh
cd examples/gothicbook
go generate
go test ./...
```

Browser checks:

- the server Button sends one `Increment` RPC and updates the count,
- Tabs and Accordion interactions send no RPC requests,
- component action messages reach the App exactly once.
