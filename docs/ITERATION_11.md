# Iteration 11: component props and Init

Status: complete. Stateful components can read their view props in `Update` and
react when props change. Component `Init` now runs on mount.

## Goal

Let a component use the props the parent already passes, without giving the
component its own server procedures or command loop.

After this iteration:

- component `Update` can take the view props as trailing parameters,
- prop changes dispatch `gothic.PropsChanged` to the component,
- component `Init` runs once after the first render,
- the UI kit uses props to clamp or reset component state.

## Props in Update

A component can opt in by declaring the view props after the message:

```go
func (m TabsModel) Update(msg gothic.Msg, items []Tab) (TabsModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case TabClicked:
		m.Active = msg.Index
	case gothic.PropsChanged:
	}
	if m.Active >= len(items) {
		m.Active = len(items) - 1
	}
	if m.Active < 0 {
		m.Active = 0
	}
	return m, nil
}
```

The view still declares the props:

```gothic
view Tabs(model TabsModel, items []Tab) {
	...
}
```

Rules:

- `Update(msg)` remains valid for components that do not need props.
- To use props, `Update` must take all props as trailing parameters, in view
  order.
- The compiler checks the count and types and reports the bad parameter with
  source context.
- Prop names and `Update` parameter names cannot collide with generated names
  such as `m`, `msg`, `model`, `dispatch`, `render`, `update`, `init`, `emit`,
  `root`, or `__gothicParentDispatch`.

The generated component receives current props:

```js
function update(m, msg, items) { ... }
```

and `dispatch` passes the current prop variables.

## PropsChanged

New sentinel:

```go
type PropsChanged struct{}
```

When the parent calls `updateProps`:

- each prop is compared with `Object.is`,
- changed props are assigned,
- if anything changed, the component dispatches `PropsChanged`,
- otherwise it re-renders without running `Update`.

Components can handle it in a type switch:

```go
case gothic.PropsChanged:
	if m.ItemID != item.ID {
		m.Expanded = false
		m.ItemID = item.ID
	}
```

The JavaScript type-switch compiler now accepts selector case types such as
`gothic.PropsChanged` and uses the selector name as the tag.

Comparison is by reference. Mutating a prop object or slice in place does not
notify the component. Assign a new value instead.

## Component Init

Component `Init` is emitted as part of the component module and called once
after the first render:

```js
render(model);
init();
```

Limits in this iteration:

- `Init` cannot take props yet,
- `Init` must return `nil`; component Init commands are not run yet,
- both cases fail generation with a hint.

`Init` can still perform synchronous browser effects through the `js` package,
such as setting the document title or focusing an element.

## UI kit

- `Tabs` clamps `Active` when the item list shrinks.
- `Accordion` resets `Open` when its items no longer contain the open index.
- `Disclosure` resets `Expanded` when its item ID changes.
- `Counter` keeps `Update(msg)` because its label is view-only.

## Gothicbook

The Tabs section gained "Fewer tabs" and "More tabs" buttons. Selecting the
third tab and then removing a tab shows the active index clamping from the
items prop.

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

- component `Init` runs and sets the document title,
- a component reads current props settings in `Update`,
- changing props dispatches `PropsChanged` and clamps component state,
- the Gothicbook Tabs clamps from active index 2 to 1 when the third tab is
  removed,
- adding the tab back keeps the clamped index,
- no console errors.

## Limits

- no component commands,
- no props in component `Init`,
- no prop-seeded initial state; component models still start at the zero value,
- prop change detection is reference-based,
- no component destroy hook,
- procedures remain App-owned.

## Follow-ups

- Component commands with a mounted guard and a destroy hook.
- Props in `Init` through a prop-aware model constructor used by SSR and client.
- Nested stateful components.
- Path parameters and route matching helpers.
- The virtualized list example.
