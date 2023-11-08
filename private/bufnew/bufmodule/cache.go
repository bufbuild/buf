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

package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/protocompile/parser/imports"
	"go.uber.org/multierr"
)

type cache struct {
	opaqueIDToModule  map[string]Module
	filePathToImports map[string]*tuple[map[string]struct{}, error]
	filePathToModule  map[string]*tuple[Module, error]
	// Just making thread-safe to future-proof a bit.
	// Could have per-map lock but then we need to deal with lock ordering, not worth it for now.
	lock sync.RWMutex
	// We do not bother locking around this variable as this is just part of construction.
	setModulesCalled bool
}

func newCache() *cache {
	return &cache{
		filePathToImports: make(map[string]*tuple[map[string]struct{}, error]),
		filePathToModule:  make(map[string]*tuple[Module, error]),
	}
}

func (c *cache) GetModuleForOpaqueID(opaqueID string) (Module, error) {
	if !c.setModulesCalled {
		return nil, errors.New("cache.SetModules never called")
	}
	// No need for lock: opaqueIDToModule is static.
	module, ok := c.opaqueIDToModule[opaqueID]
	if !ok {
		// This should never happen given how we use the cache.
		return nil, fmt.Errorf("no Module for opaqueID: %q", opaqueID)
	}
	return module, nil
}

func (c *cache) GetModuleForFilePath(ctx context.Context, filePath string) (Module, error) {
	if !c.setModulesCalled {
		return nil, errors.New("cache.SetModules never called")
	}
	return getDoubleLock(
		&c.lock,
		c.filePathToModule,
		filePath,
		func() (Module, error) {
			return c.getModuleForFilePathUncached(ctx, filePath)
		},
	)
}

func (c *cache) GetImportsForFilePath(ctx context.Context, filePath string) (map[string]struct{}, error) {
	if !c.setModulesCalled {
		return nil, errors.New("cache.SetModules never called")
	}
	return getDoubleLock(
		&c.lock,
		c.filePathToImports,
		filePath,
		func() (map[string]struct{}, error) {
			return c.getImportsForFilePathUncached(ctx, filePath)
		},
	)
}

// It is assumed that getUniqueModulesWithEarlierPreferred has been called on this Module list
// and that all Modules are unique. It is an error within the cache if any two Modules have
// overlapping files, or if any two modules have overlapping opaque IDs or overlapping digests.
func (c *cache) SetModules(modules []Module) error {
	if c.setModulesCalled {
		return errors.New("cache.SetModules already called")
	}
	opaqueIDToModule := make(map[string]Module)
	for _, module := range modules {
		opaqueID := module.opaqueID()
		if _, ok := opaqueIDToModule[opaqueID]; ok {
			// This is a system error, we should have already validated this.
			return fmt.Errorf("duplicate opaqueID: %q", opaqueID)
		}
		opaqueIDToModule[opaqueID] = module
	}
	c.opaqueIDToModule = opaqueIDToModule
	c.setModulesCalled = true
	return nil
}

// Assumed to be called within lock
func (c *cache) getModuleForFilePathUncached(ctx context.Context, filePath string) (Module, error) {
	matchingOpaqueIDs := make(map[string]struct{})
	// Note that we're effectively doing an O(num_modules * num_files) operation here, which could be prohibitive.
	for opaqueID, module := range c.opaqueIDToModule {
		if _, err := module.StatFileInfo(ctx, filePath); err == nil {
			matchingOpaqueIDs[opaqueID] = struct{}{}
		}
	}
	switch len(matchingOpaqueIDs) {
	case 0:
		// This should likely never happen given how we call the cache.
		return nil, fmt.Errorf("no Module contains file %q", filePath)
	case 1:
		var matchingOpaqueID string
		for matchingOpaqueID = range matchingOpaqueIDs {
		}
		return c.opaqueIDToModule[matchingOpaqueID], nil
	default:
		// This actually could happen, and we will want to make this error message as clear as possible.
		// The addition of opaqueID should give us clearer error messages than we have today.
		return nil, fmt.Errorf("multiple Modules contained file %q: %v", filePath, stringutil.MapToSortedSlice(matchingOpaqueIDs))
	}
}

// Assumed to be called within lock
func (c *cache) getImportsForFilePathUncached(ctx context.Context, filePath string) (_ map[string]struct{}, retErr error) {
	// Even when we know the file we want to get the imports for, we want to make sure the file
	// is not duplicated across multiple modules. By calling getModuleFileFilePathUncached,
	// we implicitly get this check for now.
	//
	// Note this basically kills the idea of only partially-lazily-loading some of the Modules
	// within a set of []Modules. We could optimize this later, and may want to. This means
	// that we're going to have to load all the modules within a workspace even if just building
	// a single module in the workspace, as an example. Luckily, modules within workspaces are
	// the cheapest to load (ie not remote).
	module, err := getWithinLock(
		c.filePathToModule,
		filePath,
		func() (Module, error) {
			return c.getModuleForFilePathUncached(ctx, filePath)
		},
	)
	if err != nil {
		return nil, err
	}
	file, err := module.GetFile(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	imports, err := imports.ScanForImports(file)
	if err != nil {
		return nil, err
	}
	return stringutil.SliceToMap(imports), nil
}

func getDoubleLock[T any](
	lock *sync.RWMutex,
	cache map[string]*tuple[T, error],
	key string,
	get func() (T, error),
) (T, error) {
	lock.RLock()
	tuple, ok := cache[key]
	lock.RUnlock()
	if ok {
		return tuple.V1, tuple.V2
	}
	lock.Lock()
	value, err := getWithinLock(cache, key, get)
	lock.Unlock()
	return value, err
}

func getWithinLock[T any](
	cache map[string]*tuple[T, error],
	key string,
	get func() (T, error),
) (T, error) {
	tuple, ok := cache[key]
	if ok {
		return tuple.V1, tuple.V2
	}
	value, err := get()
	cache[key] = newTuple(value, err)
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
