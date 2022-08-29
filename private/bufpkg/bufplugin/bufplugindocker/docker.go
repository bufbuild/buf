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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	controlapi "github.com/moby/buildkit/api/services/control"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultTarget is the default target architecture for Docker images.
	DefaultTarget = "linux/amd64"

	// pluginsImagePrefix is used to prefix all image names with the correct path for pushing to the OCI registry.
	pluginsImagePrefix = "plugins."
)

// Client is a small abstraction over a Docker API client, providing the basic APIs we need to build plugins.
// It ensures that we pass the appropriate parameters to build images (i.e. platform 'linux/amd64').
type Client interface {
	// Build creates a Docker image for the plugin using the Dockerfile.plugin and plugin config.
	Build(ctx context.Context, dockerfile io.Reader, config *bufpluginconfig.Config, options ...BuildOption) (*BuildResponse, error)
	// Push the Docker image to the remote registry.
	Push(ctx context.Context, image string, auth *RegistryAuthConfig) (*PushResponse, error)
	// Delete removes the Docker image from local Docker Engine.
	Delete(ctx context.Context, image string) (*DeleteResponse, error)
	// Tag creates a Docker image tag from an existing image and plugin config.
	Tag(ctx context.Context, image string, config *bufpluginconfig.Config) (*TagResponse, error)
	// Close releases any resources used by the underlying Docker client.
	Close() error
}

// BuildOption defines options for the image build call.
type BuildOption func(*buildOptions)

// WithConfigDirPath is a BuildOption which enables node ID persistence to the specified configuration directory.
// If not set, a new node ID will be generated with each build call.
func WithConfigDirPath(path string) BuildOption {
	return func(options *buildOptions) {
		options.configDirPath = path
	}
}

// WithTarget is a BuildOption which sets the target architecture (used for local testing on arm64).
func WithTarget(target string) BuildOption {
	return func(options *buildOptions) {
		options.target = target
	}
}

// WithCacheFrom is a BuildOption which allows using images published to an OCI registry to be used as cached state during build.
func WithCacheFrom(cacheFrom []string) BuildOption {
	return func(options *buildOptions) {
		options.cacheFrom = cacheFrom
	}
}

// WithPullParent configures whether the build will attempt to pull the latest parent images (FROM ...) prior to build.
func WithPullParent(pullParent bool) BuildOption {
	return func(options *buildOptions) {
		options.pullParent = pullParent
	}
}

// WithBuildArgs allows specifying additional Docker build args.
func WithBuildArgs(args []string) BuildOption {
	return func(options *buildOptions) {
		options.buildArgs = args
	}
}

// BuildResponse returns details of a successful image build call.
type BuildResponse struct {
	// Image contains the Docker image name in the local Docker engine including the tag (i.e. plugins.buf.build/library/some-plugin:<id>, where <id> is a random id).
	// It is created from the bufpluginconfig.Config's Name.IdentityString() and a unique id.
	Image string
	// ImageID specifies the Docker image id in the format <hash_algorithm>:<hash>.
	// Example: sha256:65001659f150f085e0b37b697a465a95cbfd885d9315b61960883b9ac588744e
	ImageID string
}

// PushResponse is a placeholder for data to be returned from a successful image push call.
type PushResponse struct {
	// Digest specifies the Docker image digest in the format <hash_algorithm>:<hash>.
	// The digest returned from Client.Push differs from the image id returned in Client.Build.
	Digest string
}

// TagResponse returns details of a successful image tag call.
type TagResponse struct {
	// Image contains the Docker image name in the local Docker engine including the tag.
	// It is created from the bufpluginconfig.Config's Name.IdentityString() and a unique id.
	Image string
}

// DeleteResponse is a placeholder for data to be returned from a successful image delete call.
type DeleteResponse struct{}

type dockerAPIClient struct {
	cli    *client.Client
	logger *zap.Logger
}

var _ Client = (*dockerAPIClient)(nil)

