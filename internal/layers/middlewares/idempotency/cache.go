package idempotency

import (
	"container/list"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CacheKey struct {
	UserID    uuid.UUID
	Path      string
	HTTPVerb  string
	RequestID string
}

type CacheValue struct {
	Response    []byte
	StatusCode  int
	ContentType string
	CreatedAt   time.Time
}

type lruEntry struct {
	key   CacheKey
	value *CacheValue
}

// InMemoryCache is a thread-safe LRU cache with O(1) Get/Set/eviction.
type InMemoryCache struct {
	mu      sync.RWMutex
	maxSize int
	items   map[CacheKey]*list.Element
	order   *list.List // front = most recently used, back = least recently used
}

func NewInMemoryCache(maxEntries int) *InMemoryCache {
	return &InMemoryCache{
		maxSize: maxEntries,
		items:   make(map[CacheKey]*list.Element, maxEntries),
		order:   list.New(),
	}
}

func (c *InMemoryCache) Get(key CacheKey) (*CacheValue, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	c.order.MoveToFront(elem)
	return elem.Value.(*lruEntry).value, true
}

func (c *InMemoryCache) Set(key CacheKey, value *CacheValue) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*lruEntry).value = value
		return
	}

	// Evict LRU if at capacity
	if c.order.Len() >= c.maxSize {
		c.evictLRU()
	}

	entry := &lruEntry{key: key, value: value}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
}

func (c *InMemoryCache) DeleteOld(before time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []*list.Element
	for elem := c.order.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(*lruEntry)
		if entry.value.CreatedAt.Before(before) {
			toRemove = append(toRemove, elem)
		}
	}
	for _, elem := range toRemove {
		entry := elem.Value.(*lruEntry)
		delete(c.items, entry.key)
		c.order.Remove(elem)
	}
}

// evictLRU removes the least recently used entry. Must be called with lock held.
func (c *InMemoryCache) evictLRU() {
	back := c.order.Back()
	if back == nil {
		return
	}
	entry := back.Value.(*lruEntry)
	delete(c.items, entry.key)
	c.order.Remove(back)
}
