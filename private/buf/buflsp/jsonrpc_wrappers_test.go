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

package buflsp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
)

func TestOrderedAsyncHandler(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		const messageCount = 10

		var (
			lock    sync.Mutex
			started []int
			handled []int
		)
		release := make(chan struct{})
		serverStream, clientStream := jsonrpc2.NewChannelStreamPair(0)
		serverConn := jsonrpc2.NewConn(serverStream)
		serverConn.Go(ctx, orderedAsyncHandler(func(_ context.Context, request *jsonrpc2.Request) (any, error) {
			var index int
			if err := json.Unmarshal(request.Params(), &index); err != nil {
				return nil, err
			}
			lock.Lock()
			started = append(started, index)
			lock.Unlock()
			<-release
			lock.Lock()
			handled = append(handled, index)
			lock.Unlock()
			return nil, nil
		}))
		clientConn := jsonrpc2.NewConn(clientStream)
		clientConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)
		t.Cleanup(func() {
			assert.NoError(t, clientConn.Close())
			assert.NoError(t, serverConn.Close())
		})
		// Release the handler before closing the connections, so a failed
		// assertion is reported instead of a deadlock.
		releaseOnce := sync.OnceFunc(func() { close(release) })
		t.Cleanup(releaseOnce)

		for index := range messageCount {
			require.NoError(t, clientConn.Notify(ctx, "test/notify", index))
		}
		// All messages are queued, and only the first has started.
		synctest.Wait()
		lock.Lock()
		assert.Equal(t, []int{0}, started)
		lock.Unlock()

		releaseOnce()
		synctest.Wait()
		expected := make([]int, messageCount)
		for index := range expected {
			expected[index] = index
		}
		lock.Lock()
		defer lock.Unlock()
		assert.Equal(t, expected, started)
		assert.Equal(t, expected, handled)
	})
}

func TestReadCancelStreamCancelsRequestsOnDisconnect(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		serverStream, clientStream := jsonrpc2.NewChannelStreamPair(0)
		readCtx, readCancel := context.WithCancel(ctx)
		t.Cleanup(readCancel)
		serverConn := jsonrpc2.NewConn(newReadCancelStream(serverStream, readCancel))
		queuedErr := make(chan error, 1)
		serverConn.Go(readCtx, orderedAsyncHandler(func(ctx context.Context, request *jsonrpc2.Request) (any, error) {
			if request.Method() == "test/block" {
				<-ctx.Done()
				return nil, nil
			}
			queuedErr <- ctx.Err()
			return nil, nil
		}))
		clientConn := jsonrpc2.NewConn(clientStream)
		clientConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

		require.NoError(t, clientConn.Notify(ctx, "test/block", nil))
		require.NoError(t, clientConn.Notify(ctx, "test/queued", nil))
		synctest.Wait()
		require.NoError(t, clientConn.Close())

		synctest.Wait()
		select {
		case <-serverConn.Done():
		default:
			require.Fail(t, "server connection is not done")
		}
		select {
		case err := <-queuedErr:
			assert.ErrorIs(t, err, context.Canceled)
		default:
			require.Fail(t, "queued request was not handled")
		}
	})
}
