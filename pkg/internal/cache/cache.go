package cache

import (
	"runtime"
	"sync"
	"time"
)

var (
	singleton *Cache
	once      sync.Once
	// TODO: make configurable
	defaultExpiry = 10 * time.Minute
	// TODO: make configurable
	defaultInterval = 10 * time.Minute
)

type Cache struct {
	sync.Mutex
	data     map[string]item
	interval time.Duration
	cancel   chan bool
}

// NewCache creates a new singleton Cache, if called multiple times, the same
// Cache is returned
func NewCache() *Cache {
	once.Do(func() {
		singleton = &Cache{
			data: make(map[string]item),
			// TODO: make configurable
			interval: defaultInterval,
		}
		runtime.SetFinalizer(singleton, singleton.stop)
		go singleton.clean()
	})
	return singleton
}

// Read returns an item from the Cache, nil if it is not found
func (c *Cache) Read(in string) (any, bool) {
	c.Lock()
	i, ok := c.data[in]
	if !ok {
		return nil, false
	}
	c.Unlock()
	if i.expired() {
		return nil, false
	}
	return i.value, true
}

// Write adds a new item to the Cache with a provided key and value
func (c *Cache) Write(key string, value any) {
	c.Lock()
	c.data[key] = item{
		value: value,
		// TODO: make configurable
		exp: time.Now().Add(defaultExpiry),
	}
	c.Unlock()
}

// Delete an item from the Cache. Does nothing if the key is not in the Cache.
func (c *Cache) Delete(key string) {
	c.Lock()
	c.delete(key)
	c.Unlock()
}

func (c *Cache) delete(key string) {
	if _, found := c.data[key]; found {
		delete(c.data, key)
	}
}

// Flush deletes all items from the Cache.
func (c *Cache) Flush() {
	c.Lock()
	c.data = make(map[string]item)
	c.Unlock()
}

// removeExpired deletes all expired items from the Cache.
func (c *Cache) removeExpired() {
	c.Lock()
	for k, v := range c.data {
		if v.expired() {
			c.delete(k)
		}
	}
	c.Unlock()
}

// clean deletes expired items from the cache
func (c *Cache) clean() {
	ticker := time.NewTicker(c.interval)
	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.cancel:
			ticker.Stop()
			return
		}
	}
}

func (c *Cache) stop() {
	c.cancel <- true
}

type item struct {
	value any
	exp   time.Time
}

func (i *item) expired() bool {
	return time.Now().After(i.exp)
}
