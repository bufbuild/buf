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
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func TestH2CProxy(t *testing.T) {
	proxyLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer proxyLis.Close()

	// setup a proxy server, it doesn't actually have to proxy anywhere
	h2s := &http2.Server{}
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	proxy := &http.Server{
		Addr: proxyLis.Addr().String(),
		// enable the http2 server to handle h2c requests
		Handler: h2c.NewHandler(proxyHandler, h2s),
	}
	go func() {
		err := proxy.Serve(proxyLis)
		require.NoError(t, err)
	}()
	defer proxy.Close()

	// setup the client to proxy all requests to the proxy server
	proxyURL, err := url.Parse("https://" + proxyLis.Addr().String())
	require.NoError(t, err)
	client := NewClient(WithH2C(), WithProxy(http.ProxyURL(proxyURL)))

	req, err := http.NewRequest("GET", "https://www.example.com", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
