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
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
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
		baseURL: baseURL,
		client:  client,
	}
}

// RegisterDevice registers a new device with the authorization server.
func (c *Client) RegisterDevice(ctx context.Context, args *DeviceRegistrationRequest) (*DeviceRegistrationResponse, error) {
	input, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+DeviceRegistrationPath, bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	payload := &struct {
		Error
		DeviceRegistrationResponse
	}{}
	if err := c.parseJSONResponse(rsp, payload); err != nil {
		return nil, fmt.Errorf("oauth2: invalid response: %w", err)
	}
	if payload.ErrorCode != "" {
		return nil, &payload.Error
	}
	if code := rsp.StatusCode; code != http.StatusOK {
		return nil, fmt.Errorf("oauth2: invalid status: %v", code)
	}
	return &payload.DeviceRegistrationResponse, nil
}

// AuthorizeDevice authorizes a device with the authorization server. The authorization server
// will return a device code and a user code that the user must use to authorize the device.
func (c *Client) AuthorizeDevice(ctx context.Context, args *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+DeviceAuthorizationPath, strings.NewReader(args.ToValues().Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	payload := &struct {
		Error
		DeviceAuthorizationResponse
	}{}
	if err := c.parseJSONResponse(rsp, payload); err != nil {
		return nil, fmt.Errorf("oauth2: invalid response: %w", err)
	}
	if payload.ErrorCode != "" {
		return nil, &payload.Error
	}
	if code := rsp.StatusCode; code != http.StatusOK {
		return nil, fmt.Errorf("oauth2: invalid status: %v", code)
	}
	return &payload.DeviceAuthorizationResponse, nil
}

// AccessDeviceToken polls the authorization server for the device access token. The interval
// parameter specifies the polling interval in seconds.
func (c *Client) AccessDeviceToken(ctx context.Context, interval int, args *DeviceAccessTokenRequest) (*DeviceAccessTokenResponse, error) {
	encodedValues := args.ToValues().Encode()
	if interval <= 0 {
		interval = 5 // Default polling interval of 5 seconds.
	} else if interval > 30 {
		return nil, fmt.Errorf("oauth2: interval is too large: %v", interval)
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+DeviceTokenPath, strings.NewReader(encodedValues))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")

			rsp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			payload := &struct {
				Error
				DeviceAccessTokenResponse
			}{}
			if err := c.parseJSONResponse(rsp, payload); err != nil {
				_ = rsp.Body.Close()
				return nil, fmt.Errorf("oauth2: invalid response: %w", err)
			}
			_ = rsp.Body.Close()
			if rsp.StatusCode == http.StatusOK && payload.ErrorCode == "" {
				return &payload.DeviceAccessTokenResponse, nil
			}
			switch payload.ErrorCode {
			case ErrorCodeSlowDown:
				// If the server is rate limiting the client, increase the polling interval.
				interval += 5
				ticker.Reset(time.Duration(interval) * time.Second)
			case ErrorCodeAuthorizationPending:
				// If the user has not yet authorized the device, continue polling.
				continue
			case ErrorCodeAccessDenied, ErrorCodeExpiredToken:
				// If the user has denied the device or the token has expired, return an error.
				fallthrough
			default:
				return nil, &payload.Error
			}
		}
	}
}

func (c *Client) parseJSONResponse(rsp *http.Response, payload any) error {
	body, err := io.ReadAll(io.LimitReader(rsp.Body, 1<<20))
	if err != nil {
		return err
	}
	contentType, _, _ := mime.ParseMediaType(rsp.Header.Get("Content-Type"))
	if contentType != "application/json" {
		return fmt.Errorf("%d %s", rsp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	return nil
}
