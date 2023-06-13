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

package bufmoduleref

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverRemote(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name           string
		references     []string
		expectedRemote string
	}
	testCases := []testCase{
		{
			name:           "nil_references",
			expectedRemote: "buf.build",
		},
		{
			name:           "no_references",
			references:     []string{},
			expectedRemote: "buf.build",
		},
		{
			name: "some_references",
			references: []string{
				"buf.build/foo/bar",
				"buf.build/foo/baz",
				"buf.build/bar/baz",
			},
			expectedRemote: "buf.build",
		},
		{
			name: "some_invalid_references",
			references: []string{
				"buf.build/foo/bar",
				"",
				"buf.build/bar/baz",
			},
			expectedRemote: "buf.build",
		},
		{
			name: "all_single_tenant_references",
			references: []string{
				"buf.acme.com/foo/bar",
				"buf.acme.com/foo/baz",
				"buf.acme.com/bar/baz",
			},
			expectedRemote: "buf.acme.com",
		},
		{
			name: "some_single_tenant_references",
			references: []string{
				"buf.build/foo/bar",
				"buf.build/foo/baz",
				"buf.first.com/bar/baz",
				"buf.second.com/bar/baz",
			},
			expectedRemote: "buf.first.com",
		},
		{
			name: "some_invalid_references_with_single_tenant",
			references: []string{
				"buf.build/foo/bar",
				"",
				"buf.first.com/bar/baz",
				"buf.second.com/bar/baz",
			},
			expectedRemote: "buf.first.com",
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var references []ModuleReference
				for _, r := range tc.references {
					ref, _ := ModuleReferenceForString(r)
					references = append(references, ref)
				}
				assert.Equal(t, tc.expectedRemote, DiscoverRemote(references))
			})
		}(tc)
	}
}
