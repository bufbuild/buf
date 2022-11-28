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

package bufplugindocker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
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

const (
	examplePluginIdentity = "buf.build/library/go"
	examplePluginVersion  = "v1.0.0"
	dockerVersion         = "1.41"
)

var (
	dockerEnabled = false
	imagePattern  = regexp.MustCompile("^(?P<image>[^/]+/[^/]+/[^/:]+)(?::(?P<tag>[^/]+))?(?:/(?P<op>[^/]+))?$")
)

func TestBuildSuccess(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerClient := createClient(t)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", examplePluginIdentity, examplePluginVersion)
	require.Nilf(t, err, "failed to build docker plugin")
	assert.Truef(
		t,
		strings.HasPrefix(response.Image, pluginsImagePrefix+examplePluginIdentity+":"),
		"image name should begin with: %q, found: %q",
		examplePluginIdentity,
		response.Image,
	)
	assert.NotEmptyf(t, response.ImageID, "expected non-empty image id")
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
	_, err := buildDockerPlugin(t, dockerClient, "testdata/failure/Dockerfile", examplePluginIdentity, examplePluginVersion)
	assert.NotNil(t, err)
}

func TestPushSuccess(t *testing.T) {
	t.Parallel()
	server := newDockerServer(t, dockerVersion)
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		WithHost("tcp://"+listenerAddr),
		WithVersion(dockerVersion),
	)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go", examplePluginVersion)
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotNil(t, response)
	pushResponse, err := dockerClient.Push(context.Background(), response.Image, &RegistryAuthConfig{})
	require.Nilf(t, err, "failed to push docker plugin")
	require.NotNil(t, pushResponse)
	assert.NotEmpty(t, pushResponse.Digest)
}

func TestPushError(t *testing.T) {
	t.Parallel()
	server := newDockerServer(t, dockerVersion)
	// Send back an error on ImagePush (still return 200 OK).
	server.pushErr = errors.New("failed to push image")
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		WithHost("tcp://"+listenerAddr),
		WithVersion(dockerVersion),
	)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go", examplePluginVersion)
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotNil(t, response)
	_, err = dockerClient.Push(context.Background(), response.Image, &RegistryAuthConfig{})
	require.NotNil(t, err, "expected error")
	assert.Equal(t, server.pushErr.Error(), err.Error())
}

func TestBuildError(t *testing.T) {
	t.Parallel()
	server := newDockerServer(t, dockerVersion)
	// Send back an error on ImageBuild (still return 200 OK).
	server.buildErr = errors.New("failed to build image")
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		WithHost("tcp://"+listenerAddr),
		WithVersion(dockerVersion),
	)
	_, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go", examplePluginVersion)
	require.NotNilf(t, err, "expected error during build")
	assert.Equal(t, server.buildErr.Error(), err.Error())
}

