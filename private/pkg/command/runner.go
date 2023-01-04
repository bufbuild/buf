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
	"context"
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

func (r *runner) Run(ctx context.Context, name string, options ...RunOption) error {
	runOptions := newRunOptions()
	for _, option := range options {
		option(runOptions)
	}
	if len(runOptions.env) == 0 {
		runOptions.env = emptyEnv
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
	cmd.Env = envSlice(runOptions.env)
	cmd.Stdin = runOptions.stdin
	cmd.Stdout = runOptions.stdout
	cmd.Stderr = runOptions.stderr
	// The default behavior for dir is what we want already, i.e. the current
	// working directory.
	cmd.Dir = runOptions.dir
	r.semaphoreC <- struct{}{}
	err := cmd.Run()
	<-r.semaphoreC
	return err
}

type runOptions struct {
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
func newRunOptions() *runOptions {
	return &runOptions{}
}

func envSlice(env map[string]string) []string {
	var environ []string
	for key, value := range env {
		environ = append(environ, key+"="+value)
	}
	sort.Strings(environ)
	return environ
}
