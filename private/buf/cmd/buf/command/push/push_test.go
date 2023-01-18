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

package push

import (
	"archive/tar"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/buftransport"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	testPush(
		t,
		"success",
		&registryv1alpha1.PushResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		"",
	)
	testPush(
		t,
		"missing local module pin",
		&registryv1alpha1.PushResponse{
			LocalModulePin: nil,
		},
		"Missing local module pin",
	)
	testPush(
		t,
		"registry error",
		nil,
		"bad request",
	)
}

func TestPushManifest(t *testing.T) {
	testPushManifest(
		t,
		"success tamper proofing enabled",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		"",
	)
}

func TestPushManifestIsSmallerBucket(t *testing.T) {
	// Assert push only manifests with only the files needed to build the
	// module as described by configuration and file extension.
	t.Parallel()
	mock := newMockPushService(t)
	mock.pushManifestResponse = &registryv1alpha1.PushManifestAndBlobsResponse{
		LocalModulePin: &registryv1alpha1.LocalModulePin{},
	}
	server := createServer(t, mock)
	err := appRun(
		t,
		map[string][]byte{
			"buf.yaml":  bufYAML(t, server.URL, "owner", "repo"),
			"foo.proto": nil,
			"bar.proto": nil,
			"baz.file":  nil,
		},
		true, // tamperProofingEnabled
	)
	require.NoError(t, err)
	request := mock.PushManifestRequest()
	require.NotNil(t, request)
	requestManifest := request.Manifest
	blob, err := manifest.NewBlobFromProto(requestManifest)
	require.NoError(t, err)
	ctx := context.Background()
	reader, err := blob.Open(ctx)
	require.NoError(t, err)
	defer reader.Close()
	m, err := manifest.NewFromReader(reader)
	require.NoError(t, err)
	_, ok := m.DigestFor("baz.file")
	assert.False(t, ok, "baz.file should not be pushed")
}

func TestBucketBlobs(t *testing.T) {
	t.Parallel()
	bucket, err := storagemem.NewReadBucket(
		map[string][]byte{
			"buf.yaml":  bufYAML(t, "foo", "bar", "repo"),
			"foo.proto": nil,
			"bar.proto": nil,
		},
	)
	require.NoError(t, err)
	ctx := context.Background()
	m, blobSet, err := manifest.NewFromBucket(ctx, bucket)
	require.NoError(t, err)
	_, blobs, err := manifest.ToProtoManifestAndBlobs(ctx, m, blobSet)
	require.NoError(t, err)
	assert.Equal(t, 2, len(blobs))
	digests := make(map[string]struct{})
	for _, blob := range blobs {
		assert.Equal(
			t,
			modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
			blob.Hash.DigestType,
		)
		hexDigest := hex.EncodeToString(blob.Hash.Digest)
		assert.NotContains(t, digests, hexDigest, "duplicated blob")
		digests[hexDigest] = struct{}{}
	}
}

type mockPushService struct {
	t *testing.T

	// protects pushRequest / pushManifestRequest
	sync.RWMutex

	// for testing with tamper proofing disabled
	pushRequest  *registryv1alpha1.PushRequest
	pushResponse *registryv1alpha1.PushResponse

	// for testing with tamper proofing enabled
	pushManifestRequest  *registryv1alpha1.PushManifestAndBlobsRequest
	pushManifestResponse *registryv1alpha1.PushManifestAndBlobsResponse
}

var _ registryv1alpha1connect.PushServiceHandler = (*mockPushService)(nil)

func newMockPushService(t *testing.T) *mockPushService {
	return &mockPushService{
		t: t,
	}
}

// Push pushes.
func (m *mockPushService) Push(
	_ context.Context,
	req *connect_go.Request[registryv1alpha1.PushRequest],
) (*connect_go.Response[registryv1alpha1.PushResponse], error) {
	m.Lock()
	defer m.Unlock()
	m.pushRequest = req.Msg
	assert.NotNil(m.t, req.Msg.Module, "missing module")
	resp := m.pushResponse
	if resp == nil {
		return nil, errors.New("bad request")
	}
	return connect_go.NewResponse(resp), nil
}

func (m *mockPushService) PushRequest() *registryv1alpha1.PushRequest {
	m.RLock()
	defer m.RUnlock()
	return m.pushRequest
}

func (m *mockPushService) PushManifestAndBlobs(
	_ context.Context,
	req *connect_go.Request[registryv1alpha1.PushManifestAndBlobsRequest],
) (*connect_go.Response[registryv1alpha1.PushManifestAndBlobsResponse], error) {
	m.Lock()
	defer m.Unlock()
	m.pushManifestRequest = req.Msg
	assert.NotNil(m.t, req.Msg.Manifest, "missing manifest")
	resp := m.pushManifestResponse
	if resp == nil {
		return nil, errors.New("bad request")
	}
	return connect_go.NewResponse(resp), nil
}

