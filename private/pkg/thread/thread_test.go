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

package thread

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestParallelizeWithImmediateCancellation(t *testing.T) {
	t.Parallel()
	// The bulk of the code relies on subtle timing that's difficult to
	// reproduce, but we can test the most basic use case.
	t.Run("RegularRun", func(t *testing.T) {
		t.Parallel()
		const jobsToExecute = 10
		var (
			executed atomic.Int64
			jobs     = make([]func(context.Context) error, 0, jobsToExecute)
		)
		for i := 0; i < jobsToExecute; i++ {
			jobs = append(jobs, func(_ context.Context) error {
				executed.Inc()
				return nil
			})
		}
		err := Parallelize(context.Background(), jobs)
		assert.NoError(t, err)
		assert.Equal(t, int64(jobsToExecute), executed.Load(), "jobs executed")
	})
	t.Run("WithCtxCancellation", func(t *testing.T) {
		t.Parallel()
		var executed atomic.Int64
		var jobs []func(context.Context) error
		for i := 0; i < 10; i++ {
			jobs = append(jobs, func(_ context.Context) error {
				executed.Inc()
				return nil
			})
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := Parallelize(ctx, jobs)
		assert.Error(t, err)
		assert.Equal(t, int64(0), executed.Load(), "jobs executed")
	})
}
