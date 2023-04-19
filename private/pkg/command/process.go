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
	"errors"
	"os/exec"

	"go.uber.org/multierr"
)

type process struct {
	cmd        *exec.Cmd
	runner     *runner
	terminated bool
}

func newProcess(cmd *exec.Cmd, runner *runner) *process {
	return &process{
		cmd:    cmd,
		runner: runner,
	}
}

func (p *process) Run(ctx context.Context) error {
	if err := p.Start(); err != nil {
		return err
	}
	return p.Wait(ctx)
}

func (p *process) Start() error {
	err := p.cmd.Start()
	if err == nil {
		p.runner.incement()
	}
	return err
}

func (p *process) Wait(ctx context.Context) error {
	// There will not be a second call to wait. Always decrement process
	// counters when we return.
	if p.terminated {
		return errors.New("process already terminated")
	}
	p.terminated = true
	defer p.runner.decrement()
	// Wait for the process to exit.
	wait := make(chan error)
	go func() {
		wait <- p.cmd.Wait()
	}()
	select {
	case err := <-wait:
		// Process exited.
		return err
	case <-ctx.Done():
		// Timed out. Send a kill signal and release our handle to it.
		return multierr.Combine(
			ctx.Err(),
			p.cmd.Process.Kill(),
			p.cmd.Process.Release(),
		)
	}
}
