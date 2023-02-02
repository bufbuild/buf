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

package bandeps

import "sync"

// You can never hold more than one key at a time! We do not enforce lock ordering!

type keyRWLock struct {
	keyToRWLock map[string]*sync.RWMutex
	lock        sync.Mutex
}

func newKeyRWLock() *keyRWLock {
	return &keyRWLock{
		keyToRWLock: make(map[string]*sync.RWMutex),
	}
}

func (k *keyRWLock) RLock(key string) {
	k.getRWLock(key).Lock()
}

func (k *keyRWLock) RUnlock(key string) {
	k.getRWLock(key).Unlock()
}

func (k *keyRWLock) Lock(key string) {
	k.getRWLock(key).Lock()
}

func (k *keyRWLock) Unlock(key string) {
	k.getRWLock(key).Unlock()
}

func (k *keyRWLock) getRWLock(key string) *sync.RWMutex {
	k.lock.Lock()
	rwLock, ok := k.keyToRWLock[key]
	if !ok {
		rwLock = &sync.RWMutex{}
		k.keyToRWLock[key] = rwLock
	}
	k.lock.Unlock()
	return rwLock
}
