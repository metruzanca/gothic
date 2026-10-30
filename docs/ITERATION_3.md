# Iteration 3: classic TodoMVC on SQLite

Status: working. This iteration turns the sqlite todo example into a classic
TodoMVC implementation and adds the framework pieces it needs: procedure
arguments, command closures, keyed lists, event values, dynamic attributes,
class toggles, focus bindings, and hash routing. See
[ITERATION_2.md](ITERATION_2.md) for pure components.

## Goal

Build the TodoMVC app with:

- SQLite persistence,
- no login or accounts,
- one shared list for every visitor,
- server-rendered current state,
- add, toggle, edit, delete, mark all, and clear completed,
- active, completed, and all filters with hash routes,
- pure components and keyed updates.

The demo is [`examples/todo`](../examples/todo).

## Backend

`main.go` owns a small SQLite store and implements the procedure interface
declared in the app file:

```go
type Procedures interface {
	gothic.RPCService
	ListTodos(ctx context.Context) (TodoData, error)
	AddTodo(ctx context.Context, title string) error
	SetTodoTitle(ctx context.Context, id int64, title string) error
	ToggleTodo(ctx context.Context, id int64) error
	SetAllCompleted(ctx context.Context, completed bool) error
	DeleteTodo(ctx context.Context, id int64) error
	ClearCompleted(ctx context.Context) error
}
```

Arguments after `context.Context` are decoded from the JSON argument array sent
by the client. The generator emits the server registrar and typed client stubs.

The schema is created on startup:

```sql
CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	completed INTEGER NOT NULL DEFAULT 0
);
```

The example uses an in-memory SQLite database by default, so there is no file
to create or migrate for the demo. Set `DB_PATH` to use a file instead. The
driver is pure Go (`modernc.org/sqlite`), so no cgo is required. `ListTodos`
also returns active and completed counts, pluralized item text, and visibility
flags for the empty state and clear-completed button.

## Rendering

The root route queries the current list on every request, so the server-rendered
page reflects the shared database. The client loads the same list again through
`Init`.

The new-todo input stores its value in the model through an event value:

```templ
<input value={ model.Draft } oninput={ DraftChanged{Text: event.target.value} } onenter={ AddClicked{} } />
```

The edit input uses dynamic attributes, a focus binding, and key events:

```templ
<input
	class="edit"
	value={ editText }
	focus={ editing }
	oninput={ EditChanged{Text: event.target.value} }
	onblur={ SaveEdit{ID: todo.ID} }
	onenter={ SaveEdit{ID: todo.ID} }
	onescape={ CancelEdit{ID: todo.ID} }
/>
```

Classes are toggled without conditionals:

```templ
<li class:completed={ todo.Completed } class:editing={ editing } class:hidden={ hidden }>
```

The todo item is a separate component. The loop supplies a stable key:

```templ
for _, todo := range model.Todos {
	<TodoItem key={ todo.ID } todo={ todo } filter={ model.Filter } editing={ todo.ID == model.EditingID } editText={ model.EditText } />
}
```

Keyed lists reuse DOM nodes by key, update their text in place, and reorder or
remove nodes when the list changes.

## Routing

The app defines:

```go
type HashChanged struct{ Hash string }
```

The generated client dispatches `HashChanged` on load and on every hashchange.
`Update` maps `#/active` and `#/completed` to `model.Filter`. The filter links
use `href="#/..."`, and the selected class is a class toggle.

## Behavior

- Adding writes to SQLite, clears the input, reloads the list, and updates the DOM.
- Toggling flips `completed` on the server and reloads the list.
- Editing saves on Enter and blur, cancels on Escape, and deletes on empty save.
- Mark all sets every todo to the checkbox state.
- Clear completed removes completed rows.
- Filters hide items without changing the stored data.
- Every visitor sees the same rows. There is no login and no per-user filtering.
- There is no push channel. Other clients see changes when they reload or act.

## Limits

- One process. Data is lost on restart unless `DB_PATH` points at a file.
- No auth, ownership, or per-user lists.
- No live updates from other clients.
- No general conditionals in views. Hiding uses class toggles.
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

- add todos with Enter and see the input clear,
- toggle one item and the mark-all state,
- double-click to edit, then save with Enter, cancel with Escape, and delete with an empty title,
- switch between All, Active, and Completed filters,
- clear completed and confirm the empty state,
- reload and confirm SQLite state and route selection persist.
