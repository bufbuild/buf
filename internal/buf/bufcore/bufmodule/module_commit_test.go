// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmodule

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModuleCommit(t *testing.T) {
	t.Parallel()
	moduleCommit, err := NewModuleCommit("foo.com", "acme", "weather", bufmoduletesting.TestCommit)
	require.NoError(t, err)
	assert.Equal(t, "foo.com/acme/weather:"+bufmoduletesting.TestCommit, moduleCommit.String())
}

func TestNewModuleCommitError(t *testing.T) {
	testCases := []struct {
		Name       string
		Remote     string
		Owner      string
		Repository string
		Commit     string
	}{
		{
			Name:       "Module commit with an invalid remote",
			Remote:     "x.-",
			Owner:      "acme",
			Repository: "weather",
			Commit:     bufmoduletesting.TestCommit,
		},
		{
			Name:       "Module commit with an invalid owner",
			Remote:     "foo.com",
			Owner:      "",
			Repository: "weather",
			Commit:     bufmoduletesting.TestCommit,
		},
		{
			Name:       "Module commit with an invalid repository",
			Remote:     "foo.com",
			Owner:      "acme",
			Repository: "",
			Commit:     bufmoduletesting.TestCommit,
		},
		{
			Name:       "Module commit with an invalid commit",
			Remote:     "foo.com",
			Owner:      "acme",
			Repository: "weather",
			Commit:     "",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := NewModuleCommit(
				testCase.Remote,
				testCase.Owner,
				testCase.Repository,
				testCase.Commit,
			)
			require.Error(t, err)
		})
	}
}
