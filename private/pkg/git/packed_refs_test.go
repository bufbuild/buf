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
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPackedRefs(t *testing.T) {
	t.Parallel()

	allBytes, err := os.ReadFile(normalpath.Join("testdata", "packed-refs"))
	require.NoError(t, err)

	branches, tags, err := parsePackedRefs(allBytes)
	require.NoError(t, err)

	t.Run("branches", func(t *testing.T) {
		t.Parallel()
		hexBranches := make(map[string]map[string]string)
		for remote, branchToHash := range branches {
			hexBranches[remote] = make(map[string]string)
			for branch, hash := range branchToHash {
				hexBranches[remote][branch] = hash.Hex()
			}
		}
		assert.Equal(t, map[string]map[string]string{
			"": { // local
				"main":         "45c2edc61040013349e094663e492996e0c044e3",
				"paralleltest": "1fddd89116e24df213d43b7d837f5dd29ee9cbf0",
			},
			"origin": {
				"main":         "45c2edc61040013349e094663e492996e0c044e3",
				"paralleltest": "1fddd89116e24df213d43b7d837f5dd29ee9cbf0",
				"other/branch": "1fddd89116e24df213d43b7d837f5dd29ee9cbf1",
			},
			"otherorigin": {
				"main":               "27523d9000238e0f7fb35d6052d10016852beee3",
				"paralleltest":       "959e716b38b179bd5a4e7edfc549db2e30df3c8e",
				"yet/another/branch": "959e716b38b179bd5a4e7edfc549db2e30df3c8f",
			},
		}, hexBranches)
	})
	t.Run("tags", func(t *testing.T) {
		t.Parallel()
		hexTags := map[string]string{}
		for tag, hash := range tags {
			hexTags[tag] = hash.Hex()
		}
		assert.Equal(t, map[string]string{
			"v0.1.0":  "157c7ae554844ff7ae178536ec10787b5b74b5db",
			"v0.2.0":  "ace9301f315979bd053b7658c017391fe1af8804",
			"v1.10.0": "ebb191e8268db7cee389e3abb0d1edc1852337a3",
		}, hexTags)
	})
}
