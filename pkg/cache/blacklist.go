package cache

import (
	"sync"
	"time"
)

// Checker is implemented by any blacklist that can report whether a JTI is revoked.
type Checker interface {
	Has(jti string) bool
}

// Adder is implemented by any blacklist that can accept new revoked JTIs.
type Adder interface {
	Add(jti string, expiresAt time.Time)
}

type blacklistEntry struct {
	expiresAt time.Time
}

// Blacklist is a thread-safe in-memory TTL store for revoked token JTIs.
type Blacklist struct {
	m sync.Map
}

// NewBlacklist creates an empty Blacklist.
func NewBlacklist() *Blacklist {
	return &Blacklist{}
}

// Add marks jti as revoked until expiresAt.
func (b *Blacklist) Add(jti string, expiresAt time.Time) {
	b.m.Store(jti, blacklistEntry{expiresAt: expiresAt})
}

// Has returns true when jti is present and not yet expired.
func (b *Blacklist) Has(jti string) bool {
	val, ok := b.m.Load(jti)
	if !ok {
		return false
	}
	entry, ok := val.(blacklistEntry)
	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		b.m.Delete(jti)
		return false
	}
	return true
}

// Cleanup removes all expired entries. Call this periodically.
func (b *Blacklist) Cleanup() {
	now := time.Now()
	b.m.Range(func(key, value any) bool {
		if entry, ok := value.(blacklistEntry); ok && now.After(entry.expiresAt) {
			b.m.Delete(key)
		}
		return true
	})
}
