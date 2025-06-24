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

package plugindelete

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

const forceFlagName = "force"

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <remote/owner/plugin>",
		Short: "Delete a BSR plugin",
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
	Force bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(
		&f.Force,
		forceFlagName,
		false,
		"Force deletion without confirming. Use with caution",
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
	if !flags.Force {
		if err := bufcli.PromptUserForDelete(container, "entity", pluginFullName.Name()); err != nil {
			return err
		}
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	pluginServiceClient := bufregistryapiplugin.NewClientProvider(clientConfig).
		V1Beta1PluginServiceClient(pluginFullName.Registry())

	if _, err := pluginServiceClient.DeletePlugins(ctx, connect.NewRequest(
		&pluginv1beta1.DeletePluginsRequest{
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
	)); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewPluginNotFoundError(container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintf(container.Stdout(), "Deleted %s.\n", pluginFullName); err != nil {
		return syserror.Wrap(err)
	}
	return nil
}
