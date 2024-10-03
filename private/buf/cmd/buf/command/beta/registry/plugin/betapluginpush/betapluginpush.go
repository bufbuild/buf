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

package betapluginpush

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremoteplugindocker"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netext"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	pkgv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	formatFlagName          = "format"
	disableSymlinksFlagName = "disable-symlinks"
	overrideRemoteFlagName  = "override-remote"
	imageFlagName           = "image"
	visibilityFlagName      = "visibility"

	publicVisibility  = "public"
	privateVisibility = "private"
)

var allVisibiltyStrings = []string{
	publicVisibility,
	privateVisibility,
}

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push a plugin to a registry",
		Long:  bufcli.GetSourceDirLong(`the source to push (directory containing buf.plugin.yaml or plugin release zip)`),
		Args:  appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format          string
	DisableSymlinks bool
	OverrideRemote  string
	Image           string
	Visibility      string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
	flagSet.StringVar(
		&f.OverrideRemote,
		overrideRemoteFlagName,
		"",
		"Override the default remote found in buf.plugin.yaml name and dependencies",
	)
	flagSet.StringVar(
		&f.Image,
		imageFlagName,
		"",
		"Existing image to push",
	)
	flagSet.StringVar(
		&f.Visibility,
		visibilityFlagName,
		"",
		fmt.Sprintf(`The plugin's visibility setting. Must be one of %s`, stringutil.SliceToString(allVisibiltyStrings)),
	)
	_ = appcmd.MarkFlagRequired(flagSet, visibilityFlagName)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	bufcli.WarnBetaCommand(ctx, container)
	if len(flags.OverrideRemote) > 0 {
		if _, err := netext.ValidateHostname(flags.OverrideRemote); err != nil {
			return fmt.Errorf("%s: %w", overrideRemoteFlagName, err)
		}
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	source, err := bufcli.GetInputValue(container, "" /* The input hashtag is not supported here */, ".")
	if err != nil {
		return err
	}
	storageProvider := newStorageosProvider(flags.DisableSymlinks)
	sourceStat, err := os.Stat(source)
	if err != nil {
		return err
	}
	var sourceBucket storage.ReadWriteBucket
	if !sourceStat.IsDir() && strings.HasSuffix(strings.ToLower(sourceStat.Name()), ".zip") {
		// Unpack plugin release to temporary directory
		tmpDir, err := os.MkdirTemp(os.TempDir(), "plugin-push")
		if err != nil {
			return err
		}
		defer func() {
			if err := os.RemoveAll(tmpDir); !os.IsNotExist(err) {
				retErr = multierr.Append(retErr, err)
			}
		}()
		sourceBucket, err = storageProvider.NewReadWriteBucket(tmpDir)
		if err != nil {
			return err
		}
		if err := unzipPluginToSourceBucket(ctx, source, sourceStat.Size(), sourceBucket); err != nil {
			return err
		}
	} else {
		sourceBucket, err = storageProvider.NewReadWriteBucket(source)
		if err != nil {
			return err
		}
	}
	existingConfigFilePath, err := bufremotepluginconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return syserror.Wrap(err)
	}
	if existingConfigFilePath == "" {
		return fmt.Errorf("please define a %s configuration file in the target directory", bufremotepluginconfig.ExternalConfigFilePath)
	}
	var options []bufremotepluginconfig.ConfigOption
	if len(flags.OverrideRemote) > 0 {
		options = append(options, bufremotepluginconfig.WithOverrideRemote(flags.OverrideRemote))
	}
	pluginConfig, err := bufremotepluginconfig.GetConfigForBucket(ctx, sourceBucket, options...)
	if err != nil {
		return err
	}
	client, err := bufremoteplugindocker.NewClient(container.Logger(), bufcli.Version)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("docker client close error: %w", err))
		}
	}()
	var imageID string
	if flags.Image != "" {
		inspectResponse, err := client.Inspect(ctx, flags.Image)
		if err != nil {
			return err
		}
		imageID = inspectResponse.ImageID
	} else {
		image, err := loadDockerImage(ctx, sourceBucket)
		if err != nil {
			return err
		}
		loadResponse, err := client.Load(ctx, image)
		if err != nil {
			return err
		}
		defer func() {
			if err := image.Close(); err != nil && !errors.Is(err, storage.ErrClosed) {
				retErr = multierr.Append(retErr, fmt.Errorf("docker image close error: %w", err))
			}
		}()
		imageID = loadResponse.ImageID
	}
	visibility, err := visibilityFlagToVisibility(flags.Visibility)
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(
		clientConfig,
		pluginConfig.Name.Remote(),
		registryv1alpha1connect.NewPluginCurationServiceClient,
	)
	latestPluginResp, err := service.GetLatestCuratedPlugin(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.GetLatestCuratedPluginRequest{
				Owner:    pluginConfig.Name.Owner(),
				Name:     pluginConfig.Name.Plugin(),
				Version:  pluginConfig.PluginVersion,
				Revision: 0, // get latest revision for the plugin version.
			},
		),
	)
	var currentImageDigest string
	var nextRevision uint32
	if err != nil {
		if connect.CodeOf(err) != connect.CodeNotFound {
			return err
		}
		nextRevision = 1
	} else {
		nextRevision = latestPluginResp.Msg.Plugin.Revision + 1
		currentImageDigest = latestPluginResp.Msg.Plugin.ContainerImageDigest
	}
	machine, err := netrc.GetMachineForName(container, pluginConfig.Name.Remote())
	if err != nil {
		return err
	}
	authConfig := &bufremoteplugindocker.RegistryAuthConfig{}
	if machine != nil {
		authConfig.ServerAddress = machine.Name()
		authConfig.Username = machine.Login()
		authConfig.Password = machine.Password()
	}
	imageDigest, err := findExistingDigestForImageID(ctx, pluginConfig, authConfig, imageID, currentImageDigest)
	if err != nil {
		return err
	}
	if imageDigest == "" {
		imageDigest, err = pushImage(ctx, client, authConfig, pluginConfig, imageID)
		if err != nil {
			return err
		}
	} else {
		container.Logger().Info("image found in registry - skipping push")
	}
	plugin, err := bufremoteplugin.NewPlugin(
		pluginConfig.PluginVersion,
		pluginConfig.Dependencies,
		pluginConfig.Registry,
		imageDigest,
		pluginConfig.SourceURL,
		pluginConfig.Description,
	)
	if err != nil {
		return err
	}
	createRequest, err := createCuratedPluginRequest(pluginConfig, plugin, nextRevision, visibility)
	if err != nil {
		return err
	}
	var curatedPlugin *registryv1alpha1.CuratedPlugin
	createPluginResp, err := service.CreateCuratedPlugin(ctx, connect.NewRequest(createRequest))
	if err != nil {
		if connect.CodeOf(err) != connect.CodeAlreadyExists {
			return err
		}
		// Plugin with the same image digest and metadata already exists
		container.Logger().Info(
			"plugin already exists",
			slog.String("name", pluginConfig.Name.IdentityString()),
			slog.String("digest", plugin.ContainerImageDigest()),
		)
		curatedPlugin = latestPluginResp.Msg.Plugin
	} else {
		curatedPlugin = createPluginResp.Msg.Configuration
	}
	return bufprint.NewCuratedPluginPrinter(container.Stdout()).PrintCuratedPlugin(ctx, format, curatedPlugin)
}

