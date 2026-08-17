// Copyright 2020-2026 Buf Technologies, Inc.
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
	"io"
	"testing"

	"buf.build/go/app"
	"buf.build/go/app/appext"
	"connectrpc.com/connect/v2"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/slogapp"
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

// nopClientStream is a no-op connect.ClientStream for testing interceptors.
type nopClientStream struct{}

func (nopClientStream) SendHeaders() error { return nil }
func (nopClientStream) Send(any) error     { return nil }
func (nopClientStream) CloseSend() error   { return nil }
func (nopClientStream) Receive(any) error  { return io.EOF }
func (nopClientStream) Close() error       { return nil }

// checkAuthHeaderClientFunc returns a connect.ClientFunc that errors if the
// authentication header on the client CallInfo does not match want.
func checkAuthHeaderClientFunc(info *connect.CallInfo, want string) connect.ClientFunc {
	return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
		if info.RequestHeader().Get(AuthenticationHeader) != want {
			return nil, errors.New("error auth token")
		}
		return nopClientStream{}, nil
	}
}

func TestNewAuthorizationInterceptorProvider(t *testing.T) {
	t.Parallel()
	tokenSet, err := NewTokenProviderFromString("token1@host1,token2@host2")
	assert.NoError(t, err)
	ctx, info := connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("host1")(
		checkAuthHeaderClientFunc(info, AuthenticationTokenPrefix+"token1"),
	)(ctx, connect.Spec{})
	assert.NoError(t, err)

	getMachineForName := func(app.EnvContainer, string) (netrc.Machine, error) {
		return testMachine{}, nil
	}
	netrcTokens := &netrcTokenProvider{getMachineForName: getMachineForName}
	assert.NoError(t, err)
	ctx, info = connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider(netrcTokens)("default")(
		checkAuthHeaderClientFunc(info, AuthenticationTokenPrefix+"password"),
	)(ctx, connect.Spec{})
	assert.NoError(t, err)

	// testing using tokenSet over netrc tokenToAuthKey
	ctx, info = connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider(tokenSet, netrcTokens)("host2")(
		checkAuthHeaderClientFunc(info, AuthenticationTokenPrefix+"token2"),
	)(ctx, connect.Spec{})
	assert.NoError(t, err)

	// testing using netrc tokenToAuthKey over tokenSet
	ctx, info = connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider(netrcTokens, tokenSet)("default")(
		checkAuthHeaderClientFunc(info, AuthenticationTokenPrefix+"password"),
	)(ctx, connect.Spec{})
	assert.NoError(t, err)

	ctx, info = connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider()("default")(
		checkAuthHeaderClientFunc(info, ""),
	)(ctx, connect.Spec{})
	assert.NoError(t, err)

	tokenSet, err = NewTokenProviderFromContainer(app.NewEnvContainer(map[string]string{
		TokenEnvKey: "default",
	}))
	assert.NoError(t, err)
	ctx, _ = connect.NewClientContext(t.Context())
	_, err = NewAuthorizationInterceptorProvider(tokenSet)("default")(
		func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			return nil, errors.New("underlying cause")
		},
	)(ctx, connect.Spec{})
	authErr, ok := AsAuthError(err)
	assert.True(t, ok)
	assert.Equal(t, TokenEnvKey, authErr.tokenEnvKey)
}

func TestCLIWarningInterceptor(t *testing.T) {
	t.Parallel()
	warningMessage := "This is a warning message from the BSR"
	var buf bytes.Buffer
	logger, err := slogapp.NewLogger(&buf, appext.LogLevelWarn, appext.LogFormatText)
	require.NoError(t, err)
	// testing valid warning message
	ctx, info := connect.NewClientContext(t.Context())
	stream, err := NewCLIWarningInterceptor(appext.NewLoggerContainer(logger))(
		func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info.ResponseHeader().Set(CLIWarningHeaderName, base64.StdEncoding.EncodeToString([]byte(warningMessage)))
			return nopClientStream{}, nil
		},
	)(ctx, connect.Spec{})
	require.NoError(t, err)
	require.NoError(t, stream.Close())
	assert.Equal(t, fmt.Sprintf("WARN\t%s\n", warningMessage), buf.String())

	// testing no warning message in valid response with no header
	buf.Reset()
	ctx, _ = connect.NewClientContext(t.Context())
	stream, err = NewCLIWarningInterceptor(appext.NewLoggerContainer(logger))(
		func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			return nopClientStream{}, nil
		},
	)(ctx, connect.Spec{})
	require.NoError(t, err)
	require.NoError(t, stream.Close())
	assert.Equal(t, "", buf.String())
}

func TestCLIWarningInterceptorFromError(t *testing.T) {
	t.Parallel()
	warningMessage := "This is a warning message from the BSR"
	var buf bytes.Buffer
	logger, err := slogapp.NewLogger(&buf, appext.LogLevelWarn, appext.LogFormatText)
	require.NoError(t, err)
	// testing valid warning message from an errored call whose response
	// headers carry the warning
	ctx, info := connect.NewClientContext(t.Context())
	_, err = NewCLIWarningInterceptor(appext.NewLoggerContainer(logger))(
		func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			info.ResponseHeader().Set(CLIWarningHeaderName, base64.StdEncoding.EncodeToString([]byte(warningMessage)))
			return nil, connect.NewError(connect.CodeInternal, "error")
		},
	)(ctx, connect.Spec{})
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("WARN\t%s\n", warningMessage), buf.String())
}

func TestNewAugmentedConnectErrorInterceptor(t *testing.T) {
	t.Parallel()
	ctx, info := connect.NewClientContext(t.Context())
	info.PeerAddr = "example.com"
	_, err := NewAugmentedConnectErrorInterceptor()(
		func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			return nil, connect.NewError(connect.CodeUnknown, "405 Method Not Allowed")
		},
	)(ctx, connect.Spec{Procedure: "/service/method"})
	assert.Error(t, err)
	var augmentedConnectError *AugmentedConnectError
	assert.ErrorAs(t, err, &augmentedConnectError)
	assert.Equal(t, "example.com", augmentedConnectError.Addr())
	assert.Equal(t, "/service/method", augmentedConnectError.Procedure())
	var unwrappedError *connect.Error
	assert.ErrorAs(t, errors.Unwrap(err), &unwrappedError)
}
