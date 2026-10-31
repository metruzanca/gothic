# Iteration 22: Gothicbook navigation and theme

Status: complete. Gothicbook is now a routed, docs-style app with a scrolling
sidebar, a grouped component index, and a system-aware theme.

## Goal

Turn the single-page demo into something you can browse: pick a component from
the sidebar, land on its own URL, and share that link.

## Brand

The page no longer repeats the word "Gothicbook" as a heading. The wordmark is
followed by a styled `book` span, in the sidebar and on the social card. The
document title and OpenGraph title stay `Gothicbook`.

## Layout

- Two-column shell: a sticky sidebar and a scrolling content column.
- The sidebar groups links under Getting started, Components, and Patterns.
- The nav scrolls on its own, with the theme toggle pinned below it.
- Below 900px the sidebar stacks above the content and the nav wraps into a
  grid.
- The overview is a tile grid linking to every page.

## Routing

Flat, deep-linkable paths:

| Path | Page |
| --- | --- |
| `/` | Overview |
| `/button` | Button |
| `/card` | Card |
| `/field` | Field |
| `/tabs` | Tabs |
| `/accordion` | Accordion |
| `/disclosure` | Disclosure |
| `/counter` | Counter |
| `/virtual-list` | Virtual list |
| `/slots` | Slots |
| `/conditionals` | Conditionals |
| `/stateful-lists` | Stateful lists |
| `/server-state` | Server state |

`NewModelAt` maps the request path to the initial page, so deep links render on
the server before hydration. A `LocationChanged` handler maps the path again on
the client. Links use the `route` attribute and `class:active`, following the
router example. Unknown paths render a small Not found page.

The route switch is written out in both `NewModelAt` and `Update`. A shared
helper function is not emitted into the client module by the generator, even
though the Update call site is, so inlining is the supported form for now.

## Theme

- Light and dark values are CSS custom properties.
- `@media (prefers-color-scheme: dark)` sets the default dark palette.
- `:root[data-theme="dark"]` and `:root[data-theme="light"]` override it for a
  manual choice.
- The UI kit's `--ui-*` variables are mapped to the same palette, so buttons,
  cards, tabs, accordions, and disclosures follow along.
- `color-scheme` is set with the palette so native controls match.

The toggle is a sidebar button plus a small inline script injected with
`gothic.WithExtraHead`. The script flips `data-theme` on `documentElement` for
the current page load. Nothing is stored, so a reload returns to the system
setting. The button label and glyph are chosen with CSS from the effective
theme, so the script only handles the click.

## Tests

`main_test.go` now checks:

- the document metadata and default icon and social image
- the brand, groups, and overview tiles
- every route returns its own page inside the shell
- the active nav item on a deep link
- the button, field, tabs, slots, stateful list, and virtual list demos
- the Not found page
- the server-backed counter on `/server-state`
- pagination SSR on `/stateful-lists`
- generated modules still include `LocationChanged`

Browser checks covered clicking every nav item, reloading a deep link, toggling
the theme in both directions, reloading to confirm the system theme returns,
and the existing interactive demos on their new pages.

## Limits

- The manual theme choice is intentionally session-only.
- The client theme script is inline, not a separate asset.
- The route switch is duplicated between the server and client paths.
- Nested paths (`/components/button`) are not used, so all routes are flat.
