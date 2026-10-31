# Iteration 17: nested stateful components

Status: complete. A stateful component can render another stateful component in
a static position, an `if` branch, or a keyed list.

## Goal

Remove the last structural gap in component composition. Until this iteration,
pure components nested freely, but a stateful component could not contain
another stateful component.

## Child positions

```templ
view Dashboard(model DashboardModel, props DashboardProps) {
	<section>
		<Counter label="static" onpick={ PickedChild{Label: "static"} } />
		if model.ShowBranch {
			<Counter label="branch" onpick={ PickedChild{Label: "branch"} } />
		}
		for i, item := range model.Items {
			<Counter key={ i } label={ item } onpick={ PickedChild{Label: item} } />
		}
	</section>
}
```

- Static children mount once with their parent.
- Branch children mount when the branch opens and are destroyed when it closes.
- List children are created per item, reconciled by key, and destroyed when
  removed.
- Each child owns its own model.
- Props update on every parent render.
- A child `emit` bubbles through the parent to the App.

## Lifecycle

A parent render calls `updateProps` on static children. Branch and list children
use the same host code as App-level stateful components.

Child instances attach to their component node:

```js
if (c0Node) c0Node.__gothicInstances = { "c0": c0 };
```

The runtime's `destroyComponentTree` already walks nested component nodes, so a
branch close, list removal, or parent unmount destroys the whole subtree.

## Module wiring

Component modules now receive the full component set, not only the components
they generate.

- A local child in the same module is called directly.
- An external child adds imports for its `mount` and `create` functions.
- Component modules pass the parent's `emit` as the child's dispatch.

Example of the generated parent mount:

```js
const c0Node = root.querySelector('[data-gothic-component-id="c0"]');
const c0 = mountCounter(c0Node, {Label: "static", OnPick: {tag: "PickedChild", Label: "static"}}, emit);
if (c0Node) c0Node.__gothicInstances = { "c0": c0 };
```

## Compiler

- `GenerateComponents` accepts `AllStateful` for child lookup.
- Component view generation enables nested stateful calls.
- The Go renderer allows stateful calls in component views.
- Unknown children and external module imports keep source errors.
- Child actions still cannot be handled by the parent. They go to the App.

## Example

`examples/nested` has a stateful Dashboard with:

- a static Counter child,
- a branch Counter child,
- keyed Counter children in a list,
- a constructor that seeds list items from props,
- Pick buttons that emit through the dashboard to the App.

The README describes what to try in the browser.

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
```

Browser checks with `examples/nested`:

- static, alpha, and beta counters increment independently,
- adding a list child preserves existing child state,
- the branch child mounts at zero, increments, and resets after off and on,
- list children keep their state across the same toggle,
- picking a nested child updates the App,
- toggling the dashboard destroys and resets the whole nested tree,
- no console errors.

## Commits

- `5f30f04` static nesting
- `5550d30` branches and lists

The example and this record ship with the same iteration.

## Limits

- app-owned stateful component views must live in the app file; separate app
  `.gth` files are pure only,
- parent components cannot intercept child messages,
- stateful components still cannot receive children or slots,
- component-declared procedures remain unsupported,
- nested state resets on unmount,
- cyclic component references are not part of a supported module layout.

## Follow-ups

- Parent-local child message handling.
- Children and slots on stateful components.
- Nested routers and wildcard segments.
- The virtualized list example.
