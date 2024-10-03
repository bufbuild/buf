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

package pluginpush

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/plugin/v1beta1/pluginv1beta1connect"
	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremoteplugindocker"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	pkgv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	disableSymlinksFlagName = "disable-symlinks"
	labelFlagName           = "label"
	imageFlagName           = "image"
	imageConfigFlagName     = "image-config"
	checkBinaryFlagName     = "check-binary"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Push a plugin to a registry",
		Long:  `The first argument is the plugin full name in the format <remote/owner/plugin>.`,
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
	DisableSymlinks bool
	Labels          []string
	Image           string
	ImageConfig     string
	CheckBinary     string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.Image,
		imageFlagName,
		"",
		"Push the plugin docker image to the registry.",
	)
	flagSet.StringVar(
		&f.ImageConfig,
		imageConfigFlagName,
		"",
		fmt.Sprintf(
			"Set the plugin image config. Must be set if --%s is set.",
			imageFlagName,
		),
	)
	flagSet.StringVar(
		&f.CheckBinary,
		checkBinaryFlagName,
		"",
		"Push the check plugin binary to the registry.",
	)
	//flagSet.StringVar(
	//	&f.BinaryType,
	//	binaryTypeFlagName,
	//	"",
	//	fmt.Sprintf(
	//		"Set the binary type. Must be set if --binary is set.",
	//		binaryFlagName,
	//	),
	//)
	flagSet.StringSliceVar(
		&f.Labels,
		labelFlagName,
		nil,
		"Associate the label with the plugins pushed. Can be used multiple times.",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) (retErr error) {
	if err := validateFlags(flags); err != nil {
		return err
	}
	// We parse the plugin full name from the user-provided argument.
	pluginFullName, err := bufplugin.ParsePluginFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	fmt.Println("pluginFullName", pluginFullName)

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	uploadServiceClient := bufapi.NewClientProvider(clientConfig).
		PluginV1Beta1UploadServiceClient(pluginFullName.Registry())

	pluginKey, err := upload(ctx, container, flags, pluginFullName, uploadServiceClient)
	if err != nil {
		return err
	}
	// Only one plugin key is returned.
	if _, err := fmt.Fprintf(container.Stdout(), "%s\n", pluginKey.String()); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}

func upload(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	pluginFullName bufplugin.PluginFullName,
	uploadServiceClient pluginv1beta1connect.UploadServiceClient,
) (_ bufplugin.PluginKey, retErr error) {
	switch {
	case flags.Image != "":
		return uploadImage(ctx, container, flags, pluginFullName, uploadServiceClient)
	case flags.CheckBinary != "":
		return uploadCheckBinary(ctx, container, flags, pluginFullName, uploadServiceClient)
	default:
		// This should never happen because the flags are validated.
		return nil, syserror.Newf("either --%s or --%s must be set", imageFlagName, checkBinaryFlagName)
	}
}

func uploadCheckBinary(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	pluginFullName bufplugin.PluginFullName,
	uploadServiceClient pluginv1beta1connect.UploadServiceClient,
) (_ bufplugin.PluginKey, retErr error) {
	wasmRuntimeCacheDir, err := bufcli.CreateWasmRuntimeCacheDir(container)
	if err != nil {
		return nil, err
	}
	wasmRuntime, err := wasm.NewRuntime(ctx, wasm.WithLocalCacheDir(wasmRuntimeCacheDir))
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, wasmRuntime.Close(ctx))
	}()

	// Load the binary from the `--binary` flag.
	wasmBinary, err := os.ReadFile(flags.CheckBinary)
	if err != nil {
		return nil, fmt.Errorf("could not read binary %q: %w", flags.CheckBinary, err)
	}

	// Maybe validate the binary is a valid plugin binary?
	_, err = wasmRuntime.Compile(ctx, pluginFullName.Name(), wasmBinary)
	if err != nil {
		return nil, fmt.Errorf("could not compile binary %q: %w", flags.CheckBinary, err)
	}

	// Upload the binary to the registry.
	content := &pluginv1beta1.UploadRequest_Content{
		PluginRef: &pluginv1beta1.PluginRef{
			Value: &pluginv1beta1.PluginRef_Name_{
				Name: &pluginv1beta1.PluginRef_Name{
					Owner:  pluginFullName.Owner(),
					Plugin: pluginFullName.Name(),
				},
			},
		},
		CompressionType: pluginv1beta1.CompressionType_COMPRESSION_TYPE_ZSTD,
		Content:         zstdCompress(wasmBinary),
	}

	uploadResponse, err := uploadServiceClient.Upload(ctx, connect.NewRequest(&pluginv1beta1.UploadRequest{
		Contents: []*pluginv1beta1.UploadRequest_Content{content},
	}))
	if err != nil {
		return nil, err
	}
	if len(uploadResponse.Msg.Commits) != 1 {
		return nil, syserror.Newf("unexpected number of commits returned from server: %d", len(uploadResponse.Msg.Commits))
	}

	protoCommit := uploadResponse.Msg.Commits[0]

	commitID, err := uuidutil.FromDashless(protoCommit.Id)
	if err != nil {
		return nil, err
	}
	pluginKey, err := bufplugin.NewPluginKey(
		pluginFullName,
		commitID,
		func() (bufplugin.Digest, error) {
			return v1beta1ProtoToDigest(protoCommit.Digest)
		},
	)
	if err != nil {
		return nil, err
	}
	return pluginKey, nil
}

