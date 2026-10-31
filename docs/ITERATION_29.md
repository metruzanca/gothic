# Iteration 29: children as a prop

Status: complete. Children are opted into through `gothic.Children` and placed
with `{ props.Children }`. The `<slot />` API is removed.

## Goal

React and Solid treat children as a prop. Gothic now does the same:

```go
type FieldProps struct {
	gothic.Children
	Label string
}
```

```templ
view Field(props FieldProps) {
	<div class="ui-field">
		<span>{ props.Label }</span>
		{ props.Children }
	</div>
}
```

```templ
<ui.Field label="Your name">
	<input value={ model.Name } oninput={ NameChanged{Text: event.target.value} } />
</ui.Field>
```

## Rules

- A component accepts children only when its props type declares
  `gothic.Children`.
- An embedded `gothic.Children` and a named `Children gothic.Children` field
  both work.
- The view must render `{ props.Children }` exactly once.
- Children are replaced at that position, not appended automatically.
- Children keep the caller's scope.
- `<slot />` is removed and reports a migration error.
- Precompiled stateful components cannot accept children; use an inline
  stateful component or an App-local component.

## Implementation

- `goapp.Field` records embedded fields and names them by their base type.
- Props extraction skips the `gothic.Children` marker and sets a `Children`
  component flag.
- The library manifest records whether a component accepts children.
- Pure components replace `{ props.Children }` with the expanded child nodes at
  compile time.
- Stateful components replace `{ props.Children }` with the internal children
  region and keep the existing App-scoped build, bind, and hydration path.
- `<slot />` nodes are rejected with a migration hint.

## Existing components

- `ui.Field` now embeds `gothic.Children`.
- The nested `Panel` embeds `gothic.Children`.
- Both render `{ props.Children }` at the same position their slots used.

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
(cd examples/nested && go test ./...)
(cd examples/virtual && go test ./...)
```

Tests cover embedded and named children fields, rejection without the field,
multiple children render positions, user `<slot />` rejection, pure expansion,
stateful slot generation, and both examples.

Browser checks:

- slotted Field content renders in Gothicbook,
- the nested Panel input still updates the App text,
- Panel state still toggles,
- no console errors.

## Commits

The children-as-prop work ships in this iteration.

## Limits

- one children prop,
- children on stateful components only in static App positions,
- no named slots or fallback content,
- precompiled stateful components cannot accept children.

## Follow-ups

- Component-valued props for named slots.
- Children on stateful components in branches and lists.
