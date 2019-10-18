// Package cli contains helper functionality for applications.
package cli

import (
	"io"
	"time"
)

// RunEnv is the runtime environment for the CLI.
type RunEnv struct {
	// Args are the arguments, not including the application name.
	Args []string
	// Stdin is the stdin.
	Stdin io.Reader
	// Stdout is the stdout.
	Stdout io.Writer
	// Stderr is the stderr.
	Stderr io.Writer
	// Environ is the environment, specified as KEY=VALUE or KEY=.
	Environ []string
}

// ExecEnv is the exec environment for the CLI. This is what is passed to commands.
type ExecEnv struct {
	// Args are the arguments, not including the application name or command.
	Args []string
	// Stdin is the stdin.
	Stdin io.Reader
	// Stdout is the stdout.
	Stdout io.Writer
	// Stderr is the stderr.
	Stderr io.Writer
	// Env is the environment. For values set as KEY=, there will be an empty
	// string associated as the value.
	Env map[string]string
	// Start is the time the command started.
	Start time.Time
}
