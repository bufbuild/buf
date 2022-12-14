// Copyright 2020-2022 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/pkg/transform/internal/protoencoding"
)

var (
	singleton *Cache
	once      sync.Once
)

type Cache struct {
	data map[string]any
	mu   sync.RWMutex
}

// NewCache creates a new singleton Cache, if called multiple times, the same
// Cache is returned
func NewCache() *Cache {
	once.Do(func() {
		singleton = &Cache{
			data: make(map[string]any),
		}
	})
	return singleton
}

// Load returns an item from the Cache, nil if it is not found
func (c *Cache) Load(key string) (protoencoding.Resolver, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	x, found := c.data[key]
	if !found {
		return nil, false
	}
	pack, ok := x.(protoencoding.Resolver)
	return pack, ok
}

// Save adds a new item to the Cache with a provided key and value
func (c *Cache) Save(key string, value protoencoding.Resolver) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}
