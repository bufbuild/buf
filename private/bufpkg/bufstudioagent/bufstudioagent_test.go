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

package bufstudioagent

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	studiov1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/studio/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/proto"
)

const (
	echoPath  = "/echo.Service/EchoEcho"
	errorPath = "/error.Service/Error"
)

func TestPlainPostHandlerTLS(t *testing.T) {
	upstreamServerTLS := newTestConnectServer(t, true)
	defer upstreamServerTLS.Close()
	testPlainPostHandler(t, upstreamServerTLS)
	testPlainPostHandlerErrors(t, upstreamServerTLS)
}

func TestPlainPostHandlerH2C(t *testing.T) {
	upstreamServerH2C := newTestConnectServer(t, false)
	defer upstreamServerH2C.Close()
	testPlainPostHandler(t, upstreamServerH2C)
	testPlainPostHandlerErrors(t, upstreamServerH2C)
}

func testPlainPostHandler(t *testing.T, upstreamServer *httptest.Server) {
	agentServer := httptest.NewTLSServer(
		NewHandler(
			zaptest.NewLogger(t),
			"https://example.buf.build",
			upstreamServer.TLS,
			nil,
			map[string]string{"foo": "bar"},
			false,
		),
	)
	defer agentServer.Close()

	t.Run("content_type_grpc_proto", func(t *testing.T) {
		requestProto := &studiov1alpha1.InvokeRequest{
			Target: upstreamServer.URL + echoPath,
			Headers: goHeadersToProtoHeaders(http.Header{
				"Content-Type": []string{"application/grpc+proto"},
			}),
			Body: []byte("echothis"),
		}
		requestBytes := protoMarshalBase64(t, requestProto)
		request, err := http.NewRequest(http.MethodPost, agentServer.URL, bytes.NewReader(requestBytes))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		request.Header.Set("Origin", "https://example.buf.build")
		request.Header.Set("Foo", "foo-value")
		response, err := agentServer.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "https://example.buf.build", response.Header.Get("Access-Control-Allow-Origin"))
		responseBytes, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		invokeResponse := &studiov1alpha1.InvokeResponse{}
		protoUnmarshalBase64(t, responseBytes, invokeResponse)
		upstreamResponseHeaders := make(http.Header)
		addProtoHeadersToGoHeader(invokeResponse.Headers, upstreamResponseHeaders)
		addProtoHeadersToGoHeader(invokeResponse.Trailers, upstreamResponseHeaders)
		assert.Equal(t, "0", upstreamResponseHeaders.Get("grpc-status"))
		assert.Equal(t, []byte("echo: echothis"), invokeResponse.Body)
		assert.Equal(t, "foo-value", upstreamResponseHeaders.Get("Echo-Bar"))
	})

	t.Run("content_type_application_proto", func(t *testing.T) {
		requestProto := &studiov1alpha1.InvokeRequest{
			Target: upstreamServer.URL + echoPath,
			Headers: goHeadersToProtoHeaders(http.Header{
				"Content-Type": []string{"application/proto"},
			}),
			Body: []byte("echothis"),
		}
		requestBytes := protoMarshalBase64(t, requestProto)
		request, err := http.NewRequest(http.MethodPost, agentServer.URL, bytes.NewReader(requestBytes))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		request.Header.Set("Origin", "https://example.buf.build")
		request.Header.Set("Foo", "foo-value")
		response, err := agentServer.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "https://example.buf.build", response.Header.Get("Access-Control-Allow-Origin"))
		responseBytes, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		invokeResponse := &studiov1alpha1.InvokeResponse{}
		protoUnmarshalBase64(t, responseBytes, invokeResponse)
		upstreamResponseHeaders := make(http.Header)
		addProtoHeadersToGoHeader(invokeResponse.Headers, upstreamResponseHeaders)
		addProtoHeadersToGoHeader(invokeResponse.Trailers, upstreamResponseHeaders)
		assert.Equal(t, "", upstreamResponseHeaders.Get("grpc-status"))
		assert.Equal(t, []byte("echo: echothis"), invokeResponse.Body)
		assert.Equal(t, "foo-value", upstreamResponseHeaders.Get("Echo-Bar"))
	})
}

