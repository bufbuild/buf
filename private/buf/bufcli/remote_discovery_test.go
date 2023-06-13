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

package bufcli

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
)

func TestDiscoverRemote(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                string
		references          []string
		expectedSelectedRef string
	}
	testCases := []testCase{
		{
			name:                "nil_references",
			expectedSelectedRef: "",
		},
		{
			name:                "no_references",
			references:          []string{},
			expectedSelectedRef: "",
		},
		{
			name: "some_references",
			references: []string{
				"buf.build/foo/repo1",
				"buf.build/foo/repo2",
				"buf.build/foo/repo3",
			},
			expectedSelectedRef: "buf.build/foo/repo1",
		},
		{
			name: "some_invalid_references",
			references: []string{
				"buf.build/foo/repo1",
				"",
				"buf.build/foo/repo3",
			},
			expectedSelectedRef: "buf.build/foo/repo1",
		},
		{
			name: "all_single_tenant_references",
			references: []string{
				"buf.acme.com/foo/repo1",
				"buf.acme.com/foo/repo2",
				"buf.acme.com/foo/repo3",
			},
			expectedSelectedRef: "buf.acme.com/foo/repo1",
		},
		{
			name: "some_single_tenant_references",
			references: []string{
				"buf.build/foo/repo1",
				"buf.build/foo/repo2",
				"buf.first.com/foo/repo3",
				"buf.second.com/foo/repo4",
			},
			expectedSelectedRef: "buf.first.com/foo/repo3",
		},
		{
			name: "some_invalid_references_with_single_tenant",
			references: []string{
				"buf.build/foo/repo1",
				"buf.first.com/foo/repo2",
				"",
				"buf.second.com/foo/repo3",
			},
			expectedSelectedRef: "buf.first.com/foo/repo2",
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var references []bufmoduleref.ModuleReference
				for _, r := range tc.references {
					ref, _ := bufmoduleref.ModuleReferenceForString(r)
					references = append(references, ref)
				}
				selectedRef := SelectReferenceForRemote(references)
				if tc.expectedSelectedRef == "" {
					assert.Nil(t, selectedRef)
				} else {
					assert.Equal(t, tc.expectedSelectedRef, selectedRef.IdentityString())
				}
			})
		}(tc)
	}
}
