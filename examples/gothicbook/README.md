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
- a server-backed Button action that emits a message to the app, runs an RPC
  procedure, and receives the new count as model state
- slotted content that keeps its input event and state in the App
- a scrollable list with 20 disclosures across two pages of 10

The client counter, Tabs, Accordion, and Disclosure stay purely client-side.
The server action proves that the App owns procedures and components only emit
action messages. Paging unmounts the old list items, so expanded disclosures
reset when they return.

No database, auth, routing, or interactive playground.
