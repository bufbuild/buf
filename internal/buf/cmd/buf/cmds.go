package buf

import (
	"github.com/bufbuild/buf/internal/pkg/cli/clicobra"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newRootCommand(use string, options ...RootCommandOption) *clicobra.Command {
	flags := newFlags()
	rootCommand := &clicobra.Command{
		Use: use,
		SubCommands: []*clicobra.Command{
			newImageCmd(flags),
			newCheckCmd(flags),
			newLsFilesCmd(flags),
		},
		BindFlags: flags.bindRootCommandFlags,
	}
	for _, option := range options {
		option(rootCommand, flags)
	}
	return rootCommand
}

func newImageCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "image",
		Short: "Work with Images and FileDescriptorSets.",
		SubCommands: []*clicobra.Command{
			newImageBuildCmd(flags),
		},
	}
}

func newImageBuildCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "build",
		Short: "Build all files from the input location  and output an Image or FileDescriptorSet.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(imageBuild),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindImageBuildInput(flagSet)
			flags.bindImageBuildConfig(flagSet)
			flags.bindImageBuildOutput(flagSet)
			flags.bindImageBuildAsFileDescriptorSet(flagSet)
			flags.bindImageBuildExcludeImports(flagSet)
			flags.bindImageBuildExcludeSourceInfo(flagSet)
			flags.bindImageBuildErrorFormat(flagSet)
		},
	}
}

func newCheckCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "check",
		Short: "Run lint or breaking change checks.",
		SubCommands: []*clicobra.Command{
			newCheckLintCmd(flags),
			newCheckBreakingCmd(flags),
			newCheckLsLintCheckersCmd(flags),
			newCheckLsBreakingCheckersCmd(flags),
		},
	}
}

func newCheckLintCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "lint",
		Short: "Check that the input location passes lint checks.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(checkLint),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindCheckLintInput(flagSet)
			flags.bindCheckLintConfig(flagSet)
			flags.bindCheckFiles(flagSet)
			flags.bindCheckLintErrorFormat(flagSet)
		},
	}
}

func newCheckBreakingCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "breaking",
		Short: "Check that the input location has no breaking changes compared to the against location.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(checkBreaking),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindCheckBreakingInput(flagSet)
			flags.bindCheckBreakingConfig(flagSet)
			flags.bindCheckBreakingAgainstInput(flagSet)
			flags.bindCheckBreakingAgainstConfig(flagSet)
			flags.bindCheckBreakingLimitToInputFiles(flagSet)
			flags.bindCheckBreakingExcludeImports(flagSet)
			flags.bindCheckFiles(flagSet)
			flags.bindCheckBreakingErrorFormat(flagSet)
		},
	}
}

func newCheckLsLintCheckersCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "ls-lint-checkers",
		Short: "List lint checkers.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(checkLsLintCheckers),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindCheckLsCheckersConfig(flagSet)
			flags.bindCheckLsCheckersAll(flagSet)
			flags.bindCheckLsCheckersCategories(flagSet)
			flags.bindCheckLsCheckersFormat(flagSet)
		},
	}
}

func newCheckLsBreakingCheckersCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "ls-breaking-checkers",
		Short: "List breaking checkers.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(checkLsBreakingCheckers),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindCheckLsCheckersConfig(flagSet)
			flags.bindCheckLsCheckersAll(flagSet)
			flags.bindCheckLsCheckersCategories(flagSet)
			flags.bindCheckLsCheckersFormat(flagSet)
		},
	}
}

func newLsFilesCmd(flags *Flags) *clicobra.Command {
	return &clicobra.Command{
		Use:   "ls-files",
		Short: "List all Protobuf files for the input location.",
		Args:  cobra.NoArgs,
		Run:   flags.newRunFunc(lsFiles),
		BindFlags: func(flagSet *pflag.FlagSet) {
			flags.bindLsFilesInput(flagSet)
			flags.bindLsFilesConfig(flagSet)
		},
	}
}
