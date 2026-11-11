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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
)

func TestOrderedAsyncHandler(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	const messageCount = 10

	var (
		lock    sync.Mutex
		handled []int
	)
	allHandled := make(chan struct{})
	serverStream, clientStream := jsonrpc2.NewChannelStreamPair(0)
	serverConn := jsonrpc2.NewConn(serverStream)
	serverConn.Go(ctx, orderedAsyncHandler(func(_ context.Context, request *jsonrpc2.Request) (any, error) {
		var index int
		if err := json.Unmarshal(request.Params(), &index); err != nil {
			return nil, err
		}
		// Earlier messages take longer to handle, so they would finish last if
		// they were handled concurrently.
		time.Sleep(time.Duration(messageCount-index) * time.Millisecond)
		lock.Lock()
		defer lock.Unlock()
		handled = append(handled, index)
		if len(handled) == messageCount {
			close(allHandled)
		}
		return nil, nil
	}))
	clientConn := jsonrpc2.NewConn(clientStream)
	clientConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)
	t.Cleanup(func() {
		assert.NoError(t, clientConn.Close())
		assert.NoError(t, serverConn.Close())
	})

	for index := range messageCount {
		require.NoError(t, clientConn.Notify(ctx, "test/notify", index))
	}
	select {
	case <-allHandled:
	case <-time.After(10 * time.Second):
		require.Fail(t, "timed out waiting for messages to be handled")
	}

	expected := make([]int, messageCount)
	for index := range expected {
		expected[index] = index
	}
	lock.Lock()
	defer lock.Unlock()
	assert.Equal(t, expected, handled)
}
