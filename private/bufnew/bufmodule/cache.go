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
	"io"
	"sync"
)

type cache struct {
	modules              []Module
	filePathToImportData map[string]*importData
	// Just making thread-safe to future-proof a bit.
	lock sync.RWMutex
}

func newCache(modules []Module) *cache {
	return &cache{
		modules:              modules,
		filePathToImportData: make(map[string]*importData),
	}
}

func (c *cache) GetImports(filePath string, f func() (io.ReadCloser, error)) (map[string]struct{}, error) {
	c.lock.RLock()
	importData, ok := c.filePathToImportData[filePath]
	c.lock.RUnlock()
	if ok {
		return importData.imports, importData.err
	}
	c.lock.Lock()
	importData, ok = c.filePathToImportData[filePath]
	if ok {
		c.lock.Unlock()
		return importData.imports, importData.err
	}
	return nil, nil
}

type importData struct {
	imports map[string]struct{}
	err     error
}

func newImportData(imports map[string]struct{}, err error) *importData {
	return &importData{
		imports: imports,
		err:     err,
	}
}
