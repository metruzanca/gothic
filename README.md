# gothic

A fullstack framework that compiles `.gothic` views to Go server code and to
fine-grained JavaScript DOM updates, with application logic written in ordinary
Go.

Status: proof of concept. The current model follows
[Bubble Tea](https://github.com/charmbracelet/bubbletea): a `Model` with
`Init`, `Update`, and a generated `View`, messages as `any`, and commands as
`func() Msg`. No hand-written JavaScript and no WebAssembly.

## Quick start

```sh
go generate ./examples/pingpong
go run ./examples/pingpong
```

The generator reports each artifact it writes, and the server prints the URL it
is listening on:

```
gothic: examples/pingpong/pingpong.gothic
  -> examples/pingpong/pingpong_gothic.go
  -> examples/pingpong/pingpong_app_gothic.go
  -> examples/pingpong/pingpong_gothic.js
gothic: wrote 3 artifact(s) from 1 source file(s)

2026/10/29 11:33:10 gothic: listening on http://localhost:8080
```

Open the printed URL and click Ping. Set `PORT` to use a different port.

The example has one app file and one entrypoint:

- `examples/pingpong/pingpong.gothic`: the Go prelude (model, messages,
  `Init`, `Update`, commands, procedure interface) and the view.
- `examples/pingpong/main.go`: server wiring and the procedure implementation.

## Install as a dependency

```sh
go get github.com/metruzanca/gothic@v0.2.0
go install github.com/metruzanca/gothic/cmd/gothic@v0.2.0
```

In an application, wires the generator into `go generate`. The in-repo
`../../cmd/gothic` form only works inside this repository, so use the published
module path:

```go
//go:generate go run github.com/metruzanca/gothic/cmd/gothic@v0.2.0 generate .
```

Or, on Go 1.24 and newer, add the CLI as a tool dependency:

```sh
go get -tool github.com/metruzanca/gothic/cmd/gothic@v0.2.0
```

```go
//go:generate go tool gothic generate .
```

Run `go generate` before `go build`. Generated Go embeds its sibling JavaScript
with `//go:embed`, so the JavaScript file must exist at build time.

## Documentation

- [PLAN.md](PLAN.md): architecture, decisions, and full roadmap.
- [SYNTAX.md](SYNTAX.md): the Go-first component syntax and its current limits.
- [ITERATION_1.md](ITERATION_1.md): what iteration 1 implements, how it works,
  how it is verified, and what is deferred.

## Usage

```sh
gothic generate ./path/to/app
```

The generator reads `.gothic` files and writes generated Go and JavaScript
beside each source. When the `.gothic` file starts with a Go prelude and then a
`templ` view, the generator also writes `<name>_app_gothic.go` for the prelude
and compiles the `Model`, messages, `Update`, and commands into the JavaScript
module. Generated code registers its module with the server and embeds it in the
Go binary.

## Requirements

Go 1.26 or newer. No Node.js, npm, or wasm is required to build or run an app.
