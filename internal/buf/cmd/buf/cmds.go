// Copyright 2020 Buf Technologies Inc.
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
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/spf13/cobra"
)

func newRootCommand(use string, options ...RootCommandOption) *appcmd.Command {
	builder := newBuilder()
	rootCommand := &appcmd.Command{
		Use: use,
		SubCommands: []*appcmd.Command{
			newImageCmd(builder),
			newCheckCmd(builder),
			newLsFilesCmd(builder),
		},
		BindFlags: builder.BindRoot,
	}
	for _, option := range options {
		option(rootCommand, builder)
	}
	return rootCommand
}

func newImageCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "image",
		Short: "Work with Images and FileDescriptorSets.",
		SubCommands: []*appcmd.Command{
			newImageBuildCmd(builder),
		},
	}
}

func newImageBuildCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "build",
		Short: "Build all files from the input location  and output an Image or FileDescriptorSet.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(imageBuild),
		BindFlags: appcmd.BindMultiple(
			builder.bindImageBuildInput,
			builder.bindImageBuildConfig,
			builder.bindImageBuildOutput,
			builder.bindImageBuildAsFileDescriptorSet,
			builder.bindImageBuildExcludeImports,
			builder.bindImageBuildExcludeSourceInfo,
			builder.bindImageBuildErrorFormat,
			builder.bindExperimentalGitClone,
		),
	}
}

func newCheckCmd(builder *builder) *appcmd.Command {
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

func newCheckLintCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "lint",
		Short: "Check that the input location passes lint checks.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(checkLint),
		BindFlags: appcmd.BindMultiple(
			builder.bindCheckLintInput,
			builder.bindCheckLintConfig,
			builder.bindCheckFiles,
			builder.bindCheckLintErrorFormat,
			builder.bindExperimentalGitClone,
		),
	}
}

func newCheckBreakingCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "breaking",
		Short: "Check that the input location has no breaking changes compared to the against location.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(checkBreaking),
		BindFlags: appcmd.BindMultiple(
			builder.bindCheckBreakingInput,
			builder.bindCheckBreakingConfig,
			builder.bindCheckBreakingAgainstInput,
			builder.bindCheckBreakingAgainstConfig,
			builder.bindCheckBreakingLimitToInputFiles,
			builder.bindCheckBreakingExcludeImports,
			builder.bindCheckFiles,
			builder.bindCheckBreakingErrorFormat,
			builder.bindExperimentalGitClone,
		),
	}
}

func newCheckLsLintCheckersCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "ls-lint-checkers",
		Short: "List lint checkers.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(checkLsLintCheckers),
		BindFlags: appcmd.BindMultiple(
			builder.bindCheckLsCheckersConfig,
			builder.bindCheckLsCheckersAll,
			builder.bindCheckLsCheckersCategories,
			builder.bindCheckLsCheckersFormat,
		),
	}
}

func newCheckLsBreakingCheckersCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "ls-breaking-checkers",
		Short: "List breaking checkers.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(checkLsBreakingCheckers),
		BindFlags: appcmd.BindMultiple(
			builder.bindCheckLsCheckersConfig,
			builder.bindCheckLsCheckersAll,
			builder.bindCheckLsCheckersCategories,
			builder.bindCheckLsCheckersFormat,
		),
	}
}

func newLsFilesCmd(builder *builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "ls-files",
		Short: "List all Protobuf files for the input location.",
		Args:  cobra.NoArgs,
		Run:   builder.newRunFunc(lsFiles),
		BindFlags: appcmd.BindMultiple(
			builder.bindLsFilesInput,
			builder.bindLsFilesConfig,
			builder.bindExperimentalGitClone,
		),
	}
}
