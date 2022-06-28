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

package bufplugindocker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufplugindocker"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var dockerEnabled = false

const examplePluginIdentity = "plugins.buf.build/library/go"

func TestBuildSuccess(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerClient := createClient(t)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", examplePluginIdentity)
	require.Nilf(t, err, "failed to build docker plugin")
	assert.Truef(
		t,
		strings.HasPrefix(response.Image, examplePluginIdentity+":"),
		"image name should begin with: %q, found: %q",
		examplePluginIdentity,
		response.Image,
	)
	assert.NotEmptyf(t, response.Digest, "expected non-empty image digest")
}

func TestBuildFailure(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerClient := createClient(t)
	_, err := buildDockerPlugin(t, dockerClient, "testdata/failure/Dockerfile", examplePluginIdentity)
	assert.NotNil(t, err)
}

func TestPushSuccess(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerVersion := "1.41"
	server := newDockerServer(t, dockerVersion)
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		bufplugindocker.WithHost("tcp://"+listenerAddr),
		bufplugindocker.WithVersion(dockerVersion),
	)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go")
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotNil(t, response)
	pushResponse, err := dockerClient.Push(context.Background(), response.Image, &bufplugindocker.RegistryAuthConfig{})
	require.Nilf(t, err, "failed to push docker plugin")
	require.NotNil(t, pushResponse)
}

func TestPushError(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerVersion := "1.41"
	server := newDockerServer(t, dockerVersion)
	// Send back an error on ImagePush (still return 200 OK).
	server.pushErr = errors.New("failed to push image")
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		bufplugindocker.WithHost("tcp://"+listenerAddr),
		bufplugindocker.WithVersion(dockerVersion),
	)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go")
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotNil(t, response)
	_, err = dockerClient.Push(context.Background(), response.Image, &bufplugindocker.RegistryAuthConfig{})
	require.NotNil(t, err, "expected error")
	assert.Equal(t, server.pushErr.Error(), err.Error())
}

func TestBuildError(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerVersion := "1.41"
	server := newDockerServer(t, dockerVersion)
	// Send back an error on ImageBuild (still return 200 OK).
	server.buildErr = errors.New("failed to build image")
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		bufplugindocker.WithHost("tcp://"+listenerAddr),
		bufplugindocker.WithVersion(dockerVersion),
	)
	_, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go")
	require.NotNilf(t, err, "expected error during build")
	assert.Equal(t, server.buildErr.Error(), err.Error())
}

func TestMain(m *testing.M) {
	// TODO: We may wish to indicate we want to run Docker tests even if the CLI ping command fails.
	// This would force CI builds to run these tests, but still allow users without Docker installed to run tests.
	if cli, err := client.NewClientWithOpts(client.FromEnv); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := cli.Ping(ctx); err == nil {
			dockerEnabled = true
		}
		_ = cli.Close()
	}
	if dockerEnabled && runtime.GOOS == "windows" {
		// Windows runners don't support building Linux images - need to disable for now.
		dockerEnabled = false
	}
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func createClient(t testing.TB, options ...bufplugindocker.ClientOption) bufplugindocker.Client {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.Nilf(t, err, "failed to create zap logger")
	dockerClient, err := bufplugindocker.NewClient(logger, options...)
	require.Nilf(t, err, "failed to create client")
	t.Cleanup(func() {
		if err := dockerClient.Close(); err != nil {
			t.Errorf("failed to close client: %v", err)
		}
	})
	return dockerClient
}

func buildDockerPlugin(t testing.TB, dockerClient bufplugindocker.Client, dockerfilePath string, pluginIdentity string) (*bufplugindocker.BuildResponse, error) {
	t.Helper()
	dockerfile, err := os.Open(dockerfilePath)
	require.Nilf(t, err, "failed to open dockerfile")
	pluginName, err := bufpluginref.PluginIdentityForString(pluginIdentity)
	require.Nilf(t, err, "failed to create plugin identity")
	pluginConfig := &bufpluginconfig.Config{Name: pluginName}
	response, err := dockerClient.Build(context.Background(), dockerfile, pluginConfig)
	if err == nil {
		t.Cleanup(func() {
			if _, err := dockerClient.Delete(context.Background(), response.Image); err != nil {
				t.Errorf("failed to delete image: %q", response.Image)
			}
		})
	}
	return response, err
}

