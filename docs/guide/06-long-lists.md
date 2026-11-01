# 06 Long lists

A list with thousands of rows does not need thousands of DOM nodes. Keep only
the visible window in the DOM and let the browser scroll a tall spacer. Core
gothic does not ship a virtual list component; the window math lives in the
separate `virtual` module so your own view stays in control.

```sh
go get github.com/metruzanca/gothic/virtual
```

## The window

`virtual.WindowFor` maps a scroll offset to a slice of rows:

```go
import "github.com/metruzanca/gothic/virtual"

win := virtual.WindowFor(msg.Top, 36, 432, len(m.Items))
m.Start = win.Start
m.Count = win.Count
m.Top = win.Top
m.Bottom = win.Bottom
```

- `scrollTop` is the pixel offset of the scroll container.
- `rowHeight` is the fixed height of one row.
- `viewportHeight` is the height of the visible area.
- `total` is the number of rows.

`Window` has four fields:

- `Start` is the first visible index.
- `Count` is how many rows to render.
- `Top` is the spacer height before the window.
- `Bottom` is the spacer height after the window.

The helper snaps to a row boundary and clamps to the end of the list. The same
rules run on the server and in the browser.

## Render the window

Put the state on the model:

```go
type Model struct {
	Rows   []Row
	Window virtual.Window
}

type Scrolled struct{ Top int }

func NewModel(rows []Row) Model {
	m := Model{Rows: rows}
	m.Window = virtual.WindowFor(0, 36, 432, len(rows))
	return m
}

func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case Scrolled:
		m.Window = virtual.WindowFor(msg.Top, 36, 432, len(m.Rows))
	}
	return m, nil
}
```

Render a scroll container, a top spacer, the visible slice, and a bottom spacer:

```templ
<div class="viewport" style={ "height:" + dom.Px(432) } onscroll={ Scrolled{Top: event.target.scrollTop} }>
	<div style={ "height:" + dom.Px(model.Window.Top) }></div>
	for _, row := range model.Rows[model.Window.Start : model.Window.Start+model.Window.Count] {
		<div class="row" key={ row.ID }>{ row.Text }</div>
	}
	<div style={ "height:" + dom.Px(model.Window.Bottom) }></div>
</div>
```

`dom.Px(n)` formats an integer pixel length on the server and lowers to a `px`
helper in the browser.

The keyed list keeps rows stable while the window moves. Without a key, the
client can only append and remove from the end.

## Variable row heights

Use `virtual.VariableWindow` when rows can have different heights:

```go
win := virtual.VariableWindow(heights, estimate, scrollTop, viewportHeight, total)
```

`heights` maps a row index to its measured height. Missing or zero entries use
`estimate`, which defaults to 40px.

`virtual.WindowForRows` selects fixed or variable mode from the row height:

```go
win := virtual.WindowForRows(heights, 0, 40, scrollTop, viewportHeight, total)
```

Variable mode needs measured heights. Measure rows after they render and store
the result in the model, or use `browser.ClientHeight` for a one-off
measurement. The example uses fixed rows.

## Full example

`virtual/example` is a 5,000 row app with two lists. It shows:

- a window computed in `NewModel` so SSR matches the first paint,
- a scroll message for each list,
- a keyed slice of the visible rows,
- top and bottom spacers,
- a second list with a different row height and markup.

Run it:

```sh
cd virtual/example
go run .
```

## Limits

- the full item list still lives in the client model,
- rows outside the window are not kept alive,
- the example uses fixed row heights and fixed viewport heights,
- scroll throttling is not included.

## What next

You have the full guide. The last chapter explains what the compiler emits and
how the runtime works:

- [Architecture](07-architecture.md)
- [Virtual example](../../virtual/example)
