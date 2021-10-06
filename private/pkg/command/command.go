package command

import (
	"bytes"
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/app"
)

// Runner runs external commands.
//
// A Runner will limit the number of concurrent commands, as well as explicitly
// set stdin, stdout, stderr, and env to nil/empty values if not set with options.
//
// All external commands in buf MUST use command.Runner instead of
// exec.Command, exec.CommandContext.
type Runner interface {
	// Run runs the external command.
	//
	// This should be used instead of exec.CommandContext(...).Run().
	Run(ctx context.Context, name string, options ...RunOption) error
}

// RunOption is an option for Run.
type RunOption func(*runOptions)

// RunWithArgs returns a new RunOption that sets the arguments other
// than the name.
//
// The default is no arguments.
func RunWithArgs(args ...string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.args = args
	}
}

// RunWithEnvContainer returns a new RunOption that sets the environment variables
// container.
//
// The default is to use an EnvContainer with the single environment variable
// __EMPTY_ENV__=1 as we cannot explicitly set an empty environment with the exec package.
func RunWithEnvContainer(envContainer app.EnvContainer) RunOption {
	return func(runOptions *runOptions) {
		runOptions.envContainer = envContainer
	}
}

// RunWithStdin returns a new RunOption that sets the stdin.
//
// The default is ioextended.DiscardReader.
func RunWithStdin(stdin io.Reader) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stdin = stdin
	}
}

// RunWithStdout returns a new RunOption that sets the stdout.
//
// The default is io.Discard.
func RunWithStdout(stdout io.Writer) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stdout = stdout
	}
}

// RunWithStderr returns a new RunOption that sets the stderr.
//
// The default is io.Discard.
func RunWithStderr(stderr io.Writer) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stderr = stderr
	}
}

// RunWithDir returns a new RunOption that sets the working directory.
//
// The default is the current working directory.
func RunWithDir(dir string) RunOption {
	return func(runOptions *runOptions) {
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
	if err := runner.Run(
		ctx,
		name,
		RunWithArgs(args...),
		RunWithEnvContainer(container),
		RunWithStdin(container.Stdin()),
		RunWithStdout(buffer),
		RunWithStderr(container.Stderr()),
	); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
