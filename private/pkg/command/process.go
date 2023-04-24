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
	"os/exec"

	"go.uber.org/multierr"
)

type process struct {
	cmd          *exec.Cmd
	doneCallback func()
	wait         chan error
	procErr      error
}

// newProcess monitors cmd and will call doneCallback when the process exits.
func newProcess(cmd *exec.Cmd, doneCallback func()) *process {
	p := &process{
		cmd:          cmd,
		doneCallback: doneCallback,
		wait:         make(chan error),
	}
	go p.monitor()
	return p
}

func (p *process) monitor() {
	p.wait <- p.cmd.Wait()
	p.doneCallback()
}

func (p *process) Wait(ctx context.Context) error {
	select {
	case err, ok := <-p.wait:
		// Process exited
		if ok {
			p.procErr = err
			close(p.wait)
			return err
		}
		return p.procErr
	case <-ctx.Done():
		// Timed out. Send a kill signal and release our handle to it.
		return multierr.Combine(
			ctx.Err(),
			p.cmd.Process.Kill(),
		)
	}
}
