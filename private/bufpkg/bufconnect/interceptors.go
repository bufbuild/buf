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

package bufconnect

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/connect-go"
)

const (
	// tokenEnvKey is the environment variable key for the auth token
	tokenEnvKey = "BUF_TOKEN"
)

// NewSetCLIVersionInterceptor returns a new Connect Interceptor that sets the Buf CLI version into all request headers
func NewSetCLIVersionInterceptor(version string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			req.Header().Set(CliVersionHeaderName, version)
			return next(ctx, req)
		}
	}
	return interceptor
}

// NewAuthorizationInterceptorProvider returns a new provider function which, when invoked, returns an interceptor
// which will look up an auth token by address and set it into the request header.  This is used for registry providers
// where the token is looked up by the client address at the time of client construction (i.e. for clients where a
// user is already authenticated and the token is stored in .netrc)
//
// Note that the interceptor returned from this provider is always applied LAST in the series of interceptors added to
// a client.
func NewAuthorizationInterceptorProvider(container app.EnvContainer) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				envKey := tokenEnvKey
				token := container.Env(envKey)
				if token == "" {
					envKey = ""
					machine, err := netrc.GetMachineForName(container, address)
					if err != nil {
						return nil, fmt.Errorf("failed to read server password from netrc: %w", err)
					}
					if machine != nil {
						token = machine.Password()
					}
				}
				if token != "" {
					req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
				}
				response, err := next(ctx, req)
				if err != nil {
					err = &ErrAuth{cause: err, tokenEnvKey: envKey}
				}
				return response, err
			})
		}
		return interceptor
	}
}

// NewAuthorizationInterceptorProviderWithToken returns a new provider function which, when invoked, returns an
// interceptor which sets the provided auth token into the request header.  This is used for registry providers where
// the token is known at provider creation (i.e. when logging in and explicitly pasting a token into stdin
//
// Note that the interceptor returned from this provider is always applied LAST in the series of interceptors added to
// a client.
func NewAuthorizationInterceptorProviderWithToken(token string) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				if token != "" {
					req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
				}
				return next(ctx, req)
			})
		}
		return interceptor
	}
}
