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

package oauth2

import (
	"net/url"
)

const (
	// DeviceRegistrationPath is the path for the device registration endpoint.
	DeviceRegistrationPath = "/oauth2/device/registration"
	// DeviceAuthorizationPath is the path for the device authorization endpoint.
	DeviceAuthorizationPath = "/oauth2/device/authorization"
	// DeviceTokenPath is the path for the device token endpoint.
	DeviceTokenPath = "/oauth2/device/token"
)

const (
	// DeviceAuthorizationGrantType is the grant type for the device authorization flow.
	DeviceAuthorizationGrantType = "urn:ietf:params:oauth:grant-type:device_code"
)

// The following error codes are defined by RFC 8628 Section 3.5 Device Authorization Response.
const (
	// ErrorCodeAuthorizationPending is a pending device authorization grant as the
	// end user hasn't yet completed the user interaction steps.
	ErrorCodeAuthorizationPending ErrorCode = "authorization_pending"
	// ErrorCodeSlowDown is returned for a pending device authorization grant and
	// polling should continue, but the interval MUST be increased by 5 seconds for
	// all subsequent requests.
	ErrorCodeSlowDown ErrorCode = "slow_down"
	// ErrorCodeAccessDenied is returned when the device authorization request was denied.
	ErrorCodeAccessDenied ErrorCode = "access_denied"
	// ErrorCodeExpiredToken is the device_code has expired, and the device authorization
	// session has concluded. The client MAY commence a new device authorization request but
	// SHOULD wait for user interaction before restarting to avoid unnecessary polling.
	ErrorCodeExpiredToken ErrorCode = "expired_token"
)

// DeviceRegistrationRequest describes an OpenID Connect Dynamic Client Registration 1.0 request
// for dynamic client registration. It is a subset of the full specification.
// It does not require a redirect URI or grant types for the device authorization flow.
// https://openid.net/specs/openid-connect-registration-1_0.html#RegistrationRequest
type DeviceRegistrationRequest struct {
	// Name of the client to be presented to the end user.
	ClientName string `json:"client_name"`
}

// Devic describes a successful OpenID Connect Dynamic Client Registration 1.0 response
// for dynamic client registration.
type DeviceRegistrationResponse struct {
	// ClientID is the unique client identifier.
	ClientID string `json:"client_id"`
	// ClientSecret is the client secret. May be empty.
	ClientSecret string `json:"client_secret,omitempty"`
	// ClientIDIssuedAt is the time at which the ClientID was issued in seconds since the Unix epoch.
	ClientIDIssuedAt int `json:"client_id_issued_at"`
	// ClientSecretExpiresAt is the time at which the client_secret will expire in seconds since the Unix epoch.
	ClientSecretExpiresAt int `json:"client_secret_expires_at,omitempty"`
}

// DeviceAuthorizationRequest describes an RFC 8628 Device Authorization Request.
// https://datatracker.ietf.org/doc/html/rfc8628#section-3.1
type DeviceAuthorizationRequest struct {
	// ClientID is the unique client identifier.
	ClientID string `json:"client_id"`
	// ClientSecret is the client secret. May be empty.
	ClientSecret string `json:"client_secret,omitempty"`
}

// ToValues converts the DeviceAuthorizationRequest to url.Values.
func (d *DeviceAuthorizationRequest) ToValues() url.Values {
	values := make(url.Values, 2)
	values.Set("client_id", d.ClientID)
	if d.ClientSecret != "" {
		values.Set("client_secret", d.ClientSecret)
	}
	return values
}

// FromValues converts the url.Values to a DeviceAuthorizationRequest.
func (d *DeviceAuthorizationRequest) FromValues(values url.Values) error {
	d.ClientID = values.Get("client_id")
	d.ClientSecret = values.Get("client_secret")
	return nil
}

// DeviceAuthorizationResponse describes a successful RFC 8628 Device Authorization Response
// https://datatracker.ietf.org/doc/html/rfc8628#section-3.2
type DeviceAuthorizationResponse struct {
	// DeviceCode is the device verification code.
	DeviceCode string `json:"device_code"`
	// UserCode is the end-user verification code.
	UserCode string `json:"user_code"`
	// VerificationURI is the verification URI that the end user should visit to
	// enter the user_code.
	VerificationURI string `json:"verification_uri"`
	// VerificationURIComplete is the verification URI that includes the user_code.
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	// ExpiresIn is the lifetime in seconds of the "device_code" and "user_code".
	ExpiresIn int `json:"expires_in"`
	// Interval is the minimum amount of time in seconds that the client SHOULD wait
	// between polling requests to the token endpoint.
	Interval int `json:"interval,omitempty"`
}

// DeviceAccessTokenRequest describes an RFC 8628 Device Token Request.
// https://datatracker.ietf.org/doc/html/rfc8628#section-3.4
type DeviceAccessTokenRequest struct {
	// ClientID is the client identifier issued to the client during the registration process.
	ClientID string `json:"client_id"`
	// ClientSecret is the client secret. May be empty.
	ClientSecret string `json:"client_secret,omitempty"`
	// DeviceCode is the device verification code.
	DeviceCode string `json:"device_code"`
	// GrantType is the grant type for the device authorization flow. Must be
	// set to "urn:ietf:params:oauth:grant-type:device_code".
	GrantType string `json:"grant_type"`
}

// ToValues converts the DeviceTokenRequest to url.Values.
func (d *DeviceAccessTokenRequest) ToValues() url.Values {
	values := make(url.Values, 4)
	values.Set("client_id", d.ClientID)
	if d.ClientSecret != "" {
		values.Set("client_secret", d.ClientSecret)
	}
	values.Set("device_code", d.DeviceCode)
	values.Set("grant_type", d.GrantType)
	return values
}

// FromValues converts the url.Values to a DeviceTokenRequest.
func (d *DeviceAccessTokenRequest) FromValues(values url.Values) error {
	d.ClientID = values.Get("client_id")
	d.ClientSecret = values.Get("client_secret")
	d.DeviceCode = values.Get("device_code")
	d.GrantType = values.Get("grant_type")
	return nil
}

// DeviceAccessTokenResponse describes a successful RFC 8628 Device Token Response.
// https://datatracker.ietf.org/doc/html/rfc8628#section-3.5
type DeviceAccessTokenResponse struct {
	// AccessToken is the access token that can be used to access the protected resources.
	AccessToken string `json:"access_token"`
	// TokenType is the type of the token issued as described in RFC 6749 Section 7.1.
	// https://datatracker.ietf.org/doc/html/rfc6749#section-7.1
	TokenType string `json:"token_type"`
	// ExpiresIn is the lifetime in seconds of the access token.
	ExpiresIn int `json:"expires_in,omitempty"`
	// RefreshToken may be used to obtain new access tokens using the same authoization
	// grant. May be empty.
	RefreshToken string `json:"refresh_token,omitempty"`
	// Scope is the scope of the access token as described in RFC 6749 Section 3.3.
	// https://datatracker.ietf.org/doc/html/rfc6749#section-3.3
	Scope string `json:"scope,omitempty"`
}
