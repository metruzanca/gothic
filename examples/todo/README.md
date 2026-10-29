# Todo example

A small shared todo list backed by SQLite. There is no login. Every visitor
reads and writes the same list.

## Run

```sh
go generate ./examples/todo
go run ./examples/todo
```

Open <http://localhost:8080>. Set `PORT` to change the port and `DB_PATH` to
change the database file. The default database is `todo.db` in the current
directory.

The SQLite driver is pure Go (`modernc.org/sqlite`), so no cgo is required.

## What it shows

- SQLite persistence through `database/sql`.
- Procedures with arguments:
  `AddTodo(ctx, text)`, `ToggleTodo(ctx, id)`, `DeleteTodo(ctx, id)`, and
  `ListTodos(ctx)`.
- Command closures in `Update`:

```go
return m, func() gothic.Msg {
	err := AddTodo(text)
	if err != nil {
		return TodoFailed{Message: err.Error()}
	}
	return loadTodos()
}
```

- Keyed list reconciliation with `key={ todo.ID }`.
- A separate `TodoItem` component file.
- An input event that reads the DOM value with `event.target.value`.

## Behavior

- The page is server-rendered with the current database contents.
- Adding, toggling, and deleting call the backend and reload the list.
- The list updates by key, so existing items keep their DOM nodes.
- There is no push channel. Other users see changes when they reload or perform
  an action.
