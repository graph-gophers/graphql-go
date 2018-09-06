package common

import (
	"sync"
	"sync/atomic"
)

type Cache struct {
	value atomic.Value
	mu    sync.Mutex
}

func (cache *Cache) GetOrElseUpdate(key interface{}, create func ()interface{}) (x interface{}) {
	lastCacheMap, _ := cache.value.Load().(map[interface{}]interface{})
	value, found := lastCacheMap[key]
	if found {
		return value
	}

	// Compute fields without lock.
	// Might duplicate effort but won't hold other computations back.
	value = create()

	// Update
	cache.mu.Lock()
	lastCacheMap, _ = cache.value.Load().(map[interface{}]interface{})
	nextCacheMap := make(map[interface{}]interface{}, len(lastCacheMap)+1)
	for k, v := range lastCacheMap {
		nextCacheMap[k] = v
	}
	nextCacheMap[key] = value
	cache.value.Store(nextCacheMap)
	cache.mu.Unlock()
	return value
}
