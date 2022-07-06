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
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func TestH2CProxy(t *testing.T) {
	upstreamLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer upstreamLis.Close()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		err := server.Serve(upstreamLis)
		require.NoError(t, err)
	}()
	defer server.Close()

	proxyLis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer proxyLis.Close()

	h2s := &http2.Server{}
	upstreamURL, err := url.Parse("http://" + upstreamLis.Addr().String())
	require.NoError(t, err)
	proxyHandler := httputil.NewSingleHostReverseProxy(upstreamURL)
	proxy := &http.Server{
		Addr:    proxyLis.Addr().String(),
		Handler: h2c.NewHandler(proxyHandler, h2s),
	}
	go func() {
		err := proxy.Serve(proxyLis)
		require.NoError(t, err)
	}()
	defer proxy.Close()

	proxyURL, err := url.Parse("https://" + proxyLis.Addr().String())
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "https://www.example.com", nil)
	require.NoError(t, err)
	client := NewClient(WithH2C(), WithProxy(http.ProxyURL(proxyURL)))
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
