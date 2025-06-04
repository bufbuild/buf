// Copyright 2020-2025 Buf Technologies, Inc.
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

package sdkinfo

import (
	"context"
	"fmt"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/pflag"
)

const (
	formatFlagName  = "format"
	moduleFlagName  = "module"
	pluginFlagName  = "plugin"
	versionFlagName = "version"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " --module=<remote/owner/repository[:ref]> --plugin=<remote/owner/plugin[:version]>",
		Short: "Get SDK information for the given module, plugin, and optionally version.",
		Long: `This command returns the version information for a Generated SDK based on the specified information.
In order to resolve the SDK information, a module and plugin must be specified.

Examples:

To get the SDK information for the latest commit of a module and latest version of a plugin, you only need to specify the module and plugin.
The following will resolve the SDK information for the latest commit of the connectrpc/eliza module and the latest version of the bufbuild/es plugin:

    $ buf registry sdk info --module=buf.build/connectrpc/eliza --plugin=buf.build/connectrpc/es
    Module
    Owner:  connectrpc
    Name:   eliza
    Commit: <latest commit on default label>

    Plugin
    Owner:    bufbuild
    Name:     es
    Version:  <latest version of plugin>
    Revision: <latest revision of the plugin version>

    Version: <SDK version for the resolved module commit and plugin version>

To get the SDK information for a specific commit of a module and/or a specific version of a plugin, you can specify the commit with the module and/or the version with the plugin.
The following will resolve the SDK information for the specified commit of the connectrpc/eliza module and specified version of the bufbuild/es plugin:

    $ buf registry sdk info --module=buf.build/connectrpc/eliza:d8fbf2620c604277a0ece1ff3a26f2ff --plugin=buf.build/bufbuild/es:v1.2.1
    Module
    Owner:  connectrpc
    Name:   eliza
    Commit: d8fbf2620c604277a0ece1ff3a26f2ff

    Plugin
    Owner:    bufbuild
    Name:     es
    Version:  v1.2.1
    Revision: 1

    Version: 1.2.1-20230727062025-d8fbf2620c60.1

If you have a SDK version and want to know the corresponding module commit and plugin version information for the SDK, you can specify the module and plugin with the version string.
The following will resolve the SDK information for the specified SDK version of the connectrpc/eliza module and bufbuild/es plugin.

    $ buf registry sdk --module=buf.build/connectrpc/eliza --plugin=buf.build/bufbuild/es --version=1.2.1-20230727062025-d8fbf2620c60.1
    Module
    Owner:  connectrpc
    Name:   eliza
    Commit: d8fbf2620c604277a0ece1ff3a26f2ff

    Plugin
    Owner:    bufbuild
    Name:     es
    Version:  v1.2.1
    Revision: 1

    Version: 1.2.1-20230727062025-d8fbf2620c60.1

The module commit and plugin version information are resolved based on the specified SDK version string.

If a module reference and/or plugin version are specified along with the SDK version, then the SDK version will be validated against the specified module reference and/or plugin version.
If there is a mismatch, this command will error.

    $ buf registry sdk info  \
        --module=buf.build/connectrpc/eliza:8b8b971d6fde4dc8ba5d96f9fda7d53c   \
        --plugin=buf.build/bufbuild/es  \
        --version=1.2.1-20230727062025-d8fbf2620c60.1
    Failure: invalid_argument: invalid SDK version v1.2.1-20230727062025-d8fbf2620c60.1 with module short commit d8fbf2620c60 for resolved module reference connectrpc/eliza:8b8b971d6fde4dc8ba5d96f9fda7d53c

In this case, the SDK version provided resolves to a different commit than the commit provided for the module.`,
		Args: appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format  string
	Module  string
	Plugin  string
	Version string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf("The output format to use. Must be one of %s", bufprint.AllFormatsString),
	)
	flagSet.StringVar(&f.Module, moduleFlagName, "", "The module reference for the SDK.")
	flagSet.StringVar(&f.Plugin, pluginFlagName, "", "The plugin reference for the SDK.")
	flagSet.StringVar(&f.Version, versionFlagName, "", "The version of the SDK.")
	_ = appcmd.MarkFlagRequired(flagSet, moduleFlagName)
	_ = appcmd.MarkFlagRequired(flagSet, pluginFlagName)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	moduleRef, err := bufparse.ParseRef(flags.Module)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	pluginIdentity, pluginVersion, err := bufremotepluginref.ParsePluginIdentityOptionalVersion(flags.Plugin)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	if moduleRef.FullName().Registry() != pluginIdentity.Remote() {
		return appcmd.NewInvalidArgumentError("module and plugin must be from the same BSR instance")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	resolveServiceClient := connectclient.Make(
		clientConfig,
		moduleRef.FullName().Registry(),
		registryv1alpha1connect.NewResolveServiceClient,
	)
	res, err := resolveServiceClient.GetSDKInfo(
		ctx,
		connect.NewRequest(registryv1alpha1.GetSDKInfoRequest_builder{
			ModuleReference: registryv1alpha1.LocalModuleReference_builder{
				Owner:      moduleRef.FullName().Owner(),
				Repository: moduleRef.FullName().Name(),
				Reference:  moduleRef.Ref(),
			}.Build(),
			PluginReference: registryv1alpha1.GetRemotePackageVersionPlugin_builder{
				Owner:   pluginIdentity.Owner(),
				Name:    pluginIdentity.Plugin(),
				Version: pluginVersion,
			}.Build(),
			SdkVersion: flags.Version,
		}.Build()),
	)
	if err != nil {
		return err
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return err
	}
	return bufprint.NewSDKInfoPrinter(container.Stdout()).PrintSDKInfo(
		ctx,
		format,
		res.Msg,
	)
}
