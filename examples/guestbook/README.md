# Guestbook example

A small app that shows the whole gothic loop: state, events, a typed server
call, a conditional, a keyed list, and a pure component.

## Run

```sh
go generate
go run .
```

Open <http://localhost:8080>. Set `PORT` to change the port.

## What it shows

- `value={ model.Draft }` with `oninput` and `event.target.value`.
- `onenter` to post from the keyboard.
- A command closure that calls the `Post` procedure and turns the result into a
  message.
- An empty state through `if` / `else`.
- A keyed `for` list over a slice of structs.
- A pure `Entry` component with a props struct.
- Request-scoped models through `gothic.NewAppFunc`.

Messages live in memory behind a mutex and every visitor shares the list. They
reset when the process restarts. Reloading renders the current list on the
server before hydration, so the page never flashes an empty state.

## Behavior

- Post with Enter or the button. The input is trimmed and cleared.
- An empty message is rejected on the client before it reaches the server.
- Every posted message is appended to the list, and a failed call shows the
  error as a list entry.
