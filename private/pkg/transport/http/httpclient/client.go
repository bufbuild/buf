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

package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/bufbuild/buf/private/pkg/observability"
	"go.opencensus.io/tag"
	"golang.org/x/net/http2"
)

type clientOptions struct {
	tlsConfig         *tls.Config
	observability     bool
	observabilityTags []tag.Mutator
	proxy             Proxy
	interceptorFunc   ClientInterceptorFunc
	h2c               bool
}

func newClient(options ...ClientOption) *http.Client {
	opts := &clientOptions{
		proxy: http.ProxyFromEnvironment,
	}
	for _, opt := range options {
		opt(opts)
	}
	var roundTripper http.RoundTripper
	if opts.h2c {
		roundTripper = &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: opts.tlsConfig,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
	} else {
		roundTripper = &http.Transport{
			TLSClientConfig: opts.tlsConfig,
			Proxy:           opts.proxy,
		}
	}
	if opts.interceptorFunc != nil {
		roundTripper = opts.interceptorFunc(roundTripper)
	}
	if opts.observability {
		roundTripper = observability.NewHTTPTransport(roundTripper, opts.observabilityTags...)
	}
	return &http.Client{
		Transport: roundTripper,
	}
}

func newClientWithTransport(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
	}
}
