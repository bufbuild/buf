// Copyright 2020-2025 Buf Technologies, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestSemanticTokens(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/semantic_tokens/test.proto")
	require.NoError(t, err)
	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	var semanticTokens *protocol.SemanticTokens
	_, err = clientJSONConn.Call(ctx, "textDocument/semanticTokens/full", protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &semanticTokens)
	require.NoError(t, err)
	require.NotNil(t, semanticTokens)
	require.NotEmpty(t, semanticTokens.Data)
	// Just lock in the existing behavior, so we know if something changes.
	expected := []uint32{
		4, 8, 4, 1, 0, 1, 2, 6, 0, 2, 0, 7, 2, 2, 0, 0, 5, 1, 2, 0,
		1, 2, 6, 0, 2, 0, 7, 4, 2, 0, 0, 7, 1, 2, 0, 1, 2, 4, 2, 0,
		0, 5, 4, 2, 0, 0, 7, 1, 2, 0, 3, 5, 4, 3, 0, 1, 2, 16, 4, 0,
		0, 19, 1, 4, 0, 1, 2, 10, 4, 0, 0, 13, 1, 4, 0, 1, 2, 9, 4, 0,
		0, 12, 1, 4, 0, 3, 8, 11, 5, 0, 1, 6, 7, 6, 0, 0, 8, 14, 6, 0,
		0, 25, 15, 6, 0, 3, 8, 14, 1, 0, 1, 2, 6, 0, 2, 0, 7, 7, 2, 0,
		0, 10, 1, 2, 0, 3, 8, 15, 1, 0, 1, 2, 4, 2, 0, 0, 5, 4, 2, 0,
		0, 7, 1, 2, 0, 3, 8, 10, 1, 1, 1, 9, 10, 7, 0, 1, 2, 6, 0, 2,
		0, 7, 2, 2, 0, 0, 5, 1, 2, 0,
	}
	assert.Equal(t, expected, semanticTokens.Data)
}
