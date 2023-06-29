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
			// e -> b not part of traversal to a on purpose
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("d", "c")
			graph.AddEdge("c", "b")
			graph.AddEdge("e", "b")
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
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "b")
		},
		"a",
		[]string{"b", "c", "b"},
	)
}

func TestWalkEdges(t *testing.T) {
	t.Parallel()
	testWalkEdgesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("b", "c")
			graph.AddEdge("a", "c")
			graph.AddEdge("a", "b")
		},
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

func TestWalkEdges2(t *testing.T) {
	t.Parallel()
	testWalkEdgesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
		},
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

func TestWalkEdges3(t *testing.T) {
	t.Parallel()
	testWalkEdgesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
		},
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
			{
				From: "e",
				To:   "b",
			},
		},
	)
}

func TestWalkEdgesCycleError(t *testing.T) {
	t.Parallel()
	testWalkEdgesCycleError(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddEdge("d", "b")
		},
		[]string{"b", "c", "d", "b"},
	)
}

func TestWalkEdgesCycleError2(t *testing.T) {
	t.Parallel()
	testWalkEdgesCycleError(
		t,
		func(graph *Graph[string]) {
			// there are no sources
			graph.AddEdge("a", "b")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddEdge("d", "e")
			graph.AddEdge("d", "a")
		},
		[]string{"b", "c", "d", "e", "b"},
	)
}

func TestWalkNodes(t *testing.T) {
	t.Parallel()
	testWalkNodesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddNode("f")
		},
		[]stringNode{
			{
				Key:      "a",
				Inbound:  []string{},
				Outbound: []string{"b", "d"},
			},
			{
				Key:      "b",
				Inbound:  []string{"a", "e"},
				Outbound: []string{"c"},
			},
			{
				Key:      "d",
				Inbound:  []string{"a", "c"},
				Outbound: []string{},
			},
			{
				Key:      "c",
				Inbound:  []string{"b"},
				Outbound: []string{"d"},
			},
			{
				Key:      "e",
				Inbound:  []string{},
				Outbound: []string{"b"},
			},
			{
				Key:      "f",
				Inbound:  []string{},
				Outbound: []string{},
			},
		},
	)
}

func TestNumNodes(t *testing.T) {
	t.Parallel()
	testNumNodesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddNode("f")
		},
		6,
	)
}

func TestNumEdges(t *testing.T) {
	t.Parallel()
	testNumEdgesSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddNode("f")
		},
		5,
	)
}

func TestDOTString(t *testing.T) {
	t.Parallel()
	testDOTStringSuccess(
		t,
		func(graph *Graph[string]) {
			graph.AddEdge("a", "b")
			graph.AddEdge("a", "d")
			graph.AddEdge("b", "c")
			graph.AddEdge("c", "d")
			graph.AddEdge("e", "b")
			graph.AddNode("f")
		},
		`digraph {

  1 [label="a"]
  2 [label="b"]
  3 [label="d"]
  4 [label="c"]
  5 [label="e"]
  6 [label="f"]

  1 -> 2
  1 -> 3
  2 -> 4
  4 -> 3
  5 -> 2
  6

}`,
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

func testWalkEdgesSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expected []stringEdge,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	var results []stringEdge
	err := graph.WalkEdges(
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

func testWalkEdgesCycleError(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expectedCycle []string,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	err := graph.WalkEdges(func(string, string) error { return nil })
	require.Equal(
		t,
		&CycleError[string]{
			Keys: expectedCycle,
		},
		err,
	)
}

func testWalkNodesSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expected []stringNode,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	var results []stringNode
	err := graph.WalkNodes(
		func(key string, inbound []string, outbound []string) error {
			results = append(
				results,
				stringNode{
					Key:      key,
					Inbound:  inbound,
					Outbound: outbound,
				},
			)
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, expected, results)
}

func testNumNodesSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expected int,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	require.Equal(t, expected, graph.NumNodes())
}

func testNumEdgesSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expected int,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	require.Equal(t, expected, graph.NumEdges())
}

func testDOTStringSuccess(
	t *testing.T,
	setupGraph func(*Graph[string]),
	expected string,
) {
	graph := &Graph[string]{}
	setupGraph(graph)
	s, err := graph.DOTString(func(key string) string { return key })
	require.NoError(t, err)
	require.Equal(t, expected, s)
}

type stringEdge struct {
	From string
	To   string
}

type stringNode struct {
	Key      string
	Inbound  []string
	Outbound []string
}
