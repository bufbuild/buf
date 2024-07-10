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

// Package oauth2 contains functionality to work with OAuth2.
package oauth2

import (
	"fmt"
)

// ErrorCode is an OAuth2 error code.
type ErrorCode string

// The following error codes are defined by RFC 6749 Section 5.2 Error Response.
const (
	// ErrorCodeInvalidRequest is an invalid or malformed request error.
	ErrorCodeInvalidRequest ErrorCode = "invalid_request"
	// ErrorCodeInvalidClient is a client authentication error.
	ErrorCodeInvalidClient ErrorCode = "invalid_client"
	// ErrorCodeInvalidGrant is an invalid grant error.
	ErrorCodeInvalidGrant ErrorCode = "invalid_grant"
	// ErrorCodeUnauthorizedClient is an unauthorized client error.
	ErrorCodeUnauthorizedClient ErrorCode = "unauthorized_client"
	// ErrorCodeUnsupportedGrantType is an unsupported grant type error.
	ErrorCodeUnsupportedGrantType ErrorCode = "unsupported_grant_type"
	// ErrorCodeInvalidScope is an invalid scope error.
	ErrorCodeInvalidScope ErrorCode = "invalid_scope"
)

// Error is an OAuth2 error.
type Error struct {
	// ErrorCode is the error code.
	ErrorCode ErrorCode `json:"error"`
	// ErrorDescription is a human-readable description of the error. May be empty.
	ErrorDescription string `json:"error_description,omitempty"`
	// ErrorURI is a URI for the error. May be empty.
	ErrorURI string `json:"error_uri,omitempty"`
}

// Error implements error.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	s := fmt.Sprintf("oauth2: %q", e.ErrorCode)
	if e.ErrorDescription != "" {
		s += fmt.Sprintf(" %q", e.ErrorDescription)
	}
	if e.ErrorURI != "" {
		s += fmt.Sprintf(" %q", e.ErrorURI)
	}
	return s
}
