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
	// Tokens for different remotes are separated by `,`. Using strings.Split to separate the string into remote tokenToAuthKey.
	tokens := strings.Split(token, ",")
	if len(tokens) <= 1 {
		return staticTokenProvider{
			setBufTokenEnvVar: isFromEnvVar,
			token:             token,
		}, nil
	}
	tokenProvider := &staticTokenProvider{
		setBufTokenEnvVar: isFromEnvVar,
		keyPairs:          make(map[string]string),
	}
	for _, token := range tokens {
		key, hostname, found := strings.Cut(token, "@")
		if !found {
			return nil, fmt.Errorf("cannot parse token: %s", token)
		}
		if _, ok := tokenProvider.keyPairs[hostname]; ok {
			return nil, fmt.Errorf("cannot parse token: %s, repeasted token for same BSR remote: %s", token, hostname)
		}
		tokenProvider.keyPairs[hostname] = key
	}
	return tokenProvider, nil
}

// staticTokenProvider is used to provide set of authentication tokenToAuthKey.
type staticTokenProvider struct {
	// true: the tokenSet is generated from environment variable tokenEnvKey
	// false: otherwise
	setBufTokenEnvVar bool
	keyPairs          map[string]string
	token             string
}

// RemoteToken finds the token by the remote address
func (t staticTokenProvider) RemoteToken(address string) string {
	if t.token != "" {
		return t.token
	}
	return t.keyPairs[address]
}

func (t staticTokenProvider) IsFromEnvVar() bool {
	return t.setBufTokenEnvVar
}
