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
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufplugindocker"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
			// Not fatal - we did our best to clean up after ourselves.
			container.Logger().Warn("failed to delete image", zap.String("image", buildResponse.Image), zap.Error(err))
		}
	}()

	// TODO: Implement authentication to BSR registry using ~/.netrc
	_, err = client.Push(ctx, buildResponse.Image, &bufplugindocker.RegistryAuthConfig{})
	if err != nil {
		return err
	}

	// TODO: Now that the imageDigest is resolved, create a bufplugin.Plugin,
	// then map it into a *registryv1alpha1.RemotePlugin
	// (or *registryv1alpha1.CreateRemotePluginRequest) so that it can be pushed
	// to the BSR.
	//
	//  plugin, err := bufplugin.NewPlugin(
	//    pluginConfig.Options(),
	//    pluginConfig.Runtime(),
	//    imageDigest,
	//  )
	//  if err != nil {
	//    return err
	//  }
	//  protoPlugin, err := bufplugin.PluginToProtoPlugin(plugin)
	//  if err != nil {
	//    return err
	//  }
	//  // Use the RemotePluginService (shown below) ...
	//
	// ---
	//
	// TODO: At this point, it's possible that an OCI registry entry was created
	// without successfully creating the metadata in the BSR. If the user tries
	// to push again, the OCI registry entry might already exist. We need to explore
	// the OCI registry API to see what we can do to prevent such cases. This might
	// involve some combination of a cleanup process, a two-phase commit flow (i.e.
	// first create the metadata in the BSR, then push to the OCI registry, etc), or
	// something else entirely. This might not actually be an issue depending on the
	// OCI registry API.
	//
	//
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	service, err := apiProvider.NewPluginService(ctx, pluginConfig.Name.Remote())
	if err != nil {
		return err
	}
	// TODO: Refactor this to be a new endpoint that uses the protoPlugin
	// object. This is simply left as a placeholder.
	if _, err := service.CreatePlugin(
		ctx,
		pluginConfig.Name.Owner(),
		pluginConfig.Name.Plugin(),
		registryv1alpha1.PluginVisibility_PLUGIN_VISIBILITY_PUBLIC,
	); err != nil {
		return err
	}
	// TODO: Print out the plugin that was just created.
	return nil
}
