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

package bufplugindocker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
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
	dockerVersion = "1.41"
)

var (
	imagePattern = regexp.MustCompile("^(?P<image>[^/]+/[^/]+/[^/:]+)(?::(?P<tag>[^/]+))?(?:/(?P<op>[^/]+))?$")
)

func TestPushSuccess(t *testing.T) {
	t.Parallel()
	server := newDockerServer(t, dockerVersion)
	listenerAddr := server.httpServer.Listener.Addr().String()
	dockerClient := createClient(t, WithHost("tcp://"+listenerAddr), WithVersion(dockerVersion))
	image, err := buildDockerPlugin(t, "testdata/success/Dockerfile", listenerAddr+"/library/go")
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotEmpty(t, image)
	pushResponse, err := dockerClient.Push(context.Background(), image, &RegistryAuthConfig{})
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
	dockerClient := createClient(t, WithHost("tcp://"+listenerAddr), WithVersion(dockerVersion))
	image, err := buildDockerPlugin(t, "testdata/success/Dockerfile", listenerAddr+"/library/go")
	require.Nilf(t, err, "failed to build docker plugin")
	require.NotEmpty(t, image)
	_, err = dockerClient.Push(context.Background(), image, &RegistryAuthConfig{})
	require.NotNil(t, err, "expected error")
	assert.Equal(t, server.pushErr.Error(), err.Error())
}

func TestMain(m *testing.M) {
	var dockerEnabled bool
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
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
	if dockerEnabled {
		os.Exit(m.Run())
	}
}

func createClient(t testing.TB, options ...ClientOption) Client {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.Nilf(t, err, "failed to create zap logger")
	dockerClient, err := NewClient(logger, "buf-cli-1.11.0", options...)
	require.Nilf(t, err, "failed to create client")
	t.Cleanup(func() {
		if err := dockerClient.Close(); err != nil {
			t.Errorf("failed to close client: %v", err)
		}
	})
	return dockerClient
}

func buildDockerPlugin(t testing.TB, dockerfilePath string, pluginIdentity string) (string, error) {
	t.Helper()
	docker, err := exec.LookPath("docker")
	if err != nil {
		return "", err
	}
	imageName := fmt.Sprintf("%s:%s", pluginIdentity, stringid.GenerateRandomID())
	cmd := command.NewRunner()
	if err := cmd.Run(
		context.Background(),
		docker,
		command.RunWithArgs("build", "-t", imageName, "."),
		command.RunWithDir(filepath.Dir(dockerfilePath)),
		command.RunWithStdout(os.Stdout),
		command.RunWithStderr(os.Stderr),
	); err != nil {
		return "", err
	}
	t.Logf("created image: %s", imageName)
	t.Cleanup(func() {
		if err := cmd.Run(
			context.Background(),
			docker,
			command.RunWithArgs("rmi", "--force", imageName),
			command.RunWithDir(filepath.Dir(dockerfilePath)),
			command.RunWithStdout(os.Stdout),
			command.RunWithStderr(os.Stderr),
		); err != nil {
			t.Logf("failed to remove temporary docker image: %v", err)
		}
	})
	return imageName, nil
}

// dockerServer allows testing some failure flows by simulating the responses to Docker CLI commands.
type dockerServer struct {
	httpServer    *httptest.Server
	h2Server      *http2.Server
	h2Handler     http.Handler
	t             testing.TB
	versionPrefix string
	pushErr       error
	// protects builtImages
	mu           sync.RWMutex
	pushedImages map[string]*pushedImage
}

type pushedImage struct {
	tags []string
}

func newDockerServer(t testing.TB, version string) *dockerServer {
	t.Helper()
	versionPrefix := "/v" + version
	dockerServer := &dockerServer{
		t:             t,
		pushedImages:  make(map[string]*pushedImage),
		versionPrefix: versionPrefix,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(versionPrefix+"/images/", dockerServer.imagesHandler)
	mux.HandleFunc("/_ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&types.Ping{APIVersion: version}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})
	dockerServer.h2Server = &http2.Server{}
	dockerServer.h2Handler = h2c.NewHandler(mux, dockerServer.h2Server)
	dockerServer.httpServer = httptest.NewUnstartedServer(dockerServer.h2Handler)
	dockerServer.httpServer.Start()
	t.Cleanup(dockerServer.httpServer.Close)
	return dockerServer
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
			RepoTags: d.pushedImages[foundImageID].tags,
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
		delete(d.pushedImages, foundImageID)
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
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.pushedImages[image]; !ok {
		d.pushedImages[image] = &pushedImage{}
	}
	imageTag := r.URL.Query()["tag"][0]
	d.pushedImages[image].tags = append(d.pushedImages[image].tags, imageTag)
	auxJSON, err := json.Marshal(map[string]any{
		"Tag":    imageTag,
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
	for imageID, builtImageInfo := range d.pushedImages {
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
