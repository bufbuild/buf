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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegisterDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     *DeviceRegistrationRequest
		transport func(t *testing.T, r *http.Request) (*http.Response, error)
		output    *DeviceRegistrationResponse
		err       error
	}{{
		name: "success",
		input: &DeviceRegistrationRequest{
			ClientName: "nameOfClient",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertJSONRequest(t, r, `{"client_name":"nameOfClient"}`)
			return testNewJSONResponse(t, http.StatusOK, `{"client_id":"clientID","client_secret":"clientSecret","client_id_issued_at":10,"client_secret_expires_at":100}`), nil
		},
		output: &DeviceRegistrationResponse{
			ClientID:              "clientID",
			ClientSecret:          "clientSecret",
			ClientIDIssuedAt:      10,
			ClientSecretExpiresAt: 100,
		},
	}, {
		name: "error",
		input: &DeviceRegistrationRequest{
			ClientName: "nameOfClient",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertJSONRequest(t, r, `{"client_name":"nameOfClient"}`)
			return testNewJSONResponse(t, http.StatusBadRequest, `{"error":"invalid_request","error_description":"invalid request"}`), nil
		},
		err: &Error{
			ErrorCode:        ErrorCodeInvalidRequest,
			ErrorDescription: "invalid request",
		},
	}, {
		name: "transport error",
		input: &DeviceRegistrationRequest{
			ClientName: "nameOfClient",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			return nil, io.EOF
		},
		err: &url.Error{
			Op:  "Post",
			URL: "https://buf.build" + DeviceRegistrationPath,
			Err: io.EOF,
		},
	}, {
		name:  "server error",
		input: &DeviceRegistrationRequest{ClientName: "nameOfClient"},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "501 Not Implemented",
				StatusCode: http.StatusNotImplemented,
				Body:       io.NopCloser(strings.NewReader(`not implemented`)),
			}, nil
		},
		err: fmt.Errorf("oauth2: %w: 501 not implemented", errors.ErrUnsupported),
	}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			c := NewClient("https://buf.build", &http.Client{
				Transport: testRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, r.Method, http.MethodPost)
					assert.Equal(t, r.URL.Path, DeviceRegistrationPath)
					assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
					assert.Equal(t, r.Header.Get("Accept"), "application/json")
					return test.transport(t, r)
				}),
			})
			output, err := c.RegisterDevice(ctx, test.input)
			assert.Equal(t, test.output, output)
			assert.Equal(t, err, test.err)
		})
	}
}

func TestAuthorizeDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     *DeviceAuthorizationRequest
		transport func(t *testing.T, r *http.Request) (*http.Response, error)
		output    *DeviceAuthorizationResponse
		err       error
	}{{
		name: "success",
		input: &DeviceAuthorizationRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertFormRequest(t, r, url.Values{"client_id": {"clientID"}, "client_secret": {"clientSecret"}})
			return testNewJSONResponse(t, http.StatusOK, `{"device_code":"deviceCode","user_code":"userCode","verification_uri":"https://example.com","verification_uri_complete":"https://example.com?code=userCode","expires_in":10,"interval":5}`), nil
		},
		output: &DeviceAuthorizationResponse{
			DeviceCode:              "deviceCode",
			UserCode:                "userCode",
			VerificationURI:         "https://example.com",
			VerificationURIComplete: "https://example.com?code=userCode",
			ExpiresIn:               10,
			Interval:                5,
		},
	}, {
		name: "error",
		input: &DeviceAuthorizationRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertFormRequest(t, r, url.Values{"client_id": {"clientID"}, "client_secret": {"clientSecret"}})
			return testNewJSONResponse(t, http.StatusBadRequest, `{"error":"invalid_request","error_description":"invalid request"}`), nil
		},
		err: &Error{
			ErrorCode:        ErrorCodeInvalidRequest,
			ErrorDescription: "invalid request",
		},
	}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			c := NewClient("https://buf.build", &http.Client{
				Transport: testRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, r.Method, http.MethodPost)
					assert.Equal(t, r.URL.Path, DeviceAuthorizationPath)
					assert.Equal(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
					assert.Equal(t, r.Header.Get("Accept"), "application/json")
					return test.transport(t, r)
				}),
			})
			output, err := c.AuthorizeDevice(ctx, test.input)
			assert.Equal(t, test.output, output)
			assert.Equal(t, err, test.err)
		})
	}
}

