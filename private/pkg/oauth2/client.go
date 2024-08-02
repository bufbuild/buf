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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"go.uber.org/multierr"
)

const (
	defaultPollingInterval   = 5 * time.Second
	incrementPollingInterval = 5 * time.Second
	maxPollingInterval       = 30 * time.Second
	maxPayloadSize           = 1 << 20 // 1 MB
)

// Client is an OAuth 2.0 client that can register a device, authorize a device,
// and poll for the device access token.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient returns a new Client with the given base URL and HTTP client.
func NewClient(baseURL string, client *http.Client) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
	}
}

// RegisterDevice registers a new device with the authorization server.
func (c *Client) RegisterDevice(
	ctx context.Context,
	deviceRegistrationRequest *DeviceRegistrationRequest,
) (_ *DeviceRegistrationResponse, retErr error) {
	input, err := json.Marshal(deviceRegistrationRequest)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(input)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+DeviceRegistrationPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()

	payload := &struct {
		Error
		DeviceRegistrationResponse
	}{}
	if err := parseJSONResponse(response, payload); err != nil {
		return nil, err
	}
	if payload.ErrorCode != "" {
		return nil, &payload.Error
	}
	if code := response.StatusCode; code != http.StatusOK {
		return nil, fmt.Errorf("oauth2: invalid status: %v", code)
	}
	return &payload.DeviceRegistrationResponse, nil
}

// AuthorizeDevice authorizes a device with the authorization server. The authorization server
// will return a device code and a user code that the user must use to authorize the device.
func (c *Client) AuthorizeDevice(
	ctx context.Context,
	deviceAuthorizationRequest *DeviceAuthorizationRequest,
) (_ *DeviceAuthorizationResponse, retErr error) {
	body := strings.NewReader(deviceAuthorizationRequest.ToValues().Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+DeviceAuthorizationPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()

	payload := &struct {
		Error
		DeviceAuthorizationResponse
	}{}
	if err := parseJSONResponse(response, payload); err != nil {
		return nil, err
	}
	if payload.ErrorCode != "" {
		return nil, &payload.Error
	}
	if code := response.StatusCode; code != http.StatusOK {
		return nil, fmt.Errorf("oauth2: invalid status: %v", code)
	}
	return &payload.DeviceAuthorizationResponse, nil
}

// AccessDeviceToken polls the authorization server for the device access token. The interval
// parameter specifies the polling interval in seconds.
func (c *Client) AccessDeviceToken(
	ctx context.Context,
	deviceAccessTokenRequest *DeviceAccessTokenRequest,
	options ...AccessDeviceTokenOption,
) (*DeviceAccessTokenResponse, error) {
	accessOptions := newAccessDeviceTokenOption()
	for _, option := range options {
		option(accessOptions)
	}
	pollingInterval := accessOptions.pollingInterval
	if pollingInterval == 0 {
		pollingInterval = defaultPollingInterval
	} else if pollingInterval < 0 {
		return nil, fmt.Errorf("oauth2: polling interval must be greater than 0")
	} else if pollingInterval > maxPollingInterval {
		return nil, fmt.Errorf("oauth2: polling interval must be less than or equal to %v", maxPollingInterval)
	}
	encodedValues := deviceAccessTokenRequest.ToValues().Encode()
	timer := time.NewTimer(pollingInterval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			body := strings.NewReader(encodedValues)
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+DeviceTokenPath, body)
			if err != nil {
				return nil, err
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Accept", "application/json")

			response, err := c.client.Do(request)
			if err != nil {
				return nil, err
			}
			payload := &struct {
				Error
				DeviceAccessTokenResponse
			}{}
			if err := parseJSONResponse(response, payload); err != nil {
				if closeErr := response.Body.Close(); closeErr != nil {
					err = multierr.Append(err, closeErr)
				}
				return nil, err
			}
			if err := response.Body.Close(); err != nil {
				return nil, fmt.Errorf("oauth2: failed to close response body: %w", err)
			}
			if response.StatusCode == http.StatusOK && payload.ErrorCode == "" {
				return &payload.DeviceAccessTokenResponse, nil
			}
			switch payload.ErrorCode {
			case ErrorCodeSlowDown:
				// If the server is rate limiting the client, increase the polling interval.
				pollingInterval += incrementPollingInterval
			case ErrorCodeAuthorizationPending:
				// If the user has not yet authorized the device, continue polling.
			case ErrorCodeAccessDenied, ErrorCodeExpiredToken:
				// If the user has denied the device or the token has expired, return the error.
				return nil, &payload.Error
			default:
				return nil, &payload.Error
			}
			timer.Reset(pollingInterval)
		}
	}
}

// AccessDeviceTokenOption is an option for AccessDeviceToken.
type AccessDeviceTokenOption func(*accessDeviceTokenOptions)

// AccessDeviceTokenWithPollingInterval returns a new AccessDeviceTokenOption that sets the polling interval.
//
// The default is 5 seconds. Polling may not be longer than 30 seconds.
func AccessDeviceTokenWithPollingInterval(pollingInterval time.Duration) AccessDeviceTokenOption {
	return func(accessDeviceTokenOptions *accessDeviceTokenOptions) {
		accessDeviceTokenOptions.pollingInterval = pollingInterval
	}
}

// *** PRIVATE ***

type accessDeviceTokenOptions struct {
	pollingInterval time.Duration
}

func newAccessDeviceTokenOption() *accessDeviceTokenOptions {
	return &accessDeviceTokenOptions{
		pollingInterval: defaultPollingInterval,
	}
}

func parseJSONResponse(response *http.Response, payload any) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPayloadSize))
	if err != nil {
		return fmt.Errorf("oauth2: failed to read response body: %w", err)
	}
	if contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type")); contentType != "application/json" {
		return fmt.Errorf("oauth2: %w: %d %s", errors.ErrUnsupported, response.StatusCode, body)
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("oauth2: failed to unmarshal response: %w: %s", err, body)
	}
	return nil
}
