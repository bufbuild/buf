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

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufplugindocker"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	connect_go "github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

const (
	errorFormatFlagName     = "error-format"
	disableSymlinksFlagName = "disable-symlinks"
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
		Long:  bufcli.GetSourceDirLong(`the source to push`),
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
	ErrorFormat     string
	DisableSymlinks bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindDisableSymlinks(flagSet, &f.DisableSymlinks, disableSymlinksFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
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
	source, err := bufcli.GetInputValue(container, "" /* The input hashtag is not supported here */, ".")
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	sourceBucket, err := storageosProvider.NewReadWriteBucket(source)
	if err != nil {
		return err
	}
	existingConfigFilePath, err := bufpluginconfig.ExistingConfigFilePath(ctx, sourceBucket)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	if existingConfigFilePath == "" {
		return fmt.Errorf("please define a %s configuration file in the target directory", bufpluginconfig.ExternalConfigFilePath)
	}
	pluginConfig, err := bufpluginconfig.GetConfigForBucket(ctx, sourceBucket)
	if err != nil {
		return err
	}
	// TODO: Once we support multiple plugin source types, this could be abstracted away
	// in the bufpluginsource package. This is much simpler for now though.
	dockerfile, err := sourceBucket.Get(ctx, bufplugindocker.SourceFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("please define a %s plugin source file in the target directory", bufplugindocker.SourceFilePath)
		}
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, dockerfile.Close())
	}()

	client, err := bufplugindocker.NewClient(container.Logger())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, client.Close())
	}()

	buildResponse, err := client.Build(ctx, dockerfile, pluginConfig, bufplugindocker.WithConfigDirPath(container.ConfigDirPath()))
	if err != nil {
		return err
	}

	// We build a Docker image using a unique ID label each time.
	// After we're done publishing the image, we delete it to not leave a lot of images left behind.
	// buildkit maintains a separate build cache so removing the image doesn't appear to impact future rebuilds.
	defer func() {
		if _, err := client.Delete(ctx, buildResponse.Image); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("failed to delete image %q", buildResponse.Image))
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
	_, err = client.Push(ctx, buildResponse.Image, authConfig)
	if err != nil {
		return err
	}
	plugin, err := bufplugin.NewPlugin(
		pluginConfig.PluginVersion,
		pluginConfig.Options,
		pluginConfig.Runtime,
		buildResponse.Digest,
	)
	if err != nil {
		return err
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewPluginCurationService(ctx, pluginConfig.Name.Remote())
	if err != nil {
		return err
	}
	var nextRevision int32
	currentRevision, err := service.GetLatestCuratedPlugin(ctx, pluginConfig.Name.Owner(), pluginConfig.Name.Plugin(), pluginConfig.PluginVersion)
	if err != nil {
		if connect_go.CodeOf(err) != connect_go.CodeNotFound {
			return err
		}
		nextRevision = 0
	} else {
		nextRevision = currentRevision.Revision + 1
		if nextRevision < 0 {
			return fmt.Errorf("next plugin revision out of range")
		}
	}
	if _, err := service.CreateCuratedPlugin(
		ctx,
		pluginConfig.Name.Owner(),
		pluginConfig.Name.Plugin(),
		bufplugin.PluginToProtoPluginLanguage(plugin),
		plugin.Version(),
		plugin.ContainerImageDigest(),
		bufplugin.PluginOptionsToOptionsSlice(plugin.Options()),
		nil, // dependencies
		"",  // sourceUrl
		"",  // description
		bufplugin.PluginRuntimeToProtoRuntimeConfig(plugin.Runtime()),
		nextRevision,
	); err != nil {
		return err
	}
	// TODO: Determine how to print curated plugin - see bufprint
	return nil
}
