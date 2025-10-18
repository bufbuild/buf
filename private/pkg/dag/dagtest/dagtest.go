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

package dagtest

import (
	"cmp"
	"slices"
	"sort"
	"testing"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/stretchr/testify/require"
)

// ExpectedNode is an expected node.
type ExpectedNode[Key cmp.Ordered] struct {
	Key      Key
	Outbound []Key
}

// RequireComparableGraphEqual requires that the Comparable equals the given ExpectedNodes.
//
// The order of the input ExpectedNodes does not matter, and the order of
// the outbound Keys does not matter.
func RequireComparableGraphEqual[Key cmp.Ordered](
	t *testing.T,
	expected []ExpectedNode[Key],
	comparableGraph *dag.ComparableGraph[Key],
) {
	RequireGraphEqual(t, expected, comparableGraph.Graph(), func(key Key) Key { return key })
}

// RequireGraphEqual requires that the graph equals the given ExpectedNodes.
//
// The order of the input ExpectedNodes does not matter, and the order of
// the outbound Keys does not matter.
func RequireGraphEqual[Key cmp.Ordered, Value any](
	t *testing.T,
	expected []ExpectedNode[Key],
	graph *dag.Graph[Key, Value],
	toKey func(Value) Key,
) {
	actual := make([]ExpectedNode[Key], 0, len(expected))
	err := graph.WalkNodes(
		func(value Value, _ []Value, outbound []Value) error {
			actual = append(
				actual,
				ExpectedNode[Key]{
					Key:      toKey(value),
					Outbound: xslices.Map(outbound, toKey),
				},
			)
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, normalizeExpectedNodes(expected), normalizeExpectedNodes(actual))
}

func normalizeExpectedNodes[Key cmp.Ordered](expectedNodes []ExpectedNode[Key]) []ExpectedNode[Key] {
	if expectedNodes == nil {
		return []ExpectedNode[Key]{}
	}
	c := slices.Clone(expectedNodes)
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

func normalizeKeys[Key cmp.Ordered](keys []Key) []Key {
	if keys == nil {
		return []Key{}
	}
	keys = slices.Clone(keys)
	slices.Sort(keys)
	return keys
}
