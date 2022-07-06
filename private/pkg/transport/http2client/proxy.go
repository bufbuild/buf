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
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// This file was adapted from https://github.com/grpc/grpc-go/blob/master/internal/transport/proxy.go
//
// Copyright 2017 gRPC authors. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.
//
// https://github.com/grpc/grpc-go/blob/master/LICENSE

const proxyAuthHeaderKey = "Proxy-Authorization"

// Proxy specifies a function to return a proxy for a given
// Request. If the function returns a non-nil error, the
// request is aborted with the provided error.
type Proxy = func(*http.Request) (*url.URL, error)

func mapAddress(address string, proxyFunc Proxy) (*url.URL, error) {
	req := &http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   address,
		},
	}
	return proxyFunc(req)
}

// To read a response from a net.Conn, http.ReadResponse() takes a bufio.Reader.
// It's possible that this reader reads more than what's need for the response and stores
// those bytes in the buffer.
// bufConn wraps the original net.Conn and the bufio.Reader to make sure we don't lose the
// bytes in the buffer.
type bufConn struct {
	net.Conn

	r io.Reader
}

func (c *bufConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func basicAuth(info *url.Userinfo) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(info.String()))
}

func doHTTPConnectHandshake(conn net.Conn, backendAddr string, proxyURL *url.URL) (_ net.Conn, err error) {
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: backendAddr},
	}
	if t := proxyURL.User; t != nil {
		req.Header.Add(proxyAuthHeaderKey, basicAuth(proxyURL.User))
	}

	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("failed to write the HTTP request: %v", err)
	}

	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return nil, fmt.Errorf("reading server HTTP response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, fmt.Errorf("failed to do HTTP CONNECT handshake, status code: %s", resp.Status)
		}
		return nil, fmt.Errorf("failed to do HTTP CONNECT handshake, response: %q", dump)
	}

	return &bufConn{Conn: conn, r: r}, nil
}

// proxyDial dials, connecting to a proxy first if necessary. Checks if a proxy
// is necessary, dials, does the HTTP CONNECT handshake, and returns the
// connection.
func proxyDial(netw, addr string, proxyFunc Proxy) (net.Conn, error) {
	newAddr := addr
	proxyURL, err := mapAddress(addr, proxyFunc)
	if err != nil {
		return nil, fmt.Errorf("HTTP proxy: %w", err)
	}
	if proxyURL != nil {
		newAddr = proxyURL.Host
	}

	conn, err := net.Dial(netw, newAddr)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		// proxy is disabled if proxyURL is nil.
		conn, err = doHTTPConnectHandshake(conn, addr, proxyURL)
	}
	return conn, err
}
