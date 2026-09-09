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

package internal

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"buf.build/go/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/require"
)

func TestReaderArchiveFetchDeduplication(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server, requestCount := newTestArchiveServer(t)
	reader := newTestHTTPReader(t)
	// Reads differing only in what is applied after the fetch share one fetch.
	for _, read := range []archiveTestRead{
		{subDirPath: "svc-a"},
		{subDirPath: "svc-b"},
		{subDirPath: "svc-a"},
		{stripComponents: 1},
	} {
		archiveRef, err := newArchiveRef(
			"targz",
			server.URL+"/archive.tar.gz",
			ArchiveTypeTar,
			CompressionTypeGzip,
			read.stripComponents,
			read.subDirPath,
		)
		require.NoError(t, err)
		readBucketCloser, bucketTargeting, err := reader.GetReadBucketCloser(
			ctx,
			newTestStdinContainer(),
			archiveRef,
		)
		require.NoError(t, err)
		if read.subDirPath != "" {
			// Each read is still scoped to its own subdirectory.
			require.Equal(t, read.subDirPath, bucketTargeting.SubDirPath())
			data, err := storage.ReadPath(ctx, readBucketCloser, read.subDirPath+"/test.proto")
			require.NoError(t, err)
			require.Equal(t, read.subDirPath, string(data))
		}
		require.NoError(t, readBucketCloser.Close())
	}
	require.Equal(t, int64(1), requestCount.Load())
}

func TestReaderFileFetchDeduplication(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	var requestCount atomic.Int64
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			requestCount.Add(1)
			_, err := responseWriter.Write([]byte("image"))
			require.NoError(t, err)
		}),
	)
	t.Cleanup(server.Close)
	reader := newTestHTTPReader(t)
	for range 3 {
		singleRef, err := newSingleRef("binpb", server.URL+"/image.binpb", CompressionTypeNone, nil)
		require.NoError(t, err)
		readCloser, err := reader.GetFile(ctx, newTestStdinContainer(), singleRef)
		require.NoError(t, err)
		data, err := io.ReadAll(readCloser)
		require.NoError(t, err)
		require.NoError(t, readCloser.Close())
		require.Equal(t, "image", string(data))
	}
	require.Equal(t, int64(1), requestCount.Load())
}

func TestReaderFileFetchNoDeduplicationForStdin(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	reader := NewReader(slogtestext.NewLogger(t), storageos.NewProvider(), WithReaderStdio())
	container := app.NewContainer(nil, strings.NewReader("image"), nil, nil)
	singleRef, err := newSingleRef("binpb", "-", CompressionTypeNone, nil)
	require.NoError(t, err)
	readCloser, err := reader.GetFile(ctx, container, singleRef)
	require.NoError(t, err)
	data, err := io.ReadAll(readCloser)
	require.NoError(t, err)
	require.NoError(t, readCloser.Close())
	require.Equal(t, "image", string(data))
	// Stdin is a stream, so it is not cached: the second read is exhausted.
	readCloser, err = reader.GetFile(ctx, container, singleRef)
	require.NoError(t, err)
	data, err = io.ReadAll(readCloser)
	require.NoError(t, err)
	require.NoError(t, readCloser.Close())
	require.Empty(t, string(data))
}

