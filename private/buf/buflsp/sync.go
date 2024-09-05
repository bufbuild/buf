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
	"sync"
	"sync/atomic"
)

// mutex is a sync.Mutex with some extra features.
//
// The main feature is reentrancy. Within the LSP, we need to lock-protect many structures,
// and it is very easy to deadlock if the same request tries to lock something multiple times.
// To achieve this, Lock() takes a context, which must be modified by withReentrancy().
type mutex struct {
	mu sync.Mutex
	// This is the id of the context currently holding the lock.
	who atomic.Uint32
	// This is the number of times we have acquired this lock, assuming who is nonzero.
	lockers uint32
}

var nextReentrancyId uint32 = 1

// withMutexId enables this context to be used to non-reentrantly lock a mutex.
//
// This function essentially creates a scope in which attempting to reentrantly lock a mutex
// panics instead of deadlocking.
func withReentrancy(ctx context.Context) context.Context {
	id := nextReentrancyId
	nextReentrancyId++
	return context.WithValue(ctx, &nextReentrancyId, id)
}

// getRentrancy returns the reentrancy ID for this context, or 0 if ctx is nil or has no
// such ID.
func getReentrancy(ctx context.Context) uint32 {
	if ctx == nil {
		return 0
	}
	id, ok := ctx.Value(&nextReentrancyId).(uint32)
	if !ok {
		return 0
	}
	return id
}

// Lock attempts to acquire this mutex or blocks.
//
// Unlike sync.Mutex, this takes a Context. If that context was updated with withReentrancy,
// this function will panic when attempting to lock the mutex while it is already held by a
// goroutine using this same context.
//
// NOTE: to Lock() and Unlock() with the same context DO NOT synchronize with each other. For example,
// attempting to lock this mutex from two different goroutines with the same context will
// result in undefined behavior.
func (mu *mutex) Lock(ctx context.Context) {
	id := getReentrancy(ctx)
	if id == 0 {
		// If no ID is present, simply lock the lock.
		mu.mu.Lock()
		return
	}

	if mu.who.Load() == id {
		// This context is the one currently holding this lock. If we see any other
		// value, this means the lock is either unlocked, or currently held by a different
		// context.
		//
		// Situations where the load above would go stale are not possible, because we
		// require that callers do not attempt to lock and unlock the mutex with the same
		// context concurrently.
		mu.lockers++
		return
	}

	mu.mu.Lock()
	mu.who.Store(id)
	mu.lockers++
}

// Unlock releases this mutex.
//
// Unlock must be called with the same context that locked it, otherwise this function panics.
func (mu *mutex) Unlock(ctx context.Context) {
	id := getReentrancy(ctx)
	if id == 0 {
		// If no ID is present, simply unlock the lock.
		mu.mu.Lock()
		return
	}

	// See the comment in Lock() for why this check is sufficient.
	if mu.who.Load() != id {
		panic("attempted to unlock reentrant mutex with the wrong context")
	}

	mu.lockers--
	if mu.lockers == 0 {
		mu.who.Store(0)
		mu.mu.Unlock()
	}
}
