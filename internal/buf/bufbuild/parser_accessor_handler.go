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

package bufbuild

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/gen/embed/wkt"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
)

type parserAccessorHandler struct {
	ctx                context.Context
	module             bufcore.Module
	pathToExternalPath map[string]string
	pathToIsNonImport  map[string]bool
	lock               sync.RWMutex
}

func newParserAccessorHandler(ctx context.Context, module bufcore.Module) *parserAccessorHandler {
	return &parserAccessorHandler{
		ctx:                ctx,
		module:             module,
		pathToExternalPath: make(map[string]string),
		pathToIsNonImport:  make(map[string]bool),
	}
}

func (p *parserAccessorHandler) Open(path string) (_ io.ReadCloser, retErr error) {
	moduleFile, moduleErr := p.module.GetFile(p.ctx, path)
	if moduleErr != nil {
		if !storage.IsNotExist(moduleErr) {
			return nil, moduleErr
		}
		if wktModuleFile, wktErr := wkt.ReadBucket.Get(p.ctx, path); wktErr == nil {
			// not setting external path for now
			// we could make this an option, or have some way we do this for wkt
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
	p.lock.Lock()
	defer p.lock.Unlock()
	existingExternalPath, ok := p.pathToExternalPath[path]
	if ok {
		if existingExternalPath != moduleFile.ExternalPath() {
			return nil, fmt.Errorf("parser accessor had external paths %q and %q for path %q", existingExternalPath, moduleFile.ExternalPath(), path)
		}
	} else {
		p.pathToExternalPath[path] = moduleFile.ExternalPath()
	}
	p.pathToIsNonImport[path] = true
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

func (p *parserAccessorHandler) IsImport(path string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	_, isNotImport := p.pathToIsNonImport[path]
	return !isNotImport
}
