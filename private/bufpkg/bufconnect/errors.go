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

import "errors"

// AuthError wraps the error returned in the auth provider to add additional context.
type AuthError struct {
	cause error

	tokenEnvKey string
}

// Unwrap returns the underlying error.
func (e *AuthError) Unwrap() error {
	return e.cause
}

// Error implements the error interface and returns the error message.
func (e *AuthError) Error() string {
	if e.cause == nil {
		return "unknown error"
	}
	return e.cause.Error()
}

// TokenEnvKey returns the environment variable used, if any, for authentication.
func (e *AuthError) TokenEnvKey() string {
	return e.tokenEnvKey
}

// AsAuthError uses errors.As to unwrap any error and look for an *AuthError.
func AsAuthError(err error) (*AuthError, bool) {
	var authErr *AuthError
	ok := errors.As(err, &authErr)
	return authErr, ok
}
