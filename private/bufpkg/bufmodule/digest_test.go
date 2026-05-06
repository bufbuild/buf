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

package bufmodule_test

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/stretchr/testify/require"
)

func TestNopProvidersToCalculateDigest(t *testing.T) {
	t.Parallel()
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(
		t.Context(),
		slogtestext.NewLogger(t),
		bufmodule.NopModuleDataProvider,
		bufmodule.NopCommitProvider,
	)
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module1.proto": []byte(
					`syntax = proto3; package module1; import "module2.proto"; import "google/protobuf/timestamp.proto";`,
				),
			},
		),
		"module1",
		true,
	)
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module2; import "module4.proto";`,
				),
			},
		),
		"module2",
		true,
	)
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module3.proto": []byte(
					`syntax = proto3; package module3; import "module4.proto";`,
				),
			},
		),
		"module3",
		true,
	)
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module4.proto": []byte(
					`syntax = proto3; package module4;`,
				),
			},
		),
		"module4",
		true,
	)
	moduleSet, err := moduleSetBuilder.Build()
	require.NoError(t, err)
	require.NotNil(t, moduleSet.GetModuleForOpaqueID("module1"))
	for _, module := range moduleSet.Modules() {
		digest, err := module.Digest(bufmodule.DigestTypeB5)
		require.NoError(t, err)
		require.NotEmpty(t, digest.String())
	}
}
