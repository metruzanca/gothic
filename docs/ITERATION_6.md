# Iteration 6: conditional rendering

Status: working proof of concept. Views can choose a branch with `if`,
`else if`, and `else`. The server renders only the chosen branch, and the
client mounts, adopts, and removes branches as the model changes.

## Goal

Remove the `class:hidden` workaround for content that should not exist in the
DOM. Real interfaces need empty states, errors, and editing modes. After this
iteration:

- `.gth` view bodies accept `if`, `else if`, and `else`.
- The server renders one branch and leaves the other out of the HTML.
- The client adopts the server branch without a rebuild, swaps on change, and
  removes the unused nodes.
- Conditionals work at element child positions, inside loops, and inside pure
  components after expansion.
- Conditions use the existing Go expression subset plus `len`.

## Syntax

```templ
view App(model Model) {
	<div class="app">
		if len(model.Todos) > 0 {
			<ul>
				for _, todo := range model.Todos {
					<li>{ todo.Title }</li>
				}
			</ul>
		} else if model.Loading {
			<p>Loading todos.</p>
		} else {
			<p class="empty">Nothing to do yet.</p>
		}
	</div>
}
```

Branches can hold text, interpolation, elements, loops inside elements,
components, and nested conditionals. A view and a pure component still render
exactly one root element, so an `if` cannot be the root of a view.

## Parser

`parser.If` holds a condition and two branches:

```go
type If struct {
	Cond ast.Expr
	Then []Node
	Else []Node
}
```

`parseIf` reads the condition with `go/parser`, parses the then branch until
`}`, and checks for `else`. `else if` is a nested `If` in the else branch. A
stray `else` remains literal text, so the word still works inside prose.
Expansion substitutes component props and enclosing loop variables into the
condition and both branches.

## Server rendering

The Go renderer emits a real Go conditional and wraps it in comments:

```html
<!--g:if:0:open-->
<!--g:if:0:then-->
<p>...</p>
<!--g:if:0:close-->
```

Only the chosen branch and its marker are written. The else marker is always
emitted when the branch is chosen, even if the branch is empty, so the client
can distinguish an empty branch from a branch that was never evaluated.

## Client runtime

`ifBlock` owns one conditional region:

```js
ifBlock(root, "0", condition,
  () => { /* build then */ },
  () => { /* build else */ },
  (region) => { /* bind then */ },
  (region) => { /* bind else */ });
```

Behavior:

- On the first call, it reads the branch marker the server rendered and adopts
  that branch. If the client condition disagrees, it swaps immediately.
- On later calls, it binds the mounted branch again. If the condition changed,
  it removes every node between the open and close comments, builds the wanted
  branch into a fragment, and inserts it.
- Controllers are stored in a `WeakMap` keyed by the open comment node. The
  same call can run on every render, and list-scoped conditionals keep one
  controller per item.
- Re-binding is safe because `on` removes its `data-gothic-on` marker after
  binding. Text, attribute, class, and focus helpers are already idempotent.

`ifBlockLocal` uses the component-local helpers for conditionals inside a
stateful component.

Nested conditionals are handled by the parent branch binder. A parent branch
builder emits the nested open and close markers, and the binder starts the
nested `ifBlock` after insertion. Server-rendered nested content is adopted on
first render.

## Expression support

Conditions use the same Go subset as `Update` and events. This iteration adds
one builtin:

- `len(x)` compiles to a null-safe `.length` read on the client, so
  `len(model.Todos) > 0` works when the serialized slice is `null`.

`append` on a nil slice also became null-safe on the client:

```js
(m.Items = m.Items || []).push(item);
```

Go appending to a nil slice is common, and a conditional that clears a list
would otherwise crash the client on the next add.

## Client compiler

- Each `if` compiles to branch builders and binders.
- Branches are built as `DocumentFragment` values, so a branch can hold any
  number of nodes.
- Conditionals inside loops are emitted in both the item builder and the keyed
  hydration callback. The runtime controller deduplicates the two.
- Loops inside a branch element use the existing list helper, scoped to the
  branch region.
- Marker indexes are assigned in source order by both backends.

## Examples

TodoMVC now uses conditionals for:

- the error message, which exists only when `Failure` is set,
- the empty state, which exists only when there are no todos,
- inline editing, which swaps the read view for the edit input instead of
  hiding both with CSS.

Gothicbook gained a conditionals section with a toggle button and a pure Card
in the then branch.

## Limits

- Stateful components inside a conditional are not supported yet. The
  generator returns an explicit error instead of rendering a wrapper that never
  mounts.
- A `for` loop directly at the root of a branch is not supported. Wrap the loop
  in an element, which is the usual shape for a list with a wrapper.
- `switch` is not part of the view language.
- Remounting a branch rebuilds its DOM. Event handlers rebind, but any state
  held only in the DOM, such as caret position in a replaced subtree, is lost.
- Existing limits still apply: no nested loops in list items, no conditionals
  as a view root, and one app model per package.

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
go generate ./examples/pingpong ./examples/todo ./examples/components
(cd examples/gothicbook && go generate ./...)
node --check examples/todo/app_gothic.js
node --check examples/gothicbook/app_gothic.js
```

Modules with their own `go.mod`:

```sh
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
```

Browser checks:

- TodoMVC shows the empty state with no todos and swaps to the list after add.
- Inline editing swaps the label view for the edit input, focuses it, saves the
  new title, and swaps back.
- Gothicbook toggles the conditional panel repeatedly with no stale nodes.
- A nested conditional and an `else if` chain swap correctly.
- No duplicate event handlers fire after repeated toggles.
