# Iteration 23: one row view per virtual list call

Status: complete. Two virtual lists in one app can use different row types and
row views.

## Goal

Remove the one-row-view-per-app limit from Iteration 20.

```templ
<ui.VirtualList items={ model.Rows } row={ RowA } ... />
<ui.VirtualList items={ model.OtherRows } row={ RowB } ... />
```

## Variant generation

During app generation:

- collect every call to each inline component,
- group calls by the `row` view name,
- create one component variant per distinct row view,
- rewrite each call to its variant name,
- remove the compile-time `row` prop,
- wrap `items` with `gothic.AnySlice`,
- bake that row view into a clone of the inline component template.

Calls that use the same row view share one variant. Different row views produce
different generated mount, create, and view functions.

Variant names are internal, for example `ui.VirtualList#RowA`. The DOM display
name stays `VirtualList`.

## Example

`examples/virtual` now has two lists:

- 5,000 `RowData` rows using the `Row` view,
- 200 `OtherRowData` rows using the `OtherRow` view.

The second list proves that row types and markup are independent, and that
scroll state is isolated per list.

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

Tests cover call grouping, variant naming, call rewriting, type assertion
baking, two variants in the generated component module, and both lists in SSR.

Browser checks:

- both lists render their own row markup,
- the first list keeps 12 rows and the second keeps 6,
- scrolling one list does not affect the other,
- spacers update independently,
- no console errors.

## Limits

- one row view per list call,
- row views must be pure,
- inline components are still compiled per app.

## Follow-ups

- Multiple row views inside one list, if needed.
- Stateful row views.
- Responsive and variable-height virtual lists.
