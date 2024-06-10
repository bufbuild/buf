// Copyright 2020-2024 Buf Technologies, Inc.
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
	"strings"
	"sync"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/docker/docker/api/types"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	// Setting this value on the buf docker client allows us to propagate a custom
	// value to the OCI registry. This is a useful property that enables registries
	// to differentiate between the buf cli vs other tools like docker cli.
	// Note, this does not override the final User-Agent entirely, but instead adds
	// the value to the final outgoing User-Agent value in the form: [docker client's UA] UpstreamClient(buf-cli-1.11.0)
	//
	// Example: User-Agent = [docker/20.10.21 go/go1.18.7 git-commit/3056208 kernel/5.15.49-linuxkit os/linux arch/arm64 UpstreamClient(buf-cli-1.11.0)]
	BufUpstreamClientUserAgentPrefix = "buf-cli-"
)

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
	cli        *client.Client
	logger     *zap.Logger
	lock       sync.RWMutex // protects negotiated
	negotiated bool
}

var _ Client = (*dockerAPIClient)(nil)

func (d *dockerAPIClient) Load(ctx context.Context, image io.Reader) (_ *LoadResponse, retErr error) {
	if err := d.negotiateVersion(ctx); err != nil {
		return nil, err
	}
	response, err := d.cli.ImageLoad(ctx, image, true)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("docker load response body close error: %w", err))
		}
	}()
	imageID := ""
	responseScanner := bufio.NewScanner(response.Body)
	for responseScanner.Scan() {
		var jsonMessage jsonmessage.JSONMessage
		if err := json.Unmarshal(responseScanner.Bytes(), &jsonMessage); err == nil {
			_, loadedImageID, found := strings.Cut(strings.TrimSpace(jsonMessage.Stream), "Loaded image ID: ")
			if !found {
				continue
			}
			if !strings.HasPrefix(loadedImageID, "sha256:") {
				d.logger.Warn("Unsupported image digest", zap.String("imageID", loadedImageID))
				continue
			}
			if err := stringid.ValidateID(strings.TrimPrefix(loadedImageID, "sha256:")); err != nil {
				d.logger.Warn("Invalid image id", zap.String("imageID", loadedImageID))
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
	if err := d.negotiateVersion(ctx); err != nil {
		return nil, err
	}
	buildID := stringid.GenerateRandomID()
	imageName := config.Name.IdentityString() + ":" + buildID
	if err := d.cli.ImageTag(ctx, image, imageName); err != nil {
		return nil, err
	}
	return &TagResponse{Image: imageName}, nil
}

func (d *dockerAPIClient) Push(ctx context.Context, image string, auth *RegistryAuthConfig) (response *PushResponse, retErr error) {
	if err := d.negotiateVersion(ctx); err != nil {
		return nil, err
	}
	registryAuth, err := auth.ToHeader()
	if err != nil {
		return nil, err
	}
	pushReader, err := d.cli.ImagePush(ctx, image, imagetypes.PushOptions{
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
	if err := d.negotiateVersion(ctx); err != nil {
		return nil, err
	}
	_, err := d.cli.ImageRemove(ctx, image, imagetypes.RemoveOptions{})
	if err != nil {
		return nil, err
	}
	return &DeleteResponse{}, nil
}

func (d *dockerAPIClient) Inspect(ctx context.Context, image string) (*InspectResponse, error) {
	if err := d.negotiateVersion(ctx); err != nil {
		return nil, err
	}
	inspect, _, err := d.cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return nil, err
	}
	return &InspectResponse{ImageID: inspect.ID}, nil
}

func (d *dockerAPIClient) Close() error {
	return d.cli.Close()
}

func (d *dockerAPIClient) negotiateVersion(ctx context.Context) error {
	d.lock.RLock()
	negotiated := d.negotiated
	d.lock.RUnlock()
	if negotiated {
		return nil
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.negotiated {
		return nil
	}
	deadline := time.Now().Add(5 * time.Second)
	if existingDeadline, ok := ctx.Deadline(); ok {
		if existingDeadline.Before(deadline) {
			deadline = existingDeadline
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	ping, err := d.cli.Ping(ctx)
	if err != nil {
		return err
	}
	d.cli.NegotiateAPIVersionPing(ping)
	d.negotiated = true
	return nil
}

// NewClient creates a new Client to use to build Docker plugins.
func NewClient(logger *zap.Logger, cliVersion string, options ...ClientOption) (Client, error) {
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
