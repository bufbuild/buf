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

package templatecreate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	configFlagName     = "config"
	visibilityFlagName = "visibility"
	formatFlagName     = "format"

	publicVisibility  = "public"
	privateVisibility = "private"
)

var allVisibiltyStrings = []string{
	publicVisibility,
	privateVisibility,
}

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <buf.build/owner/" + bufremoteplugin.TemplatesPathName + "/template>",
		Short: "Create a new Buf template.",
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
	Config     string
	Visibility string
	Format     string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The template file or data to use for configuration. Must be in either YAML or JSON format.`,
	)
	_ = cobra.MarkFlagRequired(flagSet, configFlagName)
	flagSet.StringVar(
		&f.Visibility,
		visibilityFlagName,
		"",
		fmt.Sprintf(`The template's visibility setting. Must be one of %s.`, stringutil.SliceToString(allVisibiltyStrings)),
	)
	_ = cobra.MarkFlagRequired(flagSet, visibilityFlagName)
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf(`The output format to use. Must be one of %s`, bufprint.AllFormatsString),
	)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	bufcli.WarnBetaCommand(ctx, container)
	templatePath := container.Arg(0)
	visibility, err := visibilityFlagToVisibility(flags.Visibility)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	format, err := bufprint.ParseFormat(flags.Format)
	if err != nil {
		return appcmd.NewInvalidArgumentError(err.Error())
	}
	templateConfig, err := bufremoteplugin.ParseTemplateConfig(flags.Config)
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	remote, owner, name, err := bufremoteplugin.ParseTemplatePath(templatePath)
	if err != nil {
		return err
	}
	pluginService := connectclient.Make(clientConfig, remote, registryv1alpha1connect.NewPluginServiceClient)
	resp, err := pluginService.CreateTemplate(
		ctx,
		connect.NewRequest(&registryv1alpha1.CreateTemplateRequest{
			Owner:         owner,
			Name:          name,
			Visibility:    visibility,
			PluginConfigs: bufremoteplugin.TemplateConfigToProtoPluginConfigs(templateConfig),
		}),
	)
	if err != nil {
		return err
	}
	return bufprint.NewTemplatePrinter(container.Stdout()).PrintTemplate(ctx, format, resp.Msg.Template)
}

// visibilityFlagToVisibility parses the given string as a registryv1alpha1.PluginVisibility.
func visibilityFlagToVisibility(visibility string) (registryv1alpha1.PluginVisibility, error) {
	switch visibility {
	case publicVisibility:
		return registryv1alpha1.PluginVisibility_PLUGIN_VISIBILITY_PUBLIC, nil
	case privateVisibility:
		return registryv1alpha1.PluginVisibility_PLUGIN_VISIBILITY_PRIVATE, nil
	default:
		return 0, fmt.Errorf("invalid visibility: %s, expected one of %s", visibility, stringutil.SliceToString(allVisibiltyStrings))
	}
}
