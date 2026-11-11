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

package buflsp_test

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func TestShutdownExitClosesConnection(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		protoPath, err := filepath.Abs("testdata/diagnostics/valid.proto")
		require.NoError(t, err)
		clientJSONConn, _ := setupLSPServer(t, protoPath)
		ctx := t.Context()

		_, err = clientJSONConn.Call(ctx, protocol.MethodShutdown, nil, nil)
		require.NoError(t, err)
		require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodExit, nil))

		// The server closes its end of the connection on exit, which terminates
		// the client's connection.
		synctest.Wait()
		select {
		case <-clientJSONConn.Done():
		default:
			require.Fail(t, "server did not close the connection")
		}
	})
}

func TestMalformedCancelRequestKeepsConnection(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		protoPath, err := filepath.Abs("testdata/diagnostics/valid.proto")
		require.NoError(t, err)
		clientJSONConn, _ := setupLSPServer(t, protoPath)
		ctx := t.Context()

		// The id overflows an LSP integer (int32), so it cannot be parsed.
		require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodCancelRequest, map[string]any{
			"id": int64(3000000000),
		}))

		_, err = clientJSONConn.Call(ctx, protocol.MethodShutdown, nil, nil)
		require.NoError(t, err)
	})
}

func TestClientDisconnectCancelsRequests(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		protoPath, err := filepath.Abs("testdata/diagnostics/valid.proto")
		require.NoError(t, err)
		var serverStream *blockingWriteStream
		clientJSONConn, _, serverJSONConn := setupLSPServerWithStream(t, protoPath, func(stream jsonrpc2.Stream) jsonrpc2.Stream {
			serverStream = &blockingWriteStream{Stream: stream, unblock: make(chan struct{})}
			return serverStream
		})
		// Unblock writes so that the cleanup can close the connections even
		// if the server did not cancel the request.
		t.Cleanup(func() { close(serverStream.unblock) })
		ctx := t.Context()
		synctest.Wait()

		// The server's response to this call blocks until the request's
		// context is cancelled.
		serverStream.blocking.Store(true)
		callErr := make(chan error, 1)
		go func() {
			_, err := clientJSONConn.Call(ctx, protocol.MethodShutdown, nil, nil)
			callErr <- err
		}()
		synctest.Wait()
		// Disconnect the client by closing the underlying pipe.
		require.NoError(t, serverStream.Stream.Close())
		assert.Error(t, <-callErr)

		// Failing to read from the client cancels the request, so the server
		// connection finishes.
		synctest.Wait()
		select {
		case <-serverJSONConn.Done():
		default:
			require.Fail(t, "server connection is not done")
		}
	})
}

// blockingWriteStream is a [jsonrpc2.Stream] whose writes block until their
// context is done or unblock is closed, while blocking is set.
type blockingWriteStream struct {
	jsonrpc2.Stream
	blocking atomic.Bool
	unblock  chan struct{}
}

// Write implements [jsonrpc2.Stream].
func (s *blockingWriteStream) Write(ctx context.Context, message jsonrpc2.Message) (int64, error) {
	if s.blocking.Load() {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-s.unblock:
		}
	}
	return s.Stream.Write(ctx, message)
}
