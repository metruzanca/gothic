# Iteration 28: slots on stateful components

Status: complete. Inline stateful components can take one default slot whose
content stays in the App scope.

## Goal

Let stateful containers wrap app-defined content:

```templ
view Panel(model PanelModel, props PanelProps) {
	<section>
		<h2>{ props.Title }</h2>
		<slot />
	</section>
}
```

```templ
<Panel title="Slotted panel">
	<input value={ model.Text } oninput={ TextChanged{Text: event.target.value} } />
	<p class="slot-text">{ model.Text }</p>
</Panel>
```

The Panel keeps its own state. The input and text still use the App model.

## Scope

- one default slot,
- inline stateful components only,
- static App positions,
- named slots and fallback content are not implemented,
- precompiled library stateful components remain childless.

## Server rendering

The component view receives a slot component. At the slot location it renders a
container and the slot component:

```go
<div data-gothic-slot="default">
	if slot != nil {
		slot.Render(ctx, w)
	}
</div>
```

The App view builds the slot content in App scope, so App state and events are
captured by the closure.

## Client rendering

The component module creates the same slot container. The App module:

- finds the slot container after mounting the component,
- builds and inserts the slot content when the container is empty,
- binds slot text, attributes, events, conditionals, lists, and nested
  components on every App render,
- adopts server-rendered slot content on hydration.

The runtime `findEl` now treats a root inside a component as its own boundary.
This lets App-owned slot bindings attach inside a component wrapper while still
skipping nested components.

## Example

`examples/nested` gained a local `Panel` component with a slot. The App puts an
input and text inside the Panel.

- typing updates the App model and the slot text,
- Panel toggle updates Panel state,
- Dashboard child picks still work,
- the slot content is not duplicated after hydration.

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

Tests cover parser acceptance, Go slot rendering, client slot build/bind,
rejection of children without a slot, and the nested example.

Browser check:

- one slot container renders,
- typing updates the slot text,
- Panel state toggles,
- Dashboard local picks still update the parent and App,
- no console errors.

## Commits

The stateful slot work ships in this iteration.

## Limits

- one default slot,
- static App positions only,
- no fallback content,
- no named slots,
- precompiled stateful components cannot accept children.

## Follow-ups

- Named slots.
- Fallback content.
- Slotted components inside App branches and lists.
- Slots on precompiled library components through a runtime boundary.
