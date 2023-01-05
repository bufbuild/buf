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

	"github.com/stretchr/testify/require"
)

func TestModuleOwnerForString(t *testing.T) {
	t.Parallel()
	expectedModuleOwner, err := NewModuleOwner("foo.com", "barr")
	require.NoError(t, err)
	require.Equal(t, "foo.com", expectedModuleOwner.Remote())
	require.Equal(t, "barr", expectedModuleOwner.Owner())
	moduleOwner, err := ModuleOwnerForString("foo.com/barr")
	require.NoError(t, err)
	require.Equal(t, expectedModuleOwner, moduleOwner)
}

func TestModuleOwnerForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module owner without a remote",
			Input: "/foo",
		},
		{
			Name:  "Module owner with a repository",
			Input: "foo.com/bar/baz",
		},
		{
			Name:  "Module owner with a branch",
			Input: "foo.com//bar:v1",
		},
		{
			Name:  "Module owner with invalid characters",
			Input: "foo@bar.com/baz",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := ModuleOwnerForString(testCase.Input)
			require.Error(t, err)
		})
	}
}
