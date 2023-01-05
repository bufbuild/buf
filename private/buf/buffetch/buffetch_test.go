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

package buffetch

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch/internal"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRoundTripBin(t *testing.T) {
	testRoundTripLocalFile(
		t,
		"file.bin",
		[]byte("one"),
		formatBin,
		internal.CompressionTypeNone,
	)
}

func TestRoundTripBinGz(t *testing.T) {
	testRoundTripLocalFile(
		t,
		"file.bin.gz",
		[]byte("one"),
		formatBin,
		internal.CompressionTypeGzip,
	)
}

func TestRoundTripBinZst(t *testing.T) {
	testRoundTripLocalFile(
		t,
		"file.bin.zst",
		[]byte("one"),
		formatBin,
		internal.CompressionTypeZstd,
	)
}

func testRoundTripLocalFile(
	t *testing.T,
	filename string,
	expectedData []byte,
	expectedFormat string,
	expectedCompressionType internal.CompressionType,
) {
	t.Parallel()

	logger := zap.NewNop()
	refParser := newRefParser(logger)
	reader := testNewFetchReader(logger)
	writer := testNewFetchWriter(logger)

	ctx := context.Background()
	container := app.NewContainer(nil, nil, nil, nil)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, filename)

	parsedRef, err := refParser.getParsedRef(ctx, filePath, allFormats)
	require.NoError(t, err)
	require.Equal(t, expectedFormat, parsedRef.Format())
	fileRef, ok := parsedRef.(internal.FileRef)
	require.True(t, ok)
	require.Equal(t, expectedCompressionType, fileRef.CompressionType())

	writeCloser, err := writer.PutFile(ctx, container, fileRef)
	require.NoError(t, err)
	_, err = writeCloser.Write(expectedData)
	require.NoError(t, err)
	require.NoError(t, writeCloser.Close())

	readCloser, err := reader.GetFile(ctx, container, fileRef)
	require.NoError(t, err)
	actualData, err := io.ReadAll(readCloser)
	require.NoError(t, err)
	require.NoError(t, readCloser.Close())

	require.Equal(t, string(expectedData), string(actualData))
}

func testNewFetchReader(logger *zap.Logger) internal.Reader {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	return internal.NewReader(
		logger,
		storageosProvider,
		internal.WithReaderLocal(),
	)
}

func testNewFetchWriter(logger *zap.Logger) internal.Writer {
	return internal.NewWriter(
		logger,
		internal.WithWriterLocal(),
	)
}
