package command

import (
	"context"
	"io"
	"os/exec"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/ioextended"
	"github.com/bufbuild/buf/private/pkg/thread"
)

var emptyEnvContainer = app.NewEnvContainer(
	map[string]string{
		"__EMPTY_ENV": "1",
	},
)

type runner struct {
	parallelism int
}

func newRunner(options ...RunnerOption) *runner {
	runner := &runner{
		parallelism: thread.Parallelism(),
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

func (r *runner) Run(ctx context.Context, name string, options ...RunOption) error {
	runOptions := newRunOptions()
	for _, option := range options {
		option(runOptions)
	}
	if runOptions.envContainer == nil || runOptions.envContainer.Size() == 0 {
		runOptions.envContainer = emptyEnvContainer
	}
	if runOptions.stdin == nil {
		runOptions.stdin = ioextended.DiscardReader
	}
	if runOptions.stdout == nil {
		runOptions.stdout = io.Discard
	}
	if runOptions.stderr == nil {
		runOptions.stderr = io.Discard
	}
	cmd := exec.CommandContext(ctx, name, runOptions.args...)
	cmd.Env = app.Environ(runOptions.envContainer)
	cmd.Stdin = runOptions.stdin
	cmd.Stdout = runOptions.stdout
	cmd.Stderr = runOptions.stderr
	// The default behavior for dir is what we want already, i.e. the current
	// working directory.
	cmd.Dir = runOptions.dir
	err := cmd.Run()
	return err
}

type runOptions struct {
	args         []string
	envContainer app.EnvContainer
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	dir          string
}

// We set the defaults after calling any RunOptions on a runOptions struct
// so that users cannot override the empty values, which would lead to the
// default stdin, stdout, stderr, and environment being used.
func newRunOptions() *runOptions {
	return &runOptions{}
}
