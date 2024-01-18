// Copyright 2020-2024 Buf Technologies, Inc.
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
	"bytes"
	"errors"
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
	keyToNode map[Key]*node[Key]
	// need to store order so that we can create a deterministic CycleError
	// in the case of Walk where we have no source nodes, so that we can Walk
	// deterministically and find the cycle.
	keys []Key
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
	toNode := g.getOrAddNode(to)
	fromNode.addOutboundEdge(to)
	toNode.addInboundEdge(from)
}

// ContainsNode returns true if the graph contains the given node.
func (g *Graph[Key]) ContainsNode(key Key) bool {
	g.init()
	_, ok := g.keyToNode[key]
	return ok
}

// NumNodes returns the number of nodes in the graph.
func (g *Graph[Key]) NumNodes() int {
	g.init()
	return len(g.keys)
}

// NumNodes returns the number of edges in the graph.
func (g *Graph[Key]) NumEdges() int {
	g.init()
	var numEdges int
	for _, node := range g.keyToNode {
		numEdges += len(node.outboundEdges)
	}
	return numEdges
}

// WalkNodes visited each node in the Graph based on insertion order.
//
// f is called for each node. The first argument is the key for the node,
// the second argument is all inbound edges, the third argument
// is all outbound edges.
func (g *Graph[Key]) WalkNodes(f func(Key, []Key, []Key) error) error {
	g.init()
	for _, key := range g.keys {
		node, ok := g.keyToNode[key]
		if !ok {
			return fmt.Errorf("key not present: %v", key)
		}
		inboundEdges := make([]Key, len(node.inboundEdges))
		copy(inboundEdges, node.inboundEdges)
		outboundEdges := make([]Key, len(node.outboundEdges))
		copy(outboundEdges, node.outboundEdges)
		if err := f(key, inboundEdges, outboundEdges); err != nil {
			return err
		}
	}
	return nil
}