func TestReaderGitCloneDeduplication(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name           string
		reads          []gitTestRead
		expectedClones int
	}{
		{
			name: "same repository different subdirs is one clone",
			reads: []gitTestRead{
				{gitName: git.NewBranchName("main"), subDirPath: "svc-a"},
				{gitName: git.NewBranchName("main"), subDirPath: "svc-b"},
			},
			expectedClones: 1,
		},
		{
			name: "no name is one clone",
			reads: []gitTestRead{
				{subDirPath: "svc-a"},
				{subDirPath: "svc-b"},
			},
			expectedClones: 1,
		},
		{
			// These share a String, so the key holds the fetch identity.
			name: "ref and branch with the same value are separate clones",
			reads: []gitTestRead{
				{gitName: git.NewBranchName("main")},
				{gitName: git.NewRefName("main")},
				{gitName: git.NewRefNameWithBranch("main", "dev")},
			},
			expectedClones: 3,
		},
		{
			name: "different depth is separate clones",
			reads: []gitTestRead{
				{gitName: git.NewBranchName("main"), depth: 1},
				{gitName: git.NewBranchName("main"), depth: 50},
			},
			expectedClones: 2,
		},
		{
			name: "different recurse submodules is separate clones",
			reads: []gitTestRead{
				{gitName: git.NewBranchName("main")},
				{gitName: git.NewBranchName("main"), recurseSubmodules: true},
			},
			expectedClones: 2,
		},
		{
			// With a filter the subdirectory is a sparse checkout.
			name: "different subdirs with a filter are separate clones",
			reads: []gitTestRead{
				{gitName: git.NewBranchName("main"), subDirPath: "svc-a", filter: "blob:none"},
				{gitName: git.NewBranchName("main"), subDirPath: "svc-b", filter: "blob:none"},
			},
			expectedClones: 2,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			cloner := &testCloner{}
			reader := NewReader(
				slogtestext.NewLogger(t),
				storageos.NewProvider(),
				WithReaderGit(cloner),
			)
			for _, read := range testCase.reads {
				depth := read.depth
				if depth == 0 {
					depth = 1
				}
				gitRef, err := newGitRef(
					"git",
					"https://github.com/foo/bar.git",
					read.gitName,
					depth,
					read.recurseSubmodules,
					read.subDirPath,
					read.filter,
				)
				require.NoError(t, err)
				readBucketCloser, _, err := reader.GetReadBucketCloser(
					ctx,
					newTestStdinContainer(),
					gitRef,
				)
				require.NoError(t, err)
				require.NoError(t, readBucketCloser.Close())
			}
			require.Equal(t, testCase.expectedClones, cloner.cloneCount)
		})
	}
}

type archiveTestRead struct {
	subDirPath      string
	stripComponents uint32
}

type gitTestRead struct {
	gitName           git.Name
	depth             uint32
	recurseSubmodules bool
	subDirPath        string
	filter            string
}

// testCloner writes fixed contents on every clone and counts calls.
type testCloner struct {
	cloneCount int
}

func (c *testCloner) CloneToBucket(
	ctx context.Context,
	_ app.EnvContainer,
	_ string,
	_ uint32,
	writeBucket storage.WriteBucket,
	_ git.CloneToBucketOptions,
) error {
	c.cloneCount++
	return putTestArchiveFiles(ctx, writeBucket)
}

// newTestArchiveServer serves a gzipped tarball of putTestArchiveFiles, and
// counts requests.
func newTestArchiveServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	ctx := t.Context()
	readWriteBucket := storagemem.NewReadWriteBucket()
	require.NoError(t, putTestArchiveFiles(ctx, readWriteBucket))
	buffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(buffer)
	require.NoError(t, storagearchive.Tar(ctx, readWriteBucket, gzipWriter))
	require.NoError(t, gzipWriter.Close())
	archiveData := buffer.Bytes()
	var requestCount atomic.Int64
	server := httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			requestCount.Add(1)
			_, err := responseWriter.Write(archiveData)
			require.NoError(t, err)
		}),
	)
	t.Cleanup(server.Close)
	return server, &requestCount
}

// putTestArchiveFiles writes a workspace with one module per subdirectory. Each
// file's contents are the name of the subdirectory containing it.
func putTestArchiveFiles(ctx context.Context, writeBucket storage.WriteBucket) error {
	if err := storage.PutPath(ctx, writeBucket, "buf.yaml", []byte("version: v2\n")); err != nil {
		return err
	}
	for _, subDirPath := range []string{"svc-a", "svc-b"} {
		if err := storage.PutPath(
			ctx,
			writeBucket,
			filepath.ToSlash(filepath.Join(subDirPath, "test.proto")),
			[]byte(subDirPath),
		); err != nil {
			return err
		}
	}
	return nil
}

func newTestHTTPReader(t *testing.T) Reader {
	t.Helper()
	return NewReader(
		slogtestext.NewLogger(t),
		storageos.NewProvider(),
		WithReaderHTTP(http.DefaultClient, httpauth.NewNopAuthenticator()),
	)
}

func newTestStdinContainer() app.EnvStdinContainer {
	return app.NewContainer(nil, nil, nil, nil)
}
