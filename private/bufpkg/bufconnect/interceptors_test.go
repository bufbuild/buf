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

func TestNewAuthorizationInterceptorProvider(t *testing.T) {
	tokenSet, err := NewTokenSetFromString("user1:token1@remote1,token")
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)
}

func TestNewTokenSetFromEnv(t *testing.T) {
	_, err := NewTokenSetFromContainer(app.NewEnvContainer(map[string]string{}))
	assert.NoError(t, err)
}

func TestNewTokenSetFromString(t *testing.T) {
	_, err := NewTokenSetFromString("default1,default2")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("invalid@invalid@invalid")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("invalid:invalid:invalid@remote")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("user1:token1@remote,user2:token2@remote")
	assert.Error(t, err)
}
