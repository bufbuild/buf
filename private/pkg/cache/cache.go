// Copyright 2020-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"sync"
)

// Cache is a cache from K to V.
//
// It uses double-locking to get values.
type Cache[K comparable, V any] struct {
	store map[K]*result[V]
	lock  sync.RWMutex
}

// GetOrAdd gets the value for the key, or calls getUncached to get a new value,
// and then caches the value.
//
// If getUncached calls another Cache, the order of GetOrAdd calls between caches
// must be preserved for lock ordering.
func (c *Cache[K, V]) GetOrAdd(key K, getUncached func() (V, error)) (V, error) {
	c.lock.RLock()
	var result *result[V]
	var ok bool
	if c.store != nil {
		result, ok = c.store[key]
	}
	c.lock.RUnlock()
	if ok {
		return result.value, result.err
	}
	c.lock.Lock()
	value, err := c.getOrAddInsideWriteLock(key, getUncached)
	c.lock.Unlock()
	return value, err
}

func (c *Cache[K, V]) getOrAddInsideWriteLock(key K, getUncached func() (V, error)) (V, error) {
	if c.store == nil {
		c.store = make(map[K]*result[V])
	}
	result, ok := c.store[key]
	if ok {
		return result.value, result.err
	}
	value, err := getUncached()
	c.store[key] = newResult(value, err)
	return value, err
}

type result[V any] struct {
	value V
	err   error
}

func newResult[V any](value V, err error) *result[V] {
	return &result[V]{
		value: value,
		err:   err,
	}
}
