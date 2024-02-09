// Copyright 2020-2024 Buf Technologies, Inc.
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

// Generated. DO NOT EDIT.

package bufconfig

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
)

func TestWalkBasic(t *testing.T) {
	t.Parallel()

	testWalk(
		t,
		map[string][]byte{
			"buf.yaml":      []byte(`version: v1`),
			"buf.mod":       []byte(`version: v1`),
			"buf.lock":      []byte(`version: v1`),
			"buf.gen.yaml":  []byte(`version: v1`),
			"buf.work.yaml": []byte(`version: v1`),
			"buf.work":      []byte(`version: v1`),
			"unknown.yaml":  []byte(`unknown`),
		},
		map[string]FileInfo{
			"buf.yaml":      newFileInfo(FileVersionV1, FileTypeBufYAML),
			"buf.mod":       newFileInfo(FileVersionV1, FileTypeBufYAML),
			"buf.lock":      newFileInfo(FileVersionV1, FileTypeBufLock),
			"buf.gen.yaml":  newFileInfo(FileVersionV1, FileTypeBufGenYAML),
			"buf.work.yaml": newFileInfo(FileVersionV1, FileTypeBufWorkYAML),
			"buf.work":      newFileInfo(FileVersionV1, FileTypeBufWorkYAML),
		},
	)
}

func TestWalkNoVersion(t *testing.T) {
	t.Parallel()

	testWalk(
		t,
		map[string][]byte{
			"buf.yaml":      []byte(``),
			"buf.mod":       []byte(``),
			"buf.lock":      []byte(``),
			"buf.gen.yaml":  []byte(``),
			"buf.work.yaml": []byte(``),
			"buf.work":      []byte(``),
		},
		map[string]FileInfo{
			"buf.yaml":      newFileInfo(FileVersionV1Beta1, FileTypeBufYAML),
			"buf.mod":       newFileInfo(FileVersionV1Beta1, FileTypeBufYAML),
			"buf.lock":      newFileInfo(FileVersionV1Beta1, FileTypeBufLock),
			"buf.gen.yaml":  newFileInfo(FileVersionV1Beta1, FileTypeBufGenYAML),
			"buf.work.yaml": newFileInfo(FileVersionV1, FileTypeBufWorkYAML),
			"buf.work":      newFileInfo(FileVersionV1, FileTypeBufWorkYAML),
		},
	)
}

func testWalk(t *testing.T, pathToData map[string][]byte, expectedPathToFileInfo map[string]FileInfo) {
	ctx := context.Background()
	bucket, err := storagemem.NewReadBucket(pathToData)
	require.NoError(t, err)
	pathToFileInfo, err := testGetPathToFileInfo(ctx, bucket)
	require.NoError(t, err)
	require.Equal(t, expectedPathToFileInfo, pathToFileInfo)
}

func testGetPathToFileInfo(ctx context.Context, bucket storage.ReadBucket) (map[string]FileInfo, error) {
	pathToFileInfo := make(map[string]FileInfo)
	err := WalkFileInfos(ctx, bucket, func(path string, fileInfo FileInfo) error {
		pathToFileInfo[path] = fileInfo
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pathToFileInfo, nil
}
