// Copyright 2020 Buf Technologies, Inc.
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

package buf

import (
	"context"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/build"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/breaking"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/lint"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/lsbreakingcheckers"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/lslintcheckers"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/convert"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/generate"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/lsfiles"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/protoc"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
)

const (
	// Version is the version of buf.
	Version                 = "0.29.0-dev"
	imageDeprecationMessage = `"image" sub-commands are now all implemented under the top-level "buf build" command, use "buf build" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
)

// Main is the main.
func Main(name string, options ...MainOption) {
	mainOptions := &mainOptions{
		moduleResolverReaderProvider: bufcli.NopModuleResolverReaderProvider{},
	}
	for _, option := range options {
		option(mainOptions)
	}
	appcmd.Main(
		context.Background(),
		NewRootCommand(
			name,
			mainOptions.rootCommandModifier,
			mainOptions.moduleResolverReaderProvider,
		),
	)
}

// MainOption is an option for command construction.
type MainOption func(*mainOptions)

// WithModuleResolverAndReaderProvider returns a new MainOption that uses the given ModuleResolverReaderProvider.
func WithModuleResolverAndReaderProvider(moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider) MainOption {
	return func(options *mainOptions) {
		options.moduleResolverReaderProvider = moduleResolverReaderProvider
	}
}

// WithRootCommandModifier returns a new MainOption that modifies the root Command.
func WithRootCommandModifier(rootCommandModifier func(*appcmd.Command, appflag.Builder, bufcli.ModuleResolverReaderProvider)) MainOption {
	return func(mainOptions *mainOptions) {
		mainOptions.rootCommandModifier = rootCommandModifier
	}
}

// NewRootCommand returns a new root command.
//
// This is public for use in testing.
func NewRootCommand(
	name string,
	rootCommandModifier func(*appcmd.Command, appflag.Builder, bufcli.ModuleResolverReaderProvider),
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	builder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTimeout(120*time.Second),
		appflag.BuilderWithTracing(),
	)
	rootCommand := &appcmd.Command{
		Use: name,
		SubCommands: []*appcmd.Command{
			build.NewCommand("build", builder, moduleResolverReaderProvider, "", false),
			{
				Use:        "image",
				Short:      "Work with Images and FileDescriptorSets.",
				Deprecated: imageDeprecationMessage,
				Hidden:     true,
				SubCommands: []*appcmd.Command{
					build.NewCommand(
						"build",
						builder, moduleResolverReaderProvider,
						imageDeprecationMessage,
						true,
					),
				},
			},
			{
				Use:   "check",
				Short: "Run lint or breaking change checks.",
				SubCommands: []*appcmd.Command{
					lint.NewCommand("lint", builder, moduleResolverReaderProvider),
					breaking.NewCommand("breaking", builder, moduleResolverReaderProvider),
					lslintcheckers.NewCommand("ls-lint-checkers", builder),
					lsbreakingcheckers.NewCommand("ls-breaking-checkers", builder),
				},
			},
			generate.NewCommand("generate", builder, moduleResolverReaderProvider),
			protoc.NewCommand("protoc", builder, moduleResolverReaderProvider),
			lsfiles.NewCommand("ls-files", builder, moduleResolverReaderProvider),
			{
				Use:    "beta",
				Short:  "Beta commands. Unstable and will likely change.",
				Hidden: true,
				SubCommands: []*appcmd.Command{
					{
						Use:        "image",
						Short:      "Work with Images and FileDescriptorSets.",
						Deprecated: imageDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							convert.NewCommand(
								"convert",
								builder,
								imageDeprecationMessage,
								true,
							),
						},
					},
				},
			},
			{
				Use:   "experimental",
				Short: "Experimental commands. Unstable and will likely change.",
				Deprecated: `use "beta" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`,
				Hidden: true,
				SubCommands: []*appcmd.Command{
					{
						Use:        "image",
						Short:      "Work with Images and FileDescriptorSets.",
						Deprecated: imageDeprecationMessage,
						Hidden:     true,
						SubCommands: []*appcmd.Command{
							convert.NewCommand(
								"convert",
								builder,
								imageDeprecationMessage,
								true,
							),
						},
					},
				},
			},
		},
		BindPersistentFlags: builder.BindRoot,
		Version:             Version,
	}
	if rootCommandModifier != nil {
		rootCommandModifier(rootCommand, builder, moduleResolverReaderProvider)
	}
	return rootCommand
}

type mainOptions struct {
	rootCommandModifier          func(*appcmd.Command, appflag.Builder, bufcli.ModuleResolverReaderProvider)
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider
}
