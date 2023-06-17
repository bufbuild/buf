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

package dag

// Largely adopted from https://github.com/stevenle/topsort, with modifications.
//
// Copyright 2013 Steven Le. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// See https://github.com/stevenle/topsort/blob/master/LICENSE.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopoSort(t *testing.T) {
	t.Parallel()
	testTopoSortSuccess(
		t,
		func(graph *Graph[string]) {
			// a -> b -> c
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
		},
		"a",
		[]string{"c", "b", "a"},
	)
}

func TestTopoSort2(t *testing.T) {
	t.Parallel()
	testTopoSortSuccess(
		t,
		func(graph *Graph[string]) {
			// a -> c
			// a -> b
			// b -> c
			graph.AddEdge("a", "c")
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
		},
		"a",
		[]string{"c", "b", "a"},
	)
}

func TestTopoSort3(t *testing.T) {
	t.Parallel()
	testTopoSortSuccess(
		t,
		func(graph *Graph[string]) {
			// a -> b
			// a -> d
			// d -> c
			// c -> b
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("d", "c")
			graph.AddEdge("c", "b")
		},
		"a",
		[]string{"b", "c", "d", "a"},
	)
}

func TestTopoSortCycleError(t *testing.T) {
	t.Parallel()
	testTopoSortCycleError(
		t,
		func(graph *Graph[string]) {
			// a -> b
			// b -> a
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "a")
		},
		"a",
		[]string{"a", "b", "a"},
	)
}

func TestTopoSortCycleError2(t *testing.T) {
	t.Parallel()
	testTopoSortCycleError(
		t,
		func(graph *Graph[string]) {
			// a -> b
			// b -> c
			// c -> a
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "a")
		},
		"a",
		[]string{"a", "b", "c", "a"},
	)
}

func TestTopoSortCycleError3(t *testing.T) {
	t.Parallel()
	testTopoSortCycleError(
		t,
		func(graph *Graph[string]) {
			// a -> b
			// b -> c
			// c -> b
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "b")
		},
		"a",
		[]string{"b", "c", "b"},
	)
}

func TestForEachEdge(t *testing.T) {
	t.Parallel()
	testForEachEdgeSuccess(
		t,
		func(graph *Graph[string]) {
			// b -> c
			// a -> c
			// a -> b
			// purposefully not the same start key
			graph.AddEdge("b", "c")
			graph.AddEdge("a", "c")
			graph.AddEdge("a", "b")
		},
		"a",
		[]stringEdge{
			{
				From: "a",
				To:   "c",
			},
			{
				From: "a",
				To:   "b",
			},
			{
				From: "b",
				To:   "c",
			},
		},
	)
}

func TestForEachEdge2(t *testing.T) {
	t.Parallel()
	testForEachEdgeSuccess(
		t,
		func(graph *Graph[string]) {
			// a -> b
			// a -> d
			// b -> c
			// c -> d
			// purposefully not the same start key
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
		},
		"a",
		[]stringEdge{
			{
				From: "a",
				To:   "b",
			},
			{
				From: "b",
				To:   "c",
			},
			{
				From: "c",
				To:   "d",
			},
			{
				From: "a",
				To:   "d",
			},
		},
	)
}

func testTopoSortSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	start string,
	expected []string,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	results, err := graph.TopoSort(start)
	require.NoError(t, err)
	require.Equal(t, expected, results)
}

func testTopoSortCycleError(
	t *testing.T,
	setupGraph func(*Graph[string]),
	start string,
	expectedCycle []string,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	_, err := graph.TopoSort(start)
	require.Equal(
		t,
		&CycleError[string]{
			Keys: expectedCycle,
		},
		err,
	)
}

func testForEachEdgeSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	start string,
	expected []stringEdge,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	var results []stringEdge
	err := graph.ForEachEdge(
		start,
		func(from string, to string) error {
			results = append(
				results,
				stringEdge{
					From: from,
					To:   to,
				},
			)
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, expected, results)
}

type stringEdge struct {
	From string
	To   string
}