// WalkEdges visits each edge in the Graph starting at the source keys.
//
// f is called for each directed edge. The first argument is the source
// node, the second is the destination node.
//
// Returns a *CycleError if there is a cycle in the graph.
func (g *Graph[Key]) WalkEdges(f func(Key, Key) error) error {
	g.init()
	if g.NumEdges() == 0 {
		// No edges, do not walk.
		return nil
	}
	sourceKeys, err := g.getSourceKeys()
	if err != nil {
		return err
	}
	switch len(sourceKeys) {
	case 0:
		// If we have no source nodes, we have a cycle in the graph. To print the cycle,
		// we walk starting at all keys We will hit a cycle in this process, however just to check our
		// assumptions, we also verify the the walk returns a CycleError, and if not,
		// return a system error.
		allVisited := make(map[Key]struct{})
		for _, key := range g.keys {
			if err := g.edgeVisit(
				key,
				func(Key, Key) error { return nil },
				newOrderedSet[Key](),
				allVisited,
			); err != nil {
				return err
			}
		}
		return errors.New("graph had cycle based on source node count being zero, but this was not detected during edge walking")
	case 1:
		return g.edgeVisit(
			sourceKeys[0],
			f,
			newOrderedSet[Key](),
			make(map[Key]struct{}),
		)
	default:
		allVisited := make(map[Key]struct{})
		for _, key := range sourceKeys {
			if err := g.edgeVisit(
				key,
				f,
				newOrderedSet[Key](),
				allVisited,
			); err != nil {
				return err
			}
		}
		return nil
	}
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

// DOTString returns a DOT representation of the graph.
//
// keyToString is used to print out the label for each node.
// https://graphviz.org/doc/info/lang.html
func (g *Graph[Key]) DOTString(keyToString func(Key) string) (string, error) {
	keyToIndex := make(map[Key]int)
	nextIndex := 1
	var nodeStrings []string
	var edgeStrings []string
	if err := g.WalkEdges(
		func(from Key, to Key) error {
			fromIndex, ok := keyToIndex[from]
			if !ok {
				fromIndex = nextIndex
				nextIndex++
				keyToIndex[from] = fromIndex
				nodeStrings = append(
					nodeStrings,
					fmt.Sprintf("%d [label=%q]", fromIndex, keyToString(from)),
				)
			}
			toIndex, ok := keyToIndex[to]
			if !ok {
				toIndex = nextIndex
				nextIndex++
				keyToIndex[to] = toIndex
				nodeStrings = append(
					nodeStrings,
					fmt.Sprintf("%d [label=%q]", toIndex, keyToString(to)),
				)
			}
			edgeStrings = append(
				edgeStrings,
				fmt.Sprintf("%d -> %d", fromIndex, toIndex),
			)
			return nil
		},
	); err != nil {
		return "", err
	}
	// We also want to pick up any nodes that do not have edges, and display them.
	if err := g.WalkNodes(
		func(key Key, inboundEdges []Key, outboundEdges []Key) error {
			if _, ok := keyToIndex[key]; ok {
				return nil
			}
			if len(inboundEdges) == 0 && len(outboundEdges) == 0 {
				nodeStrings = append(
					nodeStrings,
					fmt.Sprintf("%d [label=%q]", nextIndex, keyToString(key)),
				)
				edgeStrings = append(
					edgeStrings,
					fmt.Sprintf("%d", nextIndex),
				)
				nextIndex++
				return nil
			}
			// This is a system error.
			return fmt.Errorf("got node %v with %d inbound edges and %d outbound edges, but this was not processed during WalkEdges", key, len(inboundEdges), len(outboundEdges))
		},
	); err != nil {
		return "", err
	}
	if len(nodeStrings) == 0 {
		return "digraph {}", nil
	}
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString("digraph {\n\n")
	for _, nodeString := range nodeStrings {
		_, _ = buffer.WriteString("  ")
		_, _ = buffer.WriteString(nodeString)
		_, _ = buffer.WriteString("\n")
	}
	_, _ = buffer.WriteString("\n")
	for _, edgeString := range edgeStrings {
		_, _ = buffer.WriteString("  ")
		_, _ = buffer.WriteString(edgeString)
		_, _ = buffer.WriteString("\n")
	}
	_, _ = buffer.WriteString("\n}")
	return buffer.String(), nil
}

func (g *Graph[Key]) init() {
	if g.keyToNode == nil {
		g.keyToNode = make(map[Key]*node[Key])
	}
}

func (g *Graph[Key]) getOrAddNode(key Key) *node[Key] {
	node, ok := g.keyToNode[key]
	if !ok {
		node = newNode[Key]()
		g.keyToNode[key] = node
		g.keys = append(g.keys, key)
	}
	return node
}

func (g *Graph[Key]) getSourceKeys() ([]Key, error) {
	var sourceKeys []Key
	// need to get in deterministic order
	for _, key := range g.keys {
		node, ok := g.keyToNode[key]
		if !ok {
			return nil, fmt.Errorf("key not present in keyToNode: %v", key)
		}
		if len(node.inboundEdgeMap) == 0 {
			sourceKeys = append(sourceKeys, key)
		}
	}
	return sourceKeys, nil
}

func (g *Graph[Key]) edgeVisit(
	from Key,
	f func(Key, Key) error,
	thisSourceVisited *orderedSet[Key],
	allSourcesVisited map[Key]struct{},
) error {
	// this is based on this source. we want to make sure we don't
	// have any cycles based on starting at a single source.
	if !thisSourceVisited.add(from) {
		index := thisSourceVisited.index(from)
		cycle := append(thisSourceVisited.keys[index:], from)
		return &CycleError[Key]{Keys: cycle}
	}
	// If we visited this from all edge visiting from other
	// sources, do nothing, we've evaluated all cycles and visited this
	// node properly. It's OK to return here, as we've already checked
	// for cycles with thisSourceVisited.
	if _, ok := allSourcesVisited[from]; ok {
		return nil
	}
	// Add to the map. We'll be needing this for future iterations.
	allSourcesVisited[from] = struct{}{}

	fromNode, ok := g.keyToNode[from]
	if !ok {
		return fmt.Errorf("key not present: %v", from)
	}
	for _, to := range fromNode.outboundEdges {
		if err := f(from, to); err != nil {
			return err
		}
		if err := g.edgeVisit(to, f, thisSourceVisited.copy(), allSourcesVisited); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graph[Key]) topoVisit(
	from Key,
	results *orderedSet[Key],
	visited *orderedSet[Key],
) error {
	if !visited.add(from) {
		index := visited.index(from)
		cycle := append(visited.keys[index:], from)
		return &CycleError[Key]{Keys: cycle}
	}

	fromNode, ok := g.keyToNode[from]
	if !ok {
		return fmt.Errorf("key not present: %v", from)
	}
	for _, to := range fromNode.outboundEdges {
		if err := g.topoVisit(to, results, visited.copy()); err != nil {
			return err
		}
	}

	results.add(from)
	return nil
}

type node[Key comparable] struct {
	outboundEdgeMap map[Key]struct{}
	// need to store order for deterministic visits
	outboundEdges  []Key
	inboundEdgeMap map[Key]struct{}
	// need to store order for deterministic visits
	inboundEdges []Key
}

func newNode[Key comparable]() *node[Key] {
	return &node[Key]{
		outboundEdgeMap: make(map[Key]struct{}),
		inboundEdgeMap:  make(map[Key]struct{}),
	}
}

func (n *node[Key]) addOutboundEdge(key Key) {
	if _, ok := n.outboundEdgeMap[key]; !ok {
		n.outboundEdgeMap[key] = struct{}{}
		n.outboundEdges = append(n.outboundEdges, key)
	}
}

func (n *node[Key]) addInboundEdge(key Key) {
	if _, ok := n.inboundEdgeMap[key]; !ok {
		n.inboundEdgeMap[key] = struct{}{}
		n.inboundEdges = append(n.inboundEdges, key)
	}
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

// returns false if already added
func (s *orderedSet[Key]) add(key Key) bool {
	if _, ok := s.keyToIndex[key]; !ok {
		s.keyToIndex[key] = s.length
		s.keys = append(s.keys, key)
		s.length++
		return true
	}
	return false
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
