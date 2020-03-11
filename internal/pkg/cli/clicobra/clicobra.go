// Package clicobra contains helper functionality for applications using Cobra.
package clicobra

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"github.com/bufbuild/buf/internal/pkg/cli/internal/output"
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
	// The Short field will be prepended to the Long field with a newline.
	// Must be unset if short is unset.
	Long string
	// Args are the expected arguments.
	Args cobra.PositionalArgs
	// BindFlags allows binding of flags on build.
	BindFlags func(*pflag.FlagSet)
	// Run is the command to run.
	// Required if there are no sub-commands.
	// Must be unset if there are sub-commands.
	Run func(clienv.Env) error
	// SubCommands are the sub-commands. Optional.
	// Must be unset if there is a run function.
	SubCommands []*Command
}

// Main runs the application using the OS runtime and calling os.Exit on the return value of Run.
func Main(rootCommand *Command, version string) {
	env, err := clienv.NewOSEnv()
	if err != nil {
		output.PrintError(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(Run(rootCommand, version, env))
}

// Run runs the application, returning the exit code.
//
// Env will be modified to have dummy values if fields are not set.
func Run(rootCommand *Command, version string, env clienv.Env) int {
	var exitCode int
	if err := runRootCommand(rootCommand, version, env, &exitCode); err != nil {
		output.PrintError(env.Stderr(), err)
		return 1
	}
	return exitCode
}

func runRootCommand(
	rootCommand *Command,
	version string,
	env clienv.Env,
	exitCodeAddr *int,
) error {
	rootCmd, err := commandToCobra(rootCommand, env, exitCodeAddr)
	if err != nil {
		return err
	}

	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Version = version

	// If the root command is not the only command, add hidden bash-completion
	// and zsh-completion commands.
	if len(rootCommand.SubCommands) > 0 {
		rootCmd.AddCommand(&cobra.Command{
			Use:    "bash-completion",
			Args:   cobra.NoArgs,
			Hidden: true,
			Run: func(*cobra.Command, []string) {
				if err := rootCmd.GenBashCompletion(env.Stdout()); err != nil {
					output.PrintError(env.Stderr(), err)
					*exitCodeAddr = 1
				}
			},
		})
		rootCmd.AddCommand(&cobra.Command{
			Use:    "zsh-completion",
			Args:   cobra.NoArgs,
			Hidden: true,
			Run: func(*cobra.Command, []string) {
				if err := rootCmd.GenZshCompletion(env.Stdout()); err != nil {
					output.PrintError(env.Stderr(), err)
					*exitCodeAddr = 1
				}
			},
		})
	}

	rootCmd.SetArgs(env.Args())
	rootCmd.SetOutput(env.Stderr())

	return rootCmd.Execute()
}

func commandToCobra(c *Command, env clienv.Env, exitCodeAddr *int) (*cobra.Command, error) {
	if err := commandValidate(c); err != nil {
		return nil, err
	}
	cmd := &cobra.Command{}
	cmd.Use = c.Use
	cmd.Short = strings.TrimSpace(c.Short)
	if c.Long != "" {
		cmd.Long = fmt.Sprintf("%s\n%s", cmd.Short, strings.TrimSpace(c.Long))
	}
	if c.BindFlags != nil {
		c.BindFlags(cmd.PersistentFlags())
	}
	cmd.Args = c.Args
	if c.Run != nil {
		cmd.Run = func(_ *cobra.Command, args []string) {
			if err := c.Run(env.WithArgs(args)); err != nil {
				output.PrintError(env.Stderr(), err)
				*exitCodeAddr = 1
			}
		}
	}
	for _, subCommand := range c.SubCommands {
		subCmd, err := commandToCobra(subCommand, env, exitCodeAddr)
		if err != nil {
			return nil, err
		}
		cmd.AddCommand(subCmd)
	}
	return cmd, nil
}

func commandValidate(c *Command) error {
	if c.Use == "" {
		return errors.New("must set Command.Use")
	}
	if c.Long != "" && c.Short == "" {
		return errors.New("must set Command.Short if Command.Long is set")
	}
	if c.Run != nil && len(c.SubCommands) > 0 {
		return errors.New("cannot set both Command.Run and Command.SubCommands")
	}
	if c.Run == nil && len(c.SubCommands) == 0 {
		return errors.New("cannot set both Command.Run and Command.SubCommands")
	}
	return nil
}
