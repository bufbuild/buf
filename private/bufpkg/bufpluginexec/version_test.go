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

package bufpluginexec

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

func TestGetSetExperimentalAllowProto3OptionalFlag(t *testing.T) {
	t.Parallel()
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(2, 12, 4, "")))
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 11, 1, "buf")))
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 11, 4, "")))
	assert.True(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 12, 1, "")))
	assert.True(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 14, 1, "")))
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 14, 1, "buf")))
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(3, 15, 0, "")))
	assert.False(t, getSetExperimentalAllowProto3OptionalFlag(newVersion(21, 0, 0, "")))
}

func TestGetFeatureProto3OptionalSupported(t *testing.T) {
	t.Parallel()
	assert.False(t, getFeatureProto3OptionalSupported(newVersion(2, 12, 4, "")))
	assert.False(t, getFeatureProto3OptionalSupported(newVersion(3, 11, 4, "")))
	assert.True(t, getFeatureProto3OptionalSupported(newVersion(3, 11, 1, "buf")))
	assert.True(t, getFeatureProto3OptionalSupported(newVersion(3, 12, 1, "")))
	assert.True(t, getFeatureProto3OptionalSupported(newVersion(3, 14, 1, "")))
	assert.True(t, getFeatureProto3OptionalSupported(newVersion(3, 15, 0, "")))
	assert.True(t, getFeatureProto3OptionalSupported(newVersion(21, 0, 0, "")))
}

func TestGetKotlinSupportedAsBuiltin(t *testing.T) {
	t.Parallel()
	assert.True(t, getKotlinSupportedAsBuiltin(newVersion(3, 11, 1, "buf")))
	assert.True(t, getKotlinSupportedAsBuiltin(newVersion(3, 17, 4, "")))
	assert.True(t, getKotlinSupportedAsBuiltin(newVersion(21, 1, 0, "")))
	assert.True(t, getKotlinSupportedAsBuiltin(newVersion(21, 1, 0, "buf")))
	assert.False(t, getKotlinSupportedAsBuiltin(newVersion(3, 12, 1, "")))
	assert.False(t, getKotlinSupportedAsBuiltin(newVersion(3, 14, 1, "")))
}

func TestGetJSSupportedAsBuiltin(t *testing.T) {
	t.Parallel()
	assert.False(t, getJSSupportedAsBuiltin(newVersion(2, 11, 1, "")))
	assert.True(t, getJSSupportedAsBuiltin(newVersion(3, 11, 1, "buf")))
	assert.True(t, getJSSupportedAsBuiltin(newVersion(3, 17, 4, "")))
	assert.True(t, getJSSupportedAsBuiltin(newVersion(3, 20, 1, "")))
	assert.False(t, getJSSupportedAsBuiltin(newVersion(3, 21, 1, "")))
	assert.False(t, getJSSupportedAsBuiltin(newVersion(3, 22, 1, "")))
	assert.False(t, getJSSupportedAsBuiltin(newVersion(21, 1, 0, "")))
	assert.True(t, getJSSupportedAsBuiltin(newVersion(21, 1, 0, "buf")))
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
