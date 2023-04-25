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

var errWaitAlreadyCalled = errors.New("wait already called")

type process struct {
	cmd          cmdCaller
	doneCallback func()
	wait         chan error
}

// newProcess monitors cmd and will call doneCallback when the process exits.
func newProcess(cmd cmdCaller, doneCallback func()) *process {
	return &process{
		cmd:          cmd,
		doneCallback: doneCallback,
		wait:         make(chan error),
	}
}

// Start runs the command and monitors for its exit.
func (p *process) Start() error {
	if err := p.cmd.Start(); err != nil {
		return err
	}
	go func() {
		err := p.cmd.Wait()
		p.doneCallback()
		p.wait <- err
	}()
	return nil
}

func (p *process) Wait(ctx context.Context) error {
	select {
	case err, ok := <-p.wait:
		// Process exited
		if ok {
			close(p.wait)
			return err
		}
		return errWaitAlreadyCalled
	case <-ctx.Done():
		// Timed out. Send a kill signal and release our handle to it.
		return multierr.Combine(
			ctx.Err(),
			p.cmd.Kill(),
		)
	}
}

type cmdCaller interface {
	Start() error
	Wait() error
	Kill() error
}

type cmdCalls struct {
	cmd *exec.Cmd
}

func newCmdCalls(cmd *exec.Cmd) *cmdCalls {
	return &cmdCalls{cmd: cmd}
}

func (c *cmdCalls) Start() error {
	return c.cmd.Start()
}

func (c *cmdCalls) Wait() error {
	return c.cmd.Wait()
}

func (c *cmdCalls) Kill() error {
	return c.cmd.Process.Kill()
}
