# UI primitives plan

Status: proposed

Scope: turn the `ui` module into an unstyled, accessible primitives package
for gothic, starting with a small subset and a combobox.

## Goal

Make `github.com/metruzanca/gothic/ui` a headless primitive library inspired by
[Base UI](https://base-ui.com), not a React wrapper and not a styled design kit.

The package should:

- ship no CSS and no bundled class names,
- render accessible markup with correct roles, ARIA attributes, and keyboard
  behavior,
- expose stable `data-slot` and `data-*` state hooks for styling,
- keep the API usable from `.gth` views without npm, refs, or React patterns,
- give the App typed messages for meaningful state changes.

## What `ui` is today

`ui` is a precompiled gothic library, not a React component package.

- Pure components (`Button`, `Field`) ship as `.gth` source and are expanded
  into the consumer's app by the generator.
- Stateful components (`Tabs`, `Accordion`, `Disclosure`, `Counter`) are
  compiled into `ui_gothic.go`, `ui_gothic.js`, and `gothic-components.json`.
- Styling is currently bundled as `ui/ui.css`, served by `ui.MountStyles`, and
  applied through `.ui-*` classes.

Gothic already supports SSR plus hydration, ARIA and `data-*` attributes,
keyed lists, `focus`, `hidden`, `onenter` and `onescape`, and nested stateful
components. It does not support portals, floating positioning, refs, render
props, function-valued props, controlled components, auto-generated instance
IDs, or arbitrary DOM access.

The result will therefore follow Base UI's principles, not its exact API.

## Principles

1. **Unstyled**: no CSS, no `MountStyles`, no visual class names.
2. **Accessible by default**: correct roles, ARIA states, labels, keyboard
   behavior, and focus handling.
3. **Composable where gothic allows it**: pure primitives accept children;
   stateful primitives expose structured props and data hooks.
4. **App owns domain state**: primitives own their widget state and emit typed
   messages for values the App needs.
5. **One supported way**: replace the old kit completely, migrate every call
   site, and do not keep compatibility shims.
6. **SSR first**: the initial HTML must be accessible before JavaScript runs.

## Proposed decisions

These are the recommended defaults for this change.

1. **API shape**: drop-in stateful primitives with fixed markup, local state,
   `data-slot` styling hooks, and typed messages. Do not build the
   App-owned-state-machine variant in this pass.
2. **Core prerequisites**: include the seven framework additions listed below.
   Without them the combobox cannot reliably report selection, filter, or
   handle keyboard navigation.
3. **Subset**: `Button`, `Field`, `Checkbox`, `Switch`, `Tabs`, `Accordion`,
   `Combobox`.
4. **IDs**: every interactive primitive that needs ARIA relationships takes a
   required, unique `ID` prop. No auto-generated instance IDs in this pass.
5. **Combobox scope**: single-select, local filtering, plain-text option labels,
   in-flow popup positioned by consumer CSS. No async items, multiple select,
   grouping, chips, portal, or floating positioning.
6. **Version order**: these changes need a core release before a `ui` release.
   Development uses the local `go.work`; do not tag releases in this change.

## Component subset

| Primitive | Kind | Provides | Emits |
|---|---|---|---|
| `Button` | pure | Native button, children, `type`, `disabled`, `ariaLabel` | App-provided `onclick` message |
| `Field` | pure | Label, description, error, `data-invalid`, children | none |
| `Checkbox` | stateful | `role="checkbox"`, `aria-checked`, indicator slot, hidden native input | `CheckboxChanged{ID, Checked, Value}` |
| `Switch` | stateful | `role="switch"`, same form behavior | `SwitchChanged{ID, Checked, Value}` |
| `Tabs` | stateful | `tablist`/`tab`/`tabpanel`, roving tabindex, Arrow/Home/End, hidden panels | `TabsChanged{ID, Index}` |
| `Accordion` | stateful | Trigger and panel, `aria-expanded`/`aria-controls`, single-open | `AccordionChanged{ID, OpenIndex}` |
| `Combobox` | stateful | Input plus listbox, filtering, keyboard navigation, `aria-activedescendant`, hidden input | `ComboboxValueChanged{ID, Value, Label}` |

### Removed from `ui`

- `Card`: presentational, not a primitive. Move the card markup into the
  gothicbook example if it is still useful.
- `Counter`: a demo component, not a primitive.
- `Disclosure`: superseded by `Accordion`. The gothicbook stateful-list demo
  can use one `Accordion` per row.

### Deferred

`Dialog`, `AlertDialog`, `Menu`, `Popover`, `Tooltip`, `Select`, `RadioGroup`,
`Slider`, and `Toast`. They need focus trapping, outside-click handling,
positioning, or controlled state that gothic does not have yet.

## Styling contract

No component ships CSS. Consumers style with these hooks.

### Parts

Every part gets `data-slot="<primitive>-<part>"`.

| Primitive | Slots |
|---|---|
| `Button` | `button` |
| `Field` | `field`, `field-label`, `field-description`, `field-error` |
| `Checkbox` | `checkbox`, `checkbox-control`, `checkbox-indicator`, `checkbox-label` |
| `Switch` | `switch`, `switch-control`, `switch-thumb`, `switch-label` |
| `Tabs` | `tabs`, `tabs-list`, `tabs-tab`, `tabs-panel` |
| `Accordion` | `accordion`, `accordion-item`, `accordion-header`, `accordion-trigger`, `accordion-panel` |
| `Combobox` | `combobox`, `combobox-label`, `combobox-input`, `combobox-popup`, `combobox-option`, `combobox-empty`, `combobox-clear`, `combobox-hidden-input` |

### State

State is exposed as presence attributes:

- `data-open`
- `data-checked`
- `data-selected`
- `data-highlighted`
- `data-disabled`
- `data-invalid`
- `data-required`
- `data-active`
- `data-empty`

ARIA attributes always carry correct values, including `"false"` where ARIA
defines a default. Presence attributes are omitted when false.

Consumers scope styles with their own wrapper class:

```css
.fruit-combobox [data-slot="combobox-input"] {
	border: 1px solid var(--border);
}

.fruit-combobox [data-slot="combobox-popup"] {
	position: absolute;
	inset-inline: 0;
	top: 100%;
}

.fruit-combobox [data-slot="combobox-option"][data-highlighted] {
	background: var(--highlight);
}

.fruit-combobox [data-slot="combobox-option"][data-selected] {
	font-weight: 600;
}
```

Components do not accept per-part `class` props in this pass. Scoping through a
wrapper is the supported approach.

## Primitive APIs

### Button

```go
type ButtonProps struct {
	gothic.Children
	Type      *string
	Disabled  *bool
	AriaLabel *string
	Onclick   gothic.Msg
}
```

- Renders a native `<button>` with `type` defaulting to `"button"`.
- Uses `data-slot="button"` and `data-disabled`.
- `Onclick` is the App's message. A nil message is harmless because dispatch
  ignores nils.
- No visual variants.

### Field

```go
type FieldProps struct {
	gothic.Children
	ID          string
	Label       string
	Description *string
	Error       *string
}
```

- Renders `field`, `field-label`, optional `field-description`, and optional
  `field-error`.
- Sets `data-invalid` when `Error` is non-nil.
- The App passes the same `ID` to the control so the label is associated.
- The App owns `aria-describedby` wiring for description and error text.

### Checkbox

```go
type CheckboxProps struct {
	ID         string
	Label      *string
	AriaLabel  *string
	Checked    *bool
	Disabled   *bool
	Required   *bool
	Invalid    *bool
	Name       *string
	Value      *string
}

type CheckboxChanged struct {
	ID      string
	Checked bool
	Value   string
}
```

- Root `data-slot="checkbox"` with `data-checked`, `data-disabled`,
  `data-invalid`, and `data-required`.
- Button with `role="checkbox"`, `aria-checked`, `tabindex` not needed because
  it is a native button.
- Visual indicator at `data-slot="checkbox-indicator"`.
- Optional label at `data-slot="checkbox-label"` associated by `for`.
- Optional hidden native checkbox with `name`, `value`, and `checked` for
  native form submission.
- Local toggle, then `gothic.Emit(CheckboxChanged{...})`.
- Seed from `Checked` on mount and when the prop changes. Local toggles do not
  overwrite the tracked prop value, so an App that ignores the emitted message
  does not reset the checkbox on unrelated prop updates.

### Switch

Same as `Checkbox`, with:

- root `data-slot="switch"`,
- control `role="switch"`,
- `data-slot="switch-thumb"`,
- `SwitchChanged{ID, Checked, Value}`.

### Tabs

```go
type Tab struct {
	Label string
	Panel string
}

type TabsProps struct {
	ID     string
	Items  []Tab
	Active *int
	Label  *string
}

type TabsChanged struct {
	ID    string
	Index int
}
```

- Root `data-slot="tabs"`, `data-orientation="horizontal"`.
- List `role="tablist"` with `aria-orientation="horizontal"` and optional
  `aria-label`.
- Tabs: `role="tab"`, `aria-selected`, `aria-controls`, `tabindex` (`0` for
  active, `-1` otherwise), `data-active`.
- Panels: `role="tabpanel"`, `aria-labelledby`, `tabindex="0"`, `hidden` when
  inactive, `data-active`.
- `onkeynav` handles ArrowLeft, ArrowRight, Home, and End. The new active tab
  receives focus through a `focus` binding that is armed only after keyboard
  navigation, so initial page load does not steal focus.
- Click updates the active tab and emits `TabsChanged`.
- `Active` seeds and updates the active index when the prop changes. Out of
  range values clamp.
- Panels remain mounted and are hidden with `hidden`, so `aria-controls` always
  resolves.

### Accordion

```go
type AccordionItem struct {
	Title string
	Body  string
}

type AccordionProps struct {
	ID          string
	Items       []AccordionItem
	DefaultOpen *int
}

type AccordionChanged struct {
	ID        string
	OpenIndex int
}
```

- Root `data-slot="accordion"`.
- Items at `data-slot="accordion-item"` with `data-open`.
- Heading element with `data-slot="accordion-header"` containing a native
  button at `data-slot="accordion-trigger"`.
- Trigger has `aria-expanded`, `aria-controls`, and `data-open`.
- Panel has `role="region"`, `aria-labelledby`, `hidden` when closed, and
  `data-open`.
- Single-open behavior. Clicking the open trigger closes it.
- `OpenIndex` is zero-based, with `-1` meaning all closed.
- `DefaultOpen` seeds the initial open item only.
- Heading level is fixed at `h3` in this pass.

### Combobox

```go
type ComboboxItem struct {
	Value    string
	Label    string
	Disabled bool
}

type ComboboxProps struct {
	ID          string
	Items       []ComboboxItem
	Value       *string
	Label       *string
	AriaLabel   *string
	Placeholder *string
	EmptyText   *string
	Name        *string
	Disabled    *bool
	Required    *bool
	Invalid     *bool
	Clearable   *bool
	ClearLabel  *string
}

type ComboboxValueChanged struct {
	ID    string
	Value string
	Label string
}
```

#### Combobox model

```go
type ComboboxModel struct {
	Open         bool
	Query        string
	Active       int
	Value        string
	Label        string
	Filtered     []ComboboxItem
	EmptyText    string
	HasPropValue bool
	PropValue    string
}
```

Internal messages:

- `comboboxInputChanged{Text string}`
- `comboboxKeyDown{Key string}`
- `comboboxOptionPressed{Value string}`
- `comboboxBlurred{}`
- `comboboxOpened{}`
- `comboboxCleared{}`

#### Combobox markup

- Root `data-slot="combobox"` with `data-open`, `data-disabled`,
  `data-invalid`, and `data-required`.
- Optional visible label at `data-slot="combobox-label"`.
- Input with `id`, `data-slot="combobox-input"`, `type="text"`,
  `role="combobox"`, `autocomplete="off"`, `spellcheck="false"`,
  `aria-expanded`, `aria-controls`, `aria-autocomplete="list"`,
  `aria-activedescendant`, `aria-label`, `aria-required`, `aria-invalid`, and
  `disabled`.
- Popup rendered only when open, at `data-slot="combobox-popup"`,
  `role="listbox"`, `id="<ID>-popup"`, `data-empty`.
- Options at `data-slot="combobox-option"`, `role="option"`, with `id`,
  `aria-selected`, `aria-disabled`, `data-highlighted`, `data-selected`, and
  `data-disabled`.
- Empty state at `data-slot="combobox-empty"`.
- Optional clear button at `data-slot="combobox-clear"` with
  `aria-label` from `ClearLabel` or `"Clear selection"`.
- Optional hidden input at `data-slot="combobox-hidden-input"` with `name` and
  the selected value.

#### Combobox behavior

- Typing filters case-insensitively on `Label` using
  `strings.Contains(strings.ToLower(label), strings.ToLower(query))`.
- Query is trimmed before filtering.
- Empty query shows all items.
- On open by click or ArrowDown, if the query equals the selected label, the
  query is cleared so the whole list is visible. Closing restores the selected
  label.
- ArrowDown and ArrowUp open the popup and move the active option, skipping
  disabled items.
- Home and End jump to the first and last enabled option.
- PageUp and PageDown move by ten enabled options.
- Enter selects the active option. With no active option, Enter opens the popup.
- Escape closes the popup and restores the selected label.
- Clicking an option selects it on `mousedown`. Blur closes the popup and
  restores the selected label.
- Selecting updates local `Value`, `Label`, and `Query`, closes the popup, and
  emits `ComboboxValueChanged`.
- Clear resets local value, label, and query, closes the popup, and emits
  `ComboboxValueChanged` with empty value and label.
- `Value` seeds and updates selection when the prop changes. Local selection
  does not overwrite the tracked prop value, so an App that ignores the emitted
  message does not reset the combobox on unrelated prop updates.
- `Items` changes recompute the filtered list and clamp the active index.
- Required IDs:
  - input: `ID`
  - popup: `ID + "-popup"`
  - option: `ID + "-option-" + index`
- Option values must be unique and non-empty because they are list keys and
  selection values.

#### Combobox limitations

- Single-select only.
- Local filtering only.
- Plain-text option labels.
- No grouping, separators, chips, or custom option rows.
- No async loading, loading state, or remote filtering.
- No portal, flip, shift, or available-height positioning. The popup is inside
  the component root and consumers position it with CSS.
- No exit animation because the popup is conditionally rendered.
- Mouse selection blurs the input because gothic has no `preventDefault` hook
  in this pass.
- The App resets the primitive by changing `Value` or remounting it with a
  `key`.

## Core prerequisites

These are additive framework changes. Each needs tests and documentation in
`docs/SYNTAX.md`.

### 1. `gothic.Emit`

```go
// Emit sends a message from a component to its parent App.
//
// It is only meaningful as a command returned from a component's Init or
// Update. The App handles the message in its own Update.
func Emit(msg Msg) Cmd {
	return func() Msg { return msg }
}
```

The client compiler lowers `gothic.Emit(x)` in component modules to:

```js
() => emit(x)
```

`emit` already forwards to the parent dispatch, so nested components bubble to
the App. Using `gothic.Emit` outside a component is a compile error.

Example:

```go
func (m ComboboxModel) Update(msg gothic.Msg, props ComboboxProps) (ComboboxModel, gothic.Cmd) {
	switch msg := msg.(type) {
	case comboboxOptionPressed:
		for _, item := range m.Filtered {
			if item.Value == msg.Value {
				m.Value = item.Value
				m.Label = item.Label
				m.Query = item.Label
				m.Open = false
				m.Active = -1
				return m, gothic.Emit(ComboboxValueChanged{
					ID:    props.ID,
					Value: item.Value,
					Label: item.Label,
				})
			}
		}
	}
	return m, nil
}
```

### 2. Client `strings` subset

Support these calls in the client compiler:

- `strings.ToLower`
- `strings.ToUpper`
- `strings.TrimSpace`
- `strings.Contains`
- `strings.HasPrefix`
- `strings.HasSuffix`
- `strings.EqualFold`

Lower them to native string operations:

```js
String(s).toLowerCase()
String(s).toUpperCase()
String(s).trim()
String(s).includes(String(sub))
String(s).startsWith(String(prefix))
String(s).endsWith(String(suffix))
String(a).toLowerCase() === String(b).toLowerCase()
```

### 3. `dom` helpers and auto-import

Add to `dom/dom.go`:

```go
func Str(s *string) string
func StrOr(s *string, fallback string) string
func True(b *bool) bool
func If(cond bool, whenTrue string, whenFalse string) string
func ID(base string, index int) string
```

Client lowering:

```js
(arg == null ? "" : arg)
(arg == null ? fallback : arg)
(arg === true)
(cond ? whenTrue : whenFalse)
(index < 0 ? "" : base + "-" + index)
```

`dom.ID` returns `""` for negative indexes so `aria-activedescendant` can be
omitted when nothing is active.

Teach the app generator to import `dom` when a view uses it. Pure library
components are expanded into the consumer's app, so their views can use `dom`
helpers without importing `dom` in the `.gth` prelude. Stateful `.gth` files
must import `dom` because their views are generated into `ui_gothic.go`.

### 4. Empty `aria-*` and `data-*` attributes are omitted

In both SSR and the client `attr` helper:

- an empty string value for an `aria-*` or `data-*` attribute removes or omits
  the attribute,
- non-empty values behave as today.

This enables presence attributes and clean optional ARIA:

```templ
<div data-highlighted={ dom.If(i == model.Active, "true", "") }></div>
<input aria-activedescendant={ dom.ID(props.ID+"-option", model.Active) } />
```

### 5. `onkeynav` event

A keydown helper that only fires for navigation keys and prevents their default
browser behavior.

Keys:

- ArrowUp
- ArrowDown
- Home
- End
- PageUp
- PageDown
- Enter
- Escape

Generated handler shape:

```js
(event) => {
	if (!["ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown", "Enter", "Escape"].includes(event.key)) return;
	event.preventDefault();
	dispatch({ tag: "comboboxKeyDown", Key: event.key });
}
```

This stops caret movement during option navigation and stops Enter from
submitting a form. `onenter` and `onescape` keep their current behavior.

### 6. `scroll` attribute

Mirror the `focus` binding:

```templ
<div scroll={ i == model.Active }>...</div>
```

Runtime helper scrolls the element into view when the expression becomes true:

```js
el.scrollIntoView({ block: "nearest" });
```

Used by combobox options so keyboard navigation keeps the active option
visible.

### 7. Nil message guard

Make App dispatch and component dispatch ignore nil messages:

```js
function dispatch(msg) {
	if (msg == null) return;
	// ...
}
```

This makes `onclick={ nil }` and similar patterns safe.

## Repo changes

### `ui/`

Delete:

- `ui/ui.css`
- `ui/styles.go`
- `ui/card.gth`
- `ui/counter.gth`
- `ui/disclosure.gth`

Add or rewrite:

- `ui/button.gth`
- `ui/field.gth`
- `ui/checkbox.gth`
- `ui/switch.gth`
- `ui/tabs.gth`
- `ui/accordion.gth`
- `ui/combobox.gth`
- `ui/doc.go`
- `ui/README.md`

Add tests:

- `ui/combobox_test.go`
- `ui/checkbox_test.go`
- `ui/switch_test.go`
- `ui/tabs_test.go`
- `ui/accordion_test.go`

Regenerate and commit:

- `ui/ui_gothic.go`
- `ui/ui_gothic.js`
- `ui/gothic-components.json`

`ui/gothic.json` stays as the library config.

### `examples/gothicbook`

- Remove the `ui` import from `main.go` if it is only used for `ui.Styles`.
- Remove `uiStyleHref` and the `/gothic-ui.css` asset.
- Remove `ui.MountStyles(s)`.
- Remove `/gothic-ui.css` from `Document.Styles`.
- Rewrite `style.css` to target `[data-slot=...]` and `[data-*]` instead of
  `.ui-*` classes. Remove the `--ui-*` aliases.
- Add combobox styles, checkbox styles, switch styles, and the new tabs,
  accordion, button, and field styles.
- Rewrite `app.gth` pages:
  - Button with children.
  - Field with a native input.
  - Checkbox and Switch demos with value readouts.
  - Tabs with the existing items and controls.
  - Accordion.
  - Combobox with local filtering, keyboard navigation, selection readout,
    clear, and a form-style hidden input.
  - Conditionals using `Accordion` instead of `Disclosure`.
  - Stateful lists using one `Accordion` per row.
  - Server state using a `Button` and the existing counter display, without
    `ui.Counter`.
- Update nav, overview tiles, code samples, and `examples/gothicbook/README.md`.
- Update `main_test.go` assertions for the new markup, pages, and modules.

Regenerate and commit:

- `examples/gothicbook/app_gothic.go`
- `examples/gothicbook/app_gothic.js`
- `examples/gothicbook/app_components_gothic.js`

### Root and docs

- `README.md`: replace "UI kit" with "Primitives", update the description and
  links.
- `docs/SYNTAX.md`:
  - add `gothic.Emit` under actions,
  - add the supported `strings` functions to the client subset,
  - add the `dom` helpers,
  - document `onkeynav`,
  - document `scroll`,
  - document empty `aria-*`/`data-*` omission,
  - remove `/gothic-ui.css` from document examples.
- `docs/guide/03-components.md`:
  - add `gothic.Emit` under actions,
  - replace the UI kit section with a short primitives section pointing at
    `ui/README.md`.
- `docs/guide/04-routing.md`: remove `/gothic-ui.css` from the styles example.
- `docs/guide/07-architecture.md`: update the `ui/` row.
- `docs/PLAN.md`: update the `ui/` and `examples/gothicbook/` rows, the
  supported list, and the limitations.
- `docs/PITCH.md`: update the UI kit bullets.
- `docs/RELEASING.md`: rename the UI kit row and wording to UI primitives.

### Versioning

- `ui` needs the new core APIs at release time.
- Keep `ui/go.work` for development against `..`.
- Bump `ui/go.mod` to the new core version during the core release, then release
  `ui`.
- Do not tag releases in this change. Releases are initiated separately.

## Testing and verification

### Core tests

- `dom/dom_test.go`: `Str`, `StrOr`, `True`, `If`, `ID`.
- `generator/appjs` tests:
  - `gothic.Emit` lowers to `() => emit(...)`,
  - `gothic.Emit` errors in App code,
  - `strings` calls lower correctly,
  - `dom` helpers lower correctly,
  - `onkeynav` generates the key filter and `preventDefault`,
  - `scroll` generates the marker and binding,
  - nil dispatch guard is emitted.
- `generator/generator_test.go`:
  - empty `aria-*`/`data-*` attributes are omitted in SSR,
  - `scroll` writes the marker,
  - `dom` is imported when a pure view uses it.
- Runtime `attr` behavior is covered by the generated integration tests and the
  manual browser pass.

### UI tests

- Combobox filtering, navigation, Enter selection, Escape restore, blur
  restore, clear, disabled items, prop seeding, and emitted messages.
- Checkbox and Switch toggling, disabled state, prop seeding, and emitted
  messages.
- Tabs clicks, arrow/home/end navigation, clamping, and emitted messages.
- Accordion toggle, single-open behavior, and emitted messages.

These tests run against the generated `ui_gothic.go` package.

### Example tests

- Document no longer references `/gothic-ui.css`.
- New pages render the expected headings and data slots.
- Combobox SSR includes `role="combobox"`, `aria-expanded`, `data-slot`, and the
  hidden input when `name` is provided.
- Checkbox and Switch SSR include the correct roles and ARIA.
- Tabs and Accordion SSR include roles, `aria-*`, and hidden panels.
- Generated JS exports `mountCombobox`, `mountCheckbox`, `mountSwitch`,
  `mountTabs`, and `mountAccordion`; old mounts are gone.
- Server-state and pagination assertions are updated for the new markup.

### Full verification

Run the commands in `AGENT.md`:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
node --check client/runtime/gothic.js
node --check examples/guestbook/app_gothic.js
node --check examples/router/app_gothic.js
node --check virtual/example/app_gothic.js
node --check ui/ui_gothic.js
```

Nested modules:

```sh
(cd ui && go test ./...)
(cd browser && go test ./...)
(cd virtual && go test ./...)
(cd virtual/example && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
```

Regenerate artifacts:

```sh
go run ./cmd/gothic generate ./ui
go generate ./examples/gothicbook
go generate ./examples/guestbook
```

### Manual browser pass

Run gothicbook and verify:

- combobox typing, filtering, ArrowUp/ArrowDown, Home/End, PageUp/PageDown,
  Enter, Escape, click selection, clear, blur, and hidden form value,
- no caret jump while navigating options,
- active option scrolls into view,
- tabs arrow navigation moves selection and focus,
- accordion toggles and closes,
- checkbox and switch toggle and submit,
- ARIA attributes in DevTools,
- SSR markup with JavaScript disabled.

## Risks and follow-ups

- Fixed markup limits customization. Per-part classes, custom option rows, and
  rich tab panels need a later framework feature.
- Local filtering does not scale to very large lists. Virtualized or remote
  filtering is a follow-up.
- Required unique IDs put a burden on the App. Auto-generated instance IDs are
  a larger framework project because gothic has no stable per-instance identity
  in lists.
- No positioning or portal. Consumers own popup placement.
- Mouse selection blurs the input. A future `prevent` event hook would fix it.
- The client Go subset grows. Every new lowering needs compiler tests and
  documentation.
- The `ui` artifacts and core version must stay in sync across releases.

## Implementation order

1. Core: `gothic.Emit`, `strings`, `dom` helpers and auto-import, empty
   aria/data omission, `onkeynav`, `scroll`, nil dispatch guard.
2. Core tests and `docs/SYNTAX.md` updates.
3. UI: primitive contract, `Button`, `Field`, `Checkbox`, `Switch`.
4. UI: `Tabs`, `Accordion`.
5. UI: `Combobox`, including clear and hidden input.
6. UI tests.
7. Regenerate `ui` artifacts.
8. Rewrite gothicbook and its tests, regenerate its artifacts.
9. Update root docs, guide, plan, pitch, and releasing docs.
10. Full verification and manual browser pass.
11. Release sequencing separately: core first, then `ui`, then gothicbook
    dependency bumps.
