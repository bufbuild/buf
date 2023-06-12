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

package resolve

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

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/repository[:ref]> <buf.build/owner/plugin[:version]>",
		Short: "Resolve module and plugin version to a specific registry version",
		Args:  cobra.ExactArgs(2),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct{}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(_ *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appflag.Container,
	_ *flags,
) error {
	bufcli.WarnAlphaCommand(ctx, container)
	moduleReference, err := bufmoduleref.ModuleReferenceForString(container.Arg(0))
	if err != nil {
		return appcmd.NewInvalidArgumentErrorf("failed parsing module reference: %s", err.Error())
	}
	pluginReference, err := bufpluginref.PluginReferenceOptionalVersion(container.Arg(1))
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
	resolvedReference, err := resolver.ResolveReference(ctx, connect.NewRequest(
		&registryv1alpha1.ResolveReferenceRequest{
			ModuleReference: &registryv1alpha1.LocalModuleReference{
				Owner:      moduleReference.Owner(),
				Repository: moduleReference.Repository(),
				Reference:  moduleReference.Reference(),
			},
			PluginReference: &registryv1alpha1.ResolveReferencePlugin{
				Owner:   pluginReference.Owner(),
				Name:    pluginReference.Plugin(),
				Version: pluginReference.Version(),
			},
		},
	))
	_, err = container.Stdout().Write([]byte(resolvedReference.Msg.Version))
	return err
}
