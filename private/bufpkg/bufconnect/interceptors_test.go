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
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
)

type testMachineImpl struct{}

func (testMachineImpl) Name() string {
	return "name"
}

func (testMachineImpl) Login() string {
	return "login"
}

func (testMachineImpl) Password() string {
	return "password"
}

func TestNewAuthorizationInterceptorProvider(t *testing.T) {
	tokenSet, err := NewTokenSetProviderFromString("user1:token1@remote1,token")
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	getMachineForName := func(app.EnvContainer, string) (netrc.Machine, error) {
		return testMachineImpl{}, nil
	}
	netrcTokens := &NetrcTokensProvider{getMachineForName: getMachineForName}
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(netrcTokens)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"password" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// testing using tokenSet over netrc tokens
	_, err = NewAuthorizationInterceptorProvider(tokenSet, netrcTokens)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// testing using netrc tokens over tokenSet
	_, err = NewAuthorizationInterceptorProvider(netrcTokens, tokenSet)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"password" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	_, err = NewAuthorizationInterceptorProvider()("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != "" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	tokenSet, err = NewTokenSetProviderFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default,user:token@remote",
	}))
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("underlying cause")
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	authErr, ok := AsAuthError(err)
	assert.True(t, ok)
	assert.Equal(t, tokenEnvKey, authErr.tokenEnvKey)
}

func TestNewTokenSetFromEnv(t *testing.T) {
	tokenSet, err := NewTokenSetProviderFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default,user:token@remote",
	}))
	assert.NoError(t, err)
	token, setFromEnvVar := tokenSet.RemoteToken("fake")
	assert.Equal(t, "default", token)
	assert.True(t, setFromEnvVar)
	token, setFromEnvVar = tokenSet.RemoteToken("remote")
	assert.Equal(t, "token", token)
	assert.True(t, setFromEnvVar)
}

func TestNewTokenSetFromString(t *testing.T) {
	_, err := NewTokenSetProviderFromString("default")
	assert.NoError(t, err)
	_, err = NewTokenSetProviderFromString("user1:token1@remote,user2:token2@remote")
	assert.Error(t, err)
	_, err = NewTokenSetProviderFromString("default1,default2")
	assert.Error(t, err)
	_, err = NewTokenSetProviderFromString("invalid@invalid@invalid")
	assert.Error(t, err)
	_, err = NewTokenSetProviderFromString("")
	assert.NoError(t, err)
}
