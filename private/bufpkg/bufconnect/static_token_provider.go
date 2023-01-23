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
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
)

// staticTokenProvider is used to provide set of authentication tokenToAuthKey.
type staticTokenProvider struct {
	// true: the tokenSet is generated from environment variable tokenEnvKey
	// false: otherwise
	setBufTokenEnvVar bool
	defaultToken      string
	tokenToAuthKey    map[string]authKey
}

// NewTokenProviderFromContainer creates a staticTokenProvider from the BUF_TOKEN environment variable
func NewTokenProviderFromContainer(container app.EnvContainer) (TokenProvider, error) {
	return newTokenProviderFromString(container.Env(tokenEnvKey), true)
}

// NewTokenProviderFromString creates a staticTokenProvider by the token provided
func NewTokenProviderFromString(token string) (TokenProvider, error) {
	return newTokenProviderFromString(token, false)
}

// newTokenProviderFromString returns a TokenProvider with auth keys from the provided token. The
// remote token is in the format: username1:token1@remote1,username2:token2@remote2,defaultToken.
// The special characters `:`, `@` and `,` are used as the splitters. The usernames, tokens, and
// remote addresses does not contain these characters since they are enforced by the rules in BSR.
func newTokenProviderFromString(token string, isFromEnvVar bool) (TokenProvider, error) {
	tokenProvider := &staticTokenProvider{
		setBufTokenEnvVar: isFromEnvVar,
		tokenToAuthKey:    make(map[string]authKey),
	}
	// Tokens for different remotes are separated by `,`. Using strings.Split to separate the string into remote tokenToAuthKey.
	tokens := strings.Split(token, ",")
	for _, token := range tokens {
		if keyPairs, remoteAddress, ok := strings.Cut(token, "@"); ok {
			ak := authKey{}
			if err := ak.unmarshalString(keyPairs); err != nil {
				return nil, err
			}
			if _, ok = tokenProvider.tokenToAuthKey[remoteAddress]; ok {
				return nil, fmt.Errorf("cannot parse token: %s, repeated token for same BSR remote: %s", token, remoteAddress)
			}
			tokenProvider.tokenToAuthKey[remoteAddress] = ak
		} else {
			if tokenProvider.defaultToken != "" {
				return nil, fmt.Errorf("cannot parse token: two buf token provided: %q and %q", token, tokenProvider.defaultToken)
			}
			tokenProvider.defaultToken = token
		}
	}
	return tokenProvider, nil
}

// RemoteToken finds the token by the remote address
func (t *staticTokenProvider) RemoteToken(address string) string {
	if authKeyPair, ok := t.tokenToAuthKey[address]; ok {
		return authKeyPair.token
	}
	return t.defaultToken
}

func (t *staticTokenProvider) IsFromEnvVar() bool {
	return t.setBufTokenEnvVar
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
