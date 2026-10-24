# gothic

A fullstack framework that compiles one `.gothic` component language to Go
server code and to fine-grained JavaScript DOM updates.

Status: iteration 1 is complete. It implements a ping/pong vertical slice with
server rendering, hydration, reactive state, and a generated typed RPC call.
No hand-written JavaScript and no WebAssembly.

## Quick start

```sh
go generate ./examples/pingpong
go run ./examples/pingpong
```

The generator reports each artifact it writes, and the server prints the URL it
is listening on:

```
gothic: views/pingpong.gothic
  -> views/pingpong_gothic.go
  -> views/pingpong_gothic.js
gothic: wrote 2 artifact(s) from 1 source file(s)

2026/10/24 12:33:00 gothic: listening on http://localhost:8080
```

Open the printed URL and click Ping. Set `PORT` to use a different port.

## Install as a dependency

```sh
go get github.com/metruzanca/gothic@v0.1.0
go install github.com/metruzanca/gothic/cmd/gothic@v0.1.0
```

In an application, wires the generator into `go generate`. The in-repo
`../../cmd/gothic` form only works inside this repository, so use the published
module path:

```go
//go:generate go run github.com/metruzanca/gothic/cmd/gothic@v0.1.0 generate .
```

Or, on Go 1.24 and newer, add the CLI as a tool dependency:

```sh
go get -tool github.com/metruzanca/gothic/cmd/gothic@v0.1.0
```

```go
//go:generate go tool gothic generate .
```

Run `go generate` before `go build`. Generated Go embeds its sibling JavaScript
with `//go:embed`, so the JavaScript file must exist at build time.

## Documentation

- [PLAN.md](PLAN.md): architecture, decisions, and full roadmap.
- [ITERATION_1.md](ITERATION_1.md): what iteration 1 implements, how it works,
  how it is verified, and what is deferred.

## Usage

```sh
gothic generate ./path/to/app
```

The generator reads `.gothic` files and writes `<name>_gothic.go` and
`<name>_gothic.js` beside each source. Generated code registers its JavaScript
module with the server and embeds it in the Go binary.

## Requirements

Go 1.26 or newer. No Node.js, npm, or wasm is required to build or run an app.
