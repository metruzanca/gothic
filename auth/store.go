package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Common store errors.
var (
	// ErrNotFound reports a missing user.
	ErrNotFound = errors.New("auth: user not found")
	// ErrEmailTaken reports a duplicate email.
	ErrEmailTaken = errors.New("auth: email already registered")
	// ErrInvalidCredentials is returned by Manager.Login.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
)

// User is the authenticated identity.
type User struct {
	ID    string
	Email string
	Name  string
}

// Store loads and creates users. Password hashes are managed by this package.
type Store interface {
	// UserByEmail returns the user and password hash for email.
	UserByEmail(ctx context.Context, email string) (User, string, error)
	// UserByID returns the user for id.
	UserByID(ctx context.Context, id string) (User, error)
	// CreateUser stores a new user and returns it.
	CreateUser(ctx context.Context, email, name, passwordHash string) (User, error)
}

// NormalizeEmail trims and lowercases an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// MemoryStore keeps users in process memory.
type MemoryStore struct {
	mu      sync.RWMutex
	byID    map[string]memoryUser
	byEmail map[string]string
}

type memoryUser struct {
	user User
	hash string
}

// NewMemoryStore returns an empty in-memory user store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: map[string]memoryUser{}, byEmail: map[string]string{}}
}

// UserByEmail implements Store.
func (s *MemoryStore) UserByEmail(ctx context.Context, email string) (User, string, error) {
	if err := ctx.Err(); err != nil {
		return User{}, "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byEmail[NormalizeEmail(email)]
	if !ok {
		return User{}, "", ErrNotFound
	}
	record := s.byID[id]
	return record.user, record.hash, nil
}

// UserByID implements Store.
func (s *MemoryStore) UserByID(ctx context.Context, id string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return record.user, nil
}

// CreateUser implements Store.
func (s *MemoryStore) CreateUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	email = NormalizeEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[email]; ok {
		return User{}, ErrEmailTaken
	}
	user := User{ID: newUserID(), Email: email, Name: strings.TrimSpace(name)}
	s.byID[user.ID] = memoryUser{user: user, hash: passwordHash}
	s.byEmail[email] = user.ID
	return user, nil
}

func newUserID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("auth: read id: %v", err))
	}
	return fmt.Sprintf("%x", b)
}
