package repository

import (
	"fmt"
	"sync"
)

type Repository interface {
	Get(hash string) (string, error)
	Set(url, hash string) error
}

type MemStorage struct {
	s  map[string]string
	mu sync.RWMutex
}

func NewStrorage() Repository {
	storage := make(map[string]string, 50)
	return &MemStorage{
		s: storage,
	}
}

func (m *MemStorage) Get(hash string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	url, exist := m.s[hash]
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}
	return url, nil
}

func (m *MemStorage) Set(url, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.s[hash] = url
	return nil
}
