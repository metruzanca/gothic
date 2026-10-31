# Gothicbook

A tiny design system demo for the installable gothic UI kit.

## Run

```sh
go generate
go run .
```

Open <http://localhost:8080>. Set `PORT` to change the port.

The example has local `replace` directives for the core and UI modules so it
builds against this checkout. A consumer would use tagged module versions.

## Shows

- `ui.Button`
- `ui.Card`
- `ui.Field` wrapping a controlled input and a stateful Counter
- `ui.Tabs`
- `ui.Accordion`
- `ui.Disclosure` in a conditional branch and in a paginated list
- `ui.Counter` as a purely client-side counter next to a server-backed action,
  with `Counter` using `Button` internally
- a short `.gth` usage snippet for each component
- props structs behind the named tag props
- a server-backed Button action that emits a message to the app, runs an RPC
  procedure, and receives the new count as model state
- slotted content that keeps its input event and state in the App
- a scrollable list with 20 disclosures across two pages of 10

The client counter, Tabs, Accordion, and Disclosure stay purely client-side.
The server action proves that the App owns procedures and components only emit
action messages. Paging unmounts the old list items, so expanded disclosures
reset when they return.

No database, auth, routing, or interactive playground.

## Social preview

The page is served by `gothic.MountApp`, so it ships a full document: title,
description, favicon links, OpenGraph tags, and an absolute `og:image`. The
1200x630 card is `assets/og.png`, rendered from `assets/og.html`. Re-render it
with headless Chrome after changing the card:

```sh
cd assets
google-chrome --headless=new --disable-gpu --hide-scrollbars \
  --window-size=1200,630 --screenshot=og.png "file://$PWD/og.html"
```

## Deploy

Railway deploys this folder as the `Gothic UI` service:

- repository `metruzanca/gothic`, branch `main`
- root directory `/examples/gothicbook`
- builder RAILPACK
- no dashboard build or start command

`railpack.json` owns the build. It installs `git`, drops the local replaces
from `go.mod`, and fetches `github.com/metruzanca/gothic` and
`github.com/metruzanca/gothic/ui` at the deployed commit
(`RAILWAY_GIT_COMMIT_SHA`, falling back to `main`). The rest of the Go build is
Railpack's default.

This means the demo runs the in-repo core rather than the released modules, so
a push that changes the core, the UI, or this example redeploys it. Published
module versions still matter for real consumers.

Locally, the replaces in `go.mod` point at this checkout, so `go test ./...`
here uses the working tree.

Do not add a `Dockerfile` to this folder. It would switch the service to the
Docker builder and bypass `railpack.json`.

