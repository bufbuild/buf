// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/stretchr/testify/require"
)

func TestModuleNameForString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name     string
		Input    string
		Expected bufmodule.ModuleName
	}{
		{
			Name:     "Module without digest",
			Input:    "foo.com/bar/baz/v1",
			Expected: newModuleName(t, "foo.com", "bar", "baz", "v1", ""),
		},
		{
			Name:     "Module with digest",
			Input:    "foo.com/bar/baz/v1:" + bufmoduletesting.TestDigest,
			Expected: newModuleName(t, "foo.com", "bar", "baz", "v1", bufmoduletesting.TestDigest),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			output, err := bufmodule.ModuleNameForString(testCase.Input)
			require.NoError(t, err)
			require.Equal(t, testCase.Expected, output)
		})
	}
}

func TestModuleNameForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                string
		Input               string
		ExpectedErrorString string
	}{
		{
			Name:                "Module without a remote",
			Input:               "/bar/baz/v1",
			ExpectedErrorString: "invalid module name: remote name is empty: /bar/baz/v1",
		},
		{
			Name:                "Module without an owner",
			Input:               "foo.com//baz/v1",
			ExpectedErrorString: "invalid module name: owner name is empty: foo.com//baz/v1",
		},
		{
			Name:                "Module without a repository",
			Input:               "foo.com/bar//v1",
			ExpectedErrorString: "invalid module name: repository name is empty: foo.com/bar//v1",
		},
		{
			Name:                "Module without a version",
			Input:               "foo.com/bar/baz/",
			ExpectedErrorString: "invalid module name: version name is empty: foo.com/bar/baz/",
		},
		{
			Name:                "Invalid module structure",
			Input:               "foo.com/bar/baz",
			ExpectedErrorString: "invalid module name: module name is not in the form remote/owner/repository/version: foo.com/bar/baz",
		},
		{
			Name:                "Invalid digest structure",
			Input:               "foo.com/bar/baz/v1:digest:digest",
			ExpectedErrorString: "invalid module name: invalid version with digest: foo.com/bar/baz/v1:digest:digest",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := bufmodule.ModuleNameForString(testCase.Input)
			require.EqualError(t, err, testCase.ExpectedErrorString)
		})
	}
}

func newModuleName(
	t *testing.T,
	remote string,
	owner string,
	repository string,
	version string,
	digest string,
) bufmodule.ModuleName {
	t.Helper()
	moduleName, err := bufmodule.NewModuleName(remote, owner, repository, version, digest)
	require.NoError(t, err)
	return moduleName
}
