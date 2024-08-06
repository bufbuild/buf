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

package bufimage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/protocompile"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/descriptorpb"
)

type fileResolver struct {
	ctx                  context.Context
	moduleReadBucket     bufmodule.ModuleReadBucket
	pathToExternalPath   map[string]string
	pathToLocalPath      map[string]string
	nonImportPaths       map[string]struct{}
	pathToModuleFullName map[string]bufmodule.ModuleFullName
	pathToCommitID       map[string]uuid.UUID
	compiledDeps         map[string]*descriptorpb.FileDescriptorProto
	lock                 sync.RWMutex
}

func newFileResolver(
	ctx context.Context,
	moduleReadBucket bufmodule.ModuleReadBucket,
	compiledDeps map[string]*descriptorpb.FileDescriptorProto,
) *fileResolver {
	return &fileResolver{
		ctx:                  ctx,
		moduleReadBucket:     moduleReadBucket,
		pathToExternalPath:   make(map[string]string),
		pathToLocalPath:      make(map[string]string),
		nonImportPaths:       make(map[string]struct{}),
		pathToModuleFullName: make(map[string]bufmodule.ModuleFullName),
		pathToCommitID:       make(map[string]uuid.UUID),
		compiledDeps:         compiledDeps,
	}
}

// FindFileByPath implements protocompile.Resolver, providing sources for
// input files. If the given path matches a pre-compiled dependency, that
// pre-compiled descriptor will be returned instead of source code.
func (f *fileResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if fileDescriptor := f.compiledDeps[path]; fileDescriptor != nil {
		return protocompile.SearchResult{
			Proto: fileDescriptor,
		}, nil
	}
	reader, err := f.Open(path)
	if err != nil {
		return protocompile.SearchResult{}, err
	}
	return protocompile.SearchResult{Source: reader}, nil
}

// Open opens the given path, and tracks the external path and import status.
//
// This function can be used as the accessor function for a protocompile.SourceResolver.
func (f *fileResolver) Open(path string) (_ io.ReadCloser, retErr error) {
	moduleFile, moduleErr := f.moduleReadBucket.GetFile(f.ctx, path)
	if moduleErr != nil {
		if !errors.Is(moduleErr, fs.ErrNotExist) {
			return nil, moduleErr
		}
		if wktModuleFile, wktErr := datawkt.ReadBucket.Get(f.ctx, path); wktErr == nil {
			if wktModuleFile.Path() != path {
				// this should never happen, but just in case
				return nil, fmt.Errorf("parser accessor requested path %q but got %q", path, wktModuleFile.Path())
			}
			if err := f.addPath(path, path, "", nil, uuid.Nil); err != nil {
				return nil, err
			}
			return wktModuleFile, nil
		}
		return nil, moduleErr
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, moduleFile.Close())
		}
	}()
	if moduleFile.Path() != path {
		// this should never happen, but just in case
		return nil, fmt.Errorf("parser accessor requested path %q but got %q", path, moduleFile.Path())
	}
	if err := f.addPath(
		path,
		moduleFile.ExternalPath(),
		moduleFile.LocalPath(),
		moduleFile.Module().ModuleFullName(),
		moduleFile.Module().CommitID(),
	); err != nil {
		return nil, err
	}
	return moduleFile, nil
}

// ExternalPath returns the external path for the input path.
//
// Returns the input path if the external path is not known.
func (f *fileResolver) ExternalPath(path string) string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	if externalPath := f.pathToExternalPath[path]; externalPath != "" {
		return externalPath
	}
	return path
}

// LocalPath returns the local path for the input path if present.
func (f *fileResolver) LocalPath(path string) string {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.pathToLocalPath[path]
}

// ModuleFullName returns nil if not available.
func (f *fileResolver) ModuleFullName(path string) bufmodule.ModuleFullName {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.pathToModuleFullName[path] // nil is a valid value.
}

// CommitID returns empty if not available.
func (f *fileResolver) CommitID(path string) uuid.UUID {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.pathToCommitID[path] // empty is a valid value.
}

func (f *fileResolver) addPath(
	path string,
	externalPath string,
	localPath string,
	moduleFullName bufmodule.ModuleFullName,
	commitID uuid.UUID,
) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	existingExternalPath, ok := f.pathToExternalPath[path]
	if ok {
		if existingExternalPath != externalPath {
			return fmt.Errorf("parser accessor had external paths %q and %q for path %q", existingExternalPath, externalPath, path)
		}
	} else {
		f.pathToExternalPath[path] = externalPath
	}
	if localPath != "" {
		existingLocalPath, ok := f.pathToLocalPath[path]
		if ok {
			if existingLocalPath != localPath {
				return fmt.Errorf("parser accessor had local paths %q and %q for path %q", existingLocalPath, localPath, path)
			}
		} else {
			f.pathToLocalPath[path] = localPath
		}
	}
	if moduleFullName != nil {
		f.pathToModuleFullName[path] = moduleFullName
	}
	if !commitID.IsNil() {
		f.pathToCommitID[path] = commitID
	}
	return nil
}
