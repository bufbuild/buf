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
	"context"
	"errors"
	"sync"
)

var errMustUseNewCache = errors.New("must use cache.NewCache to create new *Caches")

// Cache is a cache from K to V.
//
// It uses double-locking to get values.
//
// NewCache must be used to construct Caches.
type Cache[K comparable, V any] struct {
	getFunc func(context.Context, K) (V, error)

	store map[K]*tuple[V, error]
	lock  sync.RWMutex
}

// NewCache returns a new cache with the given uncached get function.
//
// This constructor must be used to construct Caches.
func NewCache[K comparable, V any](getFunc func(context.Context, K) (V, error)) *Cache[K, V] {
	return &Cache[K, V]{
		getFunc: getFunc,
		store:   make(map[K]*tuple[V, error]),
	}
}

// GetOrAdd gets the value for the key, or calls the getFunc from the constructor
// to get a new value, and then caches it.
func (c *Cache[K, V]) GetOrAdd(ctx context.Context, key K) (V, error) {
	if c.store == nil {
		var zero V
		return zero, errMustUseNewCache
	}
	c.lock.RLock()
	tuple, ok := c.store[key]
	c.lock.RUnlock()
	if ok {
		return tuple.V1, tuple.V2
	}
	c.lock.Lock()
	value, err := c.getOrAdd(ctx, key)
	c.lock.Unlock()
	return value, err
}

func (c *Cache[K, V]) getOrAdd(ctx context.Context, key K) (V, error) {
	tuple, ok := c.store[key]
	if ok {
		return tuple.V1, tuple.V2
	}
	value, err := c.getFunc(ctx, key)
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
