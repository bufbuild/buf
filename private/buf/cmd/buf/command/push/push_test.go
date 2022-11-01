// Copyright 2020-2022 Buf Technologies, Inc.
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/buftransport"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPushService struct {
	t         *testing.T
	callbacks map[string]func(*registryv1alpha1.PushRequest)
	called    map[string]struct{}
	resp      map[string]*registryv1alpha1.PushResponse
}

var _ registryv1alpha1connect.PushServiceHandler = (*mockPushService)(nil)

func newMockPushService(t *testing.T) *mockPushService {
	return &mockPushService{
		t:         t,
		callbacks: make(map[string]func(*registryv1alpha1.PushRequest)),
		called:    make(map[string]struct{}),
		resp:      make(map[string]*registryv1alpha1.PushResponse),
	}
}

// Push pushes.
func (m *mockPushService) Push(
	_ context.Context,
	req *connect_go.Request[registryv1alpha1.PushRequest],
) (*connect_go.Response[registryv1alpha1.PushResponse], error) {
	owner := req.Msg.Owner
	cb, ok := m.callbacks[owner]
	if ok {
		cb(req.Msg)
		m.called[owner] = struct{}{}
	}
	resp := m.resp[owner]
	if resp != nil {
		return &connect_go.Response[registryv1alpha1.PushResponse]{
			Msg: resp,
		}, nil
	}
	return nil, errors.New("bad request")
}

func (m *mockPushService) respond(owner string, resp *registryv1alpha1.PushResponse) {
	m.resp[owner] = resp
}

func (m *mockPushService) callback(
	owner string,
	cb func(*registryv1alpha1.PushRequest),
) {
	m.callbacks[owner] = cb
}

func (m *mockPushService) assertAllCallbacksCalled() {
	for k := range m.callbacks {
		_, ok := m.called[k]
		assert.True(m.t, ok)
	}
}

func TestPush(t *testing.T) {
	server, mock := pushService(t)
	t.Cleanup(func() {
		server.Close()
		mock.assertAllCallbacksCalled()
	})

	testPush(
		t,
		"success",
		server.URL,
		mock,
		&registryv1alpha1.PushResponse{
			LocalModulePin: &registryv1alpha1.LocalModulePin{},
		},
		"",
	)
	testPush(
		t,
		"missing local module pin",
		server.URL,
		mock,
		&registryv1alpha1.PushResponse{
			LocalModulePin: nil,
		},
		"Missing local module pin",
	)
	testPush(
		t,
		"registry error",
		server.URL,
		mock,
		nil,
		"bad request",
	)
}

func pushService(t *testing.T) (*httptest.Server, *mockPushService) {
	mock := newMockPushService(t)
	mux := http.NewServeMux()
	mux.Handle(
		registryv1alpha1connect.NewPushServiceHandler(mock),
	)
	return httptest.NewServer(mux), mock
}

func testPush(
	t *testing.T,
	desc string,
	URL string,
	mock *mockPushService,
	resp *registryv1alpha1.PushResponse,
	errorMsg string,
) {
	t.Helper()
	owner := strings.ReplaceAll(desc, " ", "_")
	mock.respond(owner, resp)
	mock.callback(owner, func(req *registryv1alpha1.PushRequest) {
		assert.NotNil(t, req.Module, "missing module")
	})
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		appName := "test"
		err := appcmd.Run(
			context.Background(),
			app.NewContainer(
				ammendEnv(
					internaltesting.NewEnvFunc(t),
					func(env map[string]string) map[string]string {
						env["BUF_TOKEN"] = "invalid"
						buftransport.SetDisableAPISubdomain(env)
						injectConfig(t, appName, env)
						return env
					},
				)(appName),
				tarball(map[string][]byte{
					"buf.yaml":  bufYAML(t, URL, owner, "repo"),
					"foo.proto": nil,
					"bar.proto": nil,
				}),
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

// ammendEnv calls sideEffects after the env generator function constructs an
// environment. The environment from the last sideEffect call is returned.
func ammendEnv(
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
