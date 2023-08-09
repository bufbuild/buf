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

package bufgen

import (
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv1"
)

func TestMigrateV1ToV2Success(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description string
		original    ExternalConfigV1
		expected    ExternalConfigV2
	}{
		{
			description: "plugins",
			original: bufgenv1.ExternalConfigV1{
				Version: "v1",
				Plugins: []bufgenv1.ExternalPluginConfigV1{
					{
						Plugin:   "buf.build/protocolbuffers/cpp",
						Out:      "gen/cpp",
						Opt:      "xyz",
						Strategy: "all",
					},
					{
						Name:     "buf.build/protocolbuffers/cpp",
						Out:      "gen/cpp",
						Opt:      "xyz",
						Strategy: "all",
					},
					{
						Plugin:   "buf.build/protocolbuffers/cpp",
						Out:      "gen/cpp",
						Opt:      "xyz",
						Strategy: "all",
					},
				},
			},
		},
		{
			description: "managed mode",
			original: bufgenv1.ExternalConfigV1{
				Version: "v1",
				Plugins: []bufgenv1.ExternalPluginConfigV1{
					{
						Plugin: "buf.build/protocolbuffers/cpp",
						Out:    "gen/cpp",
						Opt:    "xyz",
					},
				},
				Managed: bufgenv1.ExternalManagedConfigV1{
					Enabled: true,
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
		})
	}
}

func TestMigrateV1ToV2Error(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description   string
		original      ExternalConfigV1
		expectedError string
	}{}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
		})
	}
}
