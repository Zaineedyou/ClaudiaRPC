package session

import (
	"ClaudiaRPC/internal/gateway"
	"sync"
)

type SessionManager struct {
	sessions map[string]*gateway.Gateway
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*gateway.Gateway),
	}
}

func (s *SessionManager) Add(token string, gw *gateway.Gateway) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Close existing session if any
	if existing, ok := s.sessions[token]; ok {
		existing.Close()
	}
	s.sessions[token] = gw
}

func (s *SessionManager) Remove(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gw, ok := s.sessions[token]; ok {
		gw.Close()
		delete(s.sessions, token)
	}
}

func (s *SessionManager) Get(token string) (*gateway.Gateway, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gw, ok := s.sessions[token]
	return gw, ok
}
