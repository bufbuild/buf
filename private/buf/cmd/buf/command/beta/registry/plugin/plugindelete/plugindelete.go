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

package plugindelete

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/spf13/pflag"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/plugin[:version]>",
		Short: "Delete a plugin from the registry",
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct{}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	identity, version, _ := strings.Cut(container.Arg(0), ":")
	pluginIdentity, err := bufremotepluginref.PluginIdentityForString(identity)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	if version != "" {
		if err := bufremotepluginref.ValidatePluginVersion(version); err != nil {
			return appcmd.NewInvalidArgumentError(err.Error())
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	service := connectclient.Make(
		clientConfig,
		pluginIdentity.Remote(),
		registryv1alpha1connect.NewPluginCurationServiceClient,
	)
	if _, err := service.DeleteCuratedPlugin(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.DeleteCuratedPluginRequest{
				Owner:   pluginIdentity.Owner(),
				Name:    pluginIdentity.Plugin(),
				Version: version,
			},
		),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fmt.Errorf("the plugin %s does not exist", container.Arg(0))
		}
		return err
	}
	return nil
}
