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

// Package app provides application primitives.
//
// This should be used for all mains. It provides common globals for the OS and environment
// in containers, so that they can be isolated for testing.
//
// This package has three sub-packages:
//
// - appext: More containers for typical buf-related needs, to provide things like loggers.
// - appcmd: A wrapper for cobra.
// - appcmd/appcmdtesting: Testing utilities for appcmd.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/interrupt"
)

// EnvContainer provides environment variables.
type EnvContainer interface {
	// Env gets the environment variable raw string value for the key.
	//
	// Returns empty string if the key is not set or the value is empty.
	Env(key string) string
	// ForEachEnv iterates over all non-empty environment variables and calls the function.
	//
	// The value will never be empty.
	ForEachEnv(func(string, string))
}

// NewEnvContainer returns a new EnvContainer.
//
// Empty values are effectively ignored.
func NewEnvContainer(m map[string]string) EnvContainer {
	return newEnvContainer(m)
}

// NewEnvContainerForEnviron returns a new EnvContainer for the environ slice.
func NewEnvContainerForEnviron(environ []string) (EnvContainer, error) {
	return newEnvContainerForEnviron(environ)
}

// NewEnvContainerForOS returns a new EnvContainer for the operating system.
func NewEnvContainerForOS() (EnvContainer, error) {
	return newEnvContainerForEnviron(os.Environ())
}

// NewEnvContainerWithOverrides returns a new EnvContainer with the values of the input
// EnvContainer, overridden by the values in overrides.
//
// Empty values are effectively ignored. To unset a key, set the value to "" in overrides.
func NewEnvContainerWithOverrides(envContainer EnvContainer, overrides map[string]string) EnvContainer {
	m := EnvironMap(envContainer)
	for key, value := range overrides {
		m[key] = value
	}
	return newEnvContainer(m)
}

// StdinContainer provides stdin.
type StdinContainer interface {
	// Stdin provides stdin.
	//
	// If no value was passed when Stdio was created, this will return io.EOF on any call.
	Stdin() io.Reader
}

// NewStdinContainer returns a new StdinContainer.
func NewStdinContainer(reader io.Reader) StdinContainer {
	return newStdinContainer(reader)
}

// NewStdinContainerForOS returns a new StdinContainer for the operating system.
func NewStdinContainerForOS() StdinContainer {
	return newStdinContainer(os.Stdin)
}

// StdoutContainer provides stdout.
type StdoutContainer interface {
	// Stdout provides stdout.
	//
	// If no value was passed when Stdio was created, this will return io.EOF on any call.
	Stdout() io.Writer
}

// NewStdoutContainer returns a new StdoutContainer.
func NewStdoutContainer(writer io.Writer) StdoutContainer {
	return newStdoutContainer(writer)
}

// NewStdoutContainerForOS returns a new StdoutContainer for the operatoutg system.
func NewStdoutContainerForOS() StdoutContainer {
	return newStdoutContainer(os.Stdout)
}

// StderrContainer provides stderr.
type StderrContainer interface {
	// Stderr provides stderr.
	//
	// If no value was passed when Stdio was created, this will return io.EOF on any call.
	Stderr() io.Writer
}

// NewStderrContainer returns a new StderrContainer.
func NewStderrContainer(writer io.Writer) StderrContainer {
	return newStderrContainer(writer)
}

// NewStderrContainerForOS returns a new StderrContainer for the operaterrg system.
func NewStderrContainerForOS() StderrContainer {
	return newStderrContainer(os.Stderr)
}

// ArgContainer provides the arguments.
type ArgContainer interface {
	// NumArgs gets the number of arguments.
	NumArgs() int
	// Arg gets the ith argument.
	//
	// Panics if i < 0 || i >= Len().
	Arg(i int) string
}

// NewArgContainer returns a new ArgContainer.
func NewArgContainer(args ...string) ArgContainer {
	return newArgContainer(args)
}

// NewArgContainerForOS returns a new ArgContainer for the operating system.
func NewArgContainerForOS() ArgContainer {
	return newArgContainer(os.Args)
}

// Container contains environment variables, args, and stdio.
type Container interface {
	EnvContainer
	StdinContainer
	StdoutContainer
	StderrContainer
	ArgContainer
}

// NewContainer returns a new Container.
func NewContainer(
	env map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	args ...string,
) Container {
	return newContainer(
		NewEnvContainer(env),
		NewStdinContainer(stdin),
		NewStdoutContainer(stdout),
		NewStderrContainer(stderr),
		NewArgContainer(args...),
	)
}

// NewContainerForOS returns a new Container for the operating system.
func NewContainerForOS() (Container, error) {
	envContainer, err := NewEnvContainerForOS()
	if err != nil {
		return nil, err
	}
	return newContainer(
		envContainer,
		NewStdinContainerForOS(),
		NewStdoutContainerForOS(),
		NewStderrContainerForOS(),
		NewArgContainerForOS(),
	), nil
}