func (m *mockPushService) PushManifestRequest() *registryv1alpha1.PushManifestAndBlobsRequest {
	m.RLock()
	defer m.RUnlock()
	return m.pushManifestRequest
}

func createServer(t *testing.T, mock *mockPushService) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(
		registryv1alpha1connect.NewPushServiceHandler(mock),
	)
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func appRun(
	t *testing.T,
	files map[string][]byte,
	tamperProofingEnabled bool,
) error {
	const appName = "test"
	return appcmd.Run(
		context.Background(),
		app.NewContainer(
			amendEnv(
				internaltesting.NewEnvFunc(t),
				func(env map[string]string) map[string]string {
					env["BUF_TOKEN"] = "invalid"
					buftransport.SetDisableAPISubdomain(env)
					injectConfig(t, appName, env)
					if tamperProofingEnabled {
						env[bufcli.BetaEnableTamperProofingEnvKey] = "1"
					}
					return env
				},
			)(appName),
			tarball(files),
			os.Stdout,
			os.Stderr,
			appName,        // push ran as appName, aka "test"
			"-#format=tar", // using stdin as a tar
		),
		NewCommand(
			appName,
			appflag.NewBuilder(appName),
		),
	)
}

func testPush(
	t *testing.T,
	desc string,
	resp *registryv1alpha1.PushResponse,
	errorMsg string,
) {
	t.Helper()
	mock := newMockPushService(t)
	mock.pushResponse = resp
	server := createServer(t, mock)
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		err := appRun(
			t,
			map[string][]byte{
				"buf.yaml":  bufYAML(t, server.URL, "owner", "repo"),
				"foo.proto": nil,
				"bar.proto": nil,
			},
			false, // tamperProofingEnabled
		)
		if errorMsg == "" {
			assert.NoError(t, err)
		} else {
			assert.ErrorContains(t, err, errorMsg)
		}
	})
}

func testPushManifest(
	t *testing.T,
	desc string,
	resp *registryv1alpha1.PushManifestAndBlobsResponse,
	errorMsg string,
) {
	t.Helper()
	mock := newMockPushService(t)
	mock.pushManifestResponse = resp
	server := createServer(t, mock)
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		err := appRun(
			t,
			map[string][]byte{
				"buf.yaml":  bufYAML(t, server.URL, "owner", "repo"),
				"foo.proto": nil,
				"bar.proto": nil,
			},
			true,
		)
		if errorMsg == "" {
			assert.NoError(t, err)
		} else {
			assert.ErrorContains(t, err, errorMsg)
		}
	})
}

// tarball returns a tar stream of files[path] = content.
func tarball(files map[string][]byte) io.ReadCloser {
	pr, pw := io.Pipe()
	go tarballWriter(files, pw)
	return pr
}

func tarballWriter(files map[string][]byte, out io.WriteCloser) {
	tw := tar.NewWriter(out)
	defer tw.Close()
	defer out.Close()
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0666,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return
		}
		if _, err := tw.Write(content); err != nil {
			return
		}
	}
}

// bufYAML returns a buf.yaml content for a given remote URL, org, and repo.
func bufYAML(t *testing.T, remoteURL, org, repo string) []byte {
	remote, err := url.Parse(remoteURL)
	require.NoError(t, err)
	conf := "version: v1\n"
	conf += fmt.Sprintf("name: %s/%s/%s\n", remote.Host, org, repo)
	return []byte(conf)
}

// injectConfig writes an app's config.yaml that disables TLS.
func injectConfig(t *testing.T, appName string, env map[string]string) {
	configDir := env[strings.ToUpper(appName)+"_CONFIG_DIR"]
	confFile, err := os.Create(path.Join(configDir, "config.yaml"))
	require.NoError(t, err)
	defer confFile.Close()
	_, err = io.WriteString(confFile, `
version: v1
tls:
  use: false
`)
	require.NoError(t, err)
}

// amendEnv calls sideEffects after the env generator function constructs an
// environment. The environment from the last sideEffect call is returned.
func amendEnv(
	envGen func(string) map[string]string,
	sideEffects ...func(map[string]string) map[string]string,
) func(string) map[string]string {
	return func(use string) map[string]string {
		env := envGen(use)
		for _, sideEffect := range sideEffects {
			env = sideEffect(env)
		}
		return env
	}
}
