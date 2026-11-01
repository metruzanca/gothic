# Virtualized list

A 5,000 row list built with the headless `github.com/metruzanca/gothic/virtual`
package. Only the visible window exists in the DOM.

The app owns the scroll offset and asks `virtual.WindowFor` for the window:

```go
m.RowsWin = virtual.WindowFor(msg.Top, rowHeight, rowsViewport, len(m.Rows))
```

The view renders two spacers and the visible slice:

```templ
<div class="viewport" style={ "height:" + dom.Px(rowsViewport) } onscroll={ RowsScrolled{Top: event.target.scrollTop} }>
	<div style={ "height:" + dom.Px(model.RowsWin.Top) }></div>
	for _, row := range model.Rows[model.RowsWin.Start : model.RowsWin.Start+model.RowsWin.Count] {
		<div class="row" key={ row.Title }>{ row.Title }</div>
	}
	<div style={ "height:" + dom.Px(model.RowsWin.Bottom) }></div>
</div>
```

A second list on the same page uses a different row height and markup. Both
use the same helper, so the window has the same shape on the server and in the
browser.

Run it:

```sh
go run .
```

Then open <http://localhost:8080> and scroll inside either list.

Limits:

- fixed row height,
- fixed viewport height,
- the full item list still lives in the client model,
- rows outside the window are not kept alive.
