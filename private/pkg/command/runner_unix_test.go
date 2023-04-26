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

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoubleWait(t *testing.T) {
	t.Parallel()

	runner := NewRunner()
	process, err := runner.Start("echo")
	require.NoError(t, err)
	ctx := context.Background()
	_ = process.Wait(ctx)
	require.Equal(t, process.Wait(ctx), errWaitAlreadyCalled)
}

func TestNoDeadlock(t *testing.T) {
	t.Parallel()

	runner := NewRunner(RunnerWithParallelism(2))
	processes := make([]Process, 4)
	for i := 0; i < 4; i++ {
		process, err := runner.Start("sleep", StartWithArgs("1"))
		require.NoError(t, err)
		processes[i] = process
	}
	ctx := context.Background()
	for _, process := range processes {
		require.NoError(t, process.Wait(ctx))
	}
}
