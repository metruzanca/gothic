# Prompt: build the "Ping Pong" reference app

Build a small fullstack web app called **Ping Pong** in the framework named
below. It is a reference implementation for comparing how frameworks handle
server rendering, client interactivity, and a typed server call without a full
page reload.

Framework: `<fill in>`

## Observable behavior

1. `GET /` returns an HTML page. Before any JavaScript runs, the page contains a
   list with exactly one item whose text is `ready`.
2. The page has a button labeled `Ping`.
3. Clicking `Ping` must do three things in this order:
   1. Immediately append an item with the text `ping` to the list in the DOM.
      This must happen before the server responds.
   2. Send one request to the backend procedure named `Ping`.
   3. When the response arrives, append an item with the text `pong` to the
      list.
4. The page must not reload or navigate. Browser state such as a global variable
   set before the click must survive the interaction.
5. Clicking `Ping` again repeats the sequence, producing
   `ready, ping, pong, ping, pong`.
6. The server procedure `Ping` takes no arguments and returns the string
   `pong`.
7. Support a delay on the procedure response so the optimistic update is
   observable. A query parameter, environment variable, or config value such as
   `PING_DELAY_MS=500` is acceptable. Default to no delay.

After one click, the list must contain these items in order:

```
ready
ping
pong
```

## Technical requirements

- Server-side render the initial page. `curl /` must show `ready` in the HTML
  source, not only after client JavaScript executes.
- Client-side update the list without a full page reload.
- Keep the `ready` item as the initial state for both the server render and the
  client.
- Use one backend procedure for the request. The path and encoding are up to the
  framework. Record them in the report.
- Return a clear error response if the procedure fails. Handling the error in
  the UI is optional.
- Bind the server to `0.0.0.0` so other devices can reach it, and print the URL
  the app is listening on at startup.

## Hydration requirement

If the framework supports hydration, the client must adopt the
server-rendered list instead of clearing and rebuilding it.

Verify this with a `MutationObserver` installed before the client script runs:
record any removal or replacement of the initial `ready` list item during
hydration. A correct implementation changes text, attributes, or appends new
items, but does not remove the existing list item node.

If the framework cannot hydrate, say so in the report and describe what it does
instead.

## Constraints

- Do not use WebAssembly.
- Do not use a virtual DOM requirement as an excuse to skip server rendering.
  If the framework uses a virtual DOM, it must still server-render the initial
  HTML and hydrate or adopt it.
- Keep the app to the smallest idiomatic implementation. Do not add routing,
  styling, components, or state management beyond what the behavior requires.
- Do not add client-side routing, a database, authentication, or persistence.
- Prefer a single process and a single build command. If the framework requires
  a separate client build, document it.
- For frameworks that claim a single language for client and server, the app
  must not contain hand-written JavaScript or TypeScript. For frameworks that
  require a separate client language, clearly mark the client files.

## Acceptance tests

Run these checks and include the results in the report.

1. **SSR**: `curl -s http://localhost:8080/` contains `ready`.
2. **Initial list**: the rendered page contains exactly one list item before
   interaction.
3. **Click sequence**: after one click, the list text is `ready`, `ping`,
   `pong` in order.
4. **Optimistic update**: with a 500 ms procedure delay, `ping` appears in the
   DOM before `pong`, and `pong` appears after the response.
5. **No reload**: set `window.__pingPongAlive = true` in the console before
   clicking. After the click, `window.__pingPongAlive` is still `true`.
6. **One request per click**: the network log shows exactly one call to the
   procedure per click.
7. **Repeatable**: two clicks produce `ready, ping, pong, ping, pong`.
8. **Hydration**: the `MutationObserver` records no removal of the initial
   `ready` list item during hydration, or the report explains why the framework
   cannot meet this.
9. **Startup output**: the process prints the listening URL.

## Report

Produce a short report with:

- Framework name and version.
- Run instructions.
- Procedure path and request/response format.
- Languages used for server and client code.
- Total lines of application code, split by language.
- Number of build steps and whether npm, Node.js, or another toolchain is
  required.
- Whether the initial page is server-rendered.
- Whether the client hydrates or adopts existing DOM nodes.
- Client JavaScript payload size served on first load, uncompressed and gzip if
  available.
- Which acceptance tests pass, and any that cannot be expressed in the
  framework.
- One sentence on what felt unusually easy or awkward in this framework.

## Deliverables

- A runnable source tree.
- A README with the run instructions.
- The report above.
- No screenshots are required.
