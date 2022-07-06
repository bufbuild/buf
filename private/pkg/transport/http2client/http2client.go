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

package http2client

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/bufbuild/buf/private/pkg/observability"
	"github.com/bufbuild/buf/private/pkg/rpc/rpchttp"
	"golang.org/x/net/http2"
)

// NewClient returns a new HTTP2 client.
//
// To enable connections to h2c (cleartext) servers pass the allow insecure
// client option.
func NewClient(clientOptions ...ClientOption) *http.Client {
	option := &clientOption{
		proxy: http.ProxyFromEnvironment,
	}
	for _, opt := range clientOptions {
		opt(option)
	}
	var baseTransport http.RoundTripper = &http.Transport{
		TLSClientConfig: option.tlsConfig,
		Proxy:           option.proxy,
	}
	if option.useH2C {
		baseTransport = &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return proxyDial(netw, addr, option.proxy)
			},
		}
	}
	roundTripper := rpchttp.NewClientInterceptor(baseTransport)
	if option.observability {
		roundTripper = observability.NewHTTPTransport(roundTripper)
	}
	return &http.Client{
		Transport: roundTripper,
	}
}
