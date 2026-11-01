# Gothicbook

A small demo for the headless gothic UI kit.

## Run

```sh
go generate
go run .
```

Open <http://localhost:8080>. Set `PORT` to change the port.

The example has local `replace` directives for the core, UI, and browser
modules so it builds against this checkout. A consumer would use tagged module
versions.

## Shows

- `ui.Button` with primary, secondary, and disabled states
- `ui.Field` with a label, description, and inline validation
- `ui.Checkbox` and `ui.Switch` as controlled primitives
- compound `ui.Tabs` with app-owned active state and `onkeynav`
- compound `ui.Accordion` with app-owned open state
- `ui.Combobox` with local filtering, keyboard navigation, and typed messages
- a server-backed button that runs an RPC procedure
- client-side routing with server-rendered deep links
- a system-aware theme with a session-only manual toggle

The kit ships no CSS. `style.css` styles `[data-part]` and `[data-*]` directly.

## Navigation and theme

The sidebar links to flat, deep-linkable paths:

- `/` overview
- `/button`, `/field`, `/tabs`, `/accordion`, `/combobox`

`NewModelAt` maps the request path to the initial page on the server, and a
`LocationChanged` handler does the same on the client. Links use the `route`
attribute, and the active item is marked with `class:active`.

The theme follows `prefers-color-scheme`. The sidebar toggle only sets a
`data-theme` attribute for the current page load. It is not stored anywhere, so
a reload returns to the system setting. The toggle is registered by a small
script set through the `ExtraHead` field of `gothic.WithDocument`.

## Social preview

The page is served by `gothic.MountApp`, so it ships a full document: title,
description, favicon links, OpenGraph tags, and an absolute `og:image`. The
1200x630 card is `assets/og.png`, rendered from `assets/og.html`.

## Deploy

Railway deploys this folder as the `Gothic UI` service:

- repository `metruzanca/gothic`, branch `main`
- root directory `/examples/gothicbook`
- builder RAILPACK
- no dashboard build or start command

`railpack.json` owns the build. It installs `git`, then the build step drops
the local replaces from `go.mod` and fetches the released core, UI, and browser
modules at the deployed commit.

The stylesheet, wordmark, and social card URLs carry a content hash (`?v=...`).
The Railway edge caches static assets for hours, so the hash ensures a deploy is
never served the previous file. Generated JavaScript modules and the runtime are
served with `Cache-Control: no-store`.

Do not add a `Dockerfile` to this folder. It would switch the service to the
Docker builder and bypass `railpack.json`.
