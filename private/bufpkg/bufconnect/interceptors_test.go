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

type machineImpl struct{}

func (machineImpl) Name() string {
	return ""
}

func (machineImpl) Login() string {
	return ""
}

func (machineImpl) Password() string {
	return "password"
}

type testMachineFinderImpl struct{}

func (t *testMachineFinderImpl) getMachineForName(name string) (netrc.Machine, error) {
	return machineImpl{}, nil
}

func TestNewAuthorizationInterceptorProvider(t *testing.T) {
	tokenSet, err := NewTokenSetFromString("user1:token1@remote1,token")
	assert.NoError(t, err)
	container := app.NewEnvContainer(map[string]string{})
	var machineFinder MachineFinder
	machineFinder = NewMachineFinder(container)
	_, err = NewAuthorizationInterceptorProvider(tokenSet, machineFinder)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	machineFinder = &testMachineFinderImpl{}
	tokenSet, err = NewTokenSetFromString("")
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet, machineFinder)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"password" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	_, err = NewAuthorizationInterceptorProvider(nil, nil)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != "" {
			return nil, errors.New("error auth token")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	tokenSet, err = NewTokenSetFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default,user:token@remote",
	}))
	assert.NoError(t, err)
	_, err = NewAuthorizationInterceptorProvider(tokenSet, machineFinder)("default")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("underlying cause")
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	authErr, ok := AsAuthError(err)
	assert.True(t, ok)
	assert.Equal(t, tokenEnvKey, authErr.tokenEnvKey)
}

func TestNewTokenSetFromEnv(t *testing.T) {
	container, err := NewTokenSetFromContainer(app.NewEnvContainer(map[string]string{
		tokenEnvKey: "default,user:token@remote",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "default", container.lookUpRemoteToken("fake"))
	assert.Equal(t, "token", container.lookUpRemoteToken("remote"))
	assert.Equal(t, true, container.setBufToken)
}

func TestNewTokenSetFromString(t *testing.T) {
	_, err := NewTokenSetFromString("default")
	assert.NoError(t, err)
	_, err = NewTokenSetFromString("user1:token1@remote,user2:token2@remote")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("default1,default2")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("invalid@invalid@invalid")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("invalid:invalid:invalid@remote")
	assert.Error(t, err)
	_, err = NewTokenSetFromString("")
	assert.NoError(t, err)
}
