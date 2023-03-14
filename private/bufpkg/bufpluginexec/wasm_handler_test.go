package bufpluginexec

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestValidateWASMFilePath(t *testing.T) {
	wasmPath := "/tmp/test.wasm"
	notWasmPath := "/tmp/test.txt"
	assert.NoError(t, os.WriteFile(wasmPath, []byte("a"), 0644))
	assert.NoError(t, os.WriteFile(notWasmPath, []byte("a"), 0644))
	defer func() {
		assert.NoError(t, os.Remove(wasmPath))
		assert.NoError(t, os.Remove(notWasmPath))
	}()
	t.Run("pass for valid wasm", func(t *testing.T) {
		_, err := validateWASMFilePath(wasmPath)
		assert.NoError(t, err)
	})
	t.Run("fail if not found", func(t *testing.T) {
		_, err := validateWASMFilePath("notfound")
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
	t.Run("fail if invalid extension", func(t *testing.T) {
		_, err := validateWASMFilePath(notWasmPath)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
}
