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
	"errors"
	"os/exec"

	"go.uber.org/multierr"
)

var errWaitAlreadyCalled = errors.New("wait already called on process")

type process struct {
	cmd   *exec.Cmd
	done  func()
	waitC chan error
}

// newProcess wraps an *exec.Cmd and monitors it for exiting.
// When the process exits, done will be called.
//
// This implements the Process interface.
//
// The process is expected to have been started by the caller.
func newProcess(cmd *exec.Cmd, done func()) *process {
	return &process{
		cmd:   cmd,
		done:  done,
		waitC: make(chan error, 1),
	}
}

// Monitor starts monitoring of the *exec.Cmd.
func (p *process) Monitor() {
	go func() {
		p.waitC <- p.cmd.Wait()
		close(p.waitC)
		p.done()
	}()
}

// Wait waits for the process to exit.
func (p *process) Wait(ctx context.Context) error {
	select {
	case err, ok := <-p.waitC:
		// Process exited
		if ok {
			return err
		}
		return errWaitAlreadyCalled
	case <-ctx.Done():
		// Timed out. Send a kill signal and release our handle to it.
		return multierr.Combine(ctx.Err(), p.cmd.Process.Kill())
	}
}
