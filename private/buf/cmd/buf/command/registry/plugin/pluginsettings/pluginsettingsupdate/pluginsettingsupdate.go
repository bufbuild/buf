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

package pluginsettingsupdate

import (
	"context"
	"fmt"

	pluginv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/plugin/v1beta1"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	visibilityFlagName  = "visibility"
	descriptionFlagName = "description"
	urlFlagName         = "url"
)

// NewCommand returns a new Command.
func NewCommand(name string, builder appext.SubCommandBuilder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Update BSR plugin settings",
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
	Visibility   string
	Description  *string
	URL          *string
	DefaultLabel string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindVisibility(flagSet, &f.Visibility, visibilityFlagName, true)
	bufcli.BindStringPointer(
		flagSet,
		descriptionFlagName,
		&f.Description,
		"The new description for the plugin",
	)
	bufcli.BindStringPointer(
		flagSet,
		urlFlagName,
		&f.URL,
		"The new URL for the plugin",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	pluginFullName, err := bufparse.ParseFullName(container.Arg(0))
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	visibility, err := bufcli.VisibilityFlagToPluginVisibilityAllowUnspecified(flags.Visibility)
	if err != nil {
		return appcmd.WrapInvalidArgumentError(err)
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	var visibilityUpdate *pluginv1beta1.PluginVisibility
	if visibility != pluginv1beta1.PluginVisibility_PLUGIN_VISIBILITY_UNSPECIFIED {
		visibilityUpdate = &visibility
	}

	pluginServiceClient := bufregistryapiplugin.NewClientProvider(clientConfig).
		V1Beta1PluginServiceClient(pluginFullName.Registry())

	pluginResponse, err := pluginServiceClient.UpdatePlugins(ctx, connect.NewRequest(
		&pluginv1beta1.UpdatePluginsRequest{
			Values: []*pluginv1beta1.UpdatePluginsRequest_Value{
				{
					PluginRef: &pluginv1beta1.PluginRef{
						Value: &pluginv1beta1.PluginRef_Name_{
							Name: &pluginv1beta1.PluginRef_Name{
								Owner:  pluginFullName.Owner(),
								Plugin: pluginFullName.Name(),
							},
						},
					},
					Visibility: visibilityUpdate,
				},
			},
		},
	))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewModuleNotFoundError(container.Arg(0))
		}
		return err
	}
	plugins := pluginResponse.Msg.Plugins
	if len(plugins) != 1 {
		return syserror.Newf("unexpected number of plugins returned from server: %d", len(plugins))
	}
	_, err = fmt.Fprintf(container.Stdout(), "Updated %s.\n", pluginFullName)
	if err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
