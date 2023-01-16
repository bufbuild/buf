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
	"github.com/bufbuild/buf/private/pkg/netrc"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizationInterceptorProviderTokenErr(t *testing.T) {
	// test setting auth token with provided token
	_, err := NewAuthorizationInterceptorProvider(AuthorizeWithProvidedToken("test1234"))("fake")(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("underlying cause")
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.Error(t, err)

	// test on using remote token
	container := app.NewEnvContainer(map[string]string{
		tokenEnvKey: "username:token@remote,buftoken",
	})
	_, err = NewAuthorizationInterceptorProvider(AuthorizeWithAddress(container))("remote")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"token" {
			return nil, errors.New("error auth token found")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// test on using default token
	container = app.NewEnvContainer(map[string]string{
		tokenEnvKey: "username:token@remote,buftoken",
	})
	_, err = NewAuthorizationInterceptorProvider(AuthorizeWithAddress(container))("fake")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"buftoken" {
			return nil, errors.New("error auth token found")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)

	// test on zero value
	getMachineForName = newGetMachineForName
	container = app.NewEnvContainer(map[string]string{})
	_, err = NewAuthorizationInterceptorProvider(AuthorizeWithAddress(container))("fake")(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Header().Get(AuthenticationHeader) != AuthenticationTokenPrefix+"password" {
			return nil, errors.New("error auth token found")
		}
		return nil, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	getMachineForName = netrc.GetMachineForName
}

// machine is implementation of netrc.Machine for testing
type machine struct{}

func (m machine) Name() string {
	return "name"
}

func (m machine) Login() string {
	return "login"
}

func (m machine) Password() string {
	return "password"
}

func newGetMachineForName(container app.EnvContainer, name string) (netrc.Machine, error) {
	return machine{}, nil
}