func createCuratedPluginRequest(
	pluginConfig *bufremotepluginconfig.Config,
	plugin bufremoteplugin.Plugin,
	nextRevision uint32,
	visibility registryv1alpha1.CuratedPluginVisibility,
) (*registryv1alpha1.CreateCuratedPluginRequest, error) {
	outputLanguages, err := bufremoteplugin.OutputLanguagesToProtoLanguages(pluginConfig.OutputLanguages)
	if err != nil {
		return nil, err
	}
	protoRegistryConfig, err := bufremoteplugin.PluginRegistryToProtoRegistryConfig(plugin.Registry())
	if err != nil {
		return nil, err
	}
	return &registryv1alpha1.CreateCuratedPluginRequest{
		Owner:                pluginConfig.Name.Owner(),
		Name:                 pluginConfig.Name.Plugin(),
		RegistryType:         bufremoteplugin.PluginToProtoPluginRegistryType(plugin),
		Version:              plugin.Version(),
		ContainerImageDigest: plugin.ContainerImageDigest(),
		Dependencies:         bufremoteplugin.PluginReferencesToCuratedProtoPluginReferences(plugin.Dependencies()),
		SourceUrl:            plugin.SourceURL(),
		Description:          plugin.Description(),
		RegistryConfig:       protoRegistryConfig,
		Revision:             nextRevision,
		OutputLanguages:      outputLanguages,
		SpdxLicenseId:        pluginConfig.SPDXLicenseID,
		LicenseUrl:           pluginConfig.LicenseURL,
		Visibility:           visibility,
		IntegrationGuideUrl:  pluginConfig.IntegrationGuideURL,
	}, nil
}

