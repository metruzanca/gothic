# Iteration 27: parent-local child messages

Status: complete. A parent component can handle, transform, or bubble a child
action.

## Goal

Previously every child `emit` went straight to the App. Containers could not
react to their children.

## Local actions

`emit` keeps its existing behavior and bubbles to the App.

`emitLocal` sends the action to the immediate parent component:

```templ
<button onclick={ emitLocal(PickedChild{Label: props.Label}) }>Pick</button>
```

The parent receives it as a wrapper:

```go
type ChildMessage struct {
	Source string
	Inner  Msg
}
```

```go
case gothic.ChildMessage:
	inner := msg.Inner
	switch child := inner.(type) {
	case PickedChild:
		m.LastPick = child.Label
		return m, func() gothic.Msg {
			return gothic.Bubble{Msg: child}
		}
	}
```

- `Source` identifies the child component instance.
- The parent can update its own model and stop the action.
- The parent can return a `gothic.Bubble` command to forward the message to its
  parent.
- At App level, `emitLocal` falls back to App dispatch.
- Library components keep using `emit` unless they opt into local actions.

## Runtime

Each component mount receives:

- parent dispatch, for `emit`,
- local dispatch, for `emitLocal`.

For a nested child, the parent supplies a local dispatcher that wraps messages
as `ChildMessage` with the child source.

The component command runner recognizes `Bubble` and forwards the inner message
with `emit`. The App runner forwards a Bubble to App dispatch.

## Type switch fix

Handling a wrapper message exposed a compiler gap:

```go
inner := msg.Inner
switch child := inner.(type) { ... }
```

The compiler now emits:

```js
const child = inner;
switch (child.tag) { ... }
```

Before, it switched on `child.tag` without defining `child`. Existing
`switch msg := msg.(type)` still works without an extra assignment.

## Nested example

`examples/nested` now uses `emitLocal` from Counter.

Dashboard handles `gothic.ChildMessage`, stores the picked label, and bubbles
the original message to the App. The page shows both:

- `Dashboard pick: ...` from the parent component,
- `Picked: ...` from the App.

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

Tests cover `emitLocal` lowering, component-local dispatch wiring, Bubble
commands, the type switch fix, App-view rejection, and the nested example.

Browser check:

- clicking a nested Pick updates `Dashboard pick`,
- the same action bubbles to the App and updates `Picked`,
- multiple children update the right label,
- no console errors.

## Commits

The parent-local action work ships in this iteration.

## Limits

- `Source` is the component instance id, not the list item key,
- list children should carry their own item identity in local messages,
- parent-local actions currently require local components whose message types
  are in the same app prelude,
- library components do not opt into local actions yet.

## Follow-ups

- A source key that includes list item keys.
- Library components exposing local actions.
- Parent-defined action handlers as first-class props.
