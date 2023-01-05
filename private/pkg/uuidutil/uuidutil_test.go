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

package uuidutil

import (
	"testing"

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
