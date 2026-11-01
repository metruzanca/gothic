package session

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const sqlSchema = `CREATE TABLE IF NOT EXISTS gothic_sessions (
	id TEXT PRIMARY KEY,
	data BLOB NOT NULL,
	expires_at INTEGER NOT NULL
)`

// SQLStore keeps sessions in a table in an existing database.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore creates the session table when it is missing and returns a store
// backed by db.
func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("gothic/session: nil database")
	}
	if _, err := db.Exec(sqlSchema); err != nil {
		return nil, fmt.Errorf("gothic/session: create table: %w", err)
	}
	return &SQLStore{db: db}, nil
}

// Load implements Store.
func (s *SQLStore) Load(ctx context.Context, id string) ([]byte, bool, error) {
	var (
		data    []byte
		expires int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT data, expires_at FROM gothic_sessions WHERE id = ?`, id,
	).Scan(&data, &expires)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if expires > 0 && time.Now().Unix() > expires {
		_ = s.Delete(ctx, id)
		return nil, false, nil
	}
	return data, true, nil
}

// Save implements Store.
func (s *SQLStore) Save(ctx context.Context, id string, data []byte, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO gothic_sessions (id, data, expires_at) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, expires_at = excluded.expires_at`,
		id, data, expires.Unix(),
	)
	return err
}

// Delete implements Store.
func (s *SQLStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM gothic_sessions WHERE id = ?`, id)
	return err
}

// Cleanup removes expired sessions and returns how many rows were removed.
func (s *SQLStore) Cleanup(ctx context.Context, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM gothic_sessions WHERE expires_at > 0 AND expires_at < ?`, now.Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
