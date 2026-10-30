> Historical note: component props later moved to props structs in
> [Iteration 13](ITERATION_13.md). Snippets here may show the older prop syntax.

# Iteration 7: stateful composition in branches and lists

Status: complete. Stateful components can live inside conditional branches and
inside keyed lists, including components whose views contain loops. Gothicbook
demonstrates a scrollable, paginated list where paging unmounts items and resets
their state.

## Goal

Finish conditional composition, then prove it with real lists.

After this iteration:

- a stateful component inside an `if` branch mounts when the branch appears and
  unmounts when the branch leaves,
- component state resets when a branch is re-entered,
- a `for` loop can be the direct child of a branch when the loop body is one
  element,
- stateful components work as keyed list items, including components with
  internal loops,
- list items unmount on removal and remount with initial state,
- Gothicbook has a scrollable, paginated list of expandable disclosures.

## Conditional components

```gothic
if model.ShowDetails {
	<ui.Disclosure item={ model.FeatureDisclosure } />
}
```

The branch builder creates the wrapper with the component's `CreateFunc`. The
branch binder finds the wrapper, mounts it with `MountFunc`, and stores the
instance on the wrapper for later `updateProps` calls.

When the branch swaps away, `ifBlock` removes every node in the region. The
wrapper and instance go with it. When the branch returns, a new wrapper is
built and a new instance starts from the component's initial state. There is no
destroy hook yet because components do not own timers or subscriptions.

Components nested inside elements in a branch follow the same path.

## Branch loops

A loop can be the direct child of a branch when the loop body is one element:

```gothic
if model.HasItems {
	for _, item := range model.Items {
		<li>{ item }</li>
	}
}
```

The branch builder emits the `g:each` markers and the binder runs the existing
list controller scoped to the branch region.

## CreateFunc for every component

Components are now creatable anywhere, not only in lists. The generator:

- emits a real `CreateFunc` for every stateful component,
- no longer throws for components whose views contain loops,
- no longer marks a component unlistable because of a `for` loop.

The `listable` field stays in the UI manifest for compatibility and is true for
all components in this iteration.

## Marker isolation fix

Client-created component wrappers previously carried only
`data-gothic-component-id`. SSR wrappers also carried `data-gothic-component`.
Because app-level lookup helpers skip elements with `data-gothic-component`,
they could descend into client-created components and bind to the wrong
markers. An expanded Disclosure's internal `for` collided with the app's keyed
list, so paging with an expanded item removed the visible list and inserted
rows into the disclosure details.

`CreateFunc` now sets both attributes:

```js
wrapper.setAttribute('data-gothic-component', "Disclosure");
wrapper.setAttribute('data-gothic-component-id', id);
```

App-level bindings skip the component subtree, and component-local bindings use
the `*Local` helpers. A generator test locks the attribute in place.

## Type switches

The JavaScript compiler now accepts both Go forms:

```go
switch msg := msg.(type) { ... }
switch msg.(type) { ... }
```

The second form avoids an unused variable in components that toggle state
without reading message fields, such as `Disclosure`.

## UI kit: Disclosure

New `ui.Disclosure` component:

- props: one `ui.DisclosureItem{ID, Title, Details}`
- model: `Expanded bool`
- view: a toggle button and a conditional details list with an internal `for`

Its view contains both an `if` and a `for`, so it exercises the lifted listable
restriction. Styles include an expanded state, a scrollable list container,
focus-visible outlines, and a plus/minus affordance.

## Gothicbook

New "Stateful lists" section:

- 20 disclosure items across two pages of 10,
- a scroll container with a fixed height of `15rem`, showing about five rows,
- Previous and Next page buttons,
- a page label that updates between `Showing 1-10 of 20` and
  `Showing 11-20 of 20`.

The Conditionals section now uses a conditional Disclosure. Expanding it,
hiding the branch, and showing it again proves that the component unmounts and
resets.

Paging uses two explicit slices (`PageOne`, `PageTwo`) because the client
expression compiler does not support slice indexing yet. `Update` assigns
`Visible` from one of them.

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
node --check ui/ui_gothic.js
node --check examples/gothicbook/app_gothic.js
```

Modules:

```sh
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
```

Browser checks:

- 10 list rows per page, with 5.3 rows visible in the viewport and scrolling to
  the last row,
- expanding an item shows its detail list,
- paging with an expanded item shows the other page collapsed,
- paging back shows the first page collapsed,
- expanding the conditional Disclosure, hiding the branch, and showing it again
  leaves the component collapsed,
- no console errors and no duplicate handlers.

## Limits

- stateful components cannot contain other stateful components,
- a branch-level loop body must be one element,
- component state is not preserved across list page changes or branch swaps,
- no destroy hook for component cleanup,
- no virtualized list yet,
- component commands, slots, callback props, and browser interop remain
  unsupported.

## Follow-ups

- B.3: a virtualized list example app, after URL and browser interop work.
- A.2: component children and slots, the preferred replacement for callback
  props.
- C: URL state and browser interop.
- D: actionable diagnostics.
