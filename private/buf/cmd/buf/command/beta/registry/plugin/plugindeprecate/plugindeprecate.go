// Copyright 2020-2022 Buf Technologies, Inc.
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

package plugindeprecate

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
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
	messageFlagName = "message"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/" + bufremoteplugin.PluginsPathName + "/plugin>",
		Short: "Deprecate a plugin by name.",
		Args:  cobra.ExactArgs(1),
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
	Message string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Message,
		messageFlagName,
		"",
		`The message to display with deprecation warnings.`,
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	pluginPath := container.Arg(0)
	if pluginPath == "" {
		return appcmd.NewInvalidArgumentError("you must specify a plugin path")
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	remote, owner, name, err := bufremoteplugin.ParsePluginPath(pluginPath)
	if err != nil {
		return err
	}
	pluginService := connectclient.Make(clientConfig, remote, registryv1alpha1connect.NewPluginServiceClient)
	if _, err := pluginService.DeprecatePlugin(
		ctx,
		connect.NewRequest(&registryv1alpha1.DeprecatePluginRequest{
			Owner:   owner,
			Name:    name,
			Message: flags.Message,
		}),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return bufcli.NewPluginNotFoundError(owner, name)
		}
		return err
	}
	return nil
}
