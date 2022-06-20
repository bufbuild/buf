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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufplugindocker"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var dockerEnabled = false

const examplePluginIdentity = "plugins.buf.build/library/go"

func Test_BuildSuccess(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerClient := createClient(t)
	response, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile")
	require.Nil(t, err)
	assert.Truef(
		t,
		strings.HasPrefix(response.Image, examplePluginIdentity+":"),
		"image name should begin with: %q, found: %q",
		examplePluginIdentity,
		response.Image,
	)
	assert.NotEmpty(t, response.Digest)
}

func Test_BuildFailure(t *testing.T) {
	t.Parallel()
	if !dockerEnabled {
		t.Skip("docker disabled")
	}
	if testing.Short() {
		t.Skip("docker tests disabled in short mode")
	}
	dockerClient := createClient(t)
	_, err := buildDockerPlugin(t, dockerClient, "testdata/success/Dockerfile")
	require.Nil(t, err)
}

func TestMain(m *testing.M) {
	if cli, err := client.NewClientWithOpts(client.FromEnv); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := cli.Ping(ctx); err == nil {
			dockerEnabled = true
		}
		_ = cli.Close()
	}
	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

func createClient(t testing.TB) bufplugindocker.Client {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.Nil(t, err)
	dockerClient, err := bufplugindocker.NewClient(bufplugindocker.WithLogger(logger))
	require.Nil(t, err)
	t.Cleanup(func() {
		if err := dockerClient.Close(); err != nil {
			t.Errorf("failed to close docker client: %v", err)
		}
	})
	return dockerClient
}

func buildDockerPlugin(t testing.TB, dockerClient bufplugindocker.Client, dockerfilePath string) (*bufplugindocker.BuildResponse, error) {
	t.Helper()
	dockerfile, err := os.Open(dockerfilePath)
	require.Nil(t, err)
	pluginName, err := bufpluginref.PluginIdentityForString(examplePluginIdentity)
	require.Nil(t, err)
	pluginConfig := &bufpluginconfig.Config{Name: pluginName}
	response, err := dockerClient.Build(context.Background(), dockerfile, pluginConfig, bufplugindocker.BuildParams{})
	if err != nil {
		t.Cleanup(func() {
			if _, err := dockerClient.Delete(context.Background(), response.Image); err != nil {
				t.Errorf("failed to delete image: %q", response.Image)
			}
		})
	}
	return response, err
}
