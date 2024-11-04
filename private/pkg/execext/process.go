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

package execext

import (
	"context"
	"errors"
	"os/exec"
)

var errWaitAlreadyCalled = errors.New("wait already called on process")

type process struct {
	ctx   context.Context
	cmd   *exec.Cmd
	waitC chan error
}

func newProcess(ctx context.Context, cmd *exec.Cmd) *process {
	return &process{
		ctx:   ctx,
		cmd:   cmd,
		waitC: make(chan error, 1),
	}
}

func (p *process) Wait() error {
	select {
	case err, ok := <-p.waitC:
		// Process exited
		if ok {
			return err
		}
		return errWaitAlreadyCalled
	case <-p.ctx.Done():
		// Timed out. Send a kill signal and release our handle to it.
		return errors.Join(p.ctx.Err(), p.cmd.Process.Kill())
	}
}

func (p *process) monitor() {
	go func() {
		p.waitC <- p.cmd.Wait()
		close(p.waitC)
	}()
}

func (*process) isProcess() {}
