// Package clienv contains types to work with the host environment.
package clienv

import (
	"errors"
	"io"
	"os"
	"strings"
)

// Env is an execution environment for the CLI. This is what is passed to commands.
type Env interface {
	// Args are the arguments, not including the application name or command.
	//
	// Do not modify.
	Args() []string
	// Stdin is the stdin.
	//
	// If no value was passed when the Env was created, this will return io.EOF on any call.
	Stdin() io.Reader
	// Stdout is the stdout.
	//
	// If no value was passed when the Env was created, this will return io.EOF on any call.
	Stdout() io.Writer
	// Stderr is the stderr.
	//
	// If no value was passed when the Env was created, this will return io.EOF on any call.
	Stderr() io.Writer
	// Getenv is the equivalent of os.Getenv.
	Getenv(key string) string
	// WithArgs returns a copy of Env with the replacement args.
	WithArgs(args []string) Env
}

// NewOSEnv returns a new OS environment.
func NewOSEnv() (Env, error) {
	variables, err := environToVariables(os.Environ())
	if err != nil {
		return nil, err
	}
	return NewEnv(
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		os.Stderr,
		variables,
	), nil
}

// NewEnv creates a new environment.
//
// Default values will be set if any values are nil.
func NewEnv(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	variables map[string]string,
) Env {
	return newEnv(
		args,
		stdin,
		stdout,
		stderr,
		variables,
	)
}

func environToVariables(environ []string) (map[string]string, error) {
	env := make(map[string]string, len(environ))
	for _, elem := range environ {
		if !strings.ContainsRune(elem, '=') {
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("environment variable does not contain =")
		}
		split := strings.SplitN(elem, "=", 2)
		switch len(split) {
		case 1:
			env[split[0]] = ""
		case 2:
			env[split[0]] = split[1]
		default:
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("unknown environment split")
		}
	}
	return env, nil
}
