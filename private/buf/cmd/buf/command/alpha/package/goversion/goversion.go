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

package goversion

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	pluginFlagName = "plugin"
	moduleFlagName = "module"
	registryName   = "go"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " --module=<buf.build/owner/repository[:ref]> --plugin=<buf.build/owner/plugin[:version]>",
		Short: bufcli.PackageVersionShortDescription(registryName),
		Long:  bufcli.PackageVersionLongDescription(registryName, name, "buf.build/bufbuild/connect-go"),
		Args:  cobra.NoArgs,
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
	flagSet.StringVar(&f.Module, moduleFlagName, "", "The module reference to resolve")
	flagSet.StringVar(&f.Plugin, pluginFlagName, "", fmt.Sprintf("The %s plugin reference to resolve", registryName))
	_ = cobra.MarkFlagRequired(flagSet, moduleFlagName)
	_ = cobra.MarkFlagRequired(flagSet, pluginFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnAlphaCommand(ctx, container)
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	moduleReference, err := bufmoduleref.ModuleReferenceForString(flags.Module)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("failed parsing module reference: %s", err.Error())
	}
	pluginIdentity, pluginVersion, err := bufpluginref.ParsePluginIdentityOptionalVersion(flags.Plugin)
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("failed parsing plugin reference: %s", err.Error())
	}
	if pluginIdentity.Remote() != moduleReference.Remote() {
		return appcmd.NewInvalidArgumentError("module and plugin must be from the same remote")
	}
	resolver := connectclient.Make(
		clientConfig,
		moduleReference.Remote(),
		registryv1alpha1connect.NewResolveServiceClient,
	)
	packageVersion, err := resolver.GetGoVersion(ctx, connect.NewRequest(
		&registryv1alpha1.GetGoVersionRequest{
			ModuleReference: &registryv1alpha1.LocalModuleReference{
				Owner:      moduleReference.Owner(),
				Repository: moduleReference.Repository(),
				Reference:  moduleReference.Reference(),
			},
			PluginReference: &registryv1alpha1.GetRemotePackageVersionPlugin{
				Owner:   pluginIdentity.Owner(),
				Name:    pluginIdentity.Plugin(),
				Version: pluginVersion,
			},
		},
	))
	if err != nil {
		return err
	}
	if _, err := container.Stdout().Write([]byte(packageVersion.Msg.Version)); err != nil {
		return err
	}
	return nil
}
