# Iteration 14: prop-seeded component state

Status: complete. A stateful component can build its initial model from props
with an optional constructor.

## Goal

Let props seed component state on both server and client.

Before this iteration, props could be read in the view, `Init`, and `Update`,
but component state always started at the zero value. A Tabs component could
not start on a supplied active tab, and a Disclosure could not start open.

## Constructor convention

Define a constructor named `New` plus the model type:

```go
type TabsProps struct {
	Items  []Tab
	Active *int
}

type TabsModel struct {
	Active int
}

func NewTabsModel(props TabsProps) TabsModel {
	m := TabsModel{}
	if props.Active != nil {
		m.Active = *props.Active
	}
	if m.Active >= len(props.Items) {
		m.Active = len(props.Items) - 1
	}
	if m.Active < 0 {
		m.Active = 0
	}
	return m
}
```

The constructor is optional. Without it, state starts at the zero value as
before.

Rules:

- `New<Model>(props)` may take zero parameters or one parameter of the props
  struct type.
- It must return the component model type.
- It must be deterministic and use the JavaScript compiler subset. The same
  function runs on the server and in the browser.
- It must not depend on server-only data, time, randomness, or `js` interop.

## Server rendering

The generated Go component wrapper uses the constructor:

```go
model := NewTabsModel(props)
stateJSON, err := json.Marshal(model)
```

The serialized state therefore matches the props before hydration.

## Client creation and hydration

The component module emits a JavaScript constructor, for example
`newTabs(props)`.

- Hydration still reads the server-serialized state first.
- Client-created instances, such as components in a conditional branch or a
  list, use `newTabs(props)`.
- The created DOM for client-side instances uses the same constructor result.

## UI kit and Gothicbook

`TabsProps` gained an optional `Active *int`, and `ui.Tabs` now defines
`NewTabsModel`.

Gothicbook sets `model.TabStart` to 1 and passes it to Tabs. The page starts on
the Details tab on the server and keeps that state through hydration.

## Commit

- `9be189d` prop-seeded component state

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

Browser checks:

- Gothicbook starts on the Details tab,
- the Tabs state script contains `{"Active":1}`,
- tab switching and the fewer-tabs demo still work,
- no console errors.

## Limits

- constructors apply only to stateful components,
- no constructor error return,
- constructors must stay within the client Go subset,
- nondeterministic constructors would make SSR and client state drift,
- external component constructors are internal to the component library and
  are not exposed in the consumer manifest,
- component state still resets on unmount and remount.

## Follow-ups

- Nested stateful components.
- Go expression expansion: indexing, slicing, maps, and range loops.
- Path parameters and route matching.
- The virtualized list example.
