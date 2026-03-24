// Copyright 2020-2026 Buf Technologies, Inc.
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
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/google/uuid"
)

// moduleFileResolver resolves sources and module information for module files, including
// well-known types. This is used by the compiler and image building processes.
// [context.Context] is embedded in this case for [storage.ReadBucket] APIs.
type moduleFileResolver struct {
	ctx                context.Context
	moduleReadBucket   bufmodule.ModuleReadBucket
	pathToExternalPath map[string]string
	pathToLocalPath    map[string]string
	localPathToPath    map[string]string
	nonImportPaths     map[string]struct{}
	pathToFullName     map[string]bufparse.FullName
	pathToCommitID     map[string]uuid.UUID
	lock               sync.RWMutex
}

func newModuleFileResolver(
	ctx context.Context,
	moduleReadBucket bufmodule.ModuleReadBucket,
) *moduleFileResolver {
	return &moduleFileResolver{
		ctx:                ctx,
		moduleReadBucket:   moduleReadBucket,
		pathToExternalPath: make(map[string]string),
		pathToLocalPath:    make(map[string]string),
		localPathToPath:    make(map[string]string),
		nonImportPaths:     make(map[string]struct{}),
		pathToFullName:     make(map[string]bufparse.FullName),
		pathToCommitID:     make(map[string]uuid.UUID),
	}
}

// Open opens the given path, and tracks the external path and import status.
//
// This implements [source.Opener].
func (m *moduleFileResolver) Open(path string) (_ *source.File, retErr error) {
	moduleFile, moduleErr := m.moduleReadBucket.GetFile(m.ctx, path)
	if moduleErr != nil {
		if !errors.Is(moduleErr, fs.ErrNotExist) {
			return nil, moduleErr
		}
		if wktModuleFile, wktErr := datawkt.ReadBucket.Get(m.ctx, path); wktErr == nil {
			if wktModuleFile.Path() != path {
				// This should never happen, but just in case
				return nil, fmt.Errorf("requested path %q but got %q", path, wktModuleFile.Path())
			}
			if err := m.addPath(path, path, "", nil, uuid.Nil); err != nil {
				return nil, err
			}
			return readObjectCloserToSourceFile(path, wktModuleFile)
		}
		return nil, moduleErr
	}
	defer func() {
		retErr = errors.Join(retErr, moduleFile.Close())
	}()
	if moduleFile.Path() != path {
		// this should never happen, but just in case
		return nil, fmt.Errorf("requested path %q but got %q", path, moduleFile.Path())
	}
	if err := m.addPath(
		path,
		moduleFile.ExternalPath(),
		moduleFile.LocalPath(),
		moduleFile.Module().FullName(),
		moduleFile.Module().CommitID(),
	); err != nil {
		return nil, err
	}
	return readObjectCloserToSourceFile(moduleFile.LocalPath(), moduleFile)
}

// ExternalPath returns the external path for the input path.
//
// Returns the input path if the external path is not known.
func (m *moduleFileResolver) ExternalPath(path string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if externalPath := m.pathToExternalPath[path]; externalPath != "" {
		return externalPath
	}
	return path
}

// LocalPath returns the local path for the input path if present.
func (m *moduleFileResolver) LocalPath(path string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.pathToLocalPath[path]
}

// PathForLocalPath returns the import path for the given local path if present.
func (m *moduleFileResolver) PathForLocalPath(localPath string) string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.localPathToPath[localPath]
}

// FullName returns nil if not available.
func (m *moduleFileResolver) FullName(path string) bufparse.FullName {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.pathToFullName[path] // nil is a valid value.
}

// CommitID returns empty if not available.
func (m *moduleFileResolver) CommitID(path string) uuid.UUID {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.pathToCommitID[path] // empty is a valid value.
}

func (m *moduleFileResolver) addPath(
	path string,
	externalPath string,
	localPath string,
	moduleFullName bufparse.FullName,
	commitID uuid.UUID,
) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	existingExternalPath, ok := m.pathToExternalPath[path]
	if ok {
		if existingExternalPath != externalPath {
			return fmt.Errorf("had external paths %q and %q for path %q", existingExternalPath, externalPath, path)
		}
	} else {
		m.pathToExternalPath[path] = externalPath
	}
	if localPath != "" {
		existingLocalPath, ok := m.pathToLocalPath[path]
		if ok {
			if existingLocalPath != localPath {
				return fmt.Errorf("had local paths %q and %q for path %q", existingLocalPath, localPath, path)
			}
		} else {
			m.pathToLocalPath[path] = localPath
			m.localPathToPath[localPath] = path
		}
	}
	if moduleFullName != nil {
		m.pathToFullName[path] = moduleFullName
	}
	if commitID != uuid.Nil {
		m.pathToCommitID[path] = commitID
	}
	return nil
}

// readObjectCloserToSourceFile is a helper function that takes a given [storage.ReadObjectCloser]
// and returns the corresponding [*source.File].
func readObjectCloserToSourceFile(
	path string,
	readObjectCloser storage.ReadObjectCloser,
) (*source.File, error) {
	var buf strings.Builder
	if _, err := io.Copy(&buf, readObjectCloser); err != nil {
		return nil, err
	}
	return source.NewFile(path, buf.String()), nil
}