// v1beta1ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func v1beta1ProtoToDigest(protoDigest *pluginv1beta1.Digest) (bufplugin.Digest, error) {
	digestType, err := v1beta1ProtoToDigestType(protoDigest.Type)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.NewDigest(protoDigest.Value)
	if err != nil {
		return nil, err
	}
	return bufplugin.NewDigest(digestType, bufcasDigest)
}

var (
	v1beta1ProtoDigestTypeToDigestType = map[pluginv1beta1.DigestType]bufplugin.DigestType{
		pluginv1beta1.DigestType_DIGEST_TYPE_P1: bufplugin.DigestTypeP1,
	}
)

func v1beta1ProtoToDigestType(protoDigestType pluginv1beta1.DigestType) (bufplugin.DigestType, error) {
	digestType, ok := v1beta1ProtoDigestTypeToDigestType[protoDigestType]
	if !ok {
		return 0, fmt.Errorf("unknown pluginv1beta1.DigestType: %v", protoDigestType)
	}
	return digestType, nil
}

func uploadImage(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	pluginFullName bufplugin.PluginFullName,
	uploadServiceClient pluginv1beta1connect.UploadServiceClient,
) (_ bufplugin.PluginKey, retErr error) {
	if flags.ImageConfig == "" {
		return nil, appcmd.NewInvalidArgumentErrorf("--%s is required", imageConfigFlagName)
	}
	source, err := bufcli.GetInputValue(container, "" /* The input hashtag is not supported here */, ".")
	if err != nil {
		return nil, err
	}
	storageProvider := newStorageosProvider(flags.DisableSymlinks)
	sourceBucket, err := storageProvider.NewReadWriteBucket(source)
	if err != nil {
		return nil, err
	}
	options := []bufremotepluginconfig.ConfigOption{
		bufremotepluginconfig.WithOverrideRemote(pluginFullName.Registry()),
	}
	pluginConfig, err := bufremotepluginconfig.GetConfigForBucket(ctx, sourceBucket, options...)
	if err != nil {
		return nil, err
	}

	dockerClient, err := bufremoteplugindocker.NewClient(container.Logger(), bufcli.Version)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, dockerClient.Close())
	}()
	machine, err := netrc.GetMachineForName(container, pluginFullName.Registry())
	if err != nil {
		return nil, err
	}
	authConfig := &bufremoteplugindocker.RegistryAuthConfig{}
	if machine != nil {
		authConfig.ServerAddress = machine.Name()
		authConfig.Username = machine.Login()
		authConfig.Password = machine.Password()
	}
	// Resolve the image reference.
	dockerInspectResponse, err := dockerClient.Inspect(ctx, flags.Image)
	if err != nil {
		return nil, err
	}
	imageID := dockerInspectResponse.ImageID

	currentImageDigest := ""
	{
		// TODO: need to resolve the current image digest.
	}
	imageDigest, err := findExistingDigestForImageID(ctx, pluginFullName, authConfig, imageID, currentImageDigest)
	if err != nil {
		return nil, err
	}
	if imageDigest == "" {
		imageDigest, err = pushImage(ctx, dockerClient, authConfig, pluginConfig, imageID)
		if err != nil {
			return nil, err
		}
	}
	// TODO: log image digest wasn't pushed.

	plugin, err := bufremoteplugin.NewPlugin(
		pluginConfig.PluginVersion,
		pluginConfig.Dependencies,
		pluginConfig.Registry,
		imageDigest,
		pluginConfig.SourceURL,
		pluginConfig.Description,
	)
	if err != nil {
		return nil, err
	}
	// TODO: upload the image to the BSR
	_ = plugin
	content := &pluginv1beta1.UploadImageRequest_Content{
		PluginRef: &pluginv1beta1.PluginRef{
			Value: &pluginv1beta1.PluginRef_Name_{
				Name: &pluginv1beta1.PluginRef_Name{
					Owner:  pluginFullName.Owner(),
					Plugin: pluginFullName.Name(),
				},
			},
		},
		Version:               "",
		Revision:              0,
		LicenseUrl:            "",
		LicenseSpdxIdentifier: "",
		CodeGeneration:        &pluginv1beta1.CodeGenerationConfig{},
	}
	b, _ := protojson.MarshalOptions{Multiline: true}.Marshal(content)
	fmt.Println(string(b))

	return nil, fmt.Errorf("not implemented")
}

