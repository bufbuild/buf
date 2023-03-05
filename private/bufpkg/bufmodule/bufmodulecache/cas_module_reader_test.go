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
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/connect-go"
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
	moduleManifest, blobs := createSampleManifestAndBlobs(t)
	moduleBlob, err := moduleManifest.Blob()
	require.NoError(t, err)
	testModule, err := bufmodule.NewModuleForManifestAndBlobSet(context.Background(), moduleManifest, blobs)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)

	moduleReader := newCASModuleReader(storageBucket, &testModuleReader{module: testModule}, func(_ string) registryv1alpha1connect.RepositoryServiceClient {
		return &testRepositoryServiceClient{}
	}, zaptest.NewLogger(t), &testVerbosePrinter{t: t})
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"",
		"abcd",
		moduleBlob.Digest().String(),
		time.Now(),
	)
	require.NoError(t, err)
	_, err = moduleReader.GetModule(context.Background(), pin)
	require.NoError(t, err)
	assert.Equal(t, 1, moduleReader.stats.Count())
	assert.Equal(t, 0, moduleReader.stats.Hits())
	verifyCache(t, storageBucket, pin, moduleManifest, blobs)

	_, err = moduleReader.GetModule(context.Background(), pin)
	require.NoError(t, err)
	assert.Equal(t, 2, moduleReader.stats.Count())
	assert.Equal(t, 1, moduleReader.stats.Hits()) // We should have a cache hit the second time
	verifyCache(t, storageBucket, pin, moduleManifest, blobs)
}

func TestCASModuleReaderNoDigest(t *testing.T) {
	t.Parallel()
	moduleManifest, blobs := createSampleManifestAndBlobs(t)
	testModule, err := bufmodule.NewModuleForManifestAndBlobSet(context.Background(), moduleManifest, blobs)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	moduleReader := newCASModuleReader(storageBucket, &testModuleReader{module: testModule}, func(_ string) registryv1alpha1connect.RepositoryServiceClient {
		return &testRepositoryServiceClient{}
	}, zaptest.NewLogger(t), &testVerbosePrinter{t: t})
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"",
		"abcd",
		"",
		time.Now(),
	)
	require.NoError(t, err)
	_, err = moduleReader.GetModule(context.Background(), pin)
	require.NoError(t, err)
	assert.Equal(t, 1, moduleReader.stats.Count())
	assert.Equal(t, 0, moduleReader.stats.Hits())
	verifyCache(t, storageBucket, pin, moduleManifest, blobs)
}

func TestCASModuleReaderDigestMismatch(t *testing.T) {
	t.Parallel()
	moduleManifest, blobs := createSampleManifestAndBlobs(t)
	testModule, err := bufmodule.NewModuleForManifestAndBlobSet(context.Background(), moduleManifest, blobs)
	require.NoError(t, err)
	storageProvider := storageos.NewProvider()
	storageBucket, err := storageProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	moduleReader := newCASModuleReader(storageBucket, &testModuleReader{module: testModule}, func(_ string) registryv1alpha1connect.RepositoryServiceClient {
		return &testRepositoryServiceClient{}
	}, zaptest.NewLogger(t), &testVerbosePrinter{t: t})
	pin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"test",
		"ping",
		"",
		"abcd",
		"shake256:"+strings.Repeat("00", 64), // Digest which doesn't match module's digest
		time.Now(),
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
	moduleManifest *manifest.Manifest,
	blobs *manifest.BlobSet,
) {
	t.Helper()
	ctx := context.Background()
	moduleCacheDir := normalpath.Join(pin.Remote(), pin.Owner(), pin.Repository())
	// {remote}/{owner}/{repo}/manifests/{..}/{....} => should return manifest contents
	moduleBlob, err := moduleManifest.Blob()
	require.NoError(t, err)
	verifyBlobContents(t, bucket, normalpath.Join(moduleCacheDir, blobsDir), moduleBlob)
	for _, path := range moduleManifest.Paths() {
		protoDigest, found := moduleManifest.DigestFor(path)
		require.True(t, found)
		protoBlob, found := blobs.BlobFor(protoDigest.String())
		require.True(t, found)
		// {remote}/{owner}/{repo}/blobs/{..}/{....} => should return proto blob contents
		verifyBlobContents(t, bucket, normalpath.Join(moduleCacheDir, blobsDir), protoBlob)
	}
	f, err := bucket.Get(ctx, normalpath.Join(moduleCacheDir, commitsDir, pin.Commit()))
	require.NoError(t, err)
	defer f.Close()
	commitContents, err := io.ReadAll(f)
	require.NoError(t, err)
	// {remote}/{owner}/{repo}/commits/{commit} => should return digest string format
	assert.Equal(t, []byte(moduleBlob.Digest().String()), commitContents)
}

func createSampleManifestAndBlobs(t *testing.T) (*manifest.Manifest, *manifest.BlobSet) {
	t.Helper()
	blob, err := manifest.NewMemoryBlobFromReader(strings.NewReader(pingProto))
	require.NoError(t, err)
	var moduleManifest manifest.Manifest
	err = moduleManifest.AddEntry("connect/ping/v1/ping.proto", *blob.Digest())
	require.NoError(t, err)
	blobSet, err := manifest.NewBlobSet(context.Background(), []manifest.Blob{blob})
	require.NoError(t, err)
	return &moduleManifest, blobSet
}

func verifyBlobContents(t *testing.T, bucket storage.ReadWriteBucket, basedir string, blob manifest.Blob) {
	t.Helper()
	moduleHexDigest := blob.Digest().Hex()
	r, err := blob.Open(context.Background())
	require.NoError(t, err)
	var bb bytes.Buffer
	_, err = io.Copy(&bb, r)
	require.NoError(t, err)
	f, err := bucket.Get(context.Background(), normalpath.Join(basedir, moduleHexDigest[:2], moduleHexDigest[2:]))
	require.NoError(t, err)
	defer f.Close()
	cachedModule, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, bb.Bytes(), cachedModule)
}

type testModuleReader struct {
	module bufmodule.Module
}

var _ bufmodule.ModuleReader = (*testModuleReader)(nil)

func (t *testModuleReader) GetModule(_ context.Context, _ bufmoduleref.ModulePin) (bufmodule.Module, error) {
	return t.module, nil
}

type testRepositoryServiceClient struct {
	registryv1alpha1connect.UnimplementedRepositoryServiceHandler
}

var _ registryv1alpha1connect.RepositoryServiceClient = (*testRepositoryServiceClient)(nil)

func (t *testRepositoryServiceClient) GetRepositoryByFullName(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetRepositoryByFullNameRequest],
) (*connect.Response[registryv1alpha1.GetRepositoryByFullNameResponse], error) {
	return connect.NewResponse(&registryv1alpha1.GetRepositoryByFullNameResponse{
		Repository: &registryv1alpha1.Repository{},
	}), nil
}

type testVerbosePrinter struct {
	t *testing.T
}

var _ verbose.Printer = (*testVerbosePrinter)(nil)

func (t testVerbosePrinter) Printf(format string, args ...interface{}) {
	t.t.Logf(format, args...)
}
