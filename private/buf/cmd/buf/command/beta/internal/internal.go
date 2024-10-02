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

package internal

import (
	"context"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/spf13/pflag"
	"pluginrpc.com/pluginrpc"
)

// NewPluginCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
	spec *check.Spec,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Run buf as a check plugin.",
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags, spec)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	Protocol bool
	Spec     bool
	Format   string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.Protocol, pluginrpc.ProtocolFlagName, false, "Passed through to plugin.")
	flagSet.BoolVar(&f.Spec, pluginrpc.SpecFlagName, false, "Passed through to plugin.")
	flagSet.StringVar(&f.Format, pluginrpc.FormatFlagName, pluginrpc.FormatBinary.String(), "Passed through to plugin.")
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
	spec *check.Spec,
) error {
	server, err := check.NewServer(spec)
	if err != nil {
		return err
	}
	args := app.Args(container)
	if flags.Protocol {
		args = append(args, "--"+pluginrpc.ProtocolFlagName)
	}
	if flags.Spec {
		args = append(args, "--"+pluginrpc.SpecFlagName)
	}
	if flags.Format != "" {
		args = append(args, "--"+pluginrpc.FormatFlagName+"="+flags.Format)
	}
	return server.Serve(
		ctx,
		pluginrpc.Env{
			Args:   args,
			Stdin:  container.Stdin(),
			Stdout: container.Stdout(),
			Stderr: container.Stderr(),
		},
	)
}
