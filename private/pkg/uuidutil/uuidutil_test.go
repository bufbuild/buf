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

package uuidutil

import (
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		id, err := New()
		require.NoError(t, err)
		dashless, err := ToDashless(id)
		require.NoError(t, err)
		roundTripID, err := FromDashless(dashless)
		require.NoError(t, err)
		require.Equal(t, id, roundTripID)
	}
}

func TestFromStringFailsWithDashless(t *testing.T) {
	t.Parallel()
	id, err := New()
	require.NoError(t, err)
	dashless, err := ToDashless(id)
	require.NoError(t, err)
	_, err = FromString(dashless)
	require.Error(t, err)
}

func TestFromDashlessFailsWithUUID(t *testing.T) {
	t.Parallel()
	id, err := New()
	require.NoError(t, err)
	_, err = FromDashless(id.String())
	require.Error(t, err)
}

func TestValidateFailsWithDashless(t *testing.T) {
	t.Parallel()
	id, err := New()
	require.NoError(t, err)
	dashless, err := ToDashless(id)
	require.NoError(t, err)
	err = Validate(dashless)
	require.Error(t, err)
}

func TestValidateDashlessFailsWithUUID(t *testing.T) {
	t.Parallel()
	id, err := New()
	require.NoError(t, err)
	err = ValidateDashless(id.String())
	require.Error(t, err)
}

func TestFromStringSliceFailsWithDashless(t *testing.T) {
	t.Parallel()
	id1, err := New()
	require.NoError(t, err)
	id2, err := New()
	require.NoError(t, err)
	dashless1, err := ToDashless(id1)
	require.NoError(t, err)
	dashless2, err := ToDashless(id2)
	require.NoError(t, err)
	dashless := []string{dashless1, dashless2}
	_, err = FromStringSlice(dashless)
	require.Error(t, err)
}

func TestFromStringSlice(t *testing.T) {
	t.Parallel()
	id1, err := New()
	require.NoError(t, err)
	id2, err := New()
	require.NoError(t, err)
	ids := []string{id1.String(), id2.String()}
	uuids, err := FromStringSlice(ids)
	require.NoError(t, err)
	require.Equal(t, 2, len(uuids))
	require.Equal(t, id1, uuids[0])
	require.Equal(t, id2, uuids[1])
}

func TestNewUlid(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	id, err := NewULID(testTime)
	require.NoError(t, err)
	parsed, err := FromString(id.String())
	require.NoError(t, err)
	require.Equal(t, id, parsed)
	require.True(t, ulid.Time(ulid.ULID(id).Time()).Equal(testTime))
}

func TestNewULIDParallel(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	numGoroutines := 100
	wait := make(chan struct{})
	errChan := make(chan error, numGoroutines)
	idChan := make(chan uuid.UUID, numGoroutines)
	wg := &sync.WaitGroup{}
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-wait
			id, err := NewULID(testTime)
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
	idsSeen := make(map[uuid.UUID]struct{})
	for id := range idChan {
		if _, ok := idsSeen[id]; ok {
			t.Errorf("duplicate UUID generated: %s appeared at least twice", id)
			continue
		}
		idsSeen[id] = struct{}{}
		require.True(t, ulid.Time(ulid.ULID(id).Time()).Equal(testTime))
	}
}