func validateFlags(flags *flags) error {
	if err := validateLabelFlags(flags); err != nil {
		return err
	}
	if err := validateTypeFlags(flags); err != nil {
		return err
	}
	return nil
}

func validateLabelFlags(flags *flags) error {
	return validateLabelFlagValues(flags)
}

func validateTypeFlags(flags *flags) error {
	var usedFlags []string
	if flags.Image != "" {
		usedFlags = append(usedFlags, imageFlagName)
	}
	if flags.CheckBinary != "" {
		usedFlags = append(usedFlags, checkBinaryFlagName)
	}
	if len(usedFlags) > 1 {
		usedFlagsErrStr := strings.Join(
			slicesext.Map(
				usedFlags,
				func(flag string) string { return fmt.Sprintf("--%s", flag) },
			),
			", ",
		)
		return appcmd.NewInvalidArgumentErrorf("These flags cannot be used in combination with one another: %s", usedFlagsErrStr)
	}
	if flags.Image != "" && flags.ImageConfig == "" {
		return appcmd.NewInvalidArgumentErrorf(
			"--%s is required if --%s is set",
			imageConfigFlagName,
			imageFlagName,
		)
	}
	//if flags.Binary != "" && flags.BinaryType == "" {
	//	return appcmd.NewInvalidArgumentErrorf(
	//		"--%s is required if --%s is set",
	//		binaryTypeFlagName,
	//		binaryFlagName,
	//	)
	//}
	return nil
}

func validateLabelFlagValues(flags *flags) error {
	for _, label := range flags.Labels {
		if label == "" {
			return appcmd.NewInvalidArgumentErrorf("--%s requires a non-empty string", labelFlagName)
		}
	}
	return nil
}

var (
	zstdEncoder, _ = zstd.NewWriter(nil)
)

func zstdCompress(src []byte) []byte {
	return zstdEncoder.EncodeAll(src, make([]byte, 0, len(src)))
}

func newStorageosProvider(disableSymlinks bool) storageos.Provider {
	var options []storageos.ProviderOption
	if !disableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	return storageos.NewProvider(options...)
}

// pushImage pushes the image to the OCI registry. It returns the digest of the
// pushed image.
func pushImage(
	ctx context.Context,
	dockerClient bufremoteplugindocker.Client,
	authConfig *bufremoteplugindocker.RegistryAuthConfig,
	pluginConfig *bufremotepluginconfig.Config,
	image string,
) (_ string, retErr error) {
	tagResponse, err := dockerClient.Tag(ctx, image, pluginConfig)
	if err != nil {
		return "", err
	}
	createdImage := tagResponse.Image
	// We tag a Docker image using a unique ID label each time.
	// After we're done publishing the image, we delete it to not leave a lot of images left behind.
	defer func() {
		if _, err := dockerClient.Delete(ctx, createdImage); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("failed to delete image %q", createdImage))
		}
	}()
	pushResponse, err := dockerClient.Push(ctx, createdImage, authConfig)
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
	pluginFullName bufplugin.PluginFullName,
	authConfig *bufremoteplugindocker.RegistryAuthConfig,
	imageID string,
	currentImageDigest string,
) (string, error) {
	repo, err := name.NewRepository(pluginFullName.String())
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
	for _, tag := range tags {
		remoteImageID, imageDigest, err := getImageIDAndDigestFromReference(ctx, repo.Tag(tag), remoteOpts...)
		if err != nil {
			return "", err
		}
		if remoteImageID == imageID {
			return imageDigest, nil
		}
	}
	return "", nil
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
