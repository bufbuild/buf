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

//go:build unix

package execext

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartSimple(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	processes := make([]Process, 4)
	for i := 0; i < 4; i++ {
		process, err := Start(ctx, "sleep", WithArgs("1"))
		require.NoError(t, err)
		processes[i] = process
	}
	for _, process := range processes {
		require.NoError(t, process.Wait())
	}
}

func TestStartDoubleWait(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	process, err := Start(ctx, "echo")
	require.NoError(t, err)
	_ = process.Wait()
	require.Equal(t, process.Wait(), errWaitAlreadyCalled)
}
