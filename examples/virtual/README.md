# Virtualized list

A 5,000 row list built with the UI kit `ui.VirtualList` component. Only the
visible window exists in the DOM.

The app passes the rows and the fixed size:

```templ
<ui.VirtualList items={ model.Rows } height={ 432 } rowheight={ 36 } />
```

The component owns its scroll offset, renders a top and bottom spacer, and
keeps 12 rows in the DOM for a 432px viewport with 36px rows.

Run it:

```sh
go run .
```

Then open <http://localhost:8080> and scroll inside the list.

The headless `github.com/metruzanca/gothic/virtual` package is available when
you need custom row markup instead of the standard Title/Detail rows.

Limits:

- fixed row height,
- fixed viewport height,
- the full item list still lives in the client model,
- rows outside the window are not kept alive.
