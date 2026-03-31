// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufremoteplugindocker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/stringid"
)

const (
	// BufUpstreamClientUserAgentPrefix is the user-agent prefix.
	//
	// Setting this value on the buf docker client allows us to propagate a custom
	// value to the OCI registry. This is a useful property that enables registries
	// to differentiate between the buf cli vs other tools like docker cli.
	// Note, this does not override the final User-Agent entirely, but instead adds
	// the value to the final outgoing User-Agent value in the form: [docker client's UA] UpstreamClient(buf-cli-1.11.0)
	//
	// Example: User-Agent = [docker/20.10.21 go/go1.18.7 git-commit/3056208 kernel/5.15.49-linuxkit os/linux arch/arm64 UpstreamClient(buf-cli-1.11.0)]
	BufUpstreamClientUserAgentPrefix = "buf-cli-"
)

// imageDigestRe is a regular expression for extracting the image digest from the output of a docker
// push command. It is tightly coupled to the string representation because the stream response
// itself is not very well defined.
//
// https://github.com/moby/moby/blob/1c282d1f1b90ff188a1b46f48548ac3151ca2ddf/daemon/containerd/image_push.go#L130
var imageDigestRe = regexp.MustCompile(`digest:\s*(\S+)`)

// Client is a small abstraction over a Docker API client, providing the basic APIs we need to build plugins.
// It ensures that we pass the appropriate parameters to build images (i.e. platform 'linux/amd64').
type Client interface {
	// Load imports a Docker image into the local Docker Engine.
	Load(ctx context.Context, image io.Reader) (*LoadResponse, error)
	// Push the Docker image to the remote registry.
	Push(ctx context.Context, image string, auth *RegistryAuthConfig) (*PushResponse, error)
	// Delete removes the Docker image from local Docker Engine.
	Delete(ctx context.Context, image string) (*DeleteResponse, error)
	// Tag creates a Docker image tag from an existing image and plugin config.
	Tag(ctx context.Context, image string, config *bufremotepluginconfig.Config) (*TagResponse, error)
	// Inspect inspects an image and returns the image id.
	Inspect(ctx context.Context, image string) (*InspectResponse, error)
	// Close releases any resources used by the underlying Docker client.
	Close() error
}

// LoadResponse returns details of a successful load image call.
type LoadResponse struct {
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
	// It is created from the bufremotepluginconfig.Config's Name.IdentityString() and a unique id.
	Image string
}

// DeleteResponse is a placeholder for data to be returned from a successful image delete call.
type DeleteResponse struct{}

// InspectResponse returns the image id for a given image.
type InspectResponse struct {
	// ImageID contains the Docker image's ID.
	ImageID string
}

type dockerAPIClient struct {
	cli    *client.Client
	logger *slog.Logger
}

var _ Client = (*dockerAPIClient)(nil)

func (d *dockerAPIClient) Load(ctx context.Context, image io.Reader) (_ *LoadResponse, retErr error) {
	response, err := d.cli.ImageLoad(ctx, image, client.ImageLoadWithQuiet(true))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := response.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("docker load response body close error: %w", err))
		}
	}()
	imageID := ""
	responseScanner := bufio.NewScanner(response)
	for responseScanner.Scan() {
		var jsonMessage jsonstream.Message
		if err := json.Unmarshal(responseScanner.Bytes(), &jsonMessage); err == nil {
			_, loadedImageID, found := strings.Cut(strings.TrimSpace(jsonMessage.Stream), "Loaded image ID: ")
			if !found {
				continue
			}
			if !strings.HasPrefix(loadedImageID, "sha256:") {
				d.logger.Warn("Unsupported image digest", slog.String("imageID", loadedImageID))
				continue
			}
			imageID = loadedImageID
		}
	}
	if err := responseScanner.Err(); err != nil {
		return nil, err
	}
	if imageID == "" {
		return nil, fmt.Errorf("failed to determine image ID of loaded image")
	}
	return &LoadResponse{ImageID: imageID}, nil
}

func (d *dockerAPIClient) Tag(ctx context.Context, image string, config *bufremotepluginconfig.Config) (*TagResponse, error) {
	buildID := stringid.GenerateRandomID()
	imageName := config.Name.IdentityString() + ":" + buildID
	if _, err := d.cli.ImageTag(ctx, client.ImageTagOptions{Source: image, Target: imageName}); err != nil {
		return nil, err
	}
	return &TagResponse{Image: imageName}, nil
}

func (d *dockerAPIClient) Push(ctx context.Context, image string, auth *RegistryAuthConfig) (response *PushResponse, retErr error) {
	registryAuth, err := auth.ToHeader()
	if err != nil {
		return nil, err
	}
	pushResponse, err := d.cli.ImagePush(ctx, image, client.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, pushResponse.Close())
	}()
	var imageDigest string
	pushScanner := bufio.NewScanner(pushResponse)
	for pushScanner.Scan() {
		d.logger.DebugContext(ctx, pushScanner.Text())
		var message jsonstream.Message
		if err := json.Unmarshal([]byte(pushScanner.Text()), &message); err == nil {
			if message.Error != nil {
				return nil, errors.New(message.Error.Message)
			}
			imageDigest = getImageDigestFromMessage(message)
		}
	}
	if err := pushScanner.Err(); err != nil {
		return nil, err
	}
	if imageDigest == "" {
		return nil, fmt.Errorf("failed to determine image digest after push")
	}
	d.logger.DebugContext(ctx, "docker image digest", slog.String("imageDigest", imageDigest))
	return &PushResponse{Digest: imageDigest}, nil
}

func getImageDigestFromMessage(message jsonstream.Message) string {
	if message.Aux != nil {
		var pushResult struct{ Digest string }
		if err := json.Unmarshal(*message.Aux, &pushResult); err == nil {
			return pushResult.Digest
		}
	}
	// If the message has no aux field, we fall back to parsing the status field.
	if message.Status != "" {
		if match := imageDigestRe.FindStringSubmatch(message.Status); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func (d *dockerAPIClient) Delete(ctx context.Context, image string) (*DeleteResponse, error) {
	_, err := d.cli.ImageRemove(ctx, image, client.ImageRemoveOptions{})
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{}, nil
}

func (d *dockerAPIClient) Inspect(ctx context.Context, image string) (*InspectResponse, error) {
	inspect, err := d.cli.ImageInspect(ctx, image)
	if err != nil {
		return nil, err
	}
	return &InspectResponse{ImageID: inspect.ID}, nil
}

func (d *dockerAPIClient) Close() error {
	return d.cli.Close()
}

// NewClient creates a new Client to use to build Docker plugins.
func NewClient(logger *slog.Logger, cliVersion string, options ...ClientOption) (Client, error) {
	if logger == nil {
		return nil, errors.New("logger required")
	}
	opts := &clientOptions{}
	for _, option := range options {
		option(opts)
	}
	dockerClientOpts := []client.Opt{
		client.FromEnv,
		client.WithHTTPHeaders(map[string]string{
			"User-Agent": BufUpstreamClientUserAgentPrefix + cliVersion,
		}),
	}
	if len(opts.host) > 0 {
		dockerClientOpts = append(dockerClientOpts, client.WithHost(opts.host))
	}
	if len(opts.version) > 0 {
		dockerClientOpts = append(dockerClientOpts, client.WithAPIVersion(opts.version))
	}
	cli, err := client.New(dockerClientOpts...)
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
