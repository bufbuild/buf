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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	proxyResponse := []byte("hello")
	// setup a proxy server, it doesn't actually have to proxy anywhere
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(proxyResponse)
		require.NoError(t, err)
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	client := NewClient(
		// setup the client to proxy all requests to the proxy server
		WithProxy(http.ProxyURL(proxyURL)),
	)

	req, err := http.NewRequest("GET", "http://www.example.com", nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, string(proxyResponse), string(respBody))
}
