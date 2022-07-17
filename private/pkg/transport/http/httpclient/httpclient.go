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

// Client is a client.
type Client interface {
	// Do matches http.Client.
	//
	// This allows Client to be dropped in for http.Client.
	Do(request *http.Request) (*http.Response, error)
	// ParseAddress parses the given address.
	//
	// If the address has a scheme, this is a no-op.
	// If the address does not have a scheme, this adds https:// if TLS was configured,
	// and http:// if TLS was not configured.
	ParseAddress(address string) string
	// Transport returns the http.RoundTripper configured on
	// this client.
	Transport() http.RoundTripper
}

// NewClient returns a new Client.
func NewClient(options ...ClientOption) Client {
	return newClient(options...)
}

// ClientOption is an option for a new Client.
type ClientOption func(*client)

// ClientInterceptorFunc is a function that wraps a RoundTripper with any interceptors
type ClientInterceptorFunc func(http.RoundTripper) http.RoundTripper

// ClientWithTLSConfig returns a new ClientOption to use the tls.Config.
//
// The default is to use no TLS.
func ClientWithTLSConfig(tlsConfig *tls.Config) ClientOption {
	return func(client *client) {
		client.tlsConfig = tlsConfig
	}
}

// ClientWithObservability returns a new ClientOption to use
// OpenCensus tracing and metrics.
//
// The default is to use no observability.
func ClientWithObservability() ClientOption {
	return func(client *client) {
		client.observability = true
	}
}

// WithProxy returns a new ClientOption to use
// a proxy.
//
// The default is to use http.ProxyFromEnvironment
func ClientWithProxy(proxyFunc Proxy) ClientOption {
	return func(client *client) {
		client.proxy = proxyFunc
	}
}

// ClientWithInterceptorFunc returns a new ClientOption to use a given interceptor.
func ClientWithInterceptorFunc(interceptorFunc ClientInterceptorFunc) ClientOption {
	return func(client *client) {
		client.interceptorFunc = interceptorFunc
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
func NewClientWithTransport(transport http.RoundTripper) Client {
	return newClientWithTransport(transport)
}

// GetResponseBody reads and closes the response body.
func GetResponseBody(client Client, request *http.Request) (_ []byte, retErr error) {
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
