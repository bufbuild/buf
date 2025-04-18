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

package thread

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

var (
	globalParallelism = runtime.GOMAXPROCS(0)
	globalLock        sync.RWMutex
)

// Parallelism returns the current parallelism.
//
// This defaults to the number of CPUs.
func Parallelism() int {
	globalLock.RLock()
	parallelism := globalParallelism
	globalLock.RUnlock()
	return parallelism
}

// SetParallelism sets the parallelism.
//
// If parallelism < 1, this sets the parallelism to 1.
func SetParallelism(parallelism int) {
	if parallelism < 1 {
		parallelism = 1
	}
	globalLock.Lock()
	globalParallelism = parallelism
	globalLock.Unlock()
}

// Parallelize runs the jobs in parallel.
//
// A max of Parallelism jobs will be run at once.
// Returns the combined error from the jobs.
func Parallelize(ctx context.Context, jobs []func(context.Context) error, options ...ParallelizeOption) error {
	parallelizeOptions := newParallelizeOptions()
	for _, option := range options {
		option(parallelizeOptions)
	}
	switch len(jobs) {
	case 0:
		return nil
	case 1:
		return jobs[0](ctx)
	}
	multiplier := max(parallelizeOptions.multiplier, 1)
	var cancel context.CancelFunc
	if parallelizeOptions.cancelOnFailure {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}
	semaphoreC := make(chan struct{}, Parallelism()*multiplier)
	var errs []error
	var lock sync.Mutex
	addError := func(err error) {
		lock.Lock()
		errs = append(errs, err)
		lock.Unlock()
	}
	var wg sync.WaitGroup
	var stop bool
	for _, job := range jobs {
		if stop {
			break
		}
		// We always want context cancellation/deadline expiration to take
		// precedence over the semaphore unblocking, but select statements choose
		// among the unblocked non-default cases pseudorandomly. To correctly
		// enforce precedence, use a similar pattern to the check-lock-check
		// pattern common with sync.RWMutex: check the context twice, and only do
		// the semaphore-protected work in the innermost default case.
		select {
		case <-ctx.Done():
			stop = true
			addError(ctx.Err())
		case semaphoreC <- struct{}{}:
			select {
			case <-ctx.Done():
				stop = true
				addError(ctx.Err())
			default:
				job := job
				wg.Add(1)
				go func() {
					if err := job(ctx); err != nil {
						addError(err)
						if cancel != nil {
							cancel()
						}
					}
					// This will never block.
					<-semaphoreC
					wg.Done()
				}()
			}
		}
	}
	wg.Wait()
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

// ParallelizeOption is an option to Parallelize.
type ParallelizeOption func(*parallelizeOptions)

// ParallelizeWithMultiplier returns a new ParallelizeOption that will use a multiple
// of Parallelism() for the number of jobs that can be run at once.
//
// The default is to only do Parallelism() number of jobs.
// A multiplier of <1 has no meaning.
func ParallelizeWithMultiplier(multiplier int) ParallelizeOption {
	return func(parallelizeOptions *parallelizeOptions) {
		parallelizeOptions.multiplier = multiplier
	}
}

// ParallelizeWithCancelOnFailure returns a new ParallelizeOption that will attempt
// to cancel all other jobs via context cancellation if any job fails.
func ParallelizeWithCancelOnFailure() ParallelizeOption {
	return func(parallelizeOptions *parallelizeOptions) {
		parallelizeOptions.cancelOnFailure = true
	}
}

type parallelizeOptions struct {
	multiplier      int
	cancelOnFailure bool
}

func newParallelizeOptions() *parallelizeOptions {
	return &parallelizeOptions{}
}
