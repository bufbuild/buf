// Copyright 2020-2025 Buf Technologies, Inc.
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

package execext

import (
	"context"
	"io"
	"os/exec"
	"slices"
)

var emptyEnv = []string{"__EMPTY_ENV__=1"}

// Run runs the external command. It blocks until the command exits.
//
// Stdin, stdout, stderr, and env will be explicitly set to nil/empty values if not set with options.
// The command will be killed if the Context is cancelled.
//
// This should be used instead of exec.CommandContext(...).Run().
func Run(ctx context.Context, name string, options ...RunOption) error {
	runStartOptions := newRunStartOptions()
	for _, option := range options {
		option.applyRun(runStartOptions)
	}
	cmd := exec.CommandContext(ctx, name, runStartOptions.args...)
	runStartOptions.applyCmd(cmd)
	return cmd.Run()
}

// Start runs the external command, returning a [Process] to track its progress.
//
// Stdin, stdout, stderr, and env will be explicitly set to nil/empty values if not set with options.
// The command will be killed if the Context is cancelled.
//
// This should be used instead of exec.Command(...).Start().
func Start(ctx context.Context, name string, options ...StartOption) (Process, error) {
	runStartOptions := newRunStartOptions()
	for _, option := range options {
		option.applyStart(runStartOptions)
	}
	cmd := exec.CommandContext(ctx, name, runStartOptions.args...)
	runStartOptions.applyCmd(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	process := newProcess(ctx, cmd)
	process.monitor()
	return process, nil
}

// Process represents a background process.
type Process interface {
	// Wait blocks to wait for the process to exit.
	Wait() error

	isProcess()
}

// RunOption is an option for [Run].
type RunOption interface {
	applyRun(*runStartOptions)
}

// StartOption is an option for [Start].
type StartOption interface {
	applyStart(*runStartOptions)
}

// RunStartOption is both a [RunOption] and a [StartOption].
//
// We split out RunOptions and StartOptions for maximum future flexibility, in case we ever want
// the options for [Run] and [Start] to deviate.
type RunStartOption interface {
	RunOption
	StartOption
}

// WithArgs returns a new option that sets the arguments other than the name.
//
// The default is no additional arguments.
func WithArgs(args ...string) RunStartOption {
	return &argsOption{args: slices.Clone(args)}
}

// WithEnv returns a new option that sets the environment variables.
//
// The default is to use the single envment variable __EMPTY_ENV__=1 as we
// cannot explicitly set an empty envment with the exec package.
//
// If this and WithEnv are specified, the last option specified wins.
func WithEnv(env []string) RunStartOption {
	return &envOption{env: slices.Clone(env)}
}

// WithStdin returns a new option that sets the stdin.
//
// The default is a [io.Reader] that always returns empty..
func WithStdin(stdin io.Reader) RunStartOption {
	return &stdinOption{stdin: stdin}
}

// WithStdout returns a new option that sets the stdout.
//
// The default is a [io.Writer] that ignores all writes..
func WithStdout(stdout io.Writer) RunStartOption {
	return &stdoutOption{stdout: stdout}
}

// WithStderr returns a new option that sets the stderr.
//
// The default is a [io.Writer] that ignores all writes..
func WithStderr(stderr io.Writer) RunStartOption {
	return &stderrOption{stderr: stderr}
}

// WithDir returns a new option that sets the working directory.
//
// The default is the current working directory.
func WithDir(dir string) RunStartOption {
	return &dirOption{dir: dir}
}

// *** PRIVATE ***

type argsOption struct {
	args []string
}

func (a *argsOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.args = a.args
}

func (a *argsOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.args = a.args
}

type envOption struct {
	env []string
}

func (e *envOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.env = e.env
}

func (e *envOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.env = e.env
}

type stdinOption struct {
	stdin io.Reader
}

func (i *stdinOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.stdin = i.stdin
}

func (i *stdinOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.stdin = i.stdin
}

type stdoutOption struct {
	stdout io.Writer
}

func (o *stdoutOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.stdout = o.stdout
}

func (o *stdoutOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.stdout = o.stdout
}

type stderrOption struct {
	stderr io.Writer
}

func (r *stderrOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.stderr = r.stderr
}

func (r *stderrOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.stderr = r.stderr
}

type dirOption struct {
	dir string
}

func (d *dirOption) applyRun(runStartOptions *runStartOptions) {
	runStartOptions.dir = d.dir
}

func (d *dirOption) applyStart(runStartOptions *runStartOptions) {
	runStartOptions.dir = d.dir
}

type runStartOptions struct {
	args   []string
	env    []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	dir    string
}

func newRunStartOptions() *runStartOptions {
	return &runStartOptions{}
}

func (rs *runStartOptions) applyCmd(cmd *exec.Cmd) {
	// If the user did not specify env vars, we want to make sure
	// the command has access to none, as the default is the current env.
	if len(rs.env) == 0 {
		cmd.Env = emptyEnv
	} else {
		cmd.Env = rs.env
	}
	// If the user did not specify any stdin, we want to make sure
	// the command has access to none, as the default is the default stdin.
	//
	// Note: This *should* be the same as just having cmd.Stdin = nil, given that
	// exec.Cmd documents that Stdin has the same behavior as Stdout/Stderr, namely
	// that os.DevNull is used. This has been the case for Stdin since at least 2014.
	// However, way back when this package was first introduced, we set up the discardReader
	// for some reason, and we can't remember why. We're terrified to change it, as there
	// *may* have been some reason to do so. os.DevNull is actually just a string such
	// as "/dev/null" on Unix systems, so how Golang actually handles this is somewhat
	// unknown. Honestly, I might just want to change Stdout/Stderr to use a discardWriter.
	if rs.stdin == nil {
		cmd.Stdin = discardReader{}
	} else {
		cmd.Stdin = rs.stdin
	}
	// If Stdout or Stderr are nil, os/exec connects the process output directly
	// to the null device.
	cmd.Stdout = rs.stdout
	cmd.Stderr = rs.stderr
	// The default behavior for dir is what we want already, i.e. the current
	// working directory.
	cmd.Dir = rs.dir
}

type discardReader struct{}

func (discardReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
