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

package bufcli

import (
	"net/http"
	"testing"

	"buf.build/go/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPAuthenticatorUsesUsernameAndPassword(t *testing.T) {
	t.Parallel()
	envContainer := app.NewEnvContainer(map[string]string{
		// Point HOME at an empty dir so the netrc authenticator finds no
		// machine and falls through to the env authenticator under test.
		"HOME":                   t.TempDir(),
		inputHTTPSUsernameEnvKey: "username",
		inputHTTPSPasswordEnvKey: "password",
	})
	request, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)
	ok, err := defaultHTTPAuthenticator.SetAuth(envContainer, request)
	require.NoError(t, err)
	require.True(t, ok)
	username, password, ok := request.BasicAuth()
	require.True(t, ok)
	assert.Equal(t, "username", username)
	assert.Equal(t, "password", password)
}
