// Package clienv contains types to work with the host environment.
package clienv

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Environ wraps environment variables.
type Environ interface {
	// Environ is the equivalent of os.Environ.
	//
	// Do not modify.
	Environ() []string
	// Getenv is the equivalent of os.Getenv.
	Getenv(key string) string
}

// Env is an execution environment for the CLI. This is what is passed to commands.
type Env interface {
	Environ

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
	// WithArgs returns a copy of Env with the replacement args.
	WithArgs(args []string) Env
}

// NewOSEnv returns a new OS environment.
func NewOSEnv() (Env, error) {
	environ := os.Environ()
	environMap, err := environToEnvironMap(environ)
	if err != nil {
		return nil, err
	}
	return newEnv(
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		os.Stderr,
		environ,
		environMap,
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
	environMap map[string]string,
) Env {
	return newEnv(
		args,
		stdin,
		stdout,
		stderr,
		environMapToEnviron(environMap),
		environMap,
	)
}

func environToEnvironMap(environ []string) (map[string]string, error) {
	environMap := make(map[string]string, len(environ))
	for _, elem := range environ {
		if !strings.ContainsRune(elem, '=') {
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("environment variable does not contain =")
		}
		split := strings.SplitN(elem, "=", 2)
		switch len(split) {
		case 1:
			environMap[split[0]] = ""
		case 2:
			environMap[split[0]] = split[1]
		default:
			// Do not print out as we don't want to mistakenly leak a secure environment variable
			return nil, errors.New("unknown environment split")
		}
	}
	return environMap, nil
}

func environMapToEnviron(environMap map[string]string) []string {
	environ := make([]string, 0, len(environMap))
	for key, value := range environMap {
		environ = append(environ, fmt.Sprintf("%s=%s", key, value))
	}
	return environ
}
