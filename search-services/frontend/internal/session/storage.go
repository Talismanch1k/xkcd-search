package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type entry struct {
	token     string
	expiresAt time.Time
}

type Storage struct {
	ttl      time.Duration
	mu       sync.RWMutex
	sessions map[string]entry
}

func NewStore(ctx context.Context, ttl time.Duration) *Storage {
	s := &Storage{
		ttl:      ttl,
		sessions: make(map[string]entry),
	}

	go s.cleanup(ctx)

	return s
}

func (s *Storage) Create(token string) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[id] = entry{token: token, expiresAt: time.Now().Add(s.ttl)}
	s.mu.Unlock()

	return id, nil
}

func (s *Storage) Get(id string) (string, bool) {
	s.mu.RLock()
	e, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok || time.Now().After(e.expiresAt) {
		s.Delete(id)
		return "", false
	}

	return e.token, true
}

func (s *Storage) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *Storage) cleanup(ctx context.Context) {
	ticker := time.NewTicker(s.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()

			for id, e := range s.sessions {
				if now.After(e.expiresAt) {
					delete(s.sessions, id)
				}
			}

			s.mu.Unlock()
		}
	}
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
