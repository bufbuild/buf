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

package appprotoexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestVersionString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.11.1-buf", versionString(newVersion(3, 11, 1, "buf")))
	assert.Equal(t, "3.15.0", versionString(newVersion(3, 15, 0, "")))
	assert.Equal(t, "21.0", versionString(newVersion(21, 0, 0, "")))
	assert.Equal(t, "21.1", versionString(newVersion(21, 1, 0, "")))
	assert.Equal(t, "21.1-rc-1", versionString(newVersion(21, 1, 0, "rc-1")))
	assert.Equal(t, "21.1.1", versionString(newVersion(21, 1, 1, "")))
	assert.Equal(t, "21.1.1-rc-1", versionString(newVersion(21, 1, 1, "rc-1")))
}

func TestGetExperimentalAllowProto3Optional(t *testing.T) {
	t.Parallel()
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(2, 12, 4, "")))
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(3, 11, 1, "buf")))
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(3, 11, 4, "")))
	assert.True(t, getExperimentalAllowProto3Optional(newVersion(3, 12, 1, "")))
	assert.True(t, getExperimentalAllowProto3Optional(newVersion(3, 14, 1, "")))
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(3, 14, 1, "buf")))
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(3, 15, 0, "")))
	assert.False(t, getExperimentalAllowProto3Optional(newVersion(21, 0, 0, "")))
}

func TestGetFeatureProto3Optional(t *testing.T) {
	t.Parallel()
	assert.False(t, getFeatureProto3Optional(newVersion(2, 12, 4, "")))
	assert.False(t, getFeatureProto3Optional(newVersion(3, 11, 4, "")))
	assert.True(t, getFeatureProto3Optional(newVersion(3, 11, 1, "buf")))
	assert.True(t, getFeatureProto3Optional(newVersion(3, 12, 1, "")))
	assert.True(t, getFeatureProto3Optional(newVersion(3, 14, 1, "")))
	assert.True(t, getFeatureProto3Optional(newVersion(3, 15, 0, "")))
	assert.True(t, getFeatureProto3Optional(newVersion(21, 0, 0, "")))
}

func TestGetKotlinSupported(t *testing.T) {
	t.Parallel()
	assert.True(t, getKotlinSupported(newVersion(3, 11, 1, "buf")))
	assert.True(t, getKotlinSupported(newVersion(3, 17, 4, "")))
	assert.True(t, getKotlinSupported(newVersion(21, 1, 0, "")))
	assert.True(t, getKotlinSupported(newVersion(21, 1, 0, "buf")))
	assert.False(t, getKotlinSupported(newVersion(3, 12, 1, "")))
	assert.False(t, getKotlinSupported(newVersion(3, 14, 1, "")))
}

func TestParseVersionForCLIVersion(t *testing.T) {
	t.Parallel()
	testParseVersionForCLIVersionSuccess(t, "libprotoc 3.14.0", newVersion(3, 14, 0, ""))
	testParseVersionForCLIVersionSuccess(t, "libprotoc 3.14.0-rc1", newVersion(3, 14, 0, "rc1"))
	testParseVersionForCLIVersionSuccess(t, "libprotoc 3.14.0-rc-1", newVersion(3, 14, 0, "rc-1"))
	testParseVersionForCLIVersionSuccess(t, "3.14.0", newVersion(3, 14, 0, ""))
	testParseVersionForCLIVersionSuccess(t, "3.14.0-rc1", newVersion(3, 14, 0, "rc1"))
	testParseVersionForCLIVersionSuccess(t, "3.14.0-buf", newVersion(3, 14, 0, "buf"))
	testParseVersionForCLIVersionSuccess(t, "libprotoc 21.1", newVersion(21, 1, 0, ""))
	testParseVersionForCLIVersionSuccess(t, "libprotoc 21.1-rc1", newVersion(21, 1, 0, "rc1"))
	testParseVersionForCLIVersionSuccess(t, "libprotoc 21.1-rc-1", newVersion(21, 1, 0, "rc-1"))
	testParseVersionForCLIVersionSuccess(t, "21.1", newVersion(21, 1, 0, ""))
	testParseVersionForCLIVersionSuccess(t, "21.1-rc1", newVersion(21, 1, 0, "rc1"))
	testParseVersionForCLIVersionSuccess(t, "21.1-rc-1", newVersion(21, 1, 0, "rc-1"))
	testParseVersionForCLIVersionSuccess(t, "21.1-buf", newVersion(21, 1, 0, "buf"))
	testParseVersionForCLIVersionError(t, "libprotoc3.14.0")
	testParseVersionForCLIVersionError(t, "libprotoc 3.14.0.1")
}

func testParseVersionForCLIVersionSuccess(
	t *testing.T,
	value string,
	expectedVersion *pluginpb.Version,
) {
	version, err := parseVersionForCLIVersion(value)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, expectedVersion, version)
	}
}

func testParseVersionForCLIVersionError(
	t *testing.T,
	value string,
) {
	_, err := parseVersionForCLIVersion(value)
	assert.Error(t, err)
}
