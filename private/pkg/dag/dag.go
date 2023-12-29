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
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
type CycleError[Key any] struct {
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

// Graph is a directed acyclic graph structure.
type Graph[Key any, Comp comparable] struct {
	getKeyToComp func(Key) Comp
	compToKey    map[Comp]Key

	compToNode map[Comp]*node[Comp]
	// need to store order so that we can create a deterministic CycleError
	// in the case of Walk where we have no source nodes, so that we can Walk
	// deterministically and find the cycle.
	comps []Comp
}

// ComparableGraph is a Graph that uses comparable Keys.
type ComparableGraph[Key comparable] struct {
	Graph[Key, Key]
}

// NewGraph returns a new Graph for an any Key.
//
// The toComparable function must convert a Key to a unique comparable type.
// It is up to the caller to make sure that the return value is unique on a per-key basis.
//
// This constructor must be used when initializing a Graph.
//
// TODO: It really stinks that we have to use the constructor. We have what amounts
// to silent errors now below with functions that don't return an error. We should
// figure out a way to do this properly, or perhaps just panic if we don't use the constructor.
func NewGraph[Key any, Comp comparable](toComparable func(Key) Comp) *Graph[Key, Comp] {
	return &Graph[Key, Comp]{
		getKeyToComp: toComparable,
		compToKey:    make(map[Comp]Key),
		compToNode:   make(map[Comp]*node[Comp]),
	}
}

// NewComparableGraph returns a new Graph for a comparable Key.
//
// This constructor must be used when initializing a ComparableGraph.
//
// TODO: It really stinks that we have to use the constructor. We have what amounts
// to silent errors now below with functions that don't return an error. We should
// figure out a way to do this properly, or perhaps just panic if we don't use the constructor.
//
// Do not use interfaces as Keys! If your Key type is an interface, use NewGraph.
func NewComparableGraph[Key comparable]() *ComparableGraph[Key] {
	return &ComparableGraph[Key]{
		Graph: Graph[Key, Key]{
			getKeyToComp: func(key Key) Key { return key },
			compToKey:    make(map[Key]Key),
			compToNode:   make(map[Key]*node[Key]),
		},
	}
}

// AddNode adds a node.
func (g *Graph[Key, Comp]) AddNode(key Key) {
	if err := g.checkInit(); err != nil {
		return
	}
	g.getOrAddNode(key)
}

// AddEdge adds an edge.
func (g *Graph[Key, Comp]) AddEdge(from Key, to Key) {
	if err := g.checkInit(); err != nil {
		return
	}
	fromNode := g.getOrAddNode(from)
	toNode := g.getOrAddNode(to)
	fromNode.addOutboundEdge(g.getKeyToComp(to))
	toNode.addInboundEdge(g.getKeyToComp(from))
}

// ContainsNode returns true if the graph contains the given node.
func (g *Graph[Key, Comp]) ContainsNode(key Key) bool {
	if err := g.checkInit(); err != nil {
		return false
	}
	_, ok := g.compToNode[g.getKeyToComp(key)]
	return ok
}

// NumNodes returns the number of nodes in the graph.
func (g *Graph[Key, Comp]) NumNodes() int {
	if err := g.checkInit(); err != nil {
		return 0
	}
	return len(g.comps)
}

// NumNodes returns the number of edges in the graph.
func (g *Graph[Key, Comp]) NumEdges() int {
	if err := g.checkInit(); err != nil {
		return 0
	}
	var numEdges int
	for _, node := range g.compToNode {
		numEdges += len(node.outboundEdges)
	}
	return numEdges
}

// WalkNodes visited each node in the Graph based on insertion order.
//
// f is called for each node. The first argument is the key for the node,
// the second argument is all inbound edges, the third argument
// is all outbound edges.
func (g *Graph[Key, Comp]) WalkNodes(f func(Key, []Key, []Key) error) error {
	if err := g.checkInit(); err != nil {
		return err
	}
	for _, comp := range g.comps {
		node, ok := g.compToNode[comp]
		if !ok {
			return fmt.Errorf("key not present: %v", comp)
		}
		key, err := g.getCompToKey(comp)
		if err != nil {
			return err
		}
		inboundKeys, err := g.getCompsToKeys(node.inboundEdges)
		if err != nil {
			return err
		}
		outboundKeys, err := g.getCompsToKeys(node.outboundEdges)
		if err != nil {
			return err
		}
		if err := f(key, inboundKeys, outboundKeys); err != nil {
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
func (g *Graph[Key, Comp]) WalkEdges(f func(Key, Key) error) error {
	if err := g.checkInit(); err != nil {
		return err
	}
	if g.NumEdges() == 0 {
		// No edges, do not walk.
		return nil
	}
	sourceComps, err := g.getSourceComps()
	if err != nil {
		return err
	}
	switch len(sourceComps) {
	case 0:
		// If we have no source nodes, we have a cycle in the graph. To print the cycle,
		// we walk starting at all keys We will hit a cycle in this process, however just to check our
		// assumptions, we also verify the the walk returns a CycleError, and if not,
		// return a system error.
		allVisited := make(map[Comp]struct{})
		for _, comp := range g.comps {
			if err := g.edgeVisit(
				comp,
				func(Key, Key) error { return nil },
				newOrderedSet[Comp](),
				allVisited,
			); err != nil {
				return err
			}
		}
		return errors.New("graph had cycle based on source node count being zero, but this was not detected during edge walking")
	case 1:
		return g.edgeVisit(
			sourceComps[0],
			f,
			newOrderedSet[Comp](),
			make(map[Comp]struct{}),
		)
	default:
		allVisited := make(map[Comp]struct{})
		for _, comp := range sourceComps {
			if err := g.edgeVisit(
				comp,
				f,
				newOrderedSet[Comp](),
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
func (g *Graph[Key, Comp]) TopoSort(start Key) ([]Key, error) {
	if err := g.checkInit(); err != nil {
		return nil, err
	}
	results := newOrderedSet[Comp]()
	if err := g.topoVisit(g.getKeyToComp(start), results, newOrderedSet[Comp]()); err != nil {
		return nil, err
	}
	return g.getCompsToKeys(results.comps)
}

// DOTString returns a DOT representation of the graph.
//
// keyToString is used to print out the label for each node.
// https://graphviz.org/doc/info/lang.html
func (g *Graph[Key, Comp]) DOTString(keyToString func(Key) string) (string, error) {
	if err := g.checkInit(); err != nil {
		return "", err
	}
	compToIndex := make(map[Comp]int)
	nextIndex := 1
	var nodeStrings []string
	var edgeStrings []string
	if err := g.WalkEdges(
		func(from Key, to Key) error {
			fromIndex, ok := compToIndex[g.getKeyToComp(from)]
			if !ok {
				fromIndex = nextIndex
				nextIndex++
				compToIndex[g.getKeyToComp(from)] = fromIndex
				nodeStrings = append(
					nodeStrings,
					fmt.Sprintf("%d [label=%q]", fromIndex, keyToString(from)),
				)
			}
			toIndex, ok := compToIndex[g.getKeyToComp(to)]
			if !ok {
				toIndex = nextIndex
				nextIndex++
				compToIndex[g.getKeyToComp(to)] = toIndex
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
			if _, ok := compToIndex[g.getKeyToComp(key)]; ok {
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
			return syserror.Newf("got node %v with %d inbound edges and %d outbound edges, but this was not processed during WalkEdges", key, len(inboundEdges), len(outboundEdges))
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

func (g *Graph[Key, Comp]) checkInit() error {
	// We have to force usage of the constructor as there is no other clean way to get
	// c.getKeyToComp into the struct. Otherwise, we could use an init function for everything,
	// but c.getKeyToComp is required. There is no sensible default.
	//
	// We also do this with ComparableGraph, as otherwise, if we expose Graph as a public
	// inheritance, we can't guarantee that an init function will be called, as even if
	// we wrapped all the public functions specifically for ComparableGraph with init
	// and then c.Graph.Func(), we could not guarantee that these wrapped functions
	// would be called.
	if g.getKeyToComp == nil || g.compToKey == nil || g.compToNode == nil {
		return errors.New("graphs must be constructed with dag.NewGraph or dag.NewComparableGraph")
	}
	return nil
}

func (g *Graph[Key, Comp]) getKeysToComps(keys []Key) []Comp {
	return slicesext.Map(keys, g.getKeyToComp)
}

func (g *Graph[Key, Comp]) getCompsToKeys(comps []Comp) ([]Key, error) {
	return slicesext.MapError(comps, g.getCompToKey)
}

func (g *Graph[Key, Comp]) getCompToKey(comp Comp) (Key, error) {
	key, ok := g.compToKey[comp]
	if !ok {
		// This should never happen.
		return key, fmt.Errorf("comp not present: %v", comp)
	}
	return key, nil
}

func (g *Graph[Key, Comp]) getOrAddNode(key Key) *node[Comp] {
	comp := g.getKeyToComp(key)
	node, ok := g.compToNode[comp]
	if !ok {
		node = newNode[Comp]()
		g.compToKey[comp] = key
		g.compToNode[comp] = node
		g.comps = append(g.comps, comp)
	}
	return node
}

func (g *Graph[Key, Comp]) getSourceComps() ([]Comp, error) {
	var sourceComps []Comp
	// need to get in deterministic order
	for _, comp := range g.comps {
		node, ok := g.compToNode[comp]
		if !ok {
			return nil, fmt.Errorf("key not present: %v", comp)
		}
		if len(node.inboundEdgeMap) == 0 {
			sourceComps = append(sourceComps, comp)
		}
	}
	return sourceComps, nil
}

func (g *Graph[Key, Comp]) edgeVisit(
	from Comp,
	f func(Key, Key) error,
	thisSourceVisited *orderedSet[Comp],
	allSourcesVisited map[Comp]struct{},
) error {
	// this is based on this source. we want to make sure we don't
	// have any cycles based on starting at a single source.
	if !thisSourceVisited.add(from) {
		index := thisSourceVisited.index(from)
		cycle := append(thisSourceVisited.comps[index:], from)
		keys, err := g.getCompsToKeys(cycle)
		if err != nil {
			return err
		}
		return &CycleError[Key]{Keys: keys}
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

	fromNode, ok := g.compToNode[from]
	if !ok {
		return fmt.Errorf("key not present: %v", from)
	}
	fromKey, err := g.getCompToKey(from)
	if err != nil {
		return err
	}
	for _, to := range fromNode.outboundEdges {
		toKey, err := g.getCompToKey(to)
		if err != nil {
			return err
		}
		if err := f(fromKey, toKey); err != nil {
			return err
		}
		if err := g.edgeVisit(to, f, thisSourceVisited.copy(), allSourcesVisited); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graph[Key, Comp]) topoVisit(
	from Comp,
	results *orderedSet[Comp],
	visited *orderedSet[Comp],
) error {
	if !visited.add(from) {
		index := visited.index(from)
		cycle := append(visited.comps[index:], from)
		keys, err := g.getCompsToKeys(cycle)
		if err != nil {
			return err
		}
		return &CycleError[Key]{Keys: keys}
	}

	fromNode, ok := g.compToNode[from]
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

type node[Comp comparable] struct {
	outboundEdgeMap map[Comp]struct{}
	// need to store order for deterministic visits
	outboundEdges  []Comp
	inboundEdgeMap map[Comp]struct{}
	// need to store order for deterministic visits
	inboundEdges []Comp
}

func newNode[Comp comparable]() *node[Comp] {
	return &node[Comp]{
		outboundEdgeMap: make(map[Comp]struct{}),
		inboundEdgeMap:  make(map[Comp]struct{}),
	}
}

func (n *node[Comp]) addOutboundEdge(comp Comp) {
	if _, ok := n.outboundEdgeMap[comp]; !ok {
		n.outboundEdgeMap[comp] = struct{}{}
		n.outboundEdges = append(n.outboundEdges, comp)
	}
}

func (n *node[Comp]) addInboundEdge(comp Comp) {
	if _, ok := n.inboundEdgeMap[comp]; !ok {
		n.inboundEdgeMap[comp] = struct{}{}
		n.inboundEdges = append(n.inboundEdges, comp)
	}
}

type orderedSet[Comp comparable] struct {
	compToIndex map[Comp]int
	comps       []Comp
	length      int
}

func newOrderedSet[Comp comparable]() *orderedSet[Comp] {
	return &orderedSet[Comp]{
		compToIndex: make(map[Comp]int),
	}
}

// returns false if already added
func (s *orderedSet[Comp]) add(comp Comp) bool {
	if _, ok := s.compToIndex[comp]; !ok {
		s.compToIndex[comp] = s.length
		s.comps = append(s.comps, comp)
		s.length++
		return true
	}
	return false
}

func (s *orderedSet[Comp]) copy() *orderedSet[Comp] {
	clone := newOrderedSet[Comp]()
	for _, item := range s.comps {
		clone.add(item)
	}
	return clone
}

func (s *orderedSet[Comp]) index(item Comp) int {
	index, ok := s.compToIndex[item]
	if ok {
		return index
	}
	return -1
}
