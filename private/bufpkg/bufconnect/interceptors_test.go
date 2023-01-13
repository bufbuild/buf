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

package bufconnect

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizationInterceptorProviderTokenErr(t *testing.T) {
	_, err := NewAuthorizationInterceptorProvider(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "test1234",
	}))("fake")(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("underlying cause")
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))

	authErr, ok := AsAuthError(err)
	assert.True(t, ok)
	assert.Equal(t, tokenEnvKey, authErr.tokenEnvKey)
}
