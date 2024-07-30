// Copyright 2020-2024 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
)

// NewTokenProviderFromContainer creates a singleTokenProvider from the BUF_TOKEN environment variable
func NewTokenProviderFromContainer(container app.EnvContainer) (TokenProvider, error) {
	return newTokenProviderFromString(container.Env(TokenEnvKey), true)
}

// NewTokenProviderFromString creates a singleTokenProvider by the token provided
func NewTokenProviderFromString(token string) (TokenProvider, error) {
	return newTokenProviderFromString(token, false)
}

// newTokenProviderFromString returns a TokenProvider with auth keys from the provided token. The
// remote token is in the format: token1@remote1,token2@remote2.
// The special characters `@` and `,` are used as the splitters. The tokens and remote addresses
// do not contain these characters since they are enforced by the rules in BSR.
func newTokenProviderFromString(token string, isFromEnvVar bool) (TokenProvider, error) {
	if token == "" {
		return nopTokenProvider{}, nil
	}
	// Tokens for different remotes are separated by `,`. Using strings.Split to separate the string into remote tokens.
	tokens := strings.Split(token, ",")
	if len(tokens) == 1 {
		if strings.Contains(tokens[0], "@") {
			return newMultipleTokenProvider(tokens, isFromEnvVar)
		}
		return newSingleTokenProvider(tokens[0], isFromEnvVar)
	}
	return newMultipleTokenProvider(tokens, isFromEnvVar)
}

// singleTokenProvider is used to provide set of authentication tokenToAuthKey.
type singleTokenProvider struct {
	// true: the tokenSet is generated from environment variable tokenEnvKey
	// false: otherwise
	setBufTokenEnvVar bool
	token             string
}

func newSingleTokenProvider(token string, isFromEnvVar bool) (*singleTokenProvider, error) {
	if strings.Contains(token, "@") {
		return nil, errors.New("token cannot contain special character `@`")
	}
	if strings.Contains(token, ",") {
		return nil, errors.New("token cannot contain special character `,`")
	}
	if token == "" {
		return nil, errors.New("single token cannot be empty")
	}
	return &singleTokenProvider{
		setBufTokenEnvVar: isFromEnvVar,
		token:             token,
	}, nil
}

// RemoteToken finds the token by the remote address
func (t *singleTokenProvider) RemoteToken(address string) string {
	return t.token
}

func (t *singleTokenProvider) IsFromEnvVar() bool {
	return t.setBufTokenEnvVar
}

type multipleTokenProvider struct {
	addressToToken map[string]string
	isFromEnvVar   bool
}

func newMultipleTokenProvider(tokens []string, isFromEnvVar bool) (*multipleTokenProvider, error) {
	addressToToken := make(map[string]string)
	for _, token := range tokens {
		split := strings.Split(token, "@")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid token: %s", token)
		}
		if split[0] == "" || split[1] == "" {
			return nil, fmt.Errorf("invalid token: %s", token)
		}
		if strings.Contains(split[0], ":") {
			return nil, fmt.Errorf("invalid token: %s, token cannot contain special character `:`", token)
		}
		if strings.Contains(split[0], ",") {
			return nil, fmt.Errorf("invalid token: %s, token cannot contain special character `,`", token)
		}
		if _, ok := addressToToken[split[1]]; ok {
			return nil, fmt.Errorf("invalid token: %s, repeated remote adddress: %s", token, split[1])
		}
		addressToToken[split[1]] = split[0]
	}
	return &multipleTokenProvider{
		addressToToken: addressToToken,
		isFromEnvVar:   isFromEnvVar,
	}, nil
}

func (m *multipleTokenProvider) RemoteToken(address string) string {
	return m.addressToToken[address]
}

func (m *multipleTokenProvider) IsFromEnvVar() bool {
	return m.isFromEnvVar
}

type nopTokenProvider struct{}

func (nopTokenProvider) RemoteToken(string) string {
	return ""
}

func (nopTokenProvider) IsFromEnvVar() bool {
	return false
}
