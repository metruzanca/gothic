# Iteration 3: the SQLite todo app

Status: working. This iteration adds the framework pieces needed for a real
shared list: procedure arguments, command closures, keyed lists, and event
values. See [ITERATION_2.md](ITERATION_2.md) for pure components.

## Goal

Build a todo list with:

- SQLite persistence,
- no login or accounts,
- one shared list for every visitor,
- server-rendered current state,
- add, toggle, and delete without a reload,
- pure components and keyed updates.

The demo is [`examples/todo`](../examples/todo).

## Backend

`main.go` owns a small SQLite store and implements the procedure interface
declared in the app file:

```go
type Procedures interface {
	gothic.RPCService
	ListTodos(ctx context.Context) ([]Todo, error)
	AddTodo(ctx context.Context, text string) error
	ToggleTodo(ctx context.Context, id int64) error
	DeleteTodo(ctx context.Context, id int64) error
}
```

Arguments after `context.Context` are decoded from the JSON argument array sent
by the client. The generator emits the server registrar and typed client stubs.

The schema is created on startup:

```sql
CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	text TEXT NOT NULL,
	done INTEGER NOT NULL DEFAULT 0
);
```

The example uses `modernc.org/sqlite`, which is pure Go and does not require
cgo.

## Rendering

The root route queries the current list on every request, so the server-rendered
page reflects the shared database. The client loads the same list again through
`Init`.

The input stores its value in the model through an event value:

```gothic
<input oninput={ DraftChanged{Text: event.target.value} } />
```

`Update` returns command closures for actions that need message or model values:

```go
id := msg.ID
return m, func() gothic.Msg {
	err := ToggleTodo(id)
	if err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return loadTodos()
}
```

The todo item is a separate component. The loop supplies a stable key:

```gothic
for _, todo := range model.Todos {
	<TodoItem key={ todo.ID } todo={ todo } />
}
```

Keyed lists reuse DOM nodes by key, update their text in place, and reorder or
remove nodes when the list changes.

## Behavior

- Adding writes to SQLite, reloads the list, and updates the DOM.
- Toggling flips `done` on the server and reloads the list.
- Deleting removes the row and the keyed list drops the node.
- Every visitor sees the same rows. There is no login and no per-user filtering.
- There is no push channel. Other clients see changes when they reload or act.

## Limits

- One process, one SQLite file.
- No auth, ownership, or per-user lists.
- No live updates from other clients.
- No conditionals, so status is rendered as text.
- No typed validation of procedure arguments before the handler runs.

## Verification

```sh
go generate ./examples/todo
go build ./...
go vet ./...
go test -count=1 ./...
node --check examples/todo/app_gothic.js
```

Browser checks:

- add an item and see it appear,
- toggle it and see the status change in place,
- delete it and see the node disappear,
- reload and confirm the SQLite state persists.
