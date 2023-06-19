// Copyright 2020-2023 Buf Technologies, Inc.
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

package versionget

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	PluginFlagName = "plugin"
	ModuleFlagName = "module"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " --module=<buf.build/owner/repository[:ref]> --plugin=<buf.build/owner/plugin[:version]>",
		Short: "Resolve module and plugin reference to a specific remote package version",
		Long: `This command returns the version of the asset to be used with one of the supported remote package registries.
For example npm, go proxy, maven, swift.

Examples:

Get the version of the eliza module and the connect-go plugin for use with go modules.
    $ buf alpha package version get --module=buf.build/bufbuild/eliza --plugin=buf.build/bufbuild/connect-go
        v1.7.0-20230609151053-e682db0d9918.1

Use a specific module version and plugin version.
    $ buf alpha package version get --module=buf.build/bufbuild/eliza:e682db0d99184be88b41c4405ea8a417 --plugin=buf.build/bufbuild/connect-go:v1.7.0
        v1.7.0-20230609151053-e682db0d9918.1
`,
		Args: cobra.NoArgs,
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
	Plugin string
	Module string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Module, ModuleFlagName, "", "The module reference to resolve")
	flagSet.StringVar(&f.Plugin, PluginFlagName, "", "The plugin reference to resolve")
	_ = cobra.MarkFlagRequired(flagSet, ModuleFlagName)
	_ = cobra.MarkFlagRequired(flagSet, PluginFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnAlphaCommand(ctx, container)
	moduleReference, err := bufmoduleref.ModuleReferenceForString(flags.Module)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("failed parsing module reference: %s", err.Error())
	}
	pluginReference, err := bufpluginref.PluginReferenceOptionalVersion(flags.Plugin)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("failed parsing plugin reference: %s", err.Error())
	}
	if pluginReference.Remote() != moduleReference.Remote() {
		return appcmd.NewInvalidArgumentError("module and plugin must be from the same remote")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	resolver := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewResolveServiceClient,
	)
	packageVersion, err := resolver.GetRemotePackageVersion(ctx, connect.NewRequest(
		&registryv1alpha1.GetRemotePackageVersionRequest{
			ModuleReference: &registryv1alpha1.LocalModuleReference{
				Owner:      moduleReference.Owner(),
				Repository: moduleReference.Repository(),
				Reference:  moduleReference.Reference(),
			},
			PluginReference: &registryv1alpha1.GetRemotePackageVersionPlugin{
				Owner:   pluginReference.Owner(),
				Name:    pluginReference.Plugin(),
				Version: pluginReference.Version(),
			},
		},
	))
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write([]byte(packageVersion.Msg.Version))
	return err
}
