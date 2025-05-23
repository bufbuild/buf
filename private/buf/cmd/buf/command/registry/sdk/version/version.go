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

package version

import (
	"context"
	"fmt"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	pluginFlagName = "plugin"
	moduleFlagName = "module"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " --module=<buf.build/owner/repository[:ref]> --plugin=<buf.build/owner/plugin[:version]>",
		Short: "Resolve module and plugin reference to a specific Generated SDK version",
		Long: `This command returns the version of the Generated SDK for the given module and plugin.
Examples:

Get the version of the eliza module and the go plugin for use with the Go module proxy.
    $ buf registry sdk version --module=buf.build/connectrpc/eliza --plugin=buf.build/protocolbuffers/go
    v1.33.0-20230913231627-233fca715f49.1

Use a specific module version and plugin version.
    $ buf registry sdk version --module=buf.build/connectrpc/eliza:233fca715f49425581ec0a1b660be886 --plugin=buf.build/protocolbuffers/go:v1.32.0
    v1.32.0-20230913231627-233fca715f49.1`,
		Args: appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

// TODO FUTURE: Add a --format flag, supports text (current behavior) and json, json will output information
// such as resolved module commit, plugin version and revision, package-ecosystem, and full version.

type flags struct {
	Plugin string
	Module string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Module, moduleFlagName, "", "The module reference to resolve")
	flagSet.StringVar(&f.Plugin, pluginFlagName, "", "The plugin reference to resolve")
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
	if pluginIdentity.Remote() != moduleRef.FullName().Registry() {
		return appcmd.NewInvalidArgumentError("module and plugin must be from the same BSR instance")
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	pluginCurationServiceClient := connectclient.Make(
		clientConfig,
		moduleRef.FullName().Registry(),
		registryv1alpha1connect.NewPluginCurationServiceClient,
	)
	resolveServiceClient := connectclient.Make(
		clientConfig,
		moduleRef.FullName().Registry(),
		registryv1alpha1connect.NewResolveServiceClient,
	)
	getLatestCuratedPluginResponse, err := pluginCurationServiceClient.GetLatestCuratedPlugin(
		ctx,
		connect.NewRequest(
			registryv1alpha1.GetLatestCuratedPluginRequest_builder{
				Owner:   pluginIdentity.Owner(),
				Name:    pluginIdentity.Plugin(),
				Version: pluginVersion,
			}.Build(),
		),
	)
	if err != nil {
		return err
	}
	pluginRegistryType := getLatestCuratedPluginResponse.Msg.GetPlugin().GetRegistryType()
	if pluginRegistryType == 0 {
		return fmt.Errorf("plugin %q is not associated with a package ecosystem", flags.Plugin)
	}
	moduleReference := registryv1alpha1.LocalModuleReference_builder{
		Owner:      moduleRef.FullName().Owner(),
		Repository: moduleRef.FullName().Name(),
		Reference:  moduleRef.Ref(),
	}.Build()
	pluginReference := registryv1alpha1.GetRemotePackageVersionPlugin_builder{
		Owner:   pluginIdentity.Owner(),
		Name:    pluginIdentity.Plugin(),
		Version: pluginVersion,
	}.Build()
	var version string
	switch pluginRegistryType {
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_GO:
		goVersionResponse, err := resolveServiceClient.GetGoVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetGoVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = goVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NPM:
		npmVersionResponse, err := resolveServiceClient.GetNPMVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetNPMVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = npmVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_MAVEN:
		mavenVersionResponse, err := resolveServiceClient.GetMavenVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetMavenVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = mavenVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_SWIFT:
		swiftVersionResponse, err := resolveServiceClient.GetSwiftVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetSwiftVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = swiftVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_PYTHON:
		pythonVersionResponse, err := resolveServiceClient.GetPythonVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetPythonVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = pythonVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CARGO:
		cargoVersionResponse, err := resolveServiceClient.GetCargoVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetCargoVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = cargoVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_NUGET:
		nugetVersionResponse, err := resolveServiceClient.GetNugetVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetNugetVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = nugetVersionResponse.Msg.GetVersion()
	case registryv1alpha1.PluginRegistryType_PLUGIN_REGISTRY_TYPE_CMAKE:
		cmakeVersionResponse, err := resolveServiceClient.GetCmakeVersion(
			ctx,
			connect.NewRequest(
				registryv1alpha1.GetCmakeVersionRequest_builder{
					ModuleReference: moduleReference,
					PluginReference: pluginReference,
				}.Build(),
			),
		)
		if err != nil {
			return err
		}
		version = cmakeVersionResponse.Msg.GetVersion()
	default:
		return syserror.Newf("unknown PluginRegistryType: %v", pluginRegistryType)
	}

	if _, err := container.Stdout().Write([]byte(version + "\n")); err != nil {
		return err
	}
	return nil
}
