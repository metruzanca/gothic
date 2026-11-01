# gothic UI kit

A headless, accessible component kit for gothic. The kit ships markup and
behavior. It does not ship CSS, class names, or a stylesheet mount.

Add the package to an app prelude:

```go
//gothic:components github.com/metruzanca/gothic/ui
```

Every part carries `data-part="<primitive>-<part>"` and state is exposed through
`data-*` attributes. Style them from your own stylesheet:

```css
[data-part="button"] { /* ... */ }
[data-part="button"][data-variant="primary"] { /* ... */ }
[data-part="tabs-trigger"][data-active="true"] { /* ... */ }
```

## Primitives

**Pure and controlled**

- `Button` with `type`, `variant`, `disabled`, `aria-label`, and `onclick`
- `Field` with `id`, `label`, optional `description` and `error`, and children
- `Checkbox` with `id`, `checked`, optional `name`, `value`, `disabled`, and `required`
- `Switch` with the same props as `Checkbox`
- `Tabs`, `TabsList`, `TabsTrigger`, `TabsPanel`
- `Accordion`, `AccordionItem`, `AccordionTrigger`, `AccordionPanel`

**Stateful**

- `Combobox` with local filtering, keyboard navigation, and a hidden input for
  native forms

The App owns state for the controlled primitives. For example:

```templ
<ui.Tabs>
	<ui.TabsList label={ dom.Ptr("Settings") } onkeynav={ TabKey{Key: event.key} }>
		for i, tab := range model.Tabs {
			<ui.TabsTrigger
				id={ dom.ID("settings-tab", i) }
				controls={ dom.Ptr(dom.ID("settings-panel", i)) }
				active={ model.ActiveTab == tab.ID }
				onclick={ TabClicked{ID: tab.ID} }
			>{ tab.Label }</ui.TabsTrigger>
		}
	</ui.TabsList>
	for i, tab := range model.Tabs {
		<ui.TabsPanel
			id={ dom.ID("settings-panel", i) }
			labelledby={ dom.Ptr(dom.ID("settings-tab", i)) }
			active={ model.ActiveTab == tab.ID }
		>{ tab.Panel }</ui.TabsPanel>
	}
</ui.Tabs>
```

## Combobox

```templ
<ui.Combobox
	id="fruit"
	items={ model.Fruits }
	fieldlabel={ dom.Ptr("Fruit") }
	value={ dom.Ptr(model.FruitValue) }
	label={ dom.Ptr(model.FruitLabel) }
	placeholder={ dom.Ptr("Pick a fruit") }
	name={ dom.Ptr("fruit") }
	clearable={ dom.BoolPtr(true) }
/>
```

The component emits `ui.ComboboxValueChanged` to the App:

```go
case ui.ComboboxValueChanged:
	m.FruitValue = msg.Value
	m.FruitLabel = msg.Label
```

## Styling contract

- `data-part` identifies the part.
- `data-open`, `data-active`, `data-checked`, `data-highlighted`,
  `data-invalid`, `data-required`, `data-disabled`, and `data-expanded` expose
  state.
- `aria-*` attributes are always correct. Empty `aria-*` and `data-*` values
  are omitted.
- IDs are caller supplied. They must be unique per page.

## Development

The generated files are committed:

- `ui_gothic.go`
- `ui_gothic.js`
- `gothic-components.json`

Regenerate them from the repository root:

```sh
go run ./cmd/gothic generate ./ui
```
