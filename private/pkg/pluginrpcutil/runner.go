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

package pluginrpcutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"

	"buf.build/go/standard/xos/xexec"
	"pluginrpc.com/pluginrpc"
)

type runner struct {
	programName string
	programArgs []string
}

func newRunner(
	programName string,
	programArgs ...string,
) *runner {
	return &runner{
		programName: programName,
		programArgs: programArgs,
	}
}

func (r *runner) Run(ctx context.Context, env pluginrpc.Env) error {
	args := env.Args
	if len(r.programArgs) > 0 {
		args = append(slices.Clone(r.programArgs), env.Args...)
	}
	if err := xexec.Run(
		ctx,
		r.programName,
		xexec.WithArgs(args...),
		xexec.WithStdin(env.Stdin),
		xexec.WithStdout(env.Stdout),
		xexec.WithStderr(env.Stderr),
		xexec.WithEnv(os.Environ()),
	); err != nil {
		execExitError := &exec.ExitError{}
		if errors.As(err, &execExitError) {
			return pluginrpc.NewExitError(execExitError.ExitCode(), execExitError)
		}
		return err
	}
	return nil
}
