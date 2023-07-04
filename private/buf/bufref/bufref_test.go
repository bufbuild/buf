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

package bufref

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		_, _, err := GetRawPathAndOptions(value)
		assert.EqualError(t, err, expectedErr.Error())
	})
}
