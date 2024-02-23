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

// Package appcmd contains helper functionality for applications using commands.
//
// This package wraps cobra. Imports should not import cobra directly - raise an issue
// internally if there is missing cobra functionality you need. It is acceptable to
// import pflag, however.
package appcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

// Command is a command.
type Command struct {
	// Use is the one-line usage message.
	// Required.
	Use string
	// Aliases are aliases that can be used instead of the first word in Use.
	Aliases []string
	// Short is the short message shown in the 'help' output.
	// Required if Long is set.
	Short string
	// Long is the long message shown in the 'help <this-command>' output.
	// The Short field will be prepended to the Long field with two newlines.
	// Must be unset if short is unset.
	Long string
	// Args are the expected arguments.
	Args PositionalArgs
	// Deprecated says to print this deprecation string.
	Deprecated string
	// Hidden says to hide this command.
	Hidden bool
	// BindFlags allows binding of flags on build.
	BindFlags func(*pflag.FlagSet)
	// BindPersistentFlags allows binding of flags on build.
	BindPersistentFlags func(*pflag.FlagSet)
	// NormalizeFlag allows for normalization of flag names.
	NormalizeFlag func(*pflag.FlagSet, string) string
	// NormalizePersistentFlag allows for normalization of flag names.
	NormalizePersistentFlag func(*pflag.FlagSet, string) string
	// Run is the command to run.
	// Required if there are no sub-commands.
	// Must be unset if there are sub-commands.
	Run func(context.Context, app.Container) error
	// SubCommands are the sub-commands. Optional.
	// Must be unset if there is a run function.
	SubCommands []*Command
	// Version the version of the command.
	//
	// If this is specified, a flag --version will be added to the command
	// that precedes all other functionality, and which prints the version
	// to stdout.
	Version string
}

// NewInvalidArgumentError creates a new invalidArgumentError, indicating that
// the error was caused by argument validation. This causes us to print the usage
// help text for the command that it is returned from.
func NewInvalidArgumentError(message string) error {
	return newInvalidArgumentError(errors.New(message))
}

// NewInvalidArgumentErrorf creates a new InvalidArgumentError, indicating that
// the error was caused by argument validation. This causes us to print the usage
// help text for the command that it is returned from.
func NewInvalidArgumentErrorf(format string, args ...any) error {
	return newInvalidArgumentError(fmt.Errorf(format, args...))
}

// Main runs the application using the OS container and calling os.Exit on the return value of Run.
func Main(ctx context.Context, command *Command) {
	app.Main(ctx, newRunFunc(command))
}

// Run runs the application using the container.
func Run(ctx context.Context, container app.Container, command *Command) error {
	return app.Run(ctx, container, newRunFunc(command))
}

// BindMultiple is a convenience function for binding multiple flag functions.
func BindMultiple(bindFuncs ...func(*pflag.FlagSet)) func(*pflag.FlagSet) {
	return func(flagSet *pflag.FlagSet) {
		for _, bindFunc := range bindFuncs {
			bindFunc(flagSet)
		}
	}
}

// MarkFlagRequired matches cobra.MarkFlagRequired so that importers of appcmd do
// not need to reference cobra (and shouldn't).
func MarkFlagRequired(flagSet *pflag.FlagSet, flagName string) error {
	return cobra.MarkFlagRequired(flagSet, flagName)
}

// *** PRIVATE ***

func newRunFunc(command *Command) func(context.Context, app.Container) error {
	return func(ctx context.Context, container app.Container) error {
		return run(ctx, container, command)
	}
}

