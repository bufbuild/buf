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

package bufmodulecache

import (
	"context"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const pingProto = `syntax = "proto3";

package connect.ping.v1;

message PingRequest {
  int64 number = 1;
  string text = 2;
}

message PingResponse {
  int64 number = 1;
  string text = 2;
}

service PingService {
  rpc Ping(PingRequest) returns (PingResponse) {}
}
`

func TestCASModuleReaderHappyPath(t *testing.T) {
	t.Parallel()
	fileSet := createSampleFileSet(t)
	manifestBlob, err := bufcas.ManifestToBlob(fileSet.Manifest())
	require.NoError(t, err)
	testModule, err := bufmodule.NewModuleForFileSet(context.Background(), fileSet)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)

	moduleReader := newCASModuleReader(
		storageBucket,
		&testModuleReader{module: testModule},
		zaptest.NewLogger(t),
		&testVerbosePrinter{t: t},
	)
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"abcd",
		manifestBlob.Digest().String(),
	)
	require.NoError(t, err)
	_, err = moduleReader.GetModule(context.Background(), pin) // non-cached
	require.NoError(t, err)
	assert.Equal(t, 1, moduleReader.stats.Count())
	assert.Equal(t, 0, moduleReader.stats.Hits())
	verifyCache(t, storageBucket, pin, fileSet)

	cachedMod, err := moduleReader.GetModule(context.Background(), pin)
	require.NoError(t, err)
	assertModuleIdentity(t, cachedMod, pin.IdentityString(), pin.Commit())
	assert.Equal(t, 2, moduleReader.stats.Count())
	assert.Equal(t, 1, moduleReader.stats.Hits()) // We should have a cache hit the second time
	verifyCache(t, storageBucket, pin, fileSet)
}

func TestCASModuleReaderNoDigest(t *testing.T) {
	t.Parallel()
	fileSet := createSampleFileSet(t)
	testModule, err := bufmodule.NewModuleForFileSet(context.Background(), fileSet)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	moduleReader := newCASModuleReader(
		storageBucket,
		&testModuleReader{module: testModule},
		zaptest.NewLogger(t),
		&testVerbosePrinter{t: t},
	)
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"abcd",
		"",
	)
	require.NoError(t, err)
	_, err = moduleReader.GetModule(context.Background(), pin)
	require.NoError(t, err)
	assert.Equal(t, 1, moduleReader.stats.Count())
	assert.Equal(t, 0, moduleReader.stats.Hits())
	verifyCache(t, storageBucket, pin, fileSet)
}

func TestCASModuleReaderDigestMismatch(t *testing.T) {
	t.Parallel()
	fileSet := createSampleFileSet(t)
	testModule, err := bufmodule.NewModuleForFileSet(context.Background(), fileSet)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	moduleReader := newCASModuleReader(
		storageBucket,
		&testModuleReader{module: testModule},
		zaptest.NewLogger(t),
		&testVerbosePrinter{t: t},
	)
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"abcd",
		"shake256:"+strings.Repeat("00", 64), // Digest which doesn't match module's digest
	)
	require.NoError(t, err)
	_, err = moduleReader.GetModule(context.Background(), pin)
	require.Error(t, err)
	numFiles := 0
	err = storageBucket.Walk(context.Background(), "", func(info storage.ObjectInfo) error {
		numFiles++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 0, numFiles) // Verify nothing written to cache on digest mismatch
}

func verifyCache(
	t *testing.T,
	bucket storage.ReadWriteBucket,
	pin bufmoduleref.ModulePin,
	fileSet bufcas.FileSet,
) {
	t.Helper()
	ctx := context.Background()
	moduleCacheDir := normalpath.Join(pin.Remote(), pin.Owner(), pin.Repository())
	// {remote}/{owner}/{repo}/manifests/{..}/{....} => should return manifest contents
	manifest := fileSet.Manifest()
	manifestBlob, err := bufcas.ManifestToBlob(manifest)
	require.NoError(t, err)
	verifyBlobContents(t, bucket, normalpath.Join(moduleCacheDir, blobsDir), manifestBlob)
	for _, fileNode := range manifest.FileNodes() {
		blob := fileSet.BlobSet().GetBlob(fileNode.Digest())
		require.NotNil(t, blob)
		// {remote}/{owner}/{repo}/blobs/{..}/{....} => should return proto blob contents
		verifyBlobContents(t, bucket, normalpath.Join(moduleCacheDir, blobsDir), blob)
	}
	f, err := bucket.Get(ctx, normalpath.Join(moduleCacheDir, commitsDir, pin.Commit()))
	require.NoError(t, err)
	defer f.Close()
	commitContents, err := io.ReadAll(f)
	require.NoError(t, err)
	// {remote}/{owner}/{repo}/commits/{commit} => should return digest string format
	assert.Equal(t, []byte(manifestBlob.Digest().String()), commitContents)
}

func createSampleFileSet(t *testing.T) bufcas.FileSet {
	t.Helper()
	blob, err := bufcas.NewBlobForContent(strings.NewReader(pingProto))
	require.NoError(t, err)
	fileNode, err := bufcas.NewFileNode("connect/ping/v1/ping.proto", blob.Digest())
	require.NoError(t, err)
	manifest, err := bufcas.NewManifest([]bufcas.FileNode{fileNode})
	require.NoError(t, err)
	blobSet, err := bufcas.NewBlobSet([]bufcas.Blob{blob})
	require.NoError(t, err)
	fileSet, err := bufcas.NewFileSet(manifest, blobSet)
	require.NoError(t, err)
	return fileSet
}

func verifyBlobContents(t *testing.T, bucket storage.ReadWriteBucket, basedir string, blob bufcas.Blob) {
	t.Helper()
	digestHex := hex.EncodeToString(blob.Digest().Value())
	f, err := bucket.Get(context.Background(), normalpath.Join(basedir, digestHex[:2], digestHex[2:]))
	require.NoError(t, err)
	defer f.Close()
	cachedModule, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, blob.Content(), cachedModule)
}

func assertModuleIdentity(t *testing.T, module bufmodule.Module, expectedModuleIdentity string, expectedCommit string) {
	require.NotNil(t, module)
	require.NotEmpty(t, expectedCommit)
	fileInfos, err := module.SourceFileInfos(context.Background())
	require.NoError(t, err)
	for _, fileInfo := range fileInfos {
		require.NotNil(t, fileInfo.ModuleIdentity())
		assert.Equalf(
			t, expectedModuleIdentity, fileInfo.ModuleIdentity().IdentityString(),
			"unexpected module identity for file %q", fileInfo.Path(),
		)
		assert.Equalf(
			t, expectedCommit, fileInfo.Commit(),
			"unexpected commit for file %q", fileInfo.Path(),
		)
	}
}

type testModuleReader struct {
	module bufmodule.Module
}

var _ bufmodule.ModuleReader = (*testModuleReader)(nil)

func (t *testModuleReader) GetModule(_ context.Context, _ bufmoduleref.ModulePin) (bufmodule.Module, error) {
	return t.module, nil
}

type testVerbosePrinter struct {
	t *testing.T
}

var _ verbose.Printer = (*testVerbosePrinter)(nil)

func (t testVerbosePrinter) Printf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
