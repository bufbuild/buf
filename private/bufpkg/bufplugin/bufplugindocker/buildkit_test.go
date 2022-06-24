// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufplugindocker

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNodeIDPersistence(t *testing.T) {
	t.Parallel()
	configDir := path.Join("testdata", t.Name())
	t.Cleanup(func() {
		if err := os.RemoveAll(configDir); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove persisted node id: %v", err)
		}
	})
	sharedKey := getBuildSharedKey(zap.L(), ".", configDir)
	assert.NotEmpty(t, sharedKey)
	st, err := os.Stat(path.Join(configDir, pathBuildkitNodeID))
	assert.Nil(t, err)
	assert.True(t, !st.IsDir())
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), st.Mode().Perm())
	}
	assert.True(t, st.Size() > 0)
	sharedKeyAgain := getBuildSharedKey(zap.L(), ".", configDir)
	assert.Equal(t, sharedKey, sharedKeyAgain)
}

func TestNodeIDSkipPersistence(t *testing.T) {
	t.Parallel()
	sharedKey := getBuildSharedKey(zap.L(), ".", "")
	assert.NotEmpty(t, sharedKey)
	sharedKeyAgain := getBuildSharedKey(zap.L(), ".", "")
	assert.NotEmpty(t, sharedKeyAgain)
	assert.NotEqual(t, sharedKey, sharedKeyAgain)
}