func pushImage(
	ctx context.Context,
	client bufremoteplugindocker.Client,
	authConfig *bufremoteplugindocker.RegistryAuthConfig,
	plugin *bufremotepluginconfig.Config,
	image string,
) (_ string, retErr error) {
	tagResponse, err := client.Tag(ctx, image, plugin)
	if err != nil {
		return "", err
	}
	createdImage := tagResponse.Image
	// We tag a Docker image using a unique ID label each time.
	// After we're done publishing the image, we delete it to not leave a lot of images left behind.
	defer func() {
		if _, err := client.Delete(ctx, createdImage); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("failed to delete image %q", createdImage))
		}
	}()
	pushResponse, err := client.Push(ctx, createdImage, authConfig)
	if err != nil {
		return "", err
	}
	return pushResponse.Digest, nil
}

// findExistingDigestForImageID will query the OCI registry to see if the imageID already exists.
// If an image is found with the same imageID, its digest will be returned (and we'll skip pushing to OCI registry).
//
// It performs the following search:
//
// - GET /v2/{owner}/{plugin}/tags/list
// - For each tag:
//   - Fetch image: GET /v2/{owner}/{plugin}/manifests/{tag}
//   - If image manifest matches imageID, we can use the image digest for the image.
func findExistingDigestForImageID(
	ctx context.Context,
	plugin *bufremotepluginconfig.Config,
	authConfig *bufremoteplugindocker.RegistryAuthConfig,
	imageID string,
	currentImageDigest string,
) (string, error) {
	repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", plugin.Name.Remote(), plugin.Name.Owner(), plugin.Name.Plugin()))
	if err != nil {
		return "", err
	}
	auth := &authn.Basic{Username: authConfig.Username, Password: authConfig.Password}
	remoteOpts := []remote.Option{remote.WithContext(ctx), remote.WithAuth(auth)}
	// First attempt to see if the current image digest matches the image ID
	if currentImageDigest != "" {
		remoteImageID, _, err := getImageIDAndDigestFromReference(ctx, repo.Digest(currentImageDigest), remoteOpts...)
		if err != nil {
			return "", err
		}
		if remoteImageID == imageID {
			return currentImageDigest, nil
		}
	}
	// List all tags and check for a match
	tags, err := remote.List(repo, remoteOpts...)
	if err != nil {
		structuredErr := new(transport.Error)
		if errors.As(err, &structuredErr) {
			if structuredErr.StatusCode == http.StatusUnauthorized {
				return "", errors.New("you are not authenticated. For details, visit https://buf.build/docs/bsr/authentication")
			}
			if structuredErr.StatusCode == http.StatusNotFound {
				return "", nil
			}
		}
		return "", err
	}
	existingImageDigest := ""
	for _, tag := range tags {
		remoteImageID, imageDigest, err := getImageIDAndDigestFromReference(ctx, repo.Tag(tag), remoteOpts...)
		if err != nil {
			return "", err
		}
		if remoteImageID == imageID {
			existingImageDigest = imageDigest
			break
		}
	}
	return existingImageDigest, nil
}

