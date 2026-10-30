> Historical note: component props later moved to props structs in
> [Iteration 13](ITERATION_13.md). Snippets here may show the older prop syntax.

# Iteration 8: component children and slots

Status: complete. Pure components can accept children and place them with a
single default `<slot />`. Children are written in the caller's scope, so App
state, events, and stateful components stay in the App.

## Goal

Compose layout without callback props.

Before this iteration, a reusable wrapper such as a form field could not accept
the App's own input. The alternatives were to duplicate markup or to introduce
function-valued props. A default slot keeps one event model and one state
owner:

```gothic
view Field(label string) {
	<div class="ui-field">
		<span class="ui-field-label">{ label }</span>
		<slot />
	</div>
}
```

```gothic
<ui.Field label="Your name">
	<input value={ model.SlotText } oninput={ SlotTextChanged{Text: event.target.value} } />
</ui.Field>
```

The input event stays in the App. `Field` only supplies layout.

## Syntax

A pure component view can contain one `<slot />`. A component call with
children uses opening and closing tags:

```gothic
<ui.Field label="Your name">
	<input ... />
</ui.Field>
```

Rules:

- one default slot per pure component,
- no named slots and no fallback content yet,
- a slot cannot have attributes or children,
- a slot cannot be the component root because pure components still render one
  element,
- a stateful component cannot accept children yet,
- calling a component with children when its view has no slot is an error,
- declaring more than one slot is an error,
- a slot outside a pure component is an error.

## Expansion

Children are expanded first, in the caller's environment. Props and enclosing
loop variables therefore resolve in the caller's scope. The component body is
expanded separately with its own prop environment, and the slot is replaced
with the already-expanded children.

This means children can contain text, interpolation, elements, loops,
conditionals, and stateful components. The compiler inserts them into the App
tree before stateful components are extracted, so normal app behavior applies.

If a pure component is called without children, the slot renders nothing.

## UI kit

New pure `ui.Field` component with a `label` prop and a default slot. The kit
stylesheet styles fields, labels, inputs, and focus states.

## Gothicbook

New "Slots" section:

- `ui.Field` wrapping a controlled text input bound to `model.SlotText`,
- the greeting updates from the App model as the user types,
- a second `ui.Field` wrapping `ui.Counter`, proving that a stateful component
  can live inside slotted content,
- a usage snippet.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
```

Generated modules:

```sh
go run ./cmd/gothic generate ./ui
(cd examples/gothicbook && go generate ./...)
node --check examples/gothicbook/app_gothic.js
```

Modules:

```sh
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
```

Browser checks:

- the Field input starts with the model value,
- typing updates the greeting through the App model,
- the Counter inside the second Field increments independently,
- both fields render inside their component layout,
- no console errors.

## Limits

- stateful components cannot accept children,
- no named slots,
- no fallback content for an empty slot,
- a slot cannot be the root of a pure component,
- `<slot />` is reserved and cannot be used as an HTML element in a view,
- stateful component children need a cross-module binding design.

## Follow-ups

- Browser JavaScript interop and first-class URL state (C).
- Actionable diagnostics (D).
- Stateful component children if the UI kit needs them.
- Named slots only if a real use case appears.
