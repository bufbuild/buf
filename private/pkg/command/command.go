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

package command

import (
	"bytes"
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/app"
)

// Process represents a background process.
type Process interface {
	// Wait blocks to wait for the process to exit. It will attempt to kill the
	// process if the passed context expires.
	Wait(ctx context.Context) error
}

// Runner runs external commands.
//
// A Runner will limit the number of concurrent commands, as well as explicitly
// set stdin, stdout, stderr, and env to nil/empty values if not set with options.
//
// All external commands in buf MUST use command.Runner instead of
// exec.Command, exec.CommandContext.
type Runner interface {
	// Run runs the external command. It blocks until the command exits.
	//
	// This should be used instead of exec.CommandContext(...).Run().
	Run(ctx context.Context, name string, options ...RunOption) error

	// Start runs the external command, returning a [Process] handle to track
	// its progress.
	//
	// This should be used instead of exec.Command(...).Start().
	Start(name string, options ...StartOption) (Process, error)

	isRunner()
}

// RunOption is an option for Run.
type RunOption func(*execOptions)

// RunWithArgs returns a new RunOption that sets the arguments other
// than the name.
//
// The default is no arguments.
func RunWithArgs(args ...string) RunOption {
	return func(execOptions *execOptions) {
		execOptions.args = args
	}
}

// RunWithEnv returns a new RunOption that sets the environment variables.
//
// The default is to use the single environment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty environment with the exec package.
//
// If this and RunWithEnviron are specified, the last option specified wins.
func RunWithEnv(env map[string]string) RunOption {
	return func(execOptions *execOptions) {
		execOptions.environ = envSlice(env)
	}
}

// RunWithEnviron returns a new RunOption that sets the environment variables.
//
// The default is to use the single environment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty environment with the exec package.
//
// If this and RunWithEnv are specified, the last option specified wins.
func RunWithEnviron(environ []string) RunOption {
	return func(execOptions *execOptions) {
		execOptions.environ = environ
	}
}

// RunWithStdin returns a new RunOption that sets the stdin.
//
// The default is ioext.DiscardReader.
func RunWithStdin(stdin io.Reader) RunOption {
	return func(execOptions *execOptions) {
		execOptions.stdin = stdin
	}
}

// RunWithStdout returns a new RunOption that sets the stdout.
//
// The default is the null device (os.DevNull).
func RunWithStdout(stdout io.Writer) RunOption {
	return func(execOptions *execOptions) {
		execOptions.stdout = stdout
	}
}

// RunWithStderr returns a new RunOption that sets the stderr.
//
// The default is the null device (os.DevNull).
func RunWithStderr(stderr io.Writer) RunOption {
	return func(execOptions *execOptions) {
		execOptions.stderr = stderr
	}
}

// RunWithDir returns a new RunOption that sets the working directory.
//
// The default is the current working directory.
func RunWithDir(dir string) RunOption {
	return func(execOptions *execOptions) {
		execOptions.dir = dir
	}
}

// StartOption is an option for Start.
type StartOption func(*execOptions)

// StartWithArgs returns a new RunOption that sets the arguments other
// than the name.
//
// The default is no arguments.
func StartWithArgs(args ...string) StartOption {
	return func(execOptions *execOptions) {
		execOptions.args = args
	}
}

// StartWithEnv returns a new RunOption that sets the environment variables.
//
// The default is to use the single environment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty environment with the exec package.
//
// If this and StartWithEnviron are specified, the last option specified wins.
func StartWithEnv(env map[string]string) StartOption {
	return func(execOptions *execOptions) {
		execOptions.environ = envSlice(env)
	}
}

// StartWithEnviron returns a new RunOption that sets the environment variables.
//
// The default is to use the single environment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty environment with the exec package.
//
// If this and StartWithEnv are specified, the last option specified wins.
func StartWithEnviron(environ []string) StartOption {
	return func(execOptions *execOptions) {
		execOptions.environ = environ
	}
}

// StartWithStdin returns a new RunOption that sets the stdin.
//
// The default is ioext.DiscardReader.
func StartWithStdin(stdin io.Reader) StartOption {
	return func(execOptions *execOptions) {
		execOptions.stdin = stdin
	}
}

// StartWithStdout returns a new RunOption that sets the stdout.
//
// The default is the null device (os.DevNull).
func StartWithStdout(stdout io.Writer) StartOption {
	return func(execOptions *execOptions) {
		execOptions.stdout = stdout
	}
}

// StartWithStderr returns a new RunOption that sets the stderr.
//
// The default is the null device (os.DevNull).
func StartWithStderr(stderr io.Writer) StartOption {
	return func(execOptions *execOptions) {
		execOptions.stderr = stderr
	}
}

// StartWithDir returns a new RunOption that sets the working directory.
//
// The default is the current working directory.
func StartWithDir(dir string) StartOption {
	return func(execOptions *execOptions) {
		execOptions.dir = dir
	}
}

// NewRunner returns a new Runner.
func NewRunner(options ...RunnerOption) Runner {
	return newRunner(options...)
}

// RunnerOption is an option for a new Runner.
type RunnerOption func(*runner)

// RunnerWithParallelism returns a new Runner that sets the number of
// external commands that can be run concurrently.
//
// The default is thread.Parallelism().
func RunnerWithParallelism(parallelism int) RunnerOption {
	if parallelism < 1 {
		parallelism = 1
	}
	return func(runner *runner) {
		runner.parallelism = parallelism
	}
}

// RunStdout is a convenience function that attaches the container environment,
// stdin, and stderr, and returns the stdout as a byte slice.
func RunStdout(
	ctx context.Context,
	container app.EnvStdioContainer,
	runner Runner,
	name string,
	args ...string,
) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		name,
		RunWithArgs(args...),
		RunWithEnv(app.EnvironMap(container)),
		RunWithStdin(container.Stdin()),
		RunWithStdout(buffer),
		RunWithStderr(container.Stderr()),
	); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
