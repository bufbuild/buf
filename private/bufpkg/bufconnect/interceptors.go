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
func NewAuthorizationInterceptorProvider(tokenSet *TokenSet) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				if tokenSet != nil {
					req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+tokenSet.obtainTokenFromRemoteAddress(address))
				}
				response, err := next(ctx, req)
				if err != nil {
					err = &ErrAuth{cause: err, tokenEnvKey: tokenEnvKey}
				}
				return response, err
			})
		}
		return interceptor
	}
}

// TokenSet is used to provide authentication token in NewAuthorizationInterceptorProvider
type TokenSet struct {
	bufToken        string
	remoteUsernames map[string]string
	remoteTokens    map[string]string
}

// NewTokenSetFromContainer creates a TokenSet from the BUF_TOKEN environment variable
func NewTokenSetFromContainer(container app.EnvContainer) (*TokenSet, error) {
	bufToken := container.Env(tokenEnvKey)
	return NewTokenSetFromString(bufToken)
}

// NewTokenSetFromString creates a TokenSet by the token provided
func NewTokenSetFromString(token string) (*TokenSet, error) {
	tokenSet := &TokenSet{
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
				return nil, fmt.Errorf("cannot parse token: %s, two buf token provided: %q and %q", token, u, tokenSet.bufToken)
			}
			tokenSet.bufToken = u
		}
	}
	return tokenSet, nil
}

func (t *TokenSet) obtainTokenFromRemoteAddress(remoteAddress string) (token string) {
	if remoteToken, ok := t.remoteTokens[remoteAddress]; ok {
		return remoteToken
	}
	return t.bufToken
}
