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

package ulidutil

import (
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		id, err := New(testTime)
		require.NoError(t, err)
		roundTripID, err := FromString(id.String())
		require.NoError(t, err)
		require.Equal(t, id, roundTripID)
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	id, err := New(testTime)
	require.NoError(t, err)
	parsed, err := FromString(id.String())
	require.NoError(t, err)
	require.Equal(t, id, parsed)
	require.True(t, ulid.Time(id.Time()).Equal(testTime))
}

func TestNewParallel(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	numGoroutines := 100
	wait := make(chan struct{})
	errChan := make(chan error, numGoroutines)
	idChan := make(chan ulid.ULID, numGoroutines)
	wg := &sync.WaitGroup{}
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-wait
			id, err := New(testTime)
			if err != nil {
				errChan <- err
				return
			}
			idChan <- id
		}()
	}
	// Start all the goroutines
	close(wait)
	wg.Wait()
	close(errChan)
	for err := range errChan {
		assert.NoError(t, err)
	}
	close(idChan)
	idsSeen := make(map[ulid.ULID]struct{})
	for id := range idChan {
		if _, ok := idsSeen[id]; ok {
			t.Errorf("duplicate ULID generated: %s appeared at least twice", id)
			continue
		}
		idsSeen[id] = struct{}{}
		require.True(t, ulid.Time(id.Time()).Equal(testTime))
	}
}
