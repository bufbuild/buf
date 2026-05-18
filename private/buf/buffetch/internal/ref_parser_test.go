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

package internal

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRawPathAndOptionsError(t *testing.T) {
	t.Parallel()
	testGetRawPathAndOptionsError(
		t,
		newValueEmptyError(),
		"",
	)
	testGetRawPathAndOptionsError(
		t,
		newValueMultipleHashtagsError("foo#format=git#branch=main"),
		"foo#format=git#branch=main",
	)
	testGetRawPathAndOptionsError(
		t,
		newValueStartsWithHashtagError("#path/to/dir"),
		"#path/to/dir",
	)
	testGetRawPathAndOptionsError(
		t,
		newValueEndsWithHashtagError("path/to/dir#"),
		"path/to/dir#",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsDuplicateKeyError("branch"),
		"path/to/foo#format=git,branch=foo,branch=bar",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsInvalidError("bar"),
		"path/to/foo#bar",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsInvalidError("bar="),
		"path/to/foo#bar=",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsInvalidError("format=bin,bar="),
		"path/to/foo#format=bin,bar=",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsInvalidError("format=bin,=bar"),
		"path/to/foo#format=bin,=bar",
	)
	testGetRawPathAndOptionsError(
		t,
		newOptionsDuplicateKeyError("strip_components"),
		"path/to/foo.tar#strip_components=0,strip_components=1",
	)
}

func testGetRawPathAndOptionsError(
	t *testing.T,
	expectedErr error,
	value string,
) {
	t.Run(value, func(t *testing.T) {
		t.Parallel()
		_, _, err := getRawPathAndOptions(value)
		assert.EqualError(t, err, expectedErr.Error())
	})
}

func TestRefParserGitMergeBaseValidation(t *testing.T) {
	t.Parallel()
	parser := newRefParser(slogtestext.NewLogger(t), WithGitFormat("git"))
	ctx := context.Background()

	t.Run("valid_merge_base", func(t *testing.T) {
		t.Parallel()
		parsedRef, err := parser.getParsedRef(ctx, "path/to/repo#format=git,merge_base=main", nil)
		require.NoError(t, err)
		gitRef, ok := parsedRef.(GitRef)
		require.True(t, ok, "expected GitRef")
		assert.Equal(t, "main", gitRef.GitMergeBase())
		assert.Equal(t, uint32(50), gitRef.Depth())
		assert.Nil(t, gitRef.GitName())
	})

	t.Run("merge_base_with_branch", func(t *testing.T) {
		t.Parallel()
		_, err := parser.getParsedRef(ctx, "path/to/repo#format=git,merge_base=main,branch=feature", nil)
		assert.EqualError(t, err, NewCannotSpecifyMergeBaseWithOtherGitOptionsError().Error())
	})

	t.Run("merge_base_with_ref", func(t *testing.T) {
		t.Parallel()
		_, err := parser.getParsedRef(ctx, "path/to/repo#format=git,merge_base=main,ref=abc123", nil)
		assert.EqualError(t, err, NewCannotSpecifyMergeBaseWithOtherGitOptionsError().Error())
	})

	t.Run("merge_base_with_commit", func(t *testing.T) {
		t.Parallel()
		_, err := parser.getParsedRef(ctx, "path/to/repo#format=git,merge_base=main,commit=abc123", nil)
		assert.EqualError(t, err, NewCannotSpecifyMergeBaseWithOtherGitOptionsError().Error())
	})
}
