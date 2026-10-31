# Iteration 20: app-defined row views

Status: complete. `ui.VirtualList` no longer owns the row shape. The app
defines the row type and a one-parameter row view.

## Goal

Replace the hardcoded `ui.VirtualItem` with:

```templ
<ui.VirtualList
	items={ model.Rows }
	height={ 432 }
	rowheight={ 36 }
	row={ Row }
/>
```

The app owns `RowData` and `Row`. The kit owns windowing and layout.

## Stage A: inline library components

A library can mark a stateful component as inline in `gothic.json`:

```json
{
  "inline": ["VirtualList"]
}
```

For inline components:

- the library still ships the model, props, constructor, and Update,
- the library does not precompile the component view or module,
- the app generator loads the component source,
- the component is generated into the app component module,
- model and props types are qualified from the library package.

This is what lets the component be compiled with the app row view.

## Stage B: row view baking

The inline `VirtualList` view contains a placeholder inside its window loop:

```templ
for i, item := range props.Items[model.Start : model.Start+model.Count] {
	<div class="ui-virtual-row" key={ model.Start + i } style={ model.RowHeightStyle }>
		<RowSlot item={ item } />
	</div>
}
```

At app generation:

- the app call must provide `row={ Row }`,
- `Row` must be a pure component with one parameter,
- the `row` prop is removed from the runtime call,
- the `items` expression is wrapped with `gothic.AnySlice`,
- `RowSlot` is replaced with a call to `Row`,
- the row argument is `item.(RowData)` on the server,
- the client compiler lowers the type assertion to the value.

The list frame keeps the absolute row index as its key and owns the fixed row
height. The app row view only supplies the inner markup.

## Type boundary

`VirtualListProps.Items` is `[]any`. The app slice stays strongly typed:

```go
type Model struct {
	Rows []RowData
}
```

Server rendering converts it at the boundary:

```go
ui.VirtualListProps{Items: gothic.AnySlice(model.Rows), ...}
```

The baked row call casts back:

```go
item.(RowData)
```

The client receives the original array and ignores the assertion, so the row
markup reads fields directly.

## Example

`examples/virtual` now defines:

```go
type RowData struct {
	Title  string
	Detail string
}
```

```templ
view Row(row RowData) {
	<div class="row-content">
		<strong class="title">{ row.Title }</strong>
		<span class="detail">{ row.Detail }</span>
	</div>
}
```

Gothicbook does the same with `VirtualRowData` and a `VirtualRow` view.

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

Tests cover inline baking, `gothic.AnySlice` and type assertion lowering, the
UI model, SSR row counts, and both examples.

Browser checks:

- the virtual example renders the app row markup,
- scrolling keeps 12 rows and updates the window,
- Gothicbook does the same with its own row view,
- no console errors.

## Commits

- `a14eb38` inline stateful library components
- the row view support and docs ship with this iteration

## Limits

- one row view per app in this iteration; variants per call landed in
  [Iteration 23](ITERATION_23.md),
- the row view must be pure,
- multiple virtual lists with different row views need generated variants,
- row height is still fixed,
- viewport height is still fixed,
- the full item slice still crosses the client state boundary.

## Follow-ups

- One row view per list call, with generated variants.
- Stateful row views.
- Responsive viewport and variable row heights.
- A clearer type parameter mechanism for inline components.
