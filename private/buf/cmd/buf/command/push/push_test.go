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

	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/bufmanifest"
	"github.com/bufbuild/buf/private/bufpkg/buftransport"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushManifest(t *testing.T) {
	t.Parallel()
	testPushManifest(
		t,
		"success",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		nil,   // no push service error set
		false, // no --create flag
		"",
		"",
	)
	testPushManifest(
		t,
		"missing local module pin",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: nil,
		},
		nil,   // no push service error set
		false, // no --create flag
		"",
		"Missing local module pin",
	)
	testPushManifest(
		t,
		"registry error",
		nil,
		nil,   // no push service error set
		false, // no --create flag
		"",
		"bad request",
	)
}

func TestPushManifestCreate(t *testing.T) {
	t.Parallel()
	testPushManifest(
		t,
		"repository not found, successfully create public repository",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		connect.NewError(connect.CodeNotFound, errors.New("repository not found")),
		true,
		"public",
		"",
	)
	testPushManifest(
		t,
		"repository not found, successfully create private repository",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		connect.NewError(connect.CodeNotFound, errors.New("repository not found")),
		true,
		"private",
		"",
	)
	testPushManifest(
		t,
		"repository not found, fail to create repository, no visibility",
		&registryv1alpha1.PushManifestAndBlobsResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		connect.NewError(connect.CodeNotFound, errors.New("repository not found")),
		true,
		"",
		"Failure: --create-visibility is required if --create is set.",
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
	server := createServer(t, mock, nil) // not testing the create on the push manifest code path
	err := appRun(
		t,
		map[string][]byte{
			"buf.yaml":  bufYAML(t, server.URL, "owner", "repo"),
			"foo.proto": nil,
			"bar.proto": nil,
			"baz.file":  nil,
		},
	)
	require.NoError(t, err)
	request := mock.PushManifestRequest()
	require.NotNil(t, request)
	requestManifest := request.Manifest
	blob, err := bufmanifest.NewBlobFromProto(requestManifest)
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
	_, blobs, err := bufmanifest.ToProtoManifestAndBlobs(ctx, m, blobSet)
	require.NoError(t, err)
	assert.Equal(t, 2, len(blobs))
	digests := make(map[string]struct{})
	for _, blob := range blobs {
		assert.Equal(
			t,
			modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
			blob.Digest.DigestType,
		)
		hexDigest := hex.EncodeToString(blob.Digest.Digest)
		assert.NotContains(t, digests, hexDigest, "duplicated blob")
		digests[hexDigest] = struct{}{}
	}
}

type mockPushService struct {
	t *testing.T

	// protects pushManifestRequest and called
	sync.RWMutex

	// number of times Push is called. we only want to return error once to test repository
	// create flow.
	called int

	pushManifestRequest       *registryv1alpha1.PushManifestAndBlobsRequest
	pushManifestResponse      *registryv1alpha1.PushManifestAndBlobsResponse
	pushManifestResponseError error
}

var _ registryv1alpha1connect.PushServiceHandler = (*mockPushService)(nil)

func newMockPushService(t *testing.T) *mockPushService {
	return &mockPushService{
		t: t,
	}
}

func (m *mockPushService) Push(
	context.Context,
	*connect.Request[registryv1alpha1.PushRequest],
) (*connect.Response[registryv1alpha1.PushResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("Push RPC should not be called, use PushManifestAndBlobs RPC instead"))
}

func (m *mockPushService) PushManifestAndBlobs(
	_ context.Context,
	req *connect.Request[registryv1alpha1.PushManifestAndBlobsRequest],
) (*connect.Response[registryv1alpha1.PushManifestAndBlobsResponse], error) {
	m.Lock()
	defer m.Unlock()
	m.called++
	m.pushManifestRequest = req.Msg
	assert.NotNil(m.t, req.Msg.Manifest, "missing manifest")
	resp := m.pushManifestResponse
	if resp == nil {
		return nil, errors.New("bad request")
	}
	if m.pushManifestResponseError != nil {
		if m.called == 1 {
			return nil, m.pushManifestResponseError
		}
	}
	return connect.NewResponse(resp), nil
}

func (m *mockPushService) PushManifestRequest() *registryv1alpha1.PushManifestAndBlobsRequest {
	m.RLock()
	defer m.RUnlock()
	return m.pushManifestRequest
}

type mockRepositoryService struct {
	t *testing.T
}

var _ registryv1alpha1connect.RepositoryServiceHandler = (*mockRepositoryService)(nil)

func newMockRepositoryService(t *testing.T) *mockRepositoryService {
	return &mockRepositoryService{
		t: t,
	}
}

func (m *mockRepositoryService) CreateRepositoryByFullName(
	_ context.Context,
	req *connect.Request[registryv1alpha1.CreateRepositoryByFullNameRequest],
) (*connect.Response[registryv1alpha1.CreateRepositoryByFullNameResponse], error) {
	if req.Msg.FullName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("full repository name required"))
	}
	if req.Msg.Visibility != registryv1alpha1.Visibility_VISIBILITY_PUBLIC && req.Msg.Visibility != registryv1alpha1.Visibility_VISIBILITY_PRIVATE {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid visibility"))
	}
	split := strings.Split(req.Msg.FullName, "/")
	assert.Equal(m.t, 2, len(split))
	return connect.NewResponse(&registryv1alpha1.CreateRepositoryByFullNameResponse{
		Repository: &registryv1alpha1.Repository{
			Name:       split[1],
			Visibility: req.Msg.Visibility,
		},
	}), nil
}

