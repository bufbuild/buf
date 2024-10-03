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

package plugincreate

import (
	"context"
	"fmt"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
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

const (
	formatFlagName      = "format"
	visibilityFlagName  = "visibility"
	defaultLabeFlagName = "default-label-name"

	defaultDefaultLabel = "main"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Create a BSR plugin",
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
	Format       string
	Visibility   string
	DefautlLabel string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindVisibility(flagSet, &f.Visibility, visibilityFlagName, false)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
	flagSet.StringVar(
		&f.DefautlLabel,
		defaultLabeFlagName,
		defaultDefaultLabel,
		"The default label name of the module",
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
	fmt.Println("pluginFullName", pluginFullName)

	visibility, err := bufcli.VisibilityFlagToPluginVisibilityAllowUnspecified(flags.Visibility)
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

	pluginResponse, err := pluginServiceClient.CreatePlugins(ctx, connect.NewRequest(
		&pluginv1beta1.CreatePluginsRequest{
			Values: []*pluginv1beta1.CreatePluginsRequest_Value{
				{
					OwnerRef: &ownerv1.OwnerRef{
						Value: &ownerv1.OwnerRef_Name{
							Name: pluginFullName.Owner(),
						},
					},
					Name:       pluginFullName.Name(),
					Visibility: visibility,
					SourceUrl:  "", // TODO: flag?
					Type:       pluginv1beta1.PluginType_PLUGIN_TYPE_CHECK,
				},
			},
		},
	))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return bufcli.NewPluginNameAlreadyExistsError(pluginFullName.String())
		}
		return err
	}
	plugins := pluginResponse.Msg.Plugins
	if len(plugins) != 1 {
		return syserror.Newf("unexpected number of plugins returned from server: %d", len(plugins))
	}
	if format == bufprint.FormatText {
		_, err = fmt.Fprintf(container.Stdout(), "Created %s.\n", pluginFullName)
		if err != nil {
			return syserror.Wrap(err)
		}
		return nil
	}
	return bufprint.PrintNames(
		container.Stdout(),
		format,
		bufprint.NewPluginEntity(plugins[0], pluginFullName),
	)
}
