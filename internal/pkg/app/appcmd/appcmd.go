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

// Package appcmd contains helper functionality for applications using commands.
package appcmd

import (
	"context"
	"errors"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Command is a command.
type Command struct {
	// Use is the one-line usage message.
	// Required.
	Use string
	// Short is the short message shown in the 'help' output.
	// Required if Long is set.
	Short string
	// Long is the long message shown in the 'help <this-command>' output.
	// The Short field will be prepended to the Long field with two newlines.
	// Must be unset if short is unset.
	Long string
	// Args are the expected arguments.
	//
	// TODO: make specific types for appcmd to limit what can be done.
	Args cobra.PositionalArgs
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
	// Version is the version.
	Version string
	// SubCommands are the sub-commands. Optional.
	// Must be unset if there is a run function.
	SubCommands []*Command
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

	// If the root command is not the only command, add hidden bash-completion
	// and zsh-completion commands.
	if len(command.SubCommands) > 0 {
		cobraCommand.AddCommand(&cobra.Command{
			Use:    "bash-completion",
			Args:   cobra.NoArgs,
			Hidden: true,
			Run: func(*cobra.Command, []string) {
				runErr = cobraCommand.GenBashCompletion(container.Stdout())
			},
		})
		cobraCommand.AddCommand(&cobra.Command{
			Use:    "zsh-completion",
			Args:   cobra.NoArgs,
			Hidden: true,
			Run: func(*cobra.Command, []string) {
				runErr = cobraCommand.GenZshCompletion(container.Stdout())
			},
		})
	}

	cobraCommand.SetArgs(app.Args(container)[1:])
	cobraCommand.SetOut(container.Stderr())
	cobraCommand.SetErr(container.Stderr())

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
	cobraCommand := &cobra.Command{
		Use:   command.Use,
		Args:  command.Args,
		Short: strings.TrimSpace(command.Short),
	}
	if command.Long != "" {
		cobraCommand.Long = cobraCommand.Short + "\n\n" + strings.TrimSpace(command.Long)
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
			*runErrAddr = command.Run(ctx, app.NewContainerForArgs(container, args...))
		}
	}
	if command.Version != "" {
		cobraCommand.SetVersionTemplate("{{.Version}}\n")
		cobraCommand.Version = command.Version
	}
	for _, subCommand := range command.SubCommands {
		subCobraCommand, err := commandToCobra(ctx, container, subCommand, runErrAddr)
		if err != nil {
			return nil, err
		}
		cobraCommand.AddCommand(subCobraCommand)
	}
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
		return errors.New("cannot set both Command.Run and Command.SubCommands")
	}
	return nil
}

func normalizeFunc(f func(*pflag.FlagSet, string) string) func(*pflag.FlagSet, string) pflag.NormalizedName {
	return func(flagSet *pflag.FlagSet, name string) pflag.NormalizedName {
		return pflag.NormalizedName(f(flagSet, name))
	}
}