func (m *mockRepositoryService) GetRepository(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositoryRequest]) (*connect.Response[registryv1alpha1.GetRepositoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) GetRepositoryByFullName(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositoryByFullNameRequest]) (*connect.Response[registryv1alpha1.GetRepositoryByFullNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) ListRepositories(_ context.Context, _ *connect.Request[registryv1alpha1.ListRepositoriesRequest]) (*connect.Response[registryv1alpha1.ListRepositoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) ListUserRepositories(_ context.Context, _ *connect.Request[registryv1alpha1.ListUserRepositoriesRequest]) (*connect.Response[registryv1alpha1.ListUserRepositoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) ListRepositoriesUserCanAccess(_ context.Context, _ *connect.Request[registryv1alpha1.ListRepositoriesUserCanAccessRequest]) (*connect.Response[registryv1alpha1.ListRepositoriesUserCanAccessResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) ListOrganizationRepositories(_ context.Context, _ *connect.Request[registryv1alpha1.ListOrganizationRepositoriesRequest]) (*connect.Response[registryv1alpha1.ListOrganizationRepositoriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) DeleteRepository(_ context.Context, _ *connect.Request[registryv1alpha1.DeleteRepositoryRequest]) (*connect.Response[registryv1alpha1.DeleteRepositoryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) DeleteRepositoryByFullName(_ context.Context, _ *connect.Request[registryv1alpha1.DeleteRepositoryByFullNameRequest]) (*connect.Response[registryv1alpha1.DeleteRepositoryByFullNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) DeprecateRepositoryByName(_ context.Context, _ *connect.Request[registryv1alpha1.DeprecateRepositoryByNameRequest]) (*connect.Response[registryv1alpha1.DeprecateRepositoryByNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) UndeprecateRepositoryByName(_ context.Context, _ *connect.Request[registryv1alpha1.UndeprecateRepositoryByNameRequest]) (*connect.Response[registryv1alpha1.UndeprecateRepositoryByNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) GetRepositoriesByFullName(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositoriesByFullNameRequest]) (*connect.Response[registryv1alpha1.GetRepositoriesByFullNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) SetRepositoryContributor(_ context.Context, _ *connect.Request[registryv1alpha1.SetRepositoryContributorRequest]) (*connect.Response[registryv1alpha1.SetRepositoryContributorResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) ListRepositoryContributors(_ context.Context, _ *connect.Request[registryv1alpha1.ListRepositoryContributorsRequest]) (*connect.Response[registryv1alpha1.ListRepositoryContributorsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) GetRepositoryContributor(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositoryContributorRequest]) (*connect.Response[registryv1alpha1.GetRepositoryContributorResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) GetRepositorySettings(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositorySettingsRequest]) (*connect.Response[registryv1alpha1.GetRepositorySettingsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) UpdateRepositorySettingsByName(_ context.Context, _ *connect.Request[registryv1alpha1.UpdateRepositorySettingsByNameRequest]) (*connect.Response[registryv1alpha1.UpdateRepositorySettingsByNameResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func (m *mockRepositoryService) GetRepositoriesMetadata(_ context.Context, _ *connect.Request[registryv1alpha1.GetRepositoriesMetadataRequest]) (*connect.Response[registryv1alpha1.GetRepositoriesMetadataResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented"))
}

func createServer(t *testing.T, mockPushService *mockPushService, mockRepositoryService *mockRepositoryService) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(
		registryv1alpha1connect.NewPushServiceHandler(mockPushService),
	)
	mux.Handle(
		registryv1alpha1connect.NewRepositoryServiceHandler(mockRepositoryService),
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
	args ...string,
) error {
	const appName = "test"
	defaultArgs := []string{appName, "-#format=tar"} // using stdin as a tar
	args = append(defaultArgs, args...)
	return appcmd.Run(
		context.Background(),
		app.NewContainer(
			amendEnv(
				internaltesting.NewEnvFunc(t),
				func(env map[string]string) map[string]string {
					env["BUF_TOKEN"] = "invalid"
					buftransport.SetDisableAPISubdomain(env)
					injectConfig(t, appName, env)
					return env
				},
			)(appName),
			tarball(files),
			os.Stdout,
			os.Stderr,
			args...,
		),
		NewCommand(
			appName,
			appflag.NewBuilder(appName),
		),
	)
}

func testPushManifest(
	t *testing.T,
	desc string,
	resp *registryv1alpha1.PushManifestAndBlobsResponse,
	pushServiceError error,
	create bool,
	createVisibility string,
	errorMsg string,
) {
	t.Helper()
	mock := newMockPushService(t)
	mock.pushManifestResponse = resp
	mock.pushManifestResponseError = pushServiceError
	mockRepositoryService := newMockRepositoryService(t)
	var args []string
	if create {
		args = append(args, "--create", "--create-visibility="+createVisibility)
	}

	server := createServer(t, mock, mockRepositoryService)
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		err := appRun(
			t,
			map[string][]byte{
				"buf.yaml":  bufYAML(t, server.URL, "owner", "repo"),
				"foo.proto": nil,
				"bar.proto": nil,
			},
			args...,
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