func (d *dockerAPIClient) Build(ctx context.Context, dockerfile io.Reader, pluginConfig *bufpluginconfig.Config, options ...BuildOption) (*BuildResponse, error) {
	params := &buildOptions{}
	for _, option := range options {
		option(params)
	}
	// TODO: Need to determine how contextDir parameter is used in Docker engine.
	buildkitSession, err := createSession(ctx, zap.L(), fmt.Sprintf("%s/%s", pluginConfig.Name.Owner(), pluginConfig.Name.Plugin()), params.configDirPath)
	if err != nil {
		return nil, err
	}

	dockerContext, err := createDockerContext(dockerfile)
	if err != nil {
		return nil, err
	}

	// Use errgroup here over pkg.Thread - we aren't using concurrency here for performance reasons.
	// We want both the buildkit session initialization to run alongside with the image build operation.
	eg, errGroupCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// This links a buildkit session with an active Docker client.
		// Behind the scenes, the session upgrades the connection to HTTP/2 and provides a gRPC server with services for advanced buildkit features (credentials, SSH, etc.).
		// We don't currently require this to build any of our plugins.
		// See https://pkg.go.dev/github.com/moby/buildkit/session#Session.Allow for more details.
		return buildkitSession.Run(errGroupCtx, func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
			return d.cli.DialHijack(ctx, "/session", proto, meta)
		})
	})

	buildID := stringid.GenerateRandomID()
	imageName := pluginConfig.Name.IdentityString() + ":" + buildID
	if !strings.HasPrefix(imageName, pluginsImagePrefix) {
		imageName = pluginsImagePrefix + imageName
	}
	eg.Go(func() (retErr error) {
		defer func() {
			if err := buildkitSession.Close(); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()

		target := params.target
		if len(target) == 0 {
			target = DefaultTarget
		}
		buildArgs := make(map[string]*string)
		for _, arg := range params.buildArgs {
			name, val, _ := strings.Cut(arg, "=")
			buildArgs[name] = &val
		}
		buildArgs["PLUGIN_VERSION"] = &pluginConfig.PluginVersion
		response, err := d.cli.ImageBuild(ctx, dockerContext, types.ImageBuildOptions{
			Tags:     []string{imageName},
			Platform: target,
			Labels: map[string]string{
				"build.buf.plugins.config.owner": pluginConfig.Name.Owner(),
				"build.buf.plugins.config.name":  pluginConfig.Name.Plugin(),
			},
			Version:    types.BuilderBuildKit, // DOCKER_BUILDKIT=1
			SessionID:  buildkitSession.ID(),
			BuildArgs:  buildArgs,
			CacheFrom:  params.cacheFrom,
			PullParent: params.pullParent,
		})
		if err != nil {
			return err
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			var message jsonmessage.JSONMessage
			if err := json.Unmarshal([]byte(scanner.Text()), &message); err != nil {
				d.logger.Debug(scanner.Text())
				continue
			}
			if message.Error != nil {
				return message.Error
			}
			handled := false
			if message.ID == "moby.buildkit.trace" && message.Aux != nil {
				b64Aux := string(*message.Aux)
				b64Aux = strings.TrimPrefix(b64Aux, `"`)
				b64Aux = strings.TrimSuffix(b64Aux, `"`)
				bdec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64Aux))
				if protoBytes, err := io.ReadAll(bdec); err == nil {
					var status controlapi.StatusResponse
					if err := status.Unmarshal(protoBytes); err == nil {
						d.logger.Debug("trace", zap.Any("status", status))
						handled = true
					}
				}
			}
			// If we fail to interpret a line of output, log the raw format
			if !handled {
				d.logger.Debug(scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	imageInfo, _, err := d.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, err
	}
	return &BuildResponse{
		Image:   imageName,
		ImageID: imageInfo.ID,
	}, nil
}

func (d *dockerAPIClient) Tag(ctx context.Context, image string, config *bufpluginconfig.Config) (*TagResponse, error) {
	buildID := stringid.GenerateRandomID()
	imageName := config.Name.IdentityString() + ":" + buildID
	if !strings.HasPrefix(imageName, pluginsImagePrefix) {
		imageName = pluginsImagePrefix + imageName
	}
	if err := d.cli.ImageTag(ctx, image, imageName); err != nil {
		return nil, err
	}
	return &TagResponse{Image: imageName}, nil
}

func (d *dockerAPIClient) Push(ctx context.Context, image string, auth *RegistryAuthConfig) (response *PushResponse, retErr error) {
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
	var imageDigest string
	pushScanner := bufio.NewScanner(pushReader)
	for pushScanner.Scan() {
		d.logger.Debug(pushScanner.Text())
		var message jsonmessage.JSONMessage
		if err := json.Unmarshal([]byte(pushScanner.Text()), &message); err == nil {
			if message.Error != nil {
				return nil, message.Error
			}
			if message.Aux != nil {
				var pushResult types.PushResult
				if err := json.Unmarshal(*message.Aux, &pushResult); err == nil {
					imageDigest = pushResult.Digest
				}
			}
		}
	}
	if err := pushScanner.Err(); err != nil {
		return nil, err
	}
	if len(imageDigest) == 0 {
		return nil, fmt.Errorf("failed to determine image digest after push")
	}
	return &PushResponse{Digest: imageDigest}, nil
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

// NewClient creates a new Client to use to build Docker plugins.
func NewClient(logger *zap.Logger, options ...ClientOption) (Client, error) {
	if logger == nil {
		return nil, errors.New("logger required")
	}
	opts := &clientOptions{}
	for _, option := range options {
		option(opts)
	}
	dockerClientOpts := []client.Opt{client.FromEnv}
	if len(opts.host) > 0 {
		dockerClientOpts = append(dockerClientOpts, client.WithHost(opts.host))
	}
	if len(opts.version) > 0 {
		dockerClientOpts = append(dockerClientOpts, client.WithVersion(opts.version))
	}
	cli, err := client.NewClientWithOpts(dockerClientOpts...)
	if err != nil {
		return nil, err
	}
	return &dockerAPIClient{
		cli:    cli,
		logger: logger,
	}, nil
}

type clientOptions struct {
	host    string
	version string
}

// ClientOption defines options for the NewClient call to customize the underlying Docker client.
type ClientOption func(options *clientOptions)

// WithHost allows specifying a Docker engine host to connect to (instead of the default lookup using DOCKER_HOST env var).
// This makes it suitable for use by parallel tests.
func WithHost(host string) ClientOption {
	return func(options *clientOptions) {
		options.host = host
	}
}

// WithVersion allows specifying a Docker API client version instead of using the default version negotiation algorithm.
// This allows tests to implement the Docker engine API using stable URLs.
func WithVersion(version string) ClientOption {
	return func(options *clientOptions) {
		options.version = version
	}
}

type buildOptions struct {
	configDirPath string
	target        string
	cacheFrom     []string
	pullParent    bool
	buildArgs     []string
}
