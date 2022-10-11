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

package bufcli_test

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/stretchr/testify/assert"
)

func TestParseInputAndType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("default", func(t *testing.T) {
		moduleReference, typeName, err := bufcli.ParseInputAndType(ctx, "", "buf.test/testuser/testrepo#buf.v1.Foo")
		assert.NoError(t, err)
		assert.Equal(t, "buf.test/testuser/testrepo", moduleReference)
		assert.Equal(t, "buf.v1.Foo", typeName)
	})
	t.Run("main track", func(t *testing.T) {
		moduleReference, typeName, err := bufcli.ParseInputAndType(ctx, "", "buf.test/testuser/testrepo:main#buf.v1.Foo")
		assert.NoError(t, err)
		assert.Equal(t, "buf.test/testuser/testrepo", moduleReference)
		assert.Equal(t, "buf.v1.Foo", typeName)
	})
	t.Run("dev track", func(t *testing.T) {
		moduleReference, typeName, err := bufcli.ParseInputAndType(ctx, "", "buf.test/testuser/testrepo:dev#buf.v1.Foo")
		assert.NoError(t, err)
		assert.Equal(t, "buf.test/testuser/testrepo:dev", moduleReference)
		assert.Equal(t, "buf.v1.Foo", typeName)
	})
	t.Run("fail with module name", func(t *testing.T) {
		_, _, err := bufcli.ParseInputAndType(ctx, "", "buf.test/testuser/testrepo")
		assert.EqualError(t, err, `if a input isn't provided, the type needs to be a fully qualified path that includes the module reference; failed to parse the type: "buf.test/testuser/testrepo" is not a valid fully qualified path`)
	})
	t.Run("fail with type name", func(t *testing.T) {
		_, _, err := bufcli.ParseInputAndType(ctx, "", "buf.v1.Foo")
		assert.EqualError(t, err, `if a input isn't provided, the type needs to be a fully qualified path that includes the module reference; failed to parse the type: "buf.v1.Foo" is not a valid fully qualified path`)
	})
}
