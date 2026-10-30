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
- `ui.Tabs`
- `ui.Accordion`
- `ui.Counter` as a purely client-side counter next to a server-backed action
- a short `.gth` usage snippet for each component
- a server-backed Button action that emits a message to the app, runs an RPC
  procedure, and receives the new count as model state

The client counter, Tabs, and Accordion stay purely client-side. The server
action proves that the App owns procedures and components only emit action
messages.

No database, auth, routing, or interactive playground.
