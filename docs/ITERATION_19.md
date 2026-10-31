> Historical note: the hardcoded `ui.VirtualItem` was replaced by
> app-defined row views in [Iteration 20](ITERATION_20.md).

# Iteration 19: virtual list package and component

Status: complete. Virtualized lists are now a headless package and a UI kit
component.

## Goal

Move fixed-height virtualization out of the example and into reusable layers:

- a headless `virtual` package for custom rows,
- a `ui.VirtualList` component for standard rows.

## Headless package

```go
win := virtual.WindowFor(msg.Top, 36, 432, len(m.Items))
m.Start = win.Start
m.Count = win.Count
m.Top = win.Top
m.Bottom = win.Bottom
```

`virtual.WindowFor` returns a `virtual.Window` with:

- `Start`: first visible row index,
- `Count`: number of visible rows,
- `Top`: height of the top spacer,
- `Bottom`: height of the bottom spacer.

The rules are shared by the server and client:

- row height must be positive,
- empty lists return a zero window,
- the viewport must show at least one row,
- the scroll offset snaps to a row boundary,
- the start index clamps to the end of the list.

Client `Update` calls lower to a `virtualWindow` runtime helper.

`examples/virtual` now uses the UI kit component, while the headless helper
remains available for custom rows.

## UI component

```templ
<ui.VirtualList items={ model.Rows } height={ 432 } rowheight={ 36 } />
```

`ui.VirtualItem` has `ID`, `Title`, and `Detail`.

The component:

- seeds its initial window from props with `NewVirtualListModel`,
- handles `onscroll` internally,
- reflows on `PropsChanged`,
- uses `virtual.WindowFor` for its window and spacers,
- uses `dom.Px` for all heights,
- renders keyed rows with a dynamic row height,
- falls back to 432px and 36px when height or row height is zero.

Component model fields stay flat because component model serialization does not
support nested named structs yet.

## Gothicbook

Gothicbook has a Virtualized list section with 1,000 rows. Only 12 rows are in
the DOM. Scrolling moves the window and updates the spacers.

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

Tests cover the Go window helper, client lowering, the UI component render and
update paths, SSR row counts, and Gothicbook integration.

Browser checks:

- Gothicbook shows rows 1-12 at the top,
- scrolling to 3,600px shows rows 101-112,
- the end shows rows 989-1000,
- the DOM keeps 12 rows,
- spacers track the window,
- no console errors.

## Commits

- `289229a` headless virtual window helpers
- `749f248` UI kit VirtualList component

The Gothicbook section and this record ship with the same iteration.

## Limits

- fixed row height,
- fixed viewport height,
- standard Title/Detail rows in the kit component,
- the full item list still travels in the client state script,
- no row click callback,
- no scroll throttling,
- no responsive viewport or variable row heights.

Custom row markup uses the headless `virtual` package. Per-item component
children or slots are still unsupported.

## Follow-ups

- Responsive viewport measurement and resize handling.
- Scroll throttling.
- Variable row heights.
- Custom row content through children or slots.
- Row selection callbacks if library-emitted App messages land.
