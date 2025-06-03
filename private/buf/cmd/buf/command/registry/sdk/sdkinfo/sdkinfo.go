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

package sdkinfo

import (
	"context"
	"errors"
	"fmt"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/buf/bufprint"
	"github.com/spf13/pflag"
)

const (
	formatFlagName  = "format"
	moduleFlagName  = "module"
	pluginFlagName  = "plugin"
	versionFlagName = "version"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	// TODO: should we use an example other than eliza?
	return &appcmd.Command{
		Use:   name + " --module=<remote/owner/repository[:ref]> --plugin=<remote/owner/plugin[:version]>",
		Short: "Get SDK information for the given module, plugin, and optionally version.",
		// TODO: complete examples
		Long: `This command returns the version information for a Generated SDK based on the specified information.
In order to resolve the SDK information, a module and plugin must be specified.

Examples:

To get the SDK information for of the latest commit of the eliza module and the latest version of the Go plugin:
    $ buf registry sdk info --module=buf.build/connectrpc/eliza --plugin=buf.build/protocolbuffers/go
    <TODO: return value>

To get the SDK information for a specific commit of the eliza module and a specific version of the Go plugin:
    $ buf registry sdk info --module=buf.build/connectrpc/eliza:<TODO: commit> --plugin=buf.build/protocolbuffers/go:v1.32.0
    <TODO: return value>

If you have a SDK version and you want to know the module commit and plugin version information for the SDK:
    $ buf registry sdk --module=buf.build/connectrpc/eliza --plugin=buf.build/protocolbuffers/go --version=<TODO: version string>
    <TODO: return value>

If a module reference and/or plugin version are specified along with the SDK version flag, --version, then the SDK version
will be validated against the provided module reference and/or plugin version. If there is a mismatch, this command will error.`,
		Args: appcmd.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Format  string
	Module  string
	Plugin  string
	Version string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Format,
		formatFlagName,
		bufprint.FormatText.String(),
		fmt.Sprintf("The output format to use. Must be one of %s", bufprint.AllFormatsString),
	)
	flagSet.StringVar(&f.Module, moduleFlagName, "", "The module reference for the SDK.")
	flagSet.StringVar(&f.Plugin, pluginFlagName, "", "The plugin reference for the SDK.")
	flagSet.StringVar(&f.Version, versionFlagName, "", "The version of the SDK.")
	_ = appcmd.MarkFlagRequired(flagSet, moduleFlagName)
	_ = appcmd.MarkFlagRequired(flagSet, pluginFlagName)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	return errors.New("unimplemented")
}
