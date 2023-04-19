// Copyright 2020-2023 Buf Technologies, Inc.
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

// Process represents a prepared command to execute.
type Process interface {
	// Run executes the command and waits for it to exit.
	Run(ctx context.Context) error

	// Start executes the command and returns. Call [Wait] to wait for the
	// process to exit.
	Start() error

	// Wait waits for the process to exit. If the context expires, it will kill
	// the process and release the resources.
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
	// Exec prepares a command to run. Use the returned process to run or start
	// the command.
	Exec(name string, options ...ExecOption) Process
}

// ExecOption is an option for Run.
type ExecOption func(*execOptions)

// ExecWithArgs returns a new RunOption that sets the arguments other
// than the name.
//
// The default is no arguments.
func ExecWithArgs(args ...string) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.args = args
	}
}

// ExecWithEnv returns a new RunOption that sets the environment variables.
//
// The default is to use the single environment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty environment with the exec package.
func ExecWithEnv(env map[string]string) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.env = env
	}
}

// ExecWithStdin returns a new RunOption that sets the stdin.
//
// The default is ioextended.DiscardReader.
func ExecWithStdin(stdin io.Reader) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.stdin = stdin
	}
}

// ExecWithStdout returns a new RunOption that sets the stdout.
//
// The default is the null device (os.DevNull).
func ExecWithStdout(stdout io.Writer) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.stdout = stdout
	}
}

// ExecWithStderr returns a new RunOption that sets the stderr.
//
// The default is the null device (os.DevNull).
func ExecWithStderr(stderr io.Writer) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.stderr = stderr
	}
}

// ExecWithDir returns a new RunOption that sets the working directory.
//
// The default is the current working directory.
func ExecWithDir(dir string) ExecOption {
	return func(runOptions *execOptions) {
		runOptions.dir = dir
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
	if err := runner.Exec(
		name,
		ExecWithArgs(args...),
		ExecWithEnv(app.EnvironMap(container)),
		ExecWithStdin(container.Stdin()),
		ExecWithStdout(buffer),
		ExecWithStderr(container.Stderr()),
	).Run(ctx); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
