# TodoMVC example

A classic TodoMVC app backed by SQLite. There is no login. Every visitor reads
and writes the same list.

## Run

```sh
go generate ./examples/todo
go run ./examples/todo
```

Open <http://localhost:8080>. Set `PORT` to change the port.

The example uses an in-memory SQLite database by default, so it creates no
files and starts with an empty list on every run. Set `DB_PATH=/path/to/todo.db`
to use a file instead.

The SQLite driver is pure Go (`modernc.org/sqlite`), so no cgo is required. The
TodoMVC CSS is vendored in `style.css`.

## What it shows

- SQLite persistence through `database/sql`.
- Procedures with arguments:
  `AddTodo`, `SetTodoTitle`, `ToggleTodo`, `SetAllCompleted`, `DeleteTodo`,
  `ClearCompleted`, and `ListTodos`.
- Command closures in `Update`.
- Keyed list reconciliation with `key={ todo.ID }`.
- A separate `TodoItem` component file.
- Dynamic `value={ ... }` and `checked={ ... }` attributes.
- `class:completed`, `class:editing`, `class:hidden`, and `class:selected`
  toggles.
- `focus={ editing }` for inline editing.
- `onenter` and `onescape` key shortcuts.
- Hash routing through a `HashChanged` message.

## Behavior

- The page is server-rendered with the current database contents.
- Add with Enter. The input is trimmed and cleared.
- Toggle items, mark all complete, and clear completed.
- Double-click a label to edit. Save on Enter or blur, cancel with Escape, and
  delete by saving an empty title.
- Filter with All, Active, and Completed links.
- The list updates by key, so existing items keep their DOM nodes.
- There is no push channel. Other users see changes when they reload or perform
  an action.