func run(
	ctx context.Context,
	container app.Container,
	command *Command,
) error {
	var runErr error

	cobraCommand, err := commandToCobra(ctx, container, command, &runErr)
	if err != nil {
		return err
	}

	// Cobra 1.2.0 introduced default completion commands under
	// "<binary> completion <bash/zsh/fish/powershell>"". Since we have
	// our own completion commands, disable the generation of the default
	// commands.
	cobraCommand.CompletionOptions.DisableDefaultCmd = true

	// If the root command is not the only command, add a hidden manpages command
	// and a visible completion command.
	if len(command.SubCommands) > 0 {
		shellCobraCommand, err := commandToCobra(
			ctx,
			container,
			&Command{
				Use:   "completion",
				Short: "Generate auto-completion scripts for commonly used shells",
				SubCommands: []*Command{
					{
						Use:   "bash",
						Short: "Generate auto-completion scripts for bash",
						Args:  NoArgs,
						Run: func(ctx context.Context, container app.Container) error {
							return cobraCommand.GenBashCompletion(container.Stdout())
						},
					},
					{
						Use:   "fish",
						Short: "Generate auto-completion scripts for fish",
						Args:  NoArgs,
						Run: func(ctx context.Context, container app.Container) error {
							return cobraCommand.GenFishCompletion(container.Stdout(), true)
						},
					},
					{
						Use:   "powershell",
						Short: "Generate auto-completion scripts for powershell",
						Args:  NoArgs,
						Run: func(ctx context.Context, container app.Container) error {
							return cobraCommand.GenPowerShellCompletion(container.Stdout())
						},
					},
					{
						Use:   "zsh",
						Short: "Generate auto-completion scripts for zsh",
						Args:  NoArgs,
						Run: func(ctx context.Context, container app.Container) error {
							return cobraCommand.GenZshCompletion(container.Stdout())
						},
					},
				},
			},
			&runErr,
		)
		if err != nil {
			return err
		}
		cobraCommand.AddCommand(shellCobraCommand)
		manpagesCobraCommand, err := commandToCobra(
			ctx,
			container,
			&Command{
				Use:    "manpages",
				Args:   ExactArgs(1),
				Hidden: true,
				Run: func(ctx context.Context, container app.Container) error {
					return doc.GenManTree(
						cobraCommand,
						&doc.GenManHeader{
							Title:   "Buf",
							Section: "1",
						},
						container.Arg(0),
					)
				},
			},
			&runErr,
		)
		if err != nil {
			return err
		}
		cobraCommand.AddCommand(manpagesCobraCommand)
		webpagesCobraCommand, err := commandToCobra(
			ctx,
			container,
			newWebpagesCommand(cobraCommand),
			&runErr,
		)
		if err != nil {
			return err
		}
		cobraCommand.AddCommand(webpagesCobraCommand)
	}

	cobraCommand.SetOut(container.Stderr())
	args := app.Args(container)[1:]
	// cobra will implicitly create __complete and __completeNoDesc subcommands
	// https://github.com/spf13/cobra/blob/4590150168e93f4b017c6e33469e26590ba839df/completions.go#L14-L17
	// at the very last possible point, to enable them to be overridden. Unfortunately
	// the creation of the subcommands uses hidden helper methods (unlike the automatic help command support).
	// See https://github.com/spf13/cobra/blob/4590150168e93f4b017c6e33469e26590ba839df/completions.go#L134.
	//
	// Additionally, the automatically generated commands inherit the output of the root command,
	// which we are ensuring is always stderr.
	// https://github.com/spf13/cobra/blob/4590150168e93f4b017c6e33469e26590ba839df/completions.go#L175
	//
	// bash completion has much more detailed code generation and doesn't rely on the __completion command
	// in most cases, the zsh and fish completion implementation however exclusively rely on these commands.
	// Those completion implementations send stderr to /dev/null
	// https://github.com/spf13/cobra/blob/4590150168e93f4b017c6e33469e26590ba839df/zsh_completions.go#L135
	// and the automatically generated __complete command sends extra data to /dev/null so we cannot
	// work around this by minimally changing the code generation commands, we would have to rewrite the
	// __completion command which is much more complicated.
	//
	// Instead of all that, we can peek at the positionals and if the sub command starts with __complete
	// we sets its output to stdout. This would mean that we cannot add a "real" sub-command that starts with
	// __complete _and_ has its output set to stderr. This shouldn't ever be a problem.
	//
	// SetOut sets the output location for usage, help, and version messages by default.
	if len(args) > 0 && strings.HasPrefix(args[0], "__complete") {
		cobraCommand.SetOut(container.Stdout())
	}
	cobraCommand.SetArgs(args)
	// SetErr sets the output location for error messages.
	cobraCommand.SetErr(container.Stderr())
	cobraCommand.SetIn(container.Stdin())

	if err := cobraCommand.Execute(); err != nil {
		return err
	}
	return runErr
}