func TestAccessDeviceToken(t *testing.T) {
	t.Parallel()

	var pollingCount int
	tests := []struct {
		name      string
		input     *DeviceAccessTokenRequest
		transport func(t *testing.T, r *http.Request) (*http.Response, error)
		output    *DeviceAccessTokenResponse
		err       error
	}{{
		name: "success",
		input: &DeviceAccessTokenRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
			DeviceCode:   "deviceCode",
			GrantType:    "urn:ietf:params:oauth:grant-type:device_code",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertFormRequest(t, r, url.Values{"client_id": {"clientID"}, "client_secret": {"clientSecret"}, "device_code": {"deviceCode"}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}})
			return testNewJSONResponse(t, http.StatusOK, `{"access_token":"accessToken","token_type":"bearer","expires_in":100,"refresh_token":"refreshToken","scope":"scope"}`), nil
		},
		output: &DeviceAccessTokenResponse{
			AccessToken:  "accessToken",
			TokenType:    "bearer",
			ExpiresIn:    100,
			RefreshToken: "refreshToken",
			Scope:        "scope",
		},
	}, {
		name: "polling",
		input: &DeviceAccessTokenRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
			DeviceCode:   "deviceCode",
			GrantType:    "urn:ietf:params:oauth:grant-type:device_code",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertFormRequest(t, r, url.Values{"client_id": {"clientID"}, "client_secret": {"clientSecret"}, "device_code": {"deviceCode"}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}})
			if pollingCount == 0 {
				pollingCount++
				return testNewJSONResponse(t, http.StatusBadRequest, `{"error":"authorization_pending","error_description":"authorization pending"}`), nil
			}
			return testNewJSONResponse(t, http.StatusOK, `{"access_token":"accessToken","token_type":"bearer","expires_in":100,"refresh_token":"refreshToken","scope":"scope"}`), nil
		},
		output: &DeviceAccessTokenResponse{
			AccessToken:  "accessToken",
			TokenType:    "bearer",
			ExpiresIn:    100,
			RefreshToken: "refreshToken",
			Scope:        "scope",
		},
	}, {
		name: "error",
		input: &DeviceAccessTokenRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
			DeviceCode:   "deviceCode",
			GrantType:    "urn:ietf:params:oauth:grant-type:device_code",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			testAssertFormRequest(t, r, url.Values{"client_id": {"clientID"}, "client_secret": {"clientSecret"}, "device_code": {"deviceCode"}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}})
			return testNewJSONResponse(t, http.StatusBadRequest, `{"error":"invalid_request","error_description":"invalid request"}`), nil
		},
		err: &Error{
			ErrorCode:        ErrorCodeInvalidRequest,
			ErrorDescription: "invalid request",
		},
	}, {
		name: "expired token",
		input: &DeviceAccessTokenRequest{
			ClientID:     "clientID",
			ClientSecret: "clientSecret",
			DeviceCode:   "deviceCode",
			GrantType:    "urn:ietf:params:oauth:grant-type:device_code",
		},
		transport: func(t *testing.T, r *http.Request) (*http.Response, error) {
			return testNewJSONResponse(t, http.StatusBadRequest, `{"error":"expired_token","error_description":"token expired"}`), nil
		},
		err: &Error{
			ErrorCode:        ErrorCodeExpiredToken,
			ErrorDescription: "token expired",
		},
	}}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			c := NewClient("https://buf.build", &http.Client{
				Transport: testRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, r.Method, http.MethodPost)
					assert.Equal(t, r.URL.Path, DeviceTokenPath)
					assert.Equal(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
					assert.Equal(t, r.Header.Get("Accept"), "application/json")
					return test.transport(t, r)
				}),
			})
			output, err := c.AccessDeviceToken(ctx, test.input, AccessDeviceTokenWithPollingInterval(time.Millisecond))
			assert.Equal(t, test.output, output)
			assert.Equal(t, err, test.err)
		})
	}
}

type testRoundTripFunc func(r *http.Request) (*http.Response, error)

func (s testRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

func testNewJSONResponse(t *testing.T, statusCode int, body string) *http.Response {
	t.Helper()
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testAssertJSONRequest(t *testing.T, r *http.Request, expect string) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if !assert.NoError(t, err) {
		return
	}
	assert.JSONEq(t, string(body), expect)
}

func testAssertFormRequest(t *testing.T, r *http.Request, values url.Values) {
	t.Helper()
	if !assert.NoError(t, r.ParseForm()) {
		return
	}
	assert.Equal(t, r.Form, values)
}