func testPlainPostHandlerErrors(t *testing.T, upstreamServer *httptest.Server) {
	agentServer := httptest.NewTLSServer(
		NewHandler(
			zaptest.NewLogger(t),
			"https://example.buf.build",
			upstreamServer.TLS,
			map[string]struct{}{"forbidden-header": {}},
			nil,
			false,
		),
	)
	defer agentServer.Close()

	t.Run("forbidden_header", func(t *testing.T) {
		requestProto := &studiov1alpha1.InvokeRequest{
			Target: upstreamServer.URL + echoPath,
			Headers: goHeadersToProtoHeaders(http.Header{
				"forbidden-header": []string{"<tokens>"},
			}),
		}
		requestBytes := protoMarshalBase64(t, requestProto)
		request, err := http.NewRequest(http.MethodPost, agentServer.URL, bytes.NewReader(requestBytes))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		response, err := agentServer.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("error_response", func(t *testing.T) {
		requestProto := &studiov1alpha1.InvokeRequest{
			Target: upstreamServer.URL + errorPath,
			Headers: goHeadersToProtoHeaders(http.Header{
				"Content-Type": []string{"application/grpc"},
			}),
			Body: []byte("something"),
		}
		requestBytes := protoMarshalBase64(t, requestProto)
		request, err := http.NewRequest(http.MethodPost, agentServer.URL, bytes.NewReader(requestBytes))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		response, err := agentServer.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		assert.Equal(t, http.StatusOK, response.StatusCode)
		responseBytes, err := io.ReadAll(response.Body)
		assert.NoError(t, err)
		invokeResponse := &studiov1alpha1.InvokeResponse{}
		protoUnmarshalBase64(t, responseBytes, invokeResponse)
		upstreamResponseHeaders := make(http.Header)
		addProtoHeadersToGoHeader(invokeResponse.Headers, upstreamResponseHeaders)
		addProtoHeadersToGoHeader(invokeResponse.Trailers, upstreamResponseHeaders)
		assert.Equal(t, strconv.Itoa(int(connect.CodeFailedPrecondition)), upstreamResponseHeaders.Get("grpc-status"))
		assert.Equal(t, "something", upstreamResponseHeaders.Get("grpc-message"))
	})

	t.Run("invalid_upstream", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		go func() {
			conn, err := listener.Accept()
			require.NoError(t, err)
			require.NoError(t, conn.Close())
		}()
		defer listener.Close()

		requestProto := &studiov1alpha1.InvokeRequest{
			Target: "http://" + listener.Addr().String(),
			Headers: goHeadersToProtoHeaders(http.Header{
				"Content-Type": []string{"application/grpc"},
			}),
		}
		requestBytes := protoMarshalBase64(t, requestProto)
		request, err := http.NewRequest(http.MethodPost, agentServer.URL, bytes.NewReader(requestBytes))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "text/plain")
		response, err := agentServer.Client().Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		assert.Equal(t, http.StatusBadGateway, response.StatusCode)
	})
}

func newTestConnectServer(t *testing.T, tls bool) *httptest.Server {
	mux := http.NewServeMux()
	// echoPath echoes all incoming headers (prefixed with "Echo-") and the
	// body bytes prefixed with "echo: "
	mux.Handle(echoPath, connect.NewUnaryHandler(
		echoPath,
		func(ctx context.Context, r *connect.Request[bytes.Buffer]) (*connect.Response[bytes.Buffer], error) {
			response := connect.NewResponse(bytes.NewBuffer(append([]byte("echo: "), r.Msg.Bytes()...)))
			for header, values := range r.Header() {
				for _, value := range values {
					response.Header().Add("Echo-"+header, value)
				}
			}
			return response, nil
		},
		connect.WithCodec(&bufferCodec{name: "proto"}),
	))
	// errorPath returns the body as error message with code failed precondition
	mux.Handle(errorPath, connect.NewUnaryHandler(
		errorPath,
		func(ctx context.Context, r *connect.Request[bytes.Buffer]) (*connect.Response[bytes.Buffer], error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(r.Msg.String()))
		},
		connect.WithCodec(&bufferCodec{name: "proto"}),
	))
	if tls {
		upstreamServerTLS := httptest.NewUnstartedServer(mux)
		upstreamServerTLS.EnableHTTP2 = true
		upstreamServerTLS.StartTLS()
		certpool := x509.NewCertPool()
		certpool.AddCert(upstreamServerTLS.Certificate())
		upstreamServerTLS.TLS.RootCAs = certpool
		return upstreamServerTLS
	}
	return httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
}

func protoMarshalBase64(t *testing.T, message proto.Message) []byte {
	protoBytes, err := protoencoding.NewWireMarshaler().Marshal(message)
	require.NoError(t, err)
	base64Bytes := make([]byte, base64.StdEncoding.EncodedLen(len(protoBytes)))
	base64.StdEncoding.Encode(base64Bytes, protoBytes)
	return base64Bytes
}

func protoUnmarshalBase64(t *testing.T, base64Bytes []byte, message proto.Message) {
	protoBytes := make([]byte, base64.StdEncoding.DecodedLen(len(base64Bytes)))
	actualLen, err := base64.StdEncoding.Decode(protoBytes, base64Bytes)
	require.NoError(t, err)
	protoBytes = protoBytes[:actualLen]
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(protoBytes, message))
}

func addProtoHeadersToGoHeader(fromHeaders []*studiov1alpha1.Headers, toHeaders http.Header) {
	for _, meta := range fromHeaders {
		for _, value := range meta.Value {
			toHeaders.Add(meta.Key, value)
		}
	}
}
