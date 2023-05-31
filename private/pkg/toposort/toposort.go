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

// Package toposort implemented topological sorting.
//
// Topological sorting algorithms are especially useful for dependency calculation,
// and so this particular implementation is mainly intended for this purpose.
// As a result, the direction of edges and the order of the results may seem
// reversed compared to other implementations of topological sorting.
//
// For example, if:
//
//   - A depends on B
//   - B depends on C
//
// The graph is represented as:
//
//	A -> B -> C
//
// Where -> represents a directed edge from one node to another.
//
// The topological ordering of dependencies results in:
//
//	[C, B, A]
//
// The code for this example would look something like:
//
//	// Initialize the graph.
//	graph := &toposort.Graph[string]{}
//
//	// Add edges.
//	graph.AddEdge("A", "B")
//	graph.AddEdge("B", "C")
//
//	// Topologically sort node A.
//	graph.TopoSort("A")  // => [C, B, A]
package toposort

// Largely copied from https://github.com/stevenle/topsort, with some modifications.
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
	"fmt"
	"strings"
)

// Graph is a directed acyclic graph structure with comparable keys.
//
// The Graph cannot have cycles.
type Graph[Key comparable] struct {
	keyToNode map[Key]node[Key]
}

// AddNode adds a node.
func (g *Graph[Key]) AddNode(key Key) {
	g.init()
	if !g.ContainsNode(key) {
		g.keyToNode[key] = make(node[Key])
	}
}

// AddEdge adds an edge.
func (g *Graph[Key]) AddEdge(from Key, to Key) error {
	g.init()
	f := g.getOrAddNode(from)
	g.AddNode(to)
	f.addEdge(to)
	return nil
}

// ContainsNode returns true if the graph contains the given node.
func (g *Graph[Key]) ContainsNode(key Key) bool {
	g.init()
	_, ok := g.keyToNode[key]
	return ok
}

// TopoSort topologically sorts starting at the given key.
//
// Returns error if there is a cycle in the graph.
func (g *Graph[Key]) TopoSort(key Key) ([]Key, error) {
	g.init()
	results := newOrderedSet[Key]()
	err := g.visit(key, results, nil)
	if err != nil {
		return nil, err
	}
	return results.items, nil
}

func (g *Graph[Key]) init() {
	if g.keyToNode == nil {
		g.keyToNode = make(map[Key]node[Key])
	}
}

func (g *Graph[Key]) getOrAddNode(key Key) node[Key] {
	n, ok := g.keyToNode[key]
	if !ok {
		n = make(node[Key])
		g.keyToNode[key] = n
	}
	return n
}

func (g *Graph[Key]) visit(key Key, results *orderedSet[Key], visited *orderedSet[Key]) error {
	if visited == nil {
		visited = newOrderedSet[Key]()
	}

	added := visited.add(key)
	if !added {
		index := visited.index(key)
		cycle := append(visited.items[index:], key)
		strs := make([]string, len(cycle))
		for i, k := range cycle {
			strs[i] = fmt.Sprintf("%v", k)
		}
		return fmt.Errorf("cycle error: %s", strings.Join(strs, " -> "))
	}

	node := g.keyToNode[key]
	for _, edge := range node.edges() {
		err := g.visit(edge, results, visited.copy())
		if err != nil {
			return err
		}
	}

	results.add(key)
	return nil
}

type node[Key comparable] map[Key]bool

func (n node[Key]) addEdge(key Key) {
	n[key] = true
}

func (n node[Key]) edges() []Key {
	var keys []Key
	for k := range n {
		keys = append(keys, k)
	}
	return keys
}

type orderedSet[Key comparable] struct {
	indexes map[Key]int
	items   []Key
	length  int
}

func newOrderedSet[Key comparable]() *orderedSet[Key] {
	return &orderedSet[Key]{
		indexes: make(map[Key]int),
		length:  0,
	}
}

func (s *orderedSet[Key]) add(item Key) bool {
	_, ok := s.indexes[item]
	if !ok {
		s.indexes[item] = s.length
		s.items = append(s.items, item)
		s.length++
	}
	return !ok
}

func (s *orderedSet[Key]) copy() *orderedSet[Key] {
	clone := newOrderedSet[Key]()
	for _, item := range s.items {
		clone.add(item)
	}
	return clone
}

func (s *orderedSet[Key]) index(item Key) int {
	index, ok := s.indexes[item]
	if ok {
		return index
	}
	return -1
}
