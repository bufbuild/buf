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
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/applog"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Parallel()
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

func TestCLIWarningInterceptor(t *testing.T) {
	warningMessage := "This is a warning message from the BSR"
	var buf bytes.Buffer
	logger, err := applog.NewLogger(&buf, "warn", "text")
	require.NoError(t, err)
	// testing valid warning message
	_, err = NewCLIWarningInterceptor(applog.NewContainer(logger))(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp := connect.NewResponse(&bytes.Buffer{})
		resp.Header().Set(CLIWarningHeaderName, base64.StdEncoding.EncodeToString([]byte(warningMessage)))
		return resp, nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("WARN\t%s\n", warningMessage), buf.String())

	// testing no warning message in valid response with no header
	buf.Reset()
	_, err = NewCLIWarningInterceptor(applog.NewContainer(logger))(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&bytes.Buffer{}), nil
	})(context.Background(), connect.NewRequest(&bytes.Buffer{}))
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
}
