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

package pluginundeprecate

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/spf13/cobra"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/" + bufplugin.PluginsPathName + "/plugin>",
		Short: "Undeprecate a plugin by name.",
		Args:  cobra.ExactArgs(1),
		Run:   builder.NewRunFunc(run, bufcli.NewErrorInterceptor()),
	}
}

func run(
	ctx context.Context,
	container appflag.Container,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	pluginPath := container.Arg(0)
	if pluginPath == "" {
		return appcmd.NewInvalidArgumentError("you must specify a plugin path")
	}
	registryProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return err
	}
	remote, owner, name, err := bufplugin.ParsePluginPath(pluginPath)
	if err != nil {
		return err
	}
	pluginService, err := registryProvider.NewPluginService(ctx, remote)
	if err != nil {
		return err
	}
	if err := pluginService.UndeprecatePlugin(ctx, owner, name); err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
			return bufcli.NewPluginNotFoundError(owner, name)
		}
		return err
	}
	return nil
}
