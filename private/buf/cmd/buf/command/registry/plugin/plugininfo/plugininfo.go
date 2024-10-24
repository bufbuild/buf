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

package plugininfo

import (
	"context"
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const formatFlagName = "format"

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Get a BSR plugin",
		Args:  appcmd.ExactArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	pluginFullName, err := bufplugin.ParsePluginFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}

	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	pluginServiceClient := bufapi.NewClientProvider(clientConfig).
		PluginV1Beta1PluginServiceClient(pluginFullName.Registry())

	pluginsResponse, err := pluginServiceClient.GetPlugins(ctx, connect.NewRequest(
		&pluginv1beta1.GetPluginsRequest{
			PluginRefs: []*pluginv1beta1.PluginRef{
				{
					Value: &pluginv1beta1.PluginRef_Name_{
						Name: &pluginv1beta1.PluginRef_Name{
							Owner:  pluginFullName.Owner(),
							Plugin: pluginFullName.Name(),
						},
					},
				},
			},
		},
	))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewPluginNotFoundError(container.Arg(0))
		}
		return err
	}
	plugins := pluginsResponse.Msg.Plugins
	if len(plugins) != 1 {
		return syserror.Newf("unexpected number of plugins returned from server: %d", len(plugins))
	}
	return bufprint.PrintEntity(
		container.Stdout(),
		format,
		bufprint.NewPluginEntity(plugins[0], pluginFullName),
	)
}
