# Iteration 18: virtualized list

Status: complete. A 5,000 row list keeps only its visible window in the DOM.

## Goal

Prove that the framework can virtualize a large list without hand-written
JavaScript. The list should render a fixed number of rows, update the window on
scroll, and keep the scrollbar stable.

## What was missing

Scroll events and keyed lists already worked:

```templ
<div class="viewport" onscroll={ Scrolled{Top: event.target.scrollTop} }>
```

Event expressions can read `event.target.scrollTop`, and keyed lists reconcile
visible rows.

The missing piece was formatting spacer heights. Go needs a string such as
`"3600px"`, while the client compiler cannot call `strconv.Itoa`.

## dom.Px

A small `dom` package formats pixel lengths:

```go
import "github.com/metruzanca/gothic/dom"

style={ "height:" + dom.Px(model.Top) }
```

- Server: `dom.Px` returns `strconv.Itoa(n) + "px"`.
- Client: the compiler lowers `dom.Px(x)` to a runtime `px(x)` helper.
- The view expression is ordinary Go and works on both sides.

## Window math

The model stores the row height, visible count, pixel offset, start index, and
bottom spacer height:

```go
m.Top = msg.Top - msg.Top%m.RowHeight
if m.Top < 0 {
	m.Top = 0
}
maxTop := (len(m.Items) - m.VisibleCount) * m.RowHeight
if m.Top > maxTop {
	m.Top = maxTop
}
m.Start = m.Top / m.RowHeight
m.VisibleItems = m.Items[m.Start : m.Start+m.VisibleCount]
m.Bottom = (len(m.Items) - m.Start - m.VisibleCount) * m.RowHeight
```

Snapping to a row boundary keeps `m.Top / m.RowHeight` exact in both Go and
JavaScript, so no float conversion is needed.

The view renders a top spacer, the visible keyed rows, and a bottom spacer. The
sum of their heights equals the full list height, so the scrollbar stays stable.

## Example

`examples/virtual` has 5,000 rows, 36px row height, and a 432px viewport with 12
visible rows.

- `onscroll` dispatches `Scrolled`.
- Update slices the visible window and recomputes spacers.
- The range label shows the current window.
- Only 12 row nodes exist at any time.

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

Browser checks:

- the initial window shows rows 1-12,
- scrolling to 3,600px shows rows 101-112,
- scrolling to the end shows rows 4,989-5,000,
- scrolling back to the top restores rows 1-12,
- the DOM contains 12 rows at each position,
- spacer heights keep the total scroll height at 180,000px,
- no console errors.

## Commits

- `5d3e7ff` `dom.Px` and the `px` runtime helper

The example and this record ship with the same iteration.

## Limits

- fixed row height,
- fixed viewport height,
- the full item list still lives in the client model and state script,
- scrolling does not preserve state for rows outside the window,
- scroll events are not throttled,
- no variable-height rows or scroll restoration.

## Follow-ups

- Responsive viewport measurement with `js.ClientHeight` and resize handling.
- Scroll throttling if the render cost becomes visible.
- Variable row heights.
- A virtualized list in the UI kit once content props or slots exist.
