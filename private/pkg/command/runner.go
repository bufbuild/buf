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
	"context"
	"io"
	"os/exec"
	"sort"

	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/thread"
)

var emptyEnv = envSlice(
	map[string]string{
		"__EMPTY_ENV": "1",
	},
)

type runner struct {
	parallelism int

	semaphoreC chan struct{}
}

func newRunner(options ...RunnerOption) *runner {
	runner := &runner{
		parallelism: thread.Parallelism(),
	}
	for _, option := range options {
		option(runner)
	}
	runner.semaphoreC = make(chan struct{}, runner.parallelism)
	return runner
}

func (r *runner) Run(ctx context.Context, name string, options ...RunOption) error {
	execOptions := newExecOptions()
	for _, option := range options {
		option(execOptions)
	}
	cmd := exec.CommandContext(ctx, name, execOptions.args...)
	execOptions.ApplyToCmd(cmd)
	r.increment()
	err := cmd.Run()
	r.decrement()
	return err
}

func (r *runner) Start(name string, options ...StartOption) (Process, error) {
	execOptions := newExecOptions()
	for _, option := range options {
		option(execOptions)
	}
	cmd := exec.Command(name, execOptions.args...)
	execOptions.ApplyToCmd(cmd)
	r.increment()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	process := newProcess(cmd, r.decrement)
	process.Monitor()
	return process, nil
}

func (r *runner) increment() {
	r.semaphoreC <- struct{}{}
}

func (r *runner) decrement() {
	<-r.semaphoreC
}

func (*runner) isRunner() {}

type execOptions struct {
	args    []string
	environ []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	dir     string
}

func newExecOptions() *execOptions {
	return &execOptions{}
}

func (e *execOptions) ApplyToCmd(cmd *exec.Cmd) {
	// If the user did not specify env vars, we want to make sure
	// the command has access to none, as the default is the current env.
	if len(e.environ) == 0 {
		cmd.Env = emptyEnv
	} else {
		cmd.Env = e.environ
	}
	// If the user did not specify any stdin, we want to make sure
	// the command has access to none, as the default is the default stdin.
	if e.stdin == nil {
		cmd.Stdin = ioext.DiscardReader
	} else {
		cmd.Stdin = e.stdin
	}
	// If Stdout or Stderr are nil, os/exec connects the process output directly
	// to the null device.
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr
	// The default behavior for dir is what we want already, i.e. the current
	// working directory.
	cmd.Dir = e.dir
}

func envSlice(env map[string]string) []string {
	var environ []string
	for key, value := range env {
		environ = append(environ, key+"="+value)
	}
	sort.Strings(environ)
	return environ
}
