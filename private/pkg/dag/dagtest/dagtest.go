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

package dagtest

import (
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/stretchr/testify/require"
)

// Ordered matches cmp.Ordered until we only support Go versions >= 1.21.
//
// TODO: remove and replace with cmp.Ordered when we only support Go versions >= 1.21.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

type ExpectedNode[Key Ordered] struct {
	Key      Key
	Outbound []Key
}

// RequireGraphEqual requires that the graph equals the given ExpectedNodes.
//
// The order of the input ExpectedNodes does not matter, and the order of
// the outbound Keys does not matter.
func RequireGraphEqual[Key Ordered](
	t *testing.T,
	expected []ExpectedNode[Key],
	graph *dag.Graph[Key],
) {
	actual := make([]ExpectedNode[Key], 0, len(expected))
	err := graph.WalkNodes(
		func(key Key, _ []Key, outbound []Key) error {
			actual = append(
				actual,
				ExpectedNode[Key]{
					Key:      key,
					Outbound: outbound,
				},
			)
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, normalizeExpectedNodes(expected), normalizeExpectedNodes(actual))
}

func normalizeExpectedNodes[Key Ordered](expectedNodes []ExpectedNode[Key]) []ExpectedNode[Key] {
	if expectedNodes == nil {
		return []ExpectedNode[Key]{}
	}
	c := slicesextended.Copy(expectedNodes)
	sort.Slice(
		c,
		func(i int, j int) bool {
			return c[i].Key < c[j].Key
		},
	)
	for i, e := range c {
		e.Outbound = normalizeKeys(e.Outbound)
		c[i] = e
	}
	return c
}

func normalizeKeys[Key Ordered](keys []Key) []Key {
	if keys == nil {
		return []Key{}
	}
	keys = slicesextended.Copy(keys)
	sort.Slice(
		keys,
		func(i int, j int) bool {
			return keys[i] < keys[j]
		},
	)
	return keys
}
