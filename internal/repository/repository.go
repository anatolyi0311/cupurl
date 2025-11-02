package repository

import (
	"fmt"
	"sync"
)

const defaultSizeMap = 0

type Repository interface {
	Get(hash string) (string, error)
	Set(url, hash string) error
}

type MemStorage struct {
	ms map[string]string
	mu sync.RWMutex
}

func NewStorage() Repository {
	storage := make(map[string]string, defaultSizeMap)
	return &MemStorage{
		ms: storage,
	}
}

func (m *MemStorage) Get(hash string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	url, exist := m.ms[hash]
	if !exist {
		return "", fmt.Errorf("%s not found", hash)
	}
	return url, nil
}

func (m *MemStorage) Set(url, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ms[hash] = url
	return nil
}
