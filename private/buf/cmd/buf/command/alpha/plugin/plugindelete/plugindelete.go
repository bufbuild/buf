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

package plugindelete

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
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

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/plugin[:version]>",
		Short: "Delete a plugin from the registry.",
		Args:  cobra.MaximumNArgs(1),
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

func (f *flags) Bind(flagSet *pflag.FlagSet) {}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnAlphaCommand(ctx, container)
	identity, version, _ := strings.Cut(container.Arg(0), ":")
	pluginIdentity, err := bufpluginref.PluginIdentityForString(identity)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
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
		connect.NewRequest(&registryv1alpha1.DeleteCuratedPluginRequest{
			Owner:   pluginIdentity.Owner(),
			Name:    pluginIdentity.Plugin(),
			Version: version,
		}),
	); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fmt.Errorf("the plugin %s does not exist", container.Arg(0))
		}
		return err
	}
	if _, err := fmt.Fprintln(container.Stdout(), "Plugin deleted."); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}
