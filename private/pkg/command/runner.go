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
	"io"
	"os/exec"
	"sort"

	"github.com/bufbuild/buf/private/pkg/ioextended"
	"github.com/bufbuild/buf/private/pkg/thread"
)

var emptyEnv = map[string]string{
	"__EMPTY_ENV": "1",
}

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

// Exec prepares a command to run. Use the returned process to run, start,
// or terminate the command.
func (r *runner) Exec(name string, options ...ExecOption) Process {
	execOptions := newExecOptions(options...)
	cmd := exec.Command(name, execOptions.args...)
	execOptions.Apply(cmd)
	return newProcess(cmd, r)
}

func (r *runner) incement() {
	r.semaphoreC <- struct{}{}
}
func (r *runner) decrement() {
	<-r.semaphoreC
}

type execOptions struct {
	args   []string
	env    map[string]string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	dir    string
}

// We set the defaults after calling any RunOptions on a runOptions struct
// so that users cannot override the empty values, which would lead to the
// default stdin, stdout, stderr, and environment being used.
func newExecOptions(options ...ExecOption) *execOptions {
	execOptions := &execOptions{}
	for _, option := range options {
		option(execOptions)
	}
	if len(execOptions.env) == 0 {
		execOptions.env = emptyEnv
	}
	if execOptions.stdin == nil {
		execOptions.stdin = ioextended.DiscardReader
	}
	return execOptions
}

func (e *execOptions) Apply(cmd *exec.Cmd) {
	cmd.Env = envSlice(e.env)
	cmd.Stdin = e.stdin
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
