// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufreflect_test

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"github.com/stretchr/testify/assert"
)

func TestParseFullyQualifiedModuleTypeName(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		moduleName, typeName, err := bufreflect.ParseFullyQualifiedPath("buf.test/testuser/testrepo/buf.v1.Foo")
		assert.NoError(t, err)
		assert.Equal(t, "buf.test/testuser/testrepo", moduleName)
		assert.Equal(t, "buf.v1.Foo", typeName)
	})
	t.Run("fail with module name", func(t *testing.T) {
		_, _, err := bufreflect.ParseFullyQualifiedPath("buf.test/testuser/testrepo")
		assert.EqualError(t, err, `"buf.test/testuser/testrepo" is not a valid fully qualified path`)
	})
	t.Run("fail with type name", func(t *testing.T) {
		_, _, err := bufreflect.ParseFullyQualifiedPath("buf.v1.Foo")
		assert.EqualError(t, err, `"buf.v1.Foo" is not a valid fully qualified path`)
	})
}
