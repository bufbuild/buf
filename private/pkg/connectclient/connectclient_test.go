// Copyright 2020-2026 Buf Technologies, Inc.
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

package connectclient

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	reflectionv1 "github.com/bufbuild/buf/private/gen/proto/go/grpc/reflection/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResponseSize = 64 * 1024

// TestMakeReadMaxBytes verifies that a read limit set on the Config applies to
// every client it builds, and that an option passed to Make overrides it.
//
// The CLI relies on both: a default bound is set once on the Config, and remote
// generation raises it for its own client.
func TestMakeReadMaxBytes(t *testing.T) {
	t.Parallel()

	server := newReflectionServer(t)
	factory := func(httpClient connect.HTTPClient, address string, options ...connect.ClientOption) *connect.Client[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse] {
		return connect.NewClient[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse](
			httpClient,
			address+"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
			options...,
		)
	}

	t.Run("ConfigLimitApplies", func(t *testing.T) {
		t.Parallel()
		config := NewConfig(server.Client(), WithClientOptions(connect.WithReadMaxBytes(testResponseSize/2)))
		_, err := call(t, Make(config, server.URL, factory))
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	})

	t.Run("MakeOptionOverridesConfigLimit", func(t *testing.T) {
		t.Parallel()
		config := NewConfig(server.Client(), WithClientOptions(connect.WithReadMaxBytes(testResponseSize/2)))
		client := Make(config, server.URL, factory, connect.WithReadMaxBytes(testResponseSize*2))
		_, err := call(t, client)
		require.NoError(t, err)
	})

	t.Run("NoLimitByDefault", func(t *testing.T) {
		t.Parallel()
		config := NewConfig(server.Client())
		_, err := call(t, Make(config, server.URL, factory))
		require.NoError(t, err)
	})
}

func call(
	t *testing.T,
	client *connect.Client[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse],
) (*connect.Response[reflectionv1.ServerReflectionResponse], error) {
	t.Helper()
	return client.CallUnary(t.Context(), connect.NewRequest(&reflectionv1.ServerReflectionRequest{}))
}

// newReflectionServer returns a server whose single unary method always replies
// with a response of at least testResponseSize bytes.
func newReflectionServer(t *testing.T) *httptest.Server {
	t.Helper()
	handler := connect.NewUnaryHandler(
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		func(
			_ context.Context,
			_ *connect.Request[reflectionv1.ServerReflectionRequest],
		) (*connect.Response[reflectionv1.ServerReflectionResponse], error) {
			return connect.NewResponse(reflectionv1.ServerReflectionResponse_builder{
				ValidHost: string(bytes.Repeat([]byte("a"), testResponseSize)),
			}.Build()), nil
		},
	)
	mux := http.NewServeMux()
	mux.Handle("/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}
