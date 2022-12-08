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

package pluginpush

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"buf.build/gen/go/bufbuild/buf/bufbuild/connect-go/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufplugindocker"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netextended"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	formatFlagName          = "format"
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
	overrideRemoteFlagName  = "override-remote"
	imageFlagName           = "image"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <source>",
		Short: "Push a plugin to a registry.",
		Long:  bufcli.GetSourceDirLong(`the source to push (directory containing buf.plugin.yaml or plugin release zip)`),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format          string
	ErrorFormat     string
	DisableSymlinks bool
	OverrideRemote  string
	Image           string
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
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.OverrideRemote,
		overrideRemoteFlagName,
		"",
		"Override the default remote found in buf.plugin.yaml name and dependencies.",
	)
	flagSet.StringVar(
		&f.Image,
		imageFlagName,
		"",
		"Existing image to push.",
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	bufcli.WarnAlphaCommand(ctx, container)
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	if len(flags.OverrideRemote) > 0 {
		if _, err := netextended.ValidateHostname(flags.OverrideRemote); err != nil {
			return fmt.Errorf("%s: %w", overrideRemoteFlagName, err)
		}
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	source, err := bufcli.GetInputValue(container, "" /* The input hashtag is not supported here */, ".")
	if err != nil {
		return err
	}
	storageProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
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
	existingConfigFilePath, err := bufpluginconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	if existingConfigFilePath == "" {
		return fmt.Errorf("please define a %s configuration file in the target directory", bufpluginconfig.ExternalConfigFilePath)
	}
	var options []bufpluginconfig.ConfigOption
	if len(flags.OverrideRemote) > 0 {
		options = append(options, bufpluginconfig.WithOverrideRemote(flags.OverrideRemote))
	}
	pluginConfig, err := bufpluginconfig.GetConfigForBucket(ctx, sourceBucket, options...)
	if err != nil {
		return err
	}
	outputLanguages, err := bufplugin.OutputLanguagesToProtoLanguages(pluginConfig.OutputLanguages)
	if err != nil {
		return err
	}

	client, err := bufplugindocker.NewClient(container.Logger())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, client.Close())
	}()

	var imageToTag string
	if flags.Image != "" {
		imageToTag = flags.Image
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
			if err := image.Close(); !errors.Is(err, os.ErrClosed) {
				retErr = multierr.Append(retErr, err)
			}
		}()
		imageToTag = loadResponse.ImageID
	}
	tagResponse, err := client.Tag(ctx, imageToTag, pluginConfig)
	if err != nil {
		return err
	}
	createdImage := tagResponse.Image

	// We tag a Docker image using a unique ID label each time.
	// After we're done publishing the image, we delete it to not leave a lot of images left behind.
	defer func() {
		if _, err := client.Delete(ctx, createdImage); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("failed to delete image %q", createdImage))
		}
	}()

	machine, err := netrc.GetMachineForName(container, pluginConfig.Name.Remote())
	if err != nil {
		return err
	}
	authConfig := &bufplugindocker.RegistryAuthConfig{}
	if machine != nil {
		authConfig.ServerAddress = machine.Name()
		authConfig.Username = machine.Login()
		authConfig.Password = machine.Password()
	}
	pushResponse, err := client.Push(ctx, createdImage, authConfig)
	if err != nil {
		return err
	}
	plugin, err := bufplugin.NewPlugin(
		pluginConfig.PluginVersion,
		pluginConfig.Dependencies,
		pluginConfig.Registry,
		pushResponse.Digest,
		pluginConfig.SourceURL,
		pluginConfig.Description,
	)
	if err != nil {
		return err
	}
	protoRegistryConfig, err := bufplugin.PluginRegistryToProtoRegistryConfig(plugin.Registry())
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
	var nextRevision uint32
	latestPluginResp, err := service.GetLatestCuratedPlugin(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetLatestCuratedPluginRequest{
			Owner:    pluginConfig.Name.Owner(),
			Name:     pluginConfig.Name.Plugin(),
			Version:  pluginConfig.PluginVersion,
			Revision: 0, // get latest revision for the plugin version.
		}),
	)
	if err != nil {
		if connect.CodeOf(err) != connect.CodeNotFound {
			return err
		}
		nextRevision = 1
	} else {
		nextRevision = latestPluginResp.Msg.Plugin.Revision + 1
	}
	var curatedPlugin *registryv1alpha1.CuratedPlugin
	createPluginResp, err := service.CreateCuratedPlugin(
		ctx,
		connect.NewRequest(&registryv1alpha1.CreateCuratedPluginRequest{
			Owner:                pluginConfig.Name.Owner(),
			Name:                 pluginConfig.Name.Plugin(),
			RegistryType:         bufplugin.PluginToProtoPluginRegistryType(plugin),
			Version:              plugin.Version(),
			ContainerImageDigest: plugin.ContainerImageDigest(),
			Dependencies:         bufplugin.PluginReferencesToCuratedProtoPluginReferences(plugin.Dependencies()),
			SourceUrl:            plugin.SourceURL(),
			Description:          plugin.Description(),
			RegistryConfig:       protoRegistryConfig,
			Revision:             nextRevision,
			OutputLanguages:      outputLanguages,
			SpdxLicenseId:        pluginConfig.SPDXLicenseID,
			LicenseUrl:           pluginConfig.LicenseURL,
		}),
	)
	if err != nil {
		if connect.CodeOf(err) != connect.CodeAlreadyExists {
			return err
		}
		// Plugin with the same image digest and metadata already exists
		container.Logger().Info(
			"plugin already exists",
			zap.String("name", pluginConfig.Name.IdentityString()),
			zap.String("digest", plugin.ContainerImageDigest()),
		)
		curatedPlugin = latestPluginResp.Msg.Plugin
	} else {
		curatedPlugin = createPluginResp.Msg.Configuration
	}
	return bufprint.NewCuratedPluginPrinter(container.Stdout()).PrintCuratedPlugin(ctx, format, curatedPlugin)
}

func unzipPluginToSourceBucket(ctx context.Context, pluginZip string, size int64, bucket storage.ReadWriteBucket) (retErr error) {
	f, err := os.Open(pluginZip)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, f.Close())
	}()
	return storagearchive.Unzip(ctx, f, size, bucket, nil, 0)
}

func loadDockerImage(ctx context.Context, bucket storage.ReadBucket) (storage.ReadObjectCloser, error) {
	image, err := bucket.Get(ctx, bufplugindocker.ImagePath)
	if storage.IsNotExist(err) {
		return nil, fmt.Errorf("unable to find a %s plugin image: %w", bufplugindocker.ImagePath, err)
	}
	return image, nil
}
