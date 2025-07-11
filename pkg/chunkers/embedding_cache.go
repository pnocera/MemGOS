package chunkers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryEmbeddingCache implements EmbeddingCache using in-memory storage
type MemoryEmbeddingCache struct {
	mu       sync.RWMutex
	cache    map[string]*cacheEntry
	maxSize  int
	hitCount int64
	missCount int64
}

type cacheEntry struct {
	embedding []float64
	timestamp time.Time
	ttl       time.Duration
}

// NewMemoryEmbeddingCache creates a new in-memory embedding cache
func NewMemoryEmbeddingCache(maxSize int) *MemoryEmbeddingCache {
	return &MemoryEmbeddingCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
}

// Get retrieves an embedding from cache
func (c *MemoryEmbeddingCache) Get(ctx context.Context, key string) ([]float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists {
		c.missCount++
		return nil, false
	}
	
	// Check if entry has expired
	if entry.ttl > 0 && time.Since(entry.timestamp) > entry.ttl {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		c.mu.RLock()
		c.missCount++
		return nil, false
	}
	
	c.hitCount++
	
	// Return a copy to prevent modification
	embedding := make([]float64, len(entry.embedding))
	copy(embedding, entry.embedding)
	
	return embedding, true
}

// Set stores an embedding in cache
func (c *MemoryEmbeddingCache) Set(ctx context.Context, key string, embedding []float64, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict entries
	if len(c.cache) >= c.maxSize {
		if err := c.evictLRU(); err != nil {
			return fmt.Errorf("failed to evict cache entry: %w", err)
		}
	}
	
	// Create a copy of the embedding to prevent modification
	embeddingCopy := make([]float64, len(embedding))
	copy(embeddingCopy, embedding)
	
	c.cache[key] = &cacheEntry{
		embedding: embeddingCopy,
		timestamp: time.Now(),
		ttl:       ttl,
	}
	
	return nil
}

// Delete removes an embedding from cache
func (c *MemoryEmbeddingCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
	return nil
}

// Clear removes all embeddings from cache
func (c *MemoryEmbeddingCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]*cacheEntry)
	c.hitCount = 0
	c.missCount = 0
	
	return nil
}

// Size returns the current cache size
func (c *MemoryEmbeddingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.cache)
}

// Stats returns cache statistics
func (c *MemoryEmbeddingCache) Stats() EmbeddingCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	hitRate := float64(0)
	if c.hitCount+c.missCount > 0 {
		hitRate = float64(c.hitCount) / float64(c.hitCount+c.missCount)
	}
	
	// Estimate memory usage
	memoryUsage := int64(0)
	for _, entry := range c.cache {
		memoryUsage += int64(len(entry.embedding) * 8) // 8 bytes per float64
	}
	
	return EmbeddingCacheStats{
		Size:        len(c.cache),
		MaxSize:     c.maxSize,
		HitCount:    c.hitCount,
		MissCount:   c.missCount,
		HitRate:     hitRate,
		MemoryUsage: memoryUsage,
	}
}

// evictLRU removes the least recently used entry
func (c *MemoryEmbeddingCache) evictLRU() error {
	if len(c.cache) == 0 {
		return nil
	}
	
	// Find the oldest entry
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.cache {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}
	
	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
	
	return nil
}

// StartCleanupRoutine starts a background routine to clean up expired entries
func (c *MemoryEmbeddingCache) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired entries from cache
func (c *MemoryEmbeddingCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, entry := range c.cache {
		if entry.ttl > 0 && now.Sub(entry.timestamp) > entry.ttl {
			delete(c.cache, key)
		}
	}
}

// NoOpEmbeddingCache implements EmbeddingCache with no-op operations
type NoOpEmbeddingCache struct{}

// NewNoOpEmbeddingCache creates a new no-op embedding cache
func NewNoOpEmbeddingCache() *NoOpEmbeddingCache {
	return &NoOpEmbeddingCache{}
}

// Get always returns false (no cache)
func (c *NoOpEmbeddingCache) Get(ctx context.Context, key string) ([]float64, bool) {
	return nil, false
}

// Set does nothing
func (c *NoOpEmbeddingCache) Set(ctx context.Context, key string, embedding []float64, ttl time.Duration) error {
	return nil
}

// Delete does nothing
func (c *NoOpEmbeddingCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Clear does nothing
func (c *NoOpEmbeddingCache) Clear(ctx context.Context) error {
	return nil
}

// Size always returns 0
func (c *NoOpEmbeddingCache) Size() int {
	return 0
}

// Stats returns empty stats
func (c *NoOpEmbeddingCache) Stats() EmbeddingCacheStats {
	return EmbeddingCacheStats{}
}