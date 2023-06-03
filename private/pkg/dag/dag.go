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

import (
	"fmt"
	"strings"
)

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

// CycleError is an error if the Graph had a cycle.
type CycleError[Key comparable] struct {
	Keys []Key
}

// Error implements error.
func (c *CycleError[Key]) Error() string {
	strs := make([]string, len(c.Keys))
	for i, key := range c.Keys {
		strs[i] = fmt.Sprintf("%v", key)
	}
	return fmt.Sprintf("cycle error: %s", strings.Join(strs, " -> "))
}

// Graph is a directed acyclic graph structure with comparable keys.
type Graph[Key comparable] struct {
	keyToNode map[Key]node[Key]
}

// NewGraph returns a new Graph.
//
// Graphs can also safely be instantiated with &Graph{}.
func NewGraph[Key comparable]() *Graph[Key] {
	graph := &Graph[Key]{}
	graph.init()
	return graph
}

// AddNode adds a node.
func (g *Graph[Key]) AddNode(key Key) {
	g.init()
	g.getOrAddNode(key)
}

// AddEdge adds an edge.
func (g *Graph[Key]) AddEdge(from Key, to Key) {
	g.init()
	fromNode := g.getOrAddNode(from)
	g.AddNode(to)
	fromNode.addEdge(to)
}

// ContainsNode returns true if the graph contains the given node.
func (g *Graph[Key]) ContainsNode(key Key) bool {
	g.init()
	_, ok := g.keyToNode[key]
	return ok
}

// TopoSort topologically sorts the nodes in the Graph starting at the given key.
//
// Returns a *CycleError if there is a cycle in the graph.
func (g *Graph[Key]) TopoSort(start Key) ([]Key, error) {
	g.init()
	results := newOrderedSet[Key]()
	if err := g.topoVisit(start, results, newOrderedSet[Key]()); err != nil {
		return nil, err
	}
	return results.keys, nil
}

func (g *Graph[Key]) init() {
	if g.keyToNode == nil {
		g.keyToNode = make(map[Key]node[Key])
	}
}

func (g *Graph[Key]) getOrAddNode(key Key) node[Key] {
	node, ok := g.keyToNode[key]
	if !ok {
		node = newNode[Key]()
		g.keyToNode[key] = node
	}
	return node
}

func (g *Graph[Key]) topoVisit(key Key, results *orderedSet[Key], visited *orderedSet[Key]) error {
	added := visited.add(key)
	if !added {
		index := visited.index(key)
		cycle := append(visited.keys[index:], key)
		return &CycleError[Key]{Keys: cycle}
	}

	node := g.keyToNode[key]
	for _, edge := range node.edges() {
		if err := g.topoVisit(edge, results, visited.copy()); err != nil {
			return err
		}
	}

	results.add(key)
	return nil
}

type node[Key comparable] map[Key]struct{}

func newNode[Key comparable]() node[Key] {
	return make(node[Key])
}

func (n node[Key]) addEdge(key Key) {
	n[key] = struct{}{}
}

func (n node[Key]) edges() []Key {
	var keys []Key
	for key := range n {
		keys = append(keys, key)
	}
	return keys
}

type orderedSet[Key comparable] struct {
	keyToIndex map[Key]int
	keys       []Key
	length     int
}

func newOrderedSet[Key comparable]() *orderedSet[Key] {
	return &orderedSet[Key]{
		keyToIndex: make(map[Key]int),
	}
}

func (s *orderedSet[Key]) add(key Key) bool {
	_, ok := s.keyToIndex[key]
	if !ok {
		s.keyToIndex[key] = s.length
		s.keys = append(s.keys, key)
		s.length++
	}
	return !ok
}

func (s *orderedSet[Key]) copy() *orderedSet[Key] {
	clone := newOrderedSet[Key]()
	for _, item := range s.keys {
		clone.add(item)
	}
	return clone
}

func (s *orderedSet[Key]) index(item Key) int {
	index, ok := s.keyToIndex[item]
	if ok {
		return index
	}
	return -1
}
