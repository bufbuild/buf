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

package httpauth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/stretchr/testify/assert"
)

type mockAuthenticator struct {
	setAuthCalled bool
	setAuthResult bool
	setAuthError  error
}

func (a *mockAuthenticator) SetAuth(envContainer app.EnvContainer, request *http.Request) (bool, error) {
	a.setAuthCalled = true
	return a.setAuthResult, a.setAuthError
}

func TestMultiAuthenticator_SetAuth(t *testing.T) {
	envContainer := app.NewEnvContainer(map[string]string{})

	// Test with no authenticators
	multiAuth := newMultiAuthenticator()
	ok, err := multiAuth.SetAuth(envContainer, nil)
	assert.NoError(t, err)
	assert.False(t, ok)

	// Test with one authenticator
	auth1 := &mockAuthenticator{setAuthResult: true}
	multiAuth = newMultiAuthenticator(auth1)
	ok, err = multiAuth.SetAuth(envContainer, nil)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, auth1.setAuthCalled)

	// Test with multiple authenticators, where the first authenticator succeeds
	auth2 := &mockAuthenticator{setAuthResult: false}
	multiAuth = newMultiAuthenticator(auth1, auth2)
	ok, err = multiAuth.SetAuth(envContainer, nil)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, auth1.setAuthCalled)
	assert.False(t, auth2.setAuthCalled)

	// Test with multiple authenticators, where no authenticator succeeds
	auth3 := &mockAuthenticator{setAuthResult: false, setAuthError: errors.New("test error")}
	multiAuth = newMultiAuthenticator(auth2, auth3)
	ok, err = multiAuth.SetAuth(envContainer, nil)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.True(t, auth2.setAuthCalled)
	assert.True(t, auth3.setAuthCalled)
}
