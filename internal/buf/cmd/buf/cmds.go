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
	"time"

	"github.com/bufbuild/buf/internal/buf/cmd/buf/internal/lsfiles"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/internal/protoc"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/spf13/cobra"
)

func newRootCommand(use string, options ...RootCommandOption) *appcmd.Command {
	builder := appflag.NewBuilder(appflag.BuilderWithTimeout(120 * time.Second))
	rootCommand := &appcmd.Command{
		Use: use,
		SubCommands: []*appcmd.Command{
			newImageCmd(builder),
			newCheckCmd(builder),
			lsfiles.NewCommand("ls-files", builder),
			protoc.NewCommand("protoc", builder),
			newExperimentalCmd(builder),
		},
		BindPersistentFlags: builder.BindRoot,
		Version:             Version,
	}
	for _, option := range options {
		option(rootCommand, builder)
	}
	return rootCommand
}

func newExperimentalCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "experimental",
		Short: "Experimental commands. Unstable and will likely change.",
		SubCommands: []*appcmd.Command{
			newExperimentalImageCmd(builder),
		},
	}
}

func newImageCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "image",
		Short: "Work with Images and FileDescriptorSets.",
		SubCommands: []*appcmd.Command{
			newImageBuildCmd(builder),
		},
	}
}

func newExperimentalImageCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "image",
		Short: "Work with Images and FileDescriptorSets.",
		SubCommands: []*appcmd.Command{
			newImageConvertCmd(builder),
		},
	}
}

func newImageBuildCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "build",
		Short: "Build all files from the input location and output an Image or FileDescriptorSet.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, imageBuild),
		BindFlags: appcmd.BindMultiple(
			flags.bindImageBuildInput,
			flags.bindImageBuildConfig,
			flags.bindImageBuildFiles,
			flags.bindImageBuildOutput,
			flags.bindImageBuildAsFileDescriptorSet,
			flags.bindImageBuildExcludeImports,
			flags.bindImageBuildExcludeSourceInfo,
			flags.bindImageBuildErrorFormat,
			flags.bindExperimentalGitClone,
		),
	}
}

func newImageConvertCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "convert",
		Short: "Convert the input Image to an output Image with the specified format and filters.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, imageConvert),
		BindFlags: appcmd.BindMultiple(
			flags.bindImageConvertInput,
			flags.bindImageConvertFiles,
			flags.bindImageConvertOutput,
			flags.bindImageConvertAsFileDescriptorSet,
			flags.bindImageConvertExcludeImports,
			flags.bindImageConvertExcludeSourceInfo,
		),
	}
}

func newCheckCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "check",
		Short: "Run lint or breaking change checks.",
		SubCommands: []*appcmd.Command{
			newCheckLintCmd(builder),
			newCheckBreakingCmd(builder),
			newCheckLsLintCheckersCmd(builder),
			newCheckLsBreakingCheckersCmd(builder),
		},
	}
}

func newCheckLintCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "lint",
		Short: "Check that the input location passes lint checks.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, checkLint),
		BindFlags: appcmd.BindMultiple(
			flags.bindCheckLintInput,
			flags.bindCheckLintConfig,
			flags.bindCheckFiles,
			flags.bindCheckLintErrorFormat,
			flags.bindExperimentalGitClone,
		),
	}
}

func newCheckBreakingCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "breaking",
		Short: "Check that the input location has no breaking changes compared to the against location.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, checkBreaking),
		BindFlags: appcmd.BindMultiple(
			flags.bindCheckBreakingInput,
			flags.bindCheckBreakingConfig,
			flags.bindCheckBreakingAgainstInput,
			flags.bindCheckBreakingAgainstConfig,
			flags.bindCheckBreakingLimitToInputFiles,
			flags.bindCheckBreakingExcludeImports,
			flags.bindCheckFiles,
			flags.bindCheckBreakingErrorFormat,
			flags.bindExperimentalGitClone,
		),
	}
}

func newCheckLsLintCheckersCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "ls-lint-checkers",
		Short: "List lint checkers.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, checkLsLintCheckers),
		BindFlags: appcmd.BindMultiple(
			flags.bindCheckLsCheckersConfig,
			flags.bindCheckLsCheckersAll,
			flags.bindCheckLsCheckersCategories,
			flags.bindCheckLsCheckersFormat,
		),
	}
}

func newCheckLsBreakingCheckersCmd(builder appflag.Builder) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   "ls-breaking-checkers",
		Short: "List breaking checkers.",
		Args:  cobra.NoArgs,
		Run:   newRunFunc(builder, flags, checkLsBreakingCheckers),
		BindFlags: appcmd.BindMultiple(
			flags.bindCheckLsCheckersConfig,
			flags.bindCheckLsCheckersAll,
			flags.bindCheckLsCheckersCategories,
			flags.bindCheckLsCheckersFormat,
		),
	}
}
