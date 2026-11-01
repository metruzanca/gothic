package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore keeps sessions in process memory. Sessions are lost on restart.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
	saves   int
}

type memoryEntry struct {
	data    []byte
	expires time.Time
}

// NewMemoryStore returns an empty in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: map[string]memoryEntry{}}
}

// Load implements Store.
func (s *MemoryStore) Load(ctx context.Context, id string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[id]
	if !ok {
		return nil, false, nil
	}
	if !entry.expires.IsZero() && time.Now().After(entry.expires) {
		delete(s.entries, id)
		return nil, false, nil
	}
	return append([]byte(nil), entry.data...), true, nil
}

// Save implements Store.
func (s *MemoryStore) Save(ctx context.Context, id string, data []byte, expires time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[id] = memoryEntry{data: append([]byte(nil), data...), expires: expires}
	s.saves++
	if s.saves%256 == 0 {
		s.cleanupLocked(time.Now())
	}
	return nil
}

// Delete implements Store.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, id)
	return nil
}

// Cleanup removes expired sessions and returns how many were removed.
func (s *MemoryStore) Cleanup(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cleanupLocked(now)
}

func (s *MemoryStore) cleanupLocked(now time.Time) int {
	removed := 0
	for id, entry := range s.entries {
		if !entry.expires.IsZero() && now.After(entry.expires) {
			delete(s.entries, id)
			removed++
		}
	}
	return removed
}

// Len returns the number of stored sessions.
func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}
