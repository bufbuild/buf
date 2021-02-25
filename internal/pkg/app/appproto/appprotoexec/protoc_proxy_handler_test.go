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

package appprotoexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFeatureProto3OptionalForVersionString(t *testing.T) {
	t.Parallel()
	assert.True(t, getFeatureProto3OptionalForVersionString("3.11.1-buf"))
	assert.True(t, getFeatureProto3OptionalForVersionString("libprotoc 3.12.1"))
	assert.True(t, getFeatureProto3OptionalForVersionString("libprotoc 3.14.1"))
	assert.False(t, getFeatureProto3OptionalForVersionString("libprotoc 3.11.4"))
	assert.False(t, getFeatureProto3OptionalForVersionString("libprotoc 2.11.4"))
	assert.False(t, getFeatureProto3OptionalForVersionString("protoc 3.12.3"))
	assert.False(t, getFeatureProto3OptionalForVersionString("protoc 3.15.0"))
}
