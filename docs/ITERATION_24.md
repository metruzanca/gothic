# Iteration 24: Batch and Sequence commands

Status: complete. Commands can be composed with `gothic.Batch` and
`gothic.Sequence`.

## Goal

Run several commands from one `Update` or `Init` return without writing manual
fan-out or sequencing logic.

```go
return m, gothic.Batch(loadItems(), loadStats())

return m, gothic.Sequence(
	func() gothic.Msg { return FirstDone{} },
	func() gothic.Msg { return SecondDone{} },
)
```

## Semantics

- `Batch` starts commands concurrently.
- `Sequence` runs commands in order.
- Each non-nil result is dispatched as a message.
- Nil commands and nil results are ignored.
- Nested Batch and Sequence calls work.
- Component commands stop after unmount.

## Compiler and runtime

The Go stubs keep `Init`, `Update`, and command functions compilable:

```go
func Batch(cmds ...Cmd) Cmd
func Sequence(cmds ...Cmd) Cmd
```

The client compiler lowers calls to marker commands:

```js
{ __gothicBatch: [cmd1, cmd2] }
{ __gothicSequence: [cmd1, cmd2] }
```

The generated `run` function handles markers before invoking a command:

```js
if (cmd.__gothicBatch) {
  await Promise.all(cmd.__gothicBatch.map((sub) => run(sub)));
  return;
}
if (cmd.__gothicSequence) {
  for (const sub of cmd.__gothicSequence) {
    await run(sub);
  }
  return;
}
const msg = await cmd();
if (msg) dispatch(msg);
```

The component `run` adds the existing `alive` guard, so a Sequence stops
between steps after unmount and a Batch step does not dispatch after unmount.

## Verification

Core:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -count=1 ./...
(cd ui && go test ./...)
(cd examples/gothicbook && go test ./...)
(cd examples/router && go test ./...)
(cd examples/nested && go test ./...)
(cd examples/virtual && go test ./...)
```

Compiler tests cover Batch and Sequence lowering, nested markers, and the
component unmount guard.

Browser check:

- Batch dispatched both messages,
- Sequence dispatched its messages in order,
- no console errors.

## Limits

- the Go functions are client-only stubs,
- commands still return one message each,
- Batch result order is whatever completes first,
- there is no cancellation of in-flight commands on unmount, only guarded
  dispatch.

## Follow-ups

- Cancellation for component commands.
- A typed effect or subscription system if commands need long-lived streams.
