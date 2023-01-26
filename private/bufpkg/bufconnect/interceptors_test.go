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

type testMachine struct{}

func (testMachine) Name() string {
	return "name"
}

func (testMachine) Login() string {
	return "login"
}

func (testMachine) Password() string {
	return "password"
}

func TestNewAuthorizationInterceptorProvider(t *testing.T) {
	tokenSet, err := NewTokenProviderFromString("token1@host1,token2@host2")
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("host1")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token1" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	getMachineForName := func(app.EnvContainer, string) (netrc.Machine, error) {
		return testMachine{}, nil
	}
	netrcTokens := &netrcTokenProvider{getMachineForName: getMachineForName}
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(netrcTokens)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"password" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// testing using tokenSet over netrc tokenToAuthKey
	_, err = NewAuthorizationInterceptorProvider(tokenSet, netrcTokens)("host2")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token2" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// testing using netrc tokenToAuthKey over tokenSet
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

	tokenSet, err = NewTokenProviderFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default",
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
	tokenSet, err := NewTokenProviderFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default",
	}))
	assert.NoError(t, err)
	token := tokenSet.RemoteToken("fake")
	assert.True(t, tokenSet.IsFromEnvVar())
	assert.Equal(t, "default", token)
}

func TestNewTokenSetFromString(t *testing.T) {
	_, err := NewTokenProviderFromString("default")
	assert.NoError(t, err)
	_, err = NewTokenProviderFromString("token1@host1")
	assert.NoError(t, err)
	_, err = NewTokenProviderFromString("user1@remote1,user2@remote2")
	assert.NoError(t, err)
	_, err = NewTokenProviderFromString("user1@remote1,user2@remote1")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("user1@remote1,user2@remote2,")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString(",token1@host1")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("token1@host1,")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("token1@")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("token1@host1@")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("@token1")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("token1@host1,token2")
	assert.Error(t, err)
	_, err = NewTokenProviderFromString("")
	assert.NoError(t, err)
}