// getImageIDAndDigestFromReference takes an image reference and returns 2 resolved digests:
//
//  1. The image config digest (https://github.com/opencontainers/image-spec/blob/v1.1.0/config.md)
//  2. The image manifest digest (https://github.com/opencontainers/image-spec/blob/v1.1.0/manifest.md)
//
// The incoming ref is expected to be either an image manifest digest or an image index digest.
func getImageIDAndDigestFromReference(
	ctx context.Context,
	ref name.Reference,
	options ...remote.Option,
) (string, string, error) {
	puller, err := remote.NewPuller(options...)
	if err != nil {
		return "", "", err
	}
	desc, err := puller.Get(ctx, ref)
	if err != nil {
		return "", "", err
	}

	switch {
	case desc.MediaType.IsIndex():
		imageIndex, err := desc.ImageIndex()
		if err != nil {
			return "", "", fmt.Errorf("failed to get image index: %w", err)
		}
		indexManifest, err := imageIndex.IndexManifest()
		if err != nil {
			return "", "", fmt.Errorf("failed to get image manifests: %w", err)
		}
		var manifest pkgv1.Descriptor
		for _, desc := range indexManifest.Manifests {
			if p := desc.Platform; p != nil {
				//  Drop attestations, which don't have a valid platform set.
				if p.OS == "unknown" && p.Architecture == "unknown" {
					continue
				}
				manifest = desc
				break
			}
		}
		refNameWithoutDigest, _, ok := strings.Cut(ref.Name(), "@")
		if !ok {
			return "", "", fmt.Errorf("failed to parse reference name %q", ref)
		}
		repository, err := name.NewRepository(refNameWithoutDigest)
		if err != nil {
			return "", "", fmt.Errorf("failed to construct repository %q: %w", refNameWithoutDigest, err)
		}
		// We resolved the image index to an image manifest digest, we can now call this function
		// again to resolve the image manifest digest to an image config digest.
		return getImageIDAndDigestFromReference(
			ctx,
			repository.Digest(manifest.Digest.String()),
			options...,
		)
	case desc.MediaType.IsImage():
		imageManifest, err := desc.Image()
		if err != nil {
			return "", "", fmt.Errorf("failed to get image: %w", err)
		}
		imageManifestDigest, err := imageManifest.Digest()
		if err != nil {
			return "", "", fmt.Errorf("failed to get image digest for %q: %w", ref, err)
		}
		manifest, err := imageManifest.Manifest()
		if err != nil {
			return "", "", fmt.Errorf("failed to get image manifest for %q: %w", ref, err)
		}
		return manifest.Config.Digest.String(), imageManifestDigest.String(), nil
	}
	return "", "", fmt.Errorf("unsupported media type: %q", desc.MediaType)
}

func unzipPluginToSourceBucket(ctx context.Context, pluginZip string, size int64, bucket storage.ReadWriteBucket) (retErr error) {
	f, err := os.Open(pluginZip)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("plugin zip close error: %w", err))
		}
	}()
	return storagearchive.Unzip(ctx, f, size, bucket)
}

func loadDockerImage(ctx context.Context, bucket storage.ReadBucket) (storage.ReadObjectCloser, error) {
	image, err := bucket.Get(ctx, bufremoteplugindocker.ImagePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("unable to find a %s plugin image: %w", bufremoteplugindocker.ImagePath, err)
	}
	return image, nil
}

func visibilityFlagToVisibility(visibility string) (registryv1alpha1.CuratedPluginVisibility, error) {
	switch visibility {
	case publicVisibility:
		return registryv1alpha1.CuratedPluginVisibility_CURATED_PLUGIN_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return registryv1alpha1.CuratedPluginVisibility_CURATED_PLUGIN_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s, expected one of %s", visibility, stringutil.SliceToString(allVisibiltyStrings))
	}
}

func newStorageosProvider(disableSymlinks bool) storageos.Provider {
	var options []storageos.ProviderOption
	if !disableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	return storageos.NewProvider(options...)
}