func TestBuildArgs(t *testing.T) {
	t.Parallel()
	server := newDockerServer(t, dockerVersion)
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(
		t,
		WithHost("tcp://"+listenerAddr),
		WithVersion(dockerVersion),
	)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile", listenerAddr+"/library/go", examplePluginVersion)
	require.Nil(t, err)
	assert.Len(t, server.builtImages, 1)
	assert.Equal(t, server.builtImages[strings.TrimPrefix(response.ImageID, "sha256:")].args, map[string]string{
		"PLUGIN_VERSION": examplePluginVersion,
	})
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

func createClient(t testing.TB, options ...ClientOption) Client {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.Nilf(t, err, "failed to create zap logger")
	dockerClient, err := NewClient(logger, options...)
	require.Nilf(t, err, "failed to create client")
	t.Cleanup(func() {
		if err := dockerClient.Close(); err != nil {
			t.Errorf("failed to close client: %v", err)
		}
	})
	return dockerClient
}

func buildDockerPlugin(t testing.TB, dockerClient Client, dockerfilePath string, pluginIdentity string, pluginVersion string) (*BuildResponse, error) {
	t.Helper()
	dockerfile, err := os.Open(dockerfilePath)
	require.Nilf(t, err, "failed to open dockerfile")
	pluginName, err := bufpluginref.PluginIdentityForString(pluginIdentity)
	require.Nilf(t, err, "failed to create plugin identity")
	pluginConfig := &bufpluginconfig.Config{Name: pluginName, PluginVersion: pluginVersion}
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
	httpServer    *httptest.Server
	h2Server      *http2.Server
	h2Handler     http.Handler
	t             testing.TB
	versionPrefix string
	buildErr      error
	pushErr       error
	// protects builtImages
	mu          sync.RWMutex
	builtImages map[string]*builtImage
}

type builtImage struct {
	tags []string
	args map[string]string
}

func newDockerServer(t testing.TB, version string) *dockerServer {
	t.Helper()
	versionPrefix := "/v" + version
	dockerServer := &dockerServer{
		t:             t,
		builtImages:   make(map[string]*builtImage),
		versionPrefix: versionPrefix,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/session", dockerServer.sessionHandler)
	mux.HandleFunc(versionPrefix+"/build", dockerServer.buildHandler)
	mux.HandleFunc(versionPrefix+"/images/", dockerServer.imagesHandler)
	dockerServer.h2Server = &http2.Server{}
	dockerServer.h2Handler = h2c.NewHandler(mux, dockerServer.h2Server)
	dockerServer.httpServer = httptest.NewUnstartedServer(dockerServer.h2Handler)
	dockerServer.httpServer.Start()
	t.Cleanup(dockerServer.httpServer.Close)
	return dockerServer
}

func (d *dockerServer) sessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if strings.EqualFold(r.Header.Get("Connection"), "upgrade") && r.ProtoMajor < 2 {
		conn, err := h2cUpgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()
		d.h2Server.ServeConn(conn, &http2.ServeConnOpts{
			Context:        r.Context(),
			Handler:        d.h2Handler,
			UpgradeRequest: r,
		})
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// h2cUpgrade taken from x/net/http2/h2c/h2c.go implementation
// Docker client doesn't send HTTP2-Settings header so upgrade doesn't work out of the box.
func h2cUpgrade(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("h2c: connection does not support Hijack")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	if _, err := rw.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: h2c\r\n\r\n")); err != nil {
		return nil, err
	}
	return newBufConn(conn, rw), nil
}

func newBufConn(conn net.Conn, rw *bufio.ReadWriter) net.Conn {
	rw.Flush()
	if rw.Reader.Buffered() == 0 {
		// If there's no buffered data to be read,
		// we can just discard the bufio.ReadWriter.
		return conn
	}
	return &bufConn{conn, rw.Reader}
}

// bufConn wraps a net.Conn, but reads drain the bufio.Reader first.
type bufConn struct {
	net.Conn
	*bufio.Reader
}

func (c *bufConn) Read(p []byte) (int, error) {
	if c.Reader == nil {
		return c.Conn.Read(p)
	}
	n := c.Reader.Buffered()
	if n == 0 {
		c.Reader = nil
		return c.Conn.Read(p)
	}
	if n < len(p) {
		p = p[:n]
	}
	return c.Reader.Read(p)
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
	buildArgs := make(map[string]string)
	if buildArgsJSON := r.URL.Query().Get("buildargs"); len(buildArgsJSON) > 0 {
		if err := json.Unmarshal([]byte(buildArgsJSON), &buildArgs); err != nil {
			d.t.Error("failed to read build args:", err)
		}
	}
	d.builtImages[imageID] = &builtImage{tags: r.URL.Query()["t"], args: buildArgs}
}

func (d *dockerServer) imagesHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		d.t.Error("failed to discard body:", err)
	}
	pathSuffix := strings.TrimPrefix(r.URL.Path, d.versionPrefix+"/images/")
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
	auxJSON, err := json.Marshal(map[string]any{
		"Tag":    r.URL.Query()["tag"][0],
		"Digest": "sha256:" + stringid.GenerateRandomID(),
		"Size":   123,
	})
	if err != nil {
		d.writeError(w, err)
		return
	}
	if err := json.NewEncoder(w).Encode(&jsonmessage.JSONMessage{
		Progress: &jsonmessage.JSONProgress{},
		Aux:      (*json.RawMessage)(&auxJSON),
	}); err != nil {
		d.t.Error("failed to write JSON:", err)
	}
	if _, err := w.Write([]byte("\r\n")); err != nil {
		d.t.Error("failed to write CRLF:", err)
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
