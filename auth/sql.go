package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const sqlSchema = `CREATE TABLE IF NOT EXISTS gothic_users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	created_at INTEGER NOT NULL
)`

// SQLStore keeps users in a table in an existing database.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore creates the user table when it is missing and returns a store
// backed by db.
func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("auth: nil database")
	}
	if _, err := db.Exec(sqlSchema); err != nil {
		return nil, fmt.Errorf("auth: create table: %w", err)
	}
	return &SQLStore{db: db}, nil
}

// UserByEmail implements Store.
func (s *SQLStore) UserByEmail(ctx context.Context, email string) (User, string, error) {
	var (
		user User
		hash string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash FROM gothic_users WHERE email = ?`,
		NormalizeEmail(email),
	).Scan(&user.ID, &user.Email, &user.Name, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", err
	}
	return user, hash, nil
}

// UserByID implements Store.
func (s *SQLStore) UserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, name FROM gothic_users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Email, &user.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// CreateUser implements Store.
func (s *SQLStore) CreateUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	user := User{ID: newUserID(), Email: NormalizeEmail(email), Name: strings.TrimSpace(name)}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO gothic_users (id, email, name, password_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.Name, passwordHash, time.Now().Unix(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return user, nil
}

// isUniqueViolation checks the common constraint error strings without
// depending on a specific driver.
func isUniqueViolation(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "constraint")
}
