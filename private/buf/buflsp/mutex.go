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

// This file defines various concurrency helpers.

package buflsp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

const poison = ^uint64(0)

var nextRequestID atomic.Uint64

// mutexPool represents a group of reentrant muteces that cannot be acquired simultaneously.
//
// A zero mutexPool is ready to use.
type mutexPool struct {
	lock sync.Mutex
	held map[uint64]*mutex
}

// NewMutex creates a new mutex in this pool.
func (mp *mutexPool) NewMutex() mutex {
	return mutex{pool: mp}
}

// check checks what id is either not holding a lock, or is holding the given
// map, depending on whether isUnlock is set.
func (mp *mutexPool) check(id uint64, mu *mutex, isUnlock bool) {
	if mp == nil {
		return
	}

	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.held == nil {
		mp.held = make(map[uint64]*mutex)
	}

	if isUnlock {
		if held := mp.held[id]; held != mu {
			panic(fmt.Sprintf("buflsp/mutex.go: attempted to unlock incorrect non-reentrant lock: %p -> %p", held, mu))
		}

		delete(mp.held, id)
	} else {
		if held := mp.held[id]; held != nil {
			panic(fmt.Sprintf("buflsp/mutex.go: attempted to acquire two non-reentrant locks at once: %p -> %p", mu, held))
		}

		mp.held[id] = mu
	}
}

// mutex is a sync.Mutex with some extra features.
//
// The main feature is reentrancy-checking. Within the LSP, we need to lock-protect many structures,
// and it is very easy to deadlock if the same request tries to lock something multiple times.
// To achieve this, Lock() takes a context, which must be modified by withRequestID().
type mutex struct {
	lock sync.Mutex
	// This is the id of the context currently holding the lock.
	who  atomic.Uint64
	pool *mutexPool
}

// Lock attempts to acquire this mutex or blocks.
//
// Unlike [sync.Mutex.Lock], this takes a Context. If that context was updated with withRequestID,
// this function will panic when attempting to lock the mutex while it is already held by a
// goroutine using this same context.
//
// NOTE: to Lock() and Unlock() with the same context DO NOT synchronize with each other. For example,
// attempting to lock this mutex from two different goroutines with the same context will
// result in undefined behavior.
//
// Also unlike [sync.Mutex.Lock], it returns an idempotent unlocker function. This can be used like
// defer mu.Lock()(). Note that only the outer function call is deferred: this is part of the
// definition of defer. See https://go.dev/play/p/RJNKRcoQRo1. This unlocker can also be used to
// defer unlocking but also unlock before the function returns.
//
// The returned unlocker is not thread-safe.
func (mu *mutex) Lock(ctx context.Context) (unlocker func()) {
	var unlocked bool
	unlocker = func() {
		if unlocked {
			return
		}
		mu.Unlock(ctx)
		unlocked = true
	}

	id := getRequestID(ctx)

	if mu.who.Load() == id && id > 0 {
		// We seem to have tried to lock this lock twice. Panic, and poison the lock.
		mu.who.Store(poison)
		panic("buflsp/mutex.go: non-reentrant lock locked twice by the same request")
	}

	mu.pool.check(id, mu, false)

	// Ok, we're definitely not holding a lock, so we can block until we acquire the lock.
	mu.lock.Lock()
	mu.storeWho(id)

	return unlocker
}

// Unlock releases this mutex.
//
// Unlock must be called with the same context that locked it, otherwise this function panics.
func (mu *mutex) Unlock(ctx context.Context) {
	id := getRequestID(ctx)
	if mu.who.Load() != id {
		panic("buflsp/mutex.go: lock was locked by one request and unlocked by another")
	}

	mu.storeWho(0)

	mu.pool.check(id, mu, true)
	mu.lock.Unlock()
}

func (mu *mutex) storeWho(id uint64) {
	for {
		// This has to be a CAS loop to avoid races with a poisoning p.
		old := mu.who.Load()
		if old == poison {
			panic("buflsp/mutex.go: non-reentrant lock locked twice by the same request")
		}
		if mu.who.CompareAndSwap(old, id) {
			break
		}
	}
}

// withRequestID assigns a unique request ID to the given context, which can be retrieved
// with with getRequestID.
func withRequestID(ctx context.Context) context.Context {
	// This will always be unique. It is impossible to increment a uint64 and wrap around before
	// the heat death of the universe.
	id := nextRequestID.Add(1)
	// We need to give the context package a unique identifier for the request; it can be
	// any value. The address of the global we mint new IDs from is actually great for this,
	// because users can't access it outside of this package, nor can they extract it out
	// of the context itself.
	return context.WithValue(ctx, &nextRequestID, id)
}

// getRequestID returns the request ID for this context, or 0 if ctx is nil or has no
// such ID.
func getRequestID(ctx context.Context) uint64 {
	if ctx == nil {
		return 0
	}
	id, ok := ctx.Value(&nextRequestID).(uint64)
	if !ok {
		return 0
	}

	// Make sure we don't return 0. This is the only place where the id is actually
	// witnessed so doing +1 won't affect anything.
	return id + 1
}
