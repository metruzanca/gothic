# Virtualized list

A 5,000 row list that keeps only the visible window in the DOM.

The scroll container is a fixed 432px tall with 36px rows, so 12 rows are
visible. Scrolling dispatches `Scrolled` with `event.target.scrollTop`. Update
calls `virtual.WindowFor` to snap the offset, slice the visible rows, and
compute the top and bottom spacer heights.

This example keeps custom row markup. The UI kit also ships
`ui.VirtualList` with standard Title/Detail rows, and the headless `virtual`
package is available at `github.com/metruzanca/gothic/virtual`.

The view renders:

- a top spacer sized with `dom.Px`,
- the visible rows, keyed by row ID,
- a bottom spacer.

The total scroll height stays constant, but the DOM holds 12 rows instead of
5,000.

Run it:

```sh
go run .
```

Then open <http://localhost:8080> and scroll. The range label should track the
visible rows, and the row count in the DOM stays at 12.

Limits:

- rows have a fixed height,
- the viewport height is fixed,
- the full item list still lives in the client model,
- rows outside the window are not kept alive.
