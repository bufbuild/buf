// Copyright 2020-2022 Buf Technologies, Inc.
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

package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/multierr"
)

// NewClient returns a new Client.
func NewClient(options ...ClientOption) *http.Client {
	return newClient(options...)
}

// ClientOption is an option for a new Client.
type ClientOption func(*clientOptions)

// WithTLSConfig returns a new ClientOption to use the tls.Config.
//
// The default is to use no TLS.
func WithTLSConfig(tlsConfig *tls.Config) ClientOption {
	return func(opts *clientOptions) {
		opts.tlsConfig = tlsConfig
	}
}

// WithObservability returns a new ClientOption to use
// OpenCensus tracing and metrics.
//
// The default is to use no observability.
func WithObservability() ClientOption {
	return func(opts *clientOptions) {
		opts.observability = true
	}
}

// WithH2C returns a new ClientOption that allows dialing
// h2c (cleartext) servers.
func WithH2C() ClientOption {
	return func(opts *clientOptions) {
		opts.h2c = true
	}
}

// WithProxy returns a new ClientOption to use
// a proxy.
//
// The default is to use http.ProxyFromEnvironment
func WithProxy(proxyFunc Proxy) ClientOption {
	return func(opts *clientOptions) {
		opts.proxy = proxyFunc
	}
}

// Proxy specifies a function to return a proxy for a given
// Request. If the function returns a non-nil error, the
// request is aborted with the provided error.
type Proxy func(req *http.Request) (*url.URL, error)

// NewClientWithTransport returns a new Client with the
// given transport. This is a separate constructor so
// that it's clear it cannot be used in combination
// with other ClientOptions.
func NewClientWithTransport(transport http.RoundTripper) *http.Client {
	return newClientWithTransport(transport)
}

// GetResponseBody reads and closes the response body.
func GetResponseBody(client *http.Client, request *http.Request) (_ []byte, retErr error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got HTTP status code %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read HTTP response: %v", err)
	}
	return data, nil
}
