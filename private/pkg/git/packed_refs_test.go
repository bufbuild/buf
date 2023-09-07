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

package git

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPackedRefs(t *testing.T) {
	t.Parallel()

	allBytes, err := os.ReadFile(path.Join("testdata", "packed-refs"))
	require.NoError(t, err)

	branches, tags, err := parsePackedRefs(allBytes)

	require.NoError(t, err)
	hexBranches := map[string]string{}
	for branch, hash := range branches {
		hexBranches[branch] = hash.Hex()
	}
	hexTags := map[string]string{}
	for tag, hash := range tags {
		hexTags[tag] = hash.Hex()
	}
	assert.Equal(t, map[string]string{
		"main":         "45c2edc61040013349e094663e492996e0c044e3",
		"paralleltest": "1fddd89116e24df213d43b7d837f5dd29ee9cbf0",
	}, hexBranches)
	assert.Equal(t, map[string]string{
		"v0.1.0":  "157c7ae554844ff7ae178536ec10787b5b74b5db",
		"v0.2.0":  "ace9301f315979bd053b7658c017391fe1af8804",
		"v1.10.0": "ebb191e8268db7cee389e3abb0d1edc1852337a3",
	}, hexTags)
}
