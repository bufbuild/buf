// Copyright 2020-2021 Buf Technologies, Inc.
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
	"runtime"
	"sync"

	"go.uber.org/multierr"
)

var (
	globalParallelism = runtime.NumCPU()
	globalLock        sync.RWMutex
)

// Parallelism returns the current parellism.
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
func Parallelize(jobs ...func() error) error {
	switch len(jobs) {
	case 0:
		return nil
	case 1:
		return jobs[0]()
	default:
		semaphoreC := make(chan struct{}, Parallelism())
		var retErr error
		var wg sync.WaitGroup
		var lock sync.Mutex
		for _, job := range jobs {
			job := job
			wg.Add(1)
			semaphoreC <- struct{}{}
			go func() {
				if err := job(); err != nil {
					lock.Lock()
					retErr = multierr.Append(retErr, err)
					lock.Unlock()
				}
				<-semaphoreC
				wg.Done()
			}()
		}
		wg.Wait()
		return retErr
	}
}
