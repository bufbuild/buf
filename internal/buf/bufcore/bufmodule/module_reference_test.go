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
	"github.com/bufbuild/buf/internal/pkg/uuidutil"
	"github.com/stretchr/testify/require"
)

func TestModuleReferenceForString(t *testing.T) {
	t.Parallel()
	expectedModuleReference, err := NewBranchModuleReference("foo.com", "bar", "baz", "main")
	require.NoError(t, err)
	require.Equal(t, "foo.com/bar/baz:main", expectedModuleReference.String())
	moduleReference, err := ModuleReferenceForString("foo.com/bar/baz")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	branchModuleReference, err := BranchModuleReferenceForString("foo.com/bar/baz")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, branchModuleReference)

	expectedModuleReference, err = NewBranchModuleReference("foo.com", "bar", "baz", "v1")
	require.NoError(t, err)
	require.Equal(t, "foo.com/bar/baz:v1", expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("foo.com/bar/baz:v1")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	branchModuleReference, err = BranchModuleReferenceForString("foo.com/bar/baz:v1")
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, branchModuleReference)

	commitUUID, err := uuidutil.New()
	require.NoError(t, err)
	commit, err := uuidutil.ToDashless(commitUUID)
	require.NoError(t, err)
	expectedModuleReference, err = NewCommitModuleReference("foo.com", "bar", "baz", commit)
	require.NoError(t, err)
	require.Equal(t, "foo.com/bar/baz:"+commit, expectedModuleReference.String())
	moduleReference, err = ModuleReferenceForString("foo.com/bar/baz:" + commit)
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, moduleReference)
	commitModuleReference, err := CommitModuleReferenceForString("foo.com/bar/baz:" + commit)
	require.NoError(t, err)
	require.Equal(t, expectedModuleReference, commitModuleReference)
}

func TestModuleReferenceForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module without a remote",
			Input: "/bar/baz:v1",
		},
		{
			Name:  "Module without an owner",
			Input: "foo.com//baz:v1",
		},
		{
			Name:  "Module without a repository",
			Input: "foo.com/bar/:v1",
		},
		{
			Name:  "Module without a branch or commit",
			Input: "foo.com/bar/baz:",
		},
		{
			Name:  "Module with invalid characters",
			Input: "foo.com/bar/baz@qux:v1",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := ModuleReferenceForString(testCase.Input)
			require.Error(t, err)
			_, err = BranchModuleReferenceForString(testCase.Input)
			require.Error(t, err)
			_, err = CommitModuleReferenceForString(testCase.Input)
			require.Error(t, err)
		})
	}
}

func TestModuleReferenceForStringRequireBranchError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module without a branch",
			Input: "foo.com/bar/baz",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := ModuleReferenceForString(testCase.Input, ModuleReferenceForStringRequireBranch())
			require.Error(t, err)
			_, err = BranchModuleReferenceForString(testCase.Input, BranchModuleReferenceForStringRequireBranch())
			require.Error(t, err)
		})
	}
}

func TestBranchModuleReferenceForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module with a commit",
			Input: "foo.com/bar/baz:" + bufmoduletesting.TestCommit,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := BranchModuleReferenceForString(testCase.Input)
			require.Error(t, err)
		})
	}
}

func TestCommitModuleReferenceForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Module without a commit",
			Input: "foo.com/bar/baz",
		},
		{
			Name:  "Module with a branch",
			Input: "foo.com/bar/baz:v1",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := CommitModuleReferenceForString(testCase.Input)
			require.Error(t, err)
		})
	}
}
