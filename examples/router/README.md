# Router example

A small app that demonstrates first-class URL state.

## Run

```sh
go generate
go run .
```

Open <http://localhost:8080>. Set `PORT` to change the port.

## Shows

- `LocationChanged` with `Path`, `Query`, and `Hash`
- `route` links for client-side navigation
- request-scoped models through `gothic.NewAppFunc`
- deep links rendered by the server before hydration
- `js.Navigate`, `js.Back`, `js.SetTitle`, and localStorage
- back and forward support through `popstate`

The server seeds the model from `r.URL.Path`, so `/about` and `/settings`
render correctly on a direct load. The client intercepts route links, pushes
history, and dispatches `LocationChanged` without a full reload.

Unknown paths render the not-found page. There are no path parameters or route
matching helpers yet.
