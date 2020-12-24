// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulestorage

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// Key is a list of strings used to uniquely identify a module
// within the Store. Each element of the key must be a valid
// path component.
type Key []string

// Store is the interface implemented by the module store.
type Store interface {
	Get(ctx context.Context, moduleKey Key) (bufmodule.Module, error)
	Put(ctx context.Context, moduleKey Key, module bufmodule.Module) error
	Delete(ctx context.Context, moduleKey Key) error
	// ForEachKey iterates over every key and calls f.
	//
	// If an error occurs during iteration, it will be passed to f. If f
	// returns nil, the iteration will continue, else if it returns error,
	// the iteration will stop and the error will be returned.
	ForEachKey(ctx context.Context, f func(Key, error) error) error
}

// NewStore creates a new module store backed by the readWriteBucket.
func NewStore(logger *zap.Logger, readWriteBucket storage.ReadWriteBucket) Store {
	return newStore(logger, readWriteBucket)
}

// NewModulePinKey returns a new Key for the ModulePin.
//
// This does not use digest.
func NewModulePinKey(modulePin bufmodule.ModulePin) Key {
	return Key{
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
		modulePin.Track(),
		modulePin.Commit(),
	}
}
