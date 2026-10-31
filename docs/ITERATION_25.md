# Iteration 25: responsive and variable-height virtual lists

Status: complete. `ui.VirtualList` follows its element height and supports
variable row heights.

## Goal

Finish the virtual list work:

- measure the viewport from CSS instead of requiring a fixed height,
- react to element resize,
- support rows with different heights,
- keep the client free of hand-written measurement code.

## Responsive height

The runtime now supports `onresize` on an element with a `ResizeObserver`:

```templ
<div onresize={ Resized{Height: event.target.clientHeight} }>
```

The observer fires when the element changes size and is disconnected when the
element or its component is destroyed.

`ui.VirtualList` listens on its root:

```templ
<div class="ui-virtual-list"
	onscroll={ VirtualScrolled{Top: event.target.scrollTop} }
	onresize={ VirtualResized{Height: event.target.clientHeight} }>
```

`VirtualResized` updates the model height and recomputes the window. The
component does not call the resize handler synchronously when binding, which
avoids re-entrant rendering while a row is still detached.

Height modes:

- `height > 0` sets a fixed inline height.
- `height={ 0 }` leaves the height to CSS.

The second mode lets an app size the list with CSS and still get a correct
visible window.

## js.ClientHeight

Apps can measure an element once from a command:

```go
height := js.ClientHeight("#rows")
```

It lowers to:

```js
(document.querySelector("#rows")?.clientHeight ?? 0)
```

## Variable rows

Set `rowheight={ 0 }` to use variable heights. An optional `estimate` prop sets
the fallback for rows that have not been measured yet, defaulting to 40px.

Each rendered row frame receives an `onresize` observer. When a row is measured,
it reports its index and height:

```go
type VirtualRowMeasured struct {
	Index  int
	Height int
}
```

Measured heights are stored in `map[int]int`. The window is recomputed with
`virtual.WindowForRows`:

```go
win := virtual.WindowForRows(heights, rowHeight, estimate, scrollTop, viewportHeight, total)
```

- `rowHeight > 0` uses the fixed `WindowFor` path.
- `rowHeight == 0` uses the variable path with measured heights and estimates.

`virtual.VariableWindow` is the explicit variable-height helper for headless
use.

## Example

`examples/virtual` now has:

- a fixed 5,000 row list with 36px rows,
- a variable-height 200 row list with `rowheight={ 0 }`,
- a CSS-driven height using `height={ 0 }` and `clamp(160px, 30vh, 216px)`,
- row content with different line counts.

The second list measures its CSS height and row heights after hydration, then
keeps only the visible variable rows in the DOM.

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

Tests cover `VariableWindow`, `WindowForRows`, resize updates, variable row
measurement, auto height styles, `js.ClientHeight` lowering, and both examples.

Browser checks through the Chrome DevTools protocol:

- the fixed list starts at 432px and keeps 12 rows,
- the variable list measures 197px from CSS and renders 7 variable rows,
- measured row heights are 20px or 39px,
- scrolling the variable list moves to later rows without touching the fixed list,
- no console errors.

## Commits

The responsive and variable-height work ships in this iteration.

## Limits

- row measurements trigger a full component update,
- the variable window scans measured heights to map scroll offsets,
- variable lists are best for hundreds or low thousands of rows,
- the full item slice still crosses the client state boundary,
- no row recycling across unmounts.

## Follow-ups

- A faster index for very large variable-height lists.
- Scroll throttling if measurement updates become visible.
- Row recycling for expensive stateful rows.
