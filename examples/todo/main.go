//go:generate go run ../../cmd/gothic generate .

package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/metruzanca/gothic"
	_ "modernc.org/sqlite"
)

type store struct {
	db *sql.DB
}

func openStore(path string) (*store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			done INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		db.Close()
		return nil, err
	}
	return &store{db: db}, nil
}

func (s *store) close() error {
	return s.db.Close()
}

func (s *store) list(ctx context.Context) ([]Todo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, text, done FROM todos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := make([]Todo, 0)
	for rows.Next() {
		var todo Todo
		var done int
		if err := rows.Scan(&todo.ID, &todo.Text, &done); err != nil {
			return nil, err
		}
		todo.Done = done != 0
		if todo.Done {
			todo.Status = "done"
		} else {
			todo.Status = "open"
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func (s *store) add(ctx context.Context, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("todo text is empty")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO todos(text, done) VALUES(?, 0)`, text)
	return err
}

func (s *store) toggle(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE todos SET done = CASE done WHEN 0 THEN 1 ELSE 0 END WHERE id = ?`, id)
	return err
}

func (s *store) delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM todos WHERE id = ?`, id)
	return err
}

type procedures struct {
	store *store
}

func (p procedures) ListTodos(ctx context.Context) ([]Todo, error) {
	return p.store.list(ctx)
}

func (p procedures) AddTodo(ctx context.Context, text string) error {
	return p.store.add(ctx, text)
}

func (p procedures) ToggleTodo(ctx context.Context, id int64) error {
	return p.store.toggle(ctx, id)
}

func (p procedures) DeleteTodo(ctx context.Context, id int64) error {
	return p.store.delete(ctx, id)
}

func main() {
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "todo.db"
	}
	st, err := openStore(path)
	if err != nil {
		log.Fatal(err)
	}
	defer st.close()

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	s := gothic.NewServer()
	RegisterProcedures(s, procedures{store: st})
	mountApp(s, st)
	log.Fatal(s.ListenAndServe(addr))
}

func mountApp(s *gothic.Server, st *store) {
	s.Get("/", func(w http.ResponseWriter, r *http.Request) {
		todos, err := st.list(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := gothic.NewApp(NewModel(todos)).Render(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
