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

// Package refcount provides utilities for working with reference-counted objects.
//
// Why would you need refcounting in a language that already has GC? The GC
// can't always tell that all references to some object are gone. For example,
// suppose that we have a map[string]*T for looking up values based on some
// string key, but we want to evict elements of that map if no structure holds
// the key to it anymore. Doing this correctly requires a separately managed
// refcount. For this, you would use [refcount.Map].
package refcount

import "sync"

// Map is a map from keys of type K to values of type *V.
//
// Unlike a built-in map, refcount.Map allows for a key to be inserted multiple
// times concurrently, and deleted multiple times. A key that is inserted n times
// will only be evicted from the map once it is deleted n times.
//
// A zero map is empty and ready to use. Like other Go concurrency primitives, it
// must not be copied after first use.
//
// recount.Map is thread-safe: insertions synchronize-before deletions.
type Map[K comparable, V any] struct {
	lock  sync.RWMutex
	table map[K]*counted[V]
}

// Insert inserts a key into the map.
//
// If the value is already present in the map, its count is incremented by one;
// otherwise, the zero value is inserted and returned. This function returns whether
// an existing entry was found.
//
// The returned pointer is never nil.
func (m *Map[K, V]) Insert(key K) (value *V, found bool) {
	// NOTE: By replacing counted[V].count with an atomic.Int64, this
	// can be downgraded to a read lock, with an upgrade only in the case
	// we are inserting a new entry.
	//
	// This optimization is not performed in the name of expediency, I have
	// only recorded it as potential future work
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.table == nil {
		m.table = make(map[K]*counted[V])
	}

	v, found := m.table[key]
	if !found {
		v = &counted[V]{}
		m.table[key] = v
	}
	v.count++
	return &v.value, found
}

// Get looks up a key in the map.
//
// This is identical to ordinary map lookup: if they key is not present, it does not
// insert and returns nil.
func (m *Map[K, V]) Get(key K) *V {
	m.lock.RLock()
	defer m.lock.RUnlock()
	value := m.table[key]
	if value == nil {
		return nil
	}
	return &value.value
}

// Delete deletes a key from the map.
//
// The key will only be evicted once [Map.Delete] has been called an equal number of times
// to prior calls to [Map.Insert] for this key.
//
// If the key is present and was actually evicted, the element it maps to is returned. Otherwise,
// this function returns nil.
func (m *Map[K, V]) Delete(key K) *V {
	m.lock.Lock()
	defer m.lock.Unlock()

	v := m.table[key]
	if v == nil {
		return nil
	}

	v.count--
	if v.count > 0 {
		return nil
	}

	delete(m.table, key)
	return &v.value
}

// counted is a reference-counted value.
type counted[T any] struct {
	count int32 // Protected by Map.lock.
	value T
}
