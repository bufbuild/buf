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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestShutdownExitClosesConnection(t *testing.T) {
	t.Parallel()

	protoPath, err := filepath.Abs("testdata/diagnostics/valid.proto")
	require.NoError(t, err)
	clientJSONConn, _ := setupLSPServer(t, protoPath)
	ctx := t.Context()

	_, err = clientJSONConn.Call(ctx, protocol.MethodShutdown, nil, nil)
	require.NoError(t, err)
	require.NoError(t, clientJSONConn.Notify(ctx, protocol.MethodExit, nil))

	// The server closes its end of the connection on exit, which terminates the
	// client's connection.
	select {
	case <-clientJSONConn.Done():
	case <-time.After(10 * time.Second):
		require.Fail(t, "timed out waiting for the server to close the connection")
	}
}
