package local_repo

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// CacheRepository wraps go-cache functionality
type CacheRepository struct {
	cache *cache.Cache
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(defaultExpiration, cleanupInterval time.Duration) *CacheRepository {
	return &CacheRepository{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Set sets a value in the cache
func (r *CacheRepository) Set(key string, value any) {
	r.cache.Set(key, value, cache.DefaultExpiration)
}

// Get retrieves a value from the cache
func (r *CacheRepository) Get(key string) (any, bool) {
	return r.cache.Get(key)
}

// Delete removes an item from the cache
func (r *CacheRepository) Delete(key string) {
	r.cache.Delete(key)
}
