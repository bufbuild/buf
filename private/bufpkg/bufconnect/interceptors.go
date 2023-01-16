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
	"context"
	"fmt"
	"strings"

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
// which will set the auth token into the request header by the provided option.
//
// Note that the interceptor returned from this provider is always applied LAST in the series of interceptors added to
// a client.
func NewAuthorizationInterceptorProvider(option SetAuthTokenOption) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				return option(req, ctx, next, address)
			})
		}
		return interceptor
	}
}

// SetAuthTokenOption is an option for NewAuthorizationInterceptorProvider
type SetAuthTokenOption func(connect.AnyRequest, context.Context, connect.UnaryFunc, string) (connect.AnyResponse, error)

// SetAuthTokenWithProvidedToken returns a new SetAuthTokenOption that will set the
// provided token into the request header This is used for registry providers where the token is known at provider
// creation (i.e. when logging in and explicitly pasting a token into stdin
func SetAuthTokenWithProvidedToken(token string) SetAuthTokenOption {
	return func(req connect.AnyRequest, ctx context.Context, next connect.UnaryFunc, address string) (connect.AnyResponse, error) {
		if token != "" {
			req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
		}
		return next(ctx, req)
	}
}

// SetAuthTokenWithAddress returns a new SetAuthTokenOption that will loop up an auth token
// and set it into the request header
func SetAuthTokenWithAddress(container app.EnvContainer) SetAuthTokenOption {
	return func(req connect.AnyRequest, ctx context.Context, next connect.UnaryFunc, address string) (connect.AnyResponse, error) {
		envKey := tokenEnvKey
		token := container.Env(envKey)
		authorizationToken := ""
		if token == "" {
			envKey = ""
			machine, err := netrc.GetMachineForName(container, address)
			if err != nil {
				return nil, fmt.Errorf("failed to read server password from netrc: %w", err)
			}
			if machine != nil {
				authorizationToken = machine.Password()
			}
		}
		if token != "" {
			tokenSet, err := newTokenSetFromString(token)
			if err != nil {
				return nil, err
			}
			_, authorizationToken = tokenSet.getRemoteUsernameAndToken(address)
		}
		if authorizationToken != "" {
			req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+authorizationToken)
		}
		response, err := next(ctx, req)
		if err != nil {
			err = &ErrAuth{cause: err, tokenEnvKey: envKey}
		}
		return response, err
	}
}

type tokenSet struct {
	bufToken        string
	remoteUsernames map[string]string
	remoteTokens    map[string]string
}

func newTokenSetFromString(token string) (*tokenSet, error) {
	tokenSet := &tokenSet{
		remoteUsernames: make(map[string]string),
		remoteTokens:    make(map[string]string),
	}
	tokens := strings.Split(token, ",")
	for _, u := range tokens {
		if contain := strings.ContainsAny(u, "@"); contain {
			keyPairsAndRemoteAddress := strings.Split(u, "@")
			if len(keyPairsAndRemoteAddress) != 2 {
				return nil, fmt.Errorf("cannot parse token: %s, invalid remote token: %s", token, u)
			}
			keyPairs := strings.Split(keyPairsAndRemoteAddress[0], ":")
			if len(keyPairs) != 2 {
				return nil, fmt.Errorf("cannot parse token: %s, invalid remote token: %s", token, u)
			}
			remoteAddress := keyPairsAndRemoteAddress[1]
			username := keyPairs[0]
			remoteToken := keyPairs[1]
			if _, ok := tokenSet.remoteTokens[remoteAddress]; ok {
				return nil, fmt.Errorf("cannot parse token: %s, repeated token for same BSR remote: %s", remoteToken, remoteAddress)
			}
			if _, ok := tokenSet.remoteUsernames[remoteAddress]; ok {
				return nil, fmt.Errorf("cannot parse token: %s, repeated token for same BSR remote: %s", remoteToken, remoteAddress)
			}
			tokenSet.remoteTokens[remoteAddress] = remoteToken
			tokenSet.remoteUsernames[remoteAddress] = username
		} else {
			if tokenSet.bufToken != "" {
				return nil, fmt.Errorf("cannot parse token: %s, two buf token provided: %s and %s", token, u, tokenSet.bufToken)
			}
			tokenSet.bufToken = u
		}
	}
	return tokenSet, nil
}

func (t *tokenSet) getRemoteUsernameAndToken(remoteAddress string) (user string, token string) {
	if remoteToken, ok := t.remoteTokens[remoteAddress]; ok {
		return t.remoteUsernames[remoteAddress], remoteToken
	}
	return "", t.bufToken
}
