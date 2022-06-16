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

	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/connect-go"
)

// NewWithTokenReaderInterceptor returns a new Connect Interceptor that looks up an auth token on every request and when
// found, sets it into the request header if not already set.
func NewWithTokenReaderInterceptor(container appflag.Container, address string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			token := container.Env(tokenEnvKey)
			if token == "" {
				machine, err := netrc.GetMachineForName(container, address)
				if err != nil {
					return nil, fmt.Errorf("failed to read server password from netrc: %w", err)
				}
				if machine != nil {
					token = machine.Password()
				}
			}

			if req.Header().Get(AuthenticationHeader) == "" {
				req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
			}
			return next(ctx, req)
		})
	}
	return interceptor
}

// NewWithVersionInterceptor returns a new Connect Interceptor that sets the Buf CLI version into all request headers
func NewWithVersionInterceptor(version string) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(
			ctx context.Context,
			req connect.AnyRequest,
		) (connect.AnyResponse, error) {
			req.Header().Set(CliVersionHeaderName, version)
			return next(ctx, req)
		})
	}
	return interceptor
}

// NewWithTokenInterceptor returns a new Connect Interceptor that sets the given token into all request headers
// This interceptor is useful for login requests where the user is explicitly providing a token (rather than expecting
// it to be read from netrc)
func NewWithTokenInterceptor(token string) connect.UnaryInterceptorFunc {
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
