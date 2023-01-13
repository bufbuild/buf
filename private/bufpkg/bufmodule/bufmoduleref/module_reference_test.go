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

	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/stretchr/testify/require"
)

func TestModuleReferenceForString(t *testing.T) {
	t.Parallel()
	expectedModuleReference, err := NewModuleReference("foo.com", "barr", "baz", "main")
	require.NoError(t, err)
	require.Equal(t, "foo.com/barr/baz", expectedModuleReference.String())
	moduleReference, err := ModuleReferenceForString("foo.com/barr/baz")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	branchModuleReference, err := ModuleReferenceForString("foo.com/barr/baz")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, branchModuleReference)
	require.False(t, IsCommitModuleReference(branchModuleReference))

	expectedModuleReference, err = NewModuleReference("foo.com", "barr", "baz", "v1")
	require.NoError(t, err)
	require.Equal(t, "foo.com/barr/baz:v1", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("foo.com/barr/baz:v1")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	branchModuleReference, err = ModuleReferenceForString("foo.com/barr/baz:v1")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, branchModuleReference)
	require.False(t, IsCommitModuleReference(branchModuleReference))

	commitUUID, err := uuidutil.New()
	require.NoError(t, err)
	commit, err := uuidutil.ToDashless(commitUUID)
	require.NoError(t, err)
	expectedModuleReference, err = NewModuleReference("foo.com", "barr", "baz", commit)
	require.NoError(t, err)
	require.Equal(t, "foo.com/barr/baz:"+commit, expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("foo.com/barr/baz:" + commit)
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	commitModuleReference, err := ModuleReferenceForString("foo.com/barr/baz:" + commit)
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, commitModuleReference)
	require.True(t, IsCommitModuleReference(commitModuleReference))

	expectedModuleReference, err = NewModuleReference("foo.com", "barr", "baz", "some/draft")
	require.NoError(t, err)
	require.Equal(t, "foo.com/barr/baz:some/draft", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("foo.com/barr/baz:some/draft")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	require.False(t, IsCommitModuleReference(moduleReference))

	expectedModuleReference, err = NewModuleReference("localhost:8080", "barr", "baz", "some/draft")
	require.NoError(t, err)
	require.Equal(t, "localhost:8080/barr/baz:some/draft", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("localhost:8080/barr/baz:some/draft")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	require.False(t, IsCommitModuleReference(moduleReference))

	expectedModuleReference, err = NewModuleReference("localhost:8080", "barr", "baz", "ref")
	require.NoError(t, err)
	require.Equal(t, "localhost:8080/barr/baz:ref", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("localhost:8080/barr/baz:ref")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	require.False(t, IsCommitModuleReference(moduleReference))

	expectedModuleReference, err = NewModuleReference("localhost:8080", "barr", "baz", "main")
	require.NoError(t, err)
	require.Equal(t, "localhost:8080/barr/baz", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("localhost:8080/barr/baz")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	require.False(t, IsCommitModuleReference(moduleReference))
}

func TestModuleReferenceForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module without a remote",
			Input: "/barr/baz:v1",
		},
		{
			Name:  "Module without an owner",
			Input: "foo.com//baz:v1",
		},
		{
			Name:  "Module without a repository",
			Input: "foo.com/barr/:v1",
		},
		{
			Name:  "Module without a branch or commit",
			Input: "foo.com/barr/baz:",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := ModuleReferenceForString(testCase.Input)
			require.Error(t, err)
		})
	}
}