// NewContainerForArgs returns a new Container with the replacement args.
func NewContainerForArgs(container Container, newArgs ...string) Container {
	return newContainer(
		container,
		container,
		container,
		container,
		NewArgContainer(newArgs...),
	)
}

// StdioContainer is a stdio container.
type StdioContainer interface {
	StdinContainer
	StdoutContainer
	StderrContainer
}

// EnvStdinContainer is an environment and stdin container.
type EnvStdinContainer interface {
	EnvContainer
	StdinContainer
}

// EnvStdoutContainer is an environment and stdout container.
type EnvStdoutContainer interface {
	EnvContainer
	StdoutContainer
}

// EnvStderrContainer is an environment and stderr container.
type EnvStderrContainer interface {
	EnvContainer
	StderrContainer
}

// EnvStdioContainer is an environment and stdio container.
type EnvStdioContainer interface {
	EnvContainer
	StdioContainer
}

// Environ returns all environment variables in the form "KEY=VALUE".
//
// Equivalent to os.Environ.
//
// Sorted.
func Environ(envContainer EnvContainer) []string {
	var environ []string
	envContainer.ForEachEnv(func(key string, value string) {
		environ = append(environ, key+"="+value)
	})
	sort.Strings(environ)
	return environ
}

// EnvironMap returns all environment variables in a map.
//
// No key will have an empty value.
func EnvironMap(envContainer EnvContainer) map[string]string {
	m := make(map[string]string)
	envContainer.ForEachEnv(func(key string, value string) {
		// This should be done anyways per the EnvContainer documentation but just to make sure
		if value != "" {
			m[key] = value
		}
	})
	return m
}

// Args returns all arguments.
//
// Equivalent to os.Args.
func Args(argList ArgContainer) []string {
	numArgs := argList.NumArgs()
	args := make([]string, numArgs)
	for i := 0; i < numArgs; i++ {
		args[i] = argList.Arg(i)
	}
	return args
}

// EnvBool EnvBoolValue gets and parses the environment variable bool value for the key.
//
// Returns error on parsing error.
func EnvBool(container EnvContainer, key string, defaultValue bool) (bool, error) {
	value := container.Env(key)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(value)
}

// IsDevStdin returns true if the path is the equivalent of /dev/stdin.
func IsDevStdin(path string) bool {
	return path != "" && path == DevStdinFilePath
}

// IsDevStdout returns true if the path is the equivalent of /dev/stdout.
func IsDevStdout(path string) bool {
	return path != "" && path == DevStdoutFilePath
}

// IsDevStderr returns true if the path is the equivalent of /dev/stderr.
func IsDevStderr(path string) bool {
	return path != "" && path == DevStderrFilePath
}

// IsDevNull returns true if the path is the equivalent of /dev/null.
func IsDevNull(path string) bool {
	return path != "" && path == DevNullFilePath
}

// IsDevPath returns true if the path is the equivalent of /dev/stdin, /dev/stdout,
// /dev/stderr, or /dev/null.
func IsDevPath(path string) bool {
	return IsDevStdin(path) || IsDevStdout(path) || IsDevStderr(path) || IsDevNull(path)
}

// Main runs the application using the OS Container and calling os.Exit on the return value of Run.
func Main(ctx context.Context, f func(context.Context, Container) error) {
	container, err := NewContainerForOS()
	if err != nil {
		printError(container, err)
		os.Exit(GetExitCode(err))
	}
	os.Exit(GetExitCode(Run(ctx, container, f)))
}

// Run runs the application using the container.
//
// The run will be stopped on interrupt signal.
// The exit code can be determined using GetExitCode.
func Run(ctx context.Context, container Container, f func(context.Context, Container) error) error {
	ctx, cancel := interrupt.WithCancel(ctx)
	defer cancel()
	if err := f(ctx, container); err != nil {
		printError(container, err)
		return err
	}
	return nil
}

// NewError returns a new Error that contains an exit code.
//
// The exit code cannot be 0.
func NewError(exitCode int, message string) error {
	return newAppError(exitCode, errors.New(message))
}

// NewErrorf returns a new error that contains an exit code.
//
// The exit code cannot be 0.
func NewErrorf(exitCode int, format string, args ...interface{}) error {
	return newAppError(exitCode, fmt.Errorf(format, args...))
}

// WrapError returns a new error that contains an exit code.
//
// The exit code cannot be 0 and the err cannot be nil.
func WrapError(exitCode int, err error) error {
	return newAppError(exitCode, err)
}

// GetExitCode gets the exit code.
//
// If err == nil, this returns 0.
// If err was created by this package, this returns the exit code from the error.
// Otherwise, this returns 1.
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}
	appErr := &appError{}
	if errors.As(err, &appErr) {
		return appErr.exitCode
	}
	return 1
}
