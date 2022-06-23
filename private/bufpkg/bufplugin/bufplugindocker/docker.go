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
	"context"
	"fmt"
	"io"
	"net"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stringid"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Client interface {
	// Build creates a Docker image for the plugin using the Dockerfile.plugin and plugin config.
	Build(ctx context.Context, dockerfile io.Reader, config *bufpluginconfig.Config, params BuildParams) (*BuildResponse, error)
	// Push the Docker image to the remote registry.
	Push(ctx context.Context, image string, auth RegistryAuth) (*PushResponse, error)
	// Delete removes the Docker image from local Docker Engine.
	Delete(ctx context.Context, image string) (*DeleteResponse, error)
	// Close releases any resources used by the underlying Docker client.
	Close() error
}

type BuildParams struct {
	// ConfigDirPath specifies the location where the .buildkit_node_id should be persisted.
	// This is used to establish a session when building images using buildkit.
	// If empty, the node id will be generated randomly and not persisted to disk.
	ConfigDirPath string
}

type BuildResponse struct {
	// Image contains the Docker image name in the local Docker engine including the tag (i.e. plugins.buf.build/library/some-plugin:<id>, where <id> is a random id).
	// It is created from the bufpluginconfig.Config's Name.IdentityString() and a unique id.
	Image string
	// Digest specifies the Docker image digest in the format <hash_algorithm>:<hash>.
	// Example: sha256:65001659f150f085e0b37b697a465a95cbfd885d9315b61960883b9ac588744e
	Digest string
}

type PushResponse struct{}

type DeleteResponse struct{}

type dockerAPIClient struct {
	cli    *client.Client
	logger *zap.Logger
}

var _ Client = (*dockerAPIClient)(nil)

func (d *dockerAPIClient) Build(ctx context.Context, dockerfile io.Reader, pluginConfig *bufpluginconfig.Config, params BuildParams) (*BuildResponse, error) {
	buildkitSession, err := createSession(fmt.Sprintf("%s/%s", pluginConfig.Name.Owner(), pluginConfig.Name.Plugin()), params.ConfigDirPath, zap.L())
	if err != nil {
		return nil, fmt.Errorf("failed to create buildkit session: %w", err)
	}

	dockerContext, err := createDockerContext(dockerfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker context: %w", err)
	}

	eg, errGroupCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		err := buildkitSession.Run(errGroupCtx, func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
			return d.cli.DialHijack(ctx, "/session", proto, meta)
		})
		if err != nil {
			err = fmt.Errorf("failed to run buildkit session: %w", err)
		}
		return err
	})

	buildID := stringid.GenerateRandomID()
	imageName := pluginConfig.Name.IdentityString() + ":" + buildID
	eg.Go(func() error {
		defer buildkitSession.Close()

		response, err := d.cli.ImageBuild(ctx, dockerContext, types.ImageBuildOptions{
			Tags:       []string{imageName},
			Dockerfile: "Dockerfile",
			Platform:   "linux/amd64",
			Labels: map[string]string{
				"build.buf.plugins.config.remote": pluginConfig.Name.Remote(),
				"build.buf.plugins.config.owner":  pluginConfig.Name.Owner(),
				"build.buf.plugins.config.name":   pluginConfig.Name.Plugin(),
			},
			Version:   types.BuilderBuildKit, // DOCKER_BUILDKIT=1
			SessionID: buildkitSession.ID(),
		})
		if err != nil {
			return fmt.Errorf("failed ImageBuild: %w", err)
		}
		defer response.Body.Close()
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			d.logger.Debug(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to scan response body: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to build image: %w", err)
	}

	imageInfo, _, err := d.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}
	return &BuildResponse{
		Image:  imageName,
		Digest: imageInfo.ID,
	}, nil
}

func (d *dockerAPIClient) Push(ctx context.Context, image string, auth RegistryAuth) (response *PushResponse, retErr error) {
	registryAuth, err := auth.ToHeader()
	if err != nil {
		return nil, err
	}
	pushReader, err := d.cli.ImagePush(ctx, image, types.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, pushReader.Close())
	}()
	pushScanner := bufio.NewScanner(pushReader)
	for pushScanner.Scan() {
		d.logger.Debug(pushScanner.Text())
	}
	if err := pushScanner.Err(); err != nil {
		return nil, err
	}
	return &PushResponse{}, nil
}

func (d *dockerAPIClient) Delete(ctx context.Context, image string) (*DeleteResponse, error) {
	_, err := d.cli.ImageRemove(ctx, image, types.ImageRemoveOptions{})
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{}, nil
}

func (d *dockerAPIClient) Close() error {
	return d.cli.Close()
}

type ClientOption func(client *dockerAPIClient)

func NewClient(options ...ClientOption) (Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	dockerClient := &dockerAPIClient{cli: cli}
	for _, option := range options {
		option(dockerClient)
	}
	if dockerClient.logger == nil {
		dockerClient.logger = zap.L()
	}
	return dockerClient, nil
}

func WithLogger(logger *zap.Logger) ClientOption {
	return func(client *dockerAPIClient) {
		client.logger = logger
	}
}
