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
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestFormatting(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := []struct {
		name           string
		protoFile      string
		expectEdits    bool
		expectNumEdits int
	}{
		{
			name:           "format_unformatted_file",
			protoFile:      "unformatted.proto",
			expectEdits:    true,
			expectNumEdits: 1,
		},
		{
			name:           "format_already_formatted_file",
			protoFile:      "formatted.proto",
			expectEdits:    false,
			expectNumEdits: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testProtoPath, err := filepath.Abs(filepath.Join("testdata/format", tt.protoFile))
			require.NoError(t, err)
			clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
			var textEdits []protocol.TextEdit
			_, formatErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentFormatting, protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
			}, &textEdits)
			require.NoError(t, formatErr)
			assert.Len(t, textEdits, tt.expectNumEdits)
			if tt.expectEdits {
				expectedFormatted := getExpectedFormattedContent(t, ctx, testProtoPath)
				assert.Equal(t, expectedFormatted, textEdits[0].NewText)
				assert.Equal(t, uint32(0), textEdits[0].Range.Start.Line)
				assert.Equal(t, uint32(0), textEdits[0].Range.Start.Character)
			}
		})
	}
}

func getExpectedFormattedContent(t *testing.T, ctx context.Context, protoPath string) string {
	t.Helper()
	dir := filepath.Dir(protoPath)
	bucket, err := storageos.NewProvider().NewReadWriteBucket(dir)
	require.NoError(t, err)
	formattedBucket, err := bufformat.FormatBucket(ctx, bucket)
	require.NoError(t, err)
	formatted, err := storage.ReadPath(ctx, formattedBucket, filepath.Base(protoPath))
	require.NoError(t, err)
	return string(formatted)
}
