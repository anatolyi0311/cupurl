package repository

import (
	"slices"
	"sync"
)

type MemCache struct {
	mu sync.Mutex
	cache map[int][]string
}

func NewCache() *MemCache {
	cache := make(map[int][]string)
	return &MemCache{
		cache: cache,
	}
}

func (c *MemCache) SaveHash(userID int, hashURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hashArray := c.cache[userID]
	hashArray = append(hashArray, hashURL)
	c.cache[userID] = hashArray
}

func (c *MemCache) CanDeleteHash(userID int, hashURL string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	hashArray, exist := c.cache[userID]
	if !exist {
		return false
	}

	return slices.Contains(hashArray, hashURL)
}
