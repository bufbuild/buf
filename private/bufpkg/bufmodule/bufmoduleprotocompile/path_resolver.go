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

package bufmoduleprotocompile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"go.uber.org/multierr"
)

// TODO: remove when we remove ModuleFileSet
type moduleFileReader interface {
	GetModuleFile(context.Context, string) (bufmodule.ModuleFile, error)
}

type parserAccessorHandler struct {
	ctx                  context.Context
	moduleFileReader     moduleFileReader
	pathToExternalPath   map[string]string
	nonImportPaths       map[string]struct{}
	pathToModuleIdentity map[string]bufmoduleref.ModuleIdentity
	pathToCommit         map[string]string
	lock                 sync.RWMutex
}

func newParserAccessorHandler(
	ctx context.Context,
	moduleFileReader moduleFileReader,
) *parserAccessorHandler {
	return &parserAccessorHandler{
		ctx:                  ctx,
		moduleFileReader:     moduleFileReader,
		pathToExternalPath:   make(map[string]string),
		nonImportPaths:       make(map[string]struct{}),
		pathToModuleIdentity: make(map[string]bufmoduleref.ModuleIdentity),
		pathToCommit:         make(map[string]string),
	}
}

func (p *parserAccessorHandler) Open(path string) (_ io.ReadCloser, retErr error) {
	moduleFile, moduleErr := p.moduleFileReader.GetModuleFile(p.ctx, path)
	if moduleErr != nil {
		if !errors.Is(moduleErr, fs.ErrNotExist) {
			return nil, moduleErr
		}
		if wktModuleFile, wktErr := datawkt.ReadBucket.Get(p.ctx, path); wktErr == nil {
			if wktModuleFile.Path() != path {
				// this should never happen, but just in case
				return nil, fmt.Errorf("parser accessor requested path %q but got %q", path, wktModuleFile.Path())
			}
			if err := p.addPath(path, path, nil, ""); err != nil {
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
	if err := p.addPath(
		path,
		moduleFile.ExternalPath(),
		moduleFile.ModuleIdentity(),
		moduleFile.Commit(),
	); err != nil {
		return nil, err
	}
	return moduleFile, nil
}

func (p *parserAccessorHandler) ExternalPath(path string) string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if externalPath := p.pathToExternalPath[path]; externalPath != "" {
		return externalPath
	}
	return path
}

func (p *parserAccessorHandler) ModuleIdentity(path string) bufmoduleref.ModuleIdentity {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.pathToModuleIdentity[path] // nil is a valid value.
}

func (p *parserAccessorHandler) Commit(path string) string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.pathToCommit[path] // empty is a valid value.
}

func (p *parserAccessorHandler) addPath(
	path string,
	externalPath string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	commit string,
) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	existingExternalPath, ok := p.pathToExternalPath[path]
	if ok {
		if existingExternalPath != externalPath {
			return fmt.Errorf("parser accessor had external paths %q and %q for path %q", existingExternalPath, externalPath, path)
		}
	} else {
		p.pathToExternalPath[path] = externalPath
	}
	if moduleIdentity != nil {
		p.pathToModuleIdentity[path] = moduleIdentity
	}
	if commit != "" {
		p.pathToCommit[path] = commit
	}
	return nil
}
