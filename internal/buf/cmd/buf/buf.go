// Copyright 2020-2021 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/breaking"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/build"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/config/configinit"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/config/configlsbreakingrules"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/config/configlslintrules"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/convert"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/generate"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/lint"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/lsfiles"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/protoc"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
)

const (
	// Version is the version of buf.
	Version                 = "0.34.0"
	checkDeprecationMessage = `"buf check" sub-commands are now all implemented with the top-level "buf lint" and "buf breaking" commands.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
	checkBreakingDeprecationMessage = `"buf check breaking" has been moved to "buf breaking", use "buf breaking" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
	checkLintDeprecationMessage = `"buf check lint" has been moved to "buf lint", use "buf lint" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
	checkLSBreakingCheckersDeprecationMessage = `"buf check ls-breaking-checkers" has been moved to "buf config ls-breaking-rules", use "buf config ls-breaking-rules" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
	checkLSLintCheckersDeprecationMessage = `"buf check ls-lint-checkers" has been moved to "buf config ls-lint-rules", use "buf config ls-lint-rules" instead.
We recommend migrating, however this command continues to work.
See https://docs.buf.build/faq for more details.`
	imageDeprecationMessage = `"buf image" sub-commands are now all implemented under the top-level "buf build" command, use "buf build" instead.
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
				Use:        "check",
				Short:      "Run linting or breaking change detection.",
				Deprecated: checkDeprecationMessage,
				Hidden:     true,
				SubCommands: []*appcmd.Command{
					lint.NewCommand("lint", builder, moduleResolverReaderProvider, checkLintDeprecationMessage, true),
					breaking.NewCommand("breaking", builder, moduleResolverReaderProvider, checkBreakingDeprecationMessage, true),
					configlslintrules.NewCommand("ls-lint-checkers", builder, checkLSLintCheckersDeprecationMessage, true),
					configlsbreakingrules.NewCommand("ls-breaking-checkers", builder, checkLSBreakingCheckersDeprecationMessage, true),
				},
			},
			lint.NewCommand("lint", builder, moduleResolverReaderProvider, "", false),
			breaking.NewCommand("breaking", builder, moduleResolverReaderProvider, "", false),
			generate.NewCommand("generate", builder, moduleResolverReaderProvider),
			protoc.NewCommand("protoc", builder, moduleResolverReaderProvider),
			lsfiles.NewCommand("ls-files", builder, moduleResolverReaderProvider),
			{
				Use:   "config",
				Short: "Interact with the configuration of Buf.",
				SubCommands: []*appcmd.Command{
					configlslintrules.NewCommand("ls-lint-rules", builder, "", false),
					configlsbreakingrules.NewCommand("ls-breaking-rules", builder, "", false),
				},
			},
			{
				Use:   "beta",
				Short: "Beta commands. Unstable and will likely change.",
				SubCommands: []*appcmd.Command{
					{
						Use:   "config",
						Short: "Interact with the configuration of Buf.",
						SubCommands: []*appcmd.Command{
							configinit.NewCommand("init", builder),
						},
					},
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
