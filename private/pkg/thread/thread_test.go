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
	assert.Nil(t, err, "parallelize error")
	assert.Equal(t, int64(0), executed.Load(), "jobs executed")
}
