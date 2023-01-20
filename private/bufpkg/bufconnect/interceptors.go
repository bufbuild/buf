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

// TokenFinder finds the token for NewAuthorizationInterceptorProvider
type TokenFinder interface {
	// RemoteToken returns the remote token from the remote address.
	// setFromEnvVar is true if the returned token is from the tokenEnvKey environment variable.
	RemoteToken(address string) (token string, setFromEnvVar bool)
}

// NetrcTokens finds remote tokens from netrc machine.
type NetrcTokens interface {
	TokenFinder

	GetMachineForName(address string) (netrc.Machine, error)
}

type netrcTokensImpl struct {
	container app.EnvContainer
}

// NewNetrcTokens returns a netrcTokensImpl
func NewNetrcTokens(container app.EnvContainer) *netrcTokensImpl {
	return &netrcTokensImpl{container: container}
}

func (nt *netrcTokensImpl) RemoteToken(address string) (string, bool) {
	machine, err := nt.GetMachineForName(address)
	if err != nil {
		return "", false
	}
	if machine != nil {
		return machine.Password(), false
	}
	return "", false
}

func (nt *netrcTokensImpl) GetMachineForName(address string) (netrc.Machine, error) {
	return netrc.GetMachineForName(nt.container, address)
}

// TokenSet is used to provide authentication token in NewAuthorizationInterceptorProvider
type TokenSet struct {
	// true: the tokenSet is generated from environment variable tokenEnvKey
	// false: otherwise
	setBufTokenEnvVar bool
	defaultToken      string
	tokens            map[string]authKey
}

var _ TokenFinder = (*TokenSet)(nil)

// NewTokenSetFromContainer creates a TokenSet from the BUF_TOKEN environment variable
func NewTokenSetFromContainer(container app.EnvContainer) (*TokenSet, error) {
	bufToken := container.Env(tokenEnvKey)
	tokenSet, err := NewTokenSetFromString(bufToken)
	if err != nil {
		return nil, err
	}
	if bufToken != "" {
		tokenSet.setBufTokenEnvVar = true
	}
	return tokenSet, nil
}

// NewTokenSetFromString creates a TokenSet by the token provided
func NewTokenSetFromString(token string) (*TokenSet, error) {
	tokenSet := &TokenSet{
		tokens: make(map[string]authKey),
	}
	tokens := strings.Split(token, ",")
	for _, u := range tokens {
		if keyPairs, remoteAddress, ok := strings.Cut(u, "@"); ok {
			ak := authKey{}
			err := ak.unmarshalString(keyPairs)
			if err != nil {
				return nil, err
			}
			if _, ok = tokenSet.tokens[remoteAddress]; ok {
				return nil, fmt.Errorf("cannot parse token: %s, repeated token for same BSR remote: %s", token, remoteAddress)
			}
			tokenSet.tokens[remoteAddress] = ak
		} else {
			if tokenSet.defaultToken != "" {
				return nil, fmt.Errorf("cannot parse token: %s, two buf token provided: %q and %q", token, u, tokenSet.defaultToken)
			}
			tokenSet.defaultToken = u
		}
	}
	return tokenSet, nil
}

func (t *TokenSet) RemoteToken(address string) (string, bool) {
	if authKeyPair, ok := t.tokens[address]; ok {
		return authKeyPair.token, t.setBufTokenEnvVar
	}
	return t.defaultToken, t.setBufTokenEnvVar
}

type authKey struct {
	username string
	token    string
}

func (ak *authKey) unmarshalString(s string) error {
	username, token, found := strings.Cut(s, ":")
	if !found {
		return fmt.Errorf("cannot parse remote token: %s", s)
	}
	ak.username = username
	ak.token = token
	return nil
}

// NewAuthorizationInterceptorProvider returns a new provider function which, when invoked, returns an interceptor
// which will set the auth token into the request header by the provided option.
//
// Note that the interceptor returned from this provider is always applied LAST in the series of interceptors added to
// a client.
func NewAuthorizationInterceptorProvider(tokenFinders ...TokenFinder) func(string) connect.UnaryInterceptorFunc {
	return func(address string) connect.UnaryInterceptorFunc {
		interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
			return connect.UnaryFunc(func(
				ctx context.Context,
				req connect.AnyRequest,
			) (connect.AnyResponse, error) {
				usingTokenEnvKey := false
				for _, tf := range tokenFinders {
					if token, setFromEnvVar := tf.RemoteToken(address); token != "" {
						req.Header().Set(AuthenticationHeader, AuthenticationTokenPrefix+token)
						usingTokenEnvKey = setFromEnvVar
						break
					}
				}
				response, err := next(ctx, req)
				if err != nil && usingTokenEnvKey {
					err = &ErrAuth{cause: err, tokenEnvKey: tokenEnvKey}
				}
				return response, err
			})
		}
		return interceptor
	}
}