func commandToCobra(
	ctx context.Context,
	container app.Container,
	command *Command,
	runErrAddr *error,
) (*cobra.Command, error) {
	if err := commandValidate(command); err != nil {
		return nil, err
	}
	var cobraPositionalArgs cobra.PositionalArgs
	if command.Args != nil {
		cobraPositionalArgs = command.Args.cobra()
	}
	cobraCommand := &cobra.Command{
		Use:        command.Use,
		Aliases:    command.Aliases,
		Args:       cobraPositionalArgs,
		Deprecated: command.Deprecated,
		Hidden:     command.Hidden,
		Short:      strings.TrimSpace(command.Short),
	}
	cobraCommand.SetHelpTemplate(`{{.Short}}

{{with .Long}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
	cobraCommand.SetHelpFunc(
		func(c *cobra.Command, _ []string) {
			if err := tmpl(container.Stdout(), c.HelpTemplate(), c); err != nil {
				c.PrintErrln(err)
			}
		},
	)
	if command.Long != "" {
		cobraCommand.Long = strings.TrimSpace(command.Long)
	}
	if command.BindFlags != nil {
		command.BindFlags(cobraCommand.Flags())
	}
	if command.BindPersistentFlags != nil {
		command.BindPersistentFlags(cobraCommand.PersistentFlags())
	}
	if command.NormalizeFlag != nil {
		cobraCommand.Flags().SetNormalizeFunc(normalizeFunc(command.NormalizeFlag))
	}
	if command.NormalizePersistentFlag != nil {
		cobraCommand.PersistentFlags().SetNormalizeFunc(normalizeFunc(command.NormalizePersistentFlag))
	}
	if command.Run != nil {
		cobraCommand.Run = func(_ *cobra.Command, args []string) {
			runErr := command.Run(ctx, app.NewContainerForArgs(container, args...))
			if asErr := (&invalidArgumentError{}); errors.As(runErr, &asErr) {
				// Print usage for failing command if an args error is returned.
				// This has to be done at this level since the usage must relate
				// to the command executed.
				printUsage(container, cobraCommand.UsageString())
			}
			*runErrAddr = runErr
		}
	}
	if len(command.SubCommands) > 0 {
		// command.Run will not be set per validation
		cobraCommand.Run = func(cmd *cobra.Command, args []string) {
			printUsage(container, cobraCommand.UsageString())
			if len(args) == 0 {
				*runErrAddr = errors.New("Sub-command required.")
			} else {
				*runErrAddr = fmt.Errorf("Unknown sub-command: %s", strings.Join(args, " "))
			}
		}
		for _, subCommand := range command.SubCommands {
			subCobraCommand, err := commandToCobra(ctx, container, subCommand, runErrAddr)
			if err != nil {
				return nil, err
			}
			cobraCommand.AddCommand(subCobraCommand)
		}
		addHelpTreeFlag(container, cobraCommand, runErrAddr)
	}
	if command.Version != "" {
		doVersion := false
		oldRun := cobraCommand.Run
		cobraCommand.Flags().BoolVar(
			&doVersion,
			"version",
			false,
			"Print the version",
		)
		cobraCommand.Run = func(cmd *cobra.Command, args []string) {
			if doVersion {
				_, err := container.Stdout().Write([]byte(command.Version + "\n"))
				*runErrAddr = err
				return
			}
			oldRun(cmd, args)
		}
	}
	// appcommand prints errors, disable to prevent duplicates.
	cobraCommand.SilenceErrors = true
	return cobraCommand, nil
}

func commandValidate(command *Command) error {
	if command.Use == "" {
		return errors.New("must set Command.Use")
	}
	if command.Long != "" && command.Short == "" {
		return errors.New("must set Command.Short if Command.Long is set")
	}
	if command.Run != nil && len(command.SubCommands) > 0 {
		return errors.New("cannot set both Command.Run and Command.SubCommands")
	}
	if command.Run == nil && len(command.SubCommands) == 0 {
		return errors.New("must set one of Command.Run and Command.SubCommands")
	}
	return nil
}

func normalizeFunc(f func(*pflag.FlagSet, string) string) func(*pflag.FlagSet, string) pflag.NormalizedName {
	return func(flagSet *pflag.FlagSet, name string) pflag.NormalizedName {
		return pflag.NormalizedName(f(flagSet, name))
	}
}

func printUsage(container app.StderrContainer, usage string) {
	_, _ = container.Stderr().Write([]byte(usage + "\n"))
}

func addHelpTreeFlag(
	container app.Container,
	cmd *cobra.Command,
	runErrAddr *error,
) {
	helpTree := false
	oldRun := cmd.Run
	cmd.Flags().BoolVar(
		&helpTree,
		"help-tree",
		false,
		"Print the entire sub-command tree",
	)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if helpTree {
			_, err := container.Stdout().Write([]byte(helpTreeString(cmd)))
			*runErrAddr = err
			return
		}
		oldRun(cmd, args)
	}
}

func helpTreeString(cmd *cobra.Command) string {
	var builder strings.Builder
	helpTreeStringRec(cmd, &builder, maxPaddingRec(cmd, 0), 0)
	return builder.String()
}

func helpTreeStringRec(cmd *cobra.Command, builder *strings.Builder, maxPadding int, curIndentCount int) {
	if cmd.Hidden {
		return
	}
	if name := cmd.Name(); name != "" {
		_, _ = builder.WriteString(strings.Repeat(" ", curIndentCount*2))
		_, _ = builder.WriteString(name)
		_, _ = builder.WriteString(strings.Repeat(" ", maxPadding-(len(cmd.Name())+(curIndentCount*2))))
		_, _ = builder.WriteString("  ")
		_, _ = builder.WriteString(cmd.Short)
		_, _ = builder.WriteString("\n")
	}
	for _, child := range cmd.Commands() {
		helpTreeStringRec(child, builder, maxPadding, curIndentCount+1)
	}
}

func maxPaddingRec(cmd *cobra.Command, curIndentCount int) int {
	maxPadding := (curIndentCount * 2) + len(cmd.Name())
	for _, child := range cmd.Commands() {
		if !child.Hidden {
			maxPadding = maxInt(maxPadding, maxPaddingRec(child, curIndentCount+1))
		}
	}
	return maxPadding
}

func maxInt(i int, j int) int {
	if i > j {
		return i
	}
	return j
}
