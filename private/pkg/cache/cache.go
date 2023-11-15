// Copyright 2020-2023 Buf Technologies, Inc.
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
	store map[K]*tuple[V, error]
	lock  sync.RWMutex
}

// GetOrAdd gets the value for the key, or calls getUncached to get a new value,
// and then caches the value.
//
// If getUnached calls another Cache, the order of GetOrAdd calls between caches
// must be preserved for lock ordering.
func (c *Cache[K, V]) GetOrAdd(key K, getUncached func() (V, error)) (V, error) {
	c.lock.RLock()
	var tuple *tuple[V, error]
	var ok bool
	if c.store != nil {
		tuple, ok = c.store[key]
	}
	c.lock.RUnlock()
	if ok {
		return tuple.V1, tuple.V2
	}
	c.lock.Lock()
	value, err := c.getOrAdd(key, getUncached)
	c.lock.Unlock()
	return value, err
}

func (c *Cache[K, V]) getOrAdd(key K, getUncached func() (V, error)) (V, error) {
	if c.store == nil {
		c.store = make(map[K]*tuple[V, error])
	}
	tuple, ok := c.store[key]
	if ok {
		return tuple.V1, tuple.V2
	}
	value, err := getUncached()
	c.store[key] = newTuple(value, err)
	return value, err
}

type tuple[T1, T2 any] struct {
	V1 T1
	V2 T2
}

func newTuple[T1, T2 any](v1 T1, v2 T2) *tuple[T1, T2] {
	return &tuple[T1, T2]{
		V1: v1,
		V2: v2,
	}
}