// dockerServer allows testing some failure flows by simulating the responses to Docker CLI commands.
type dockerServer struct {
	httpServer *httptest.Server
	t          testing.TB
	buildErr   error
	pushErr    error
	// protects builtImages
	mu          sync.RWMutex
	builtImages map[string]*builtImage
}

type builtImage struct {
	tags []string
}

func newDockerServer(t testing.TB, version string) *dockerServer {
	dockerServer := &dockerServer{t: t, builtImages: make(map[string]*builtImage)}
	versionPrefix := "/v" + version
	mux := http.NewServeMux()
	mux.HandleFunc("/session", dockerServer.sessionHandler)
	mux.HandleFunc(versionPrefix+"/build", dockerServer.buildHandler)
	mux.HandleFunc(versionPrefix+"/images/", dockerServer.imagesHandler)
	dockerServer.httpServer = httptest.NewServer(h2c.NewHandler(mux, &http2.Server{}))
	t.Cleanup(dockerServer.httpServer.Close)
	return dockerServer
}

func (d *dockerServer) sessionHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		d.t.Error("failed to discard body:", err)
	}
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if strings.EqualFold(r.Header.Get("Connection"), "upgrade") && r.ProtoMajor < 2 {
		w.WriteHeader(http.StatusSwitchingProtocols)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (d *dockerServer) buildHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		d.t.Error("failed to discard body:", err)
	}
	w.WriteHeader(http.StatusOK)
	if d.buildErr != nil {
		d.writeError(w, d.buildErr)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	imageID := stringid.GenerateRandomID()
	d.builtImages[imageID] = &builtImage{tags: r.URL.Query()["t"]}
}

func (d *dockerServer) imagesHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		d.t.Error("failed to discard body:", err)
	}
	d.t.Logf("%s %s", r.Method, r.URL.Path)
	pathSuffix := strings.TrimPrefix(r.URL.Path, "/v1.41/images/")
	imagePattern := regexp.MustCompile("^(?P<image>[^/]+/[^/]+/[^/:]+)(?::(?P<tag>[^/]+))?(?:/(?P<op>[^/]+))?$")
	submatches := imagePattern.FindStringSubmatch(pathSuffix)
	if len(submatches) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	image, tag, operation := submatches[1], submatches[2], submatches[3]
	// ImageInspectWithRaw
	if r.Method == http.MethodGet && operation == "json" {
		d.mu.RLock()
		defer d.mu.RUnlock()
		foundImageID := d.findImageIDFromName(image + ":" + tag)
		if len(foundImageID) == 0 {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(&types.ImageInspect{
			ID:       "sha256:" + foundImageID,
			RepoTags: d.builtImages[foundImageID].tags,
		}); err != nil {
			d.t.Error("failed to encode image inspect response:", err)
		}

		return
	}
	// ImageRemove
	if r.Method == http.MethodDelete {
		d.mu.Lock()
		defer d.mu.Unlock()
		foundImageID := d.findImageIDFromName(image + ":" + tag)
		if len(foundImageID) == 0 {
			http.NotFound(w, r)
			return
		}
		delete(d.builtImages, foundImageID)
		if err := json.NewEncoder(w).Encode([]types.ImageDeleteResponseItem{
			{Deleted: "sha256:" + foundImageID},
		}); err != nil {
			d.t.Error("failed to encode image delete response:", err)
		}
		return
	}
	// ImagePush
	w.WriteHeader(http.StatusOK)
	if d.pushErr != nil {
		d.writeError(w, d.pushErr)
		return
	}
}

func (d *dockerServer) findImageIDFromName(name string) string {
	for imageID, builtImageInfo := range d.builtImages {
		for _, imageTag := range builtImageInfo.tags {
			if imageTag == name {
				return imageID
			}
		}
	}
	return ""
}

func (d *dockerServer) writeError(w http.ResponseWriter, err error) {
	if err := json.NewEncoder(w).Encode(&jsonmessage.JSONMessage{Error: &jsonmessage.JSONError{Message: err.Error()}}); err != nil {
		d.t.Error("failed to write json message:", err)
	}
	if _, err := w.Write([]byte{'\r', '\n'}); err != nil {
		d.t.Error("failed to write CRLF", err)
	}
}
