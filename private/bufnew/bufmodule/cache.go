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
