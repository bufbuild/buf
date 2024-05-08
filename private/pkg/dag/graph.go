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
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// Graph is a directed acyclic graph structure.
type Graph[Key comparable, Value any] struct {
	getKeyForValue func(Value) Key
	keyToValue     map[Key]Value
	keyToNode      map[Key]*node[Key]
	// need to store order so that we can create a deterministic CycleError
	// in the case of Walk where we have no source nodes, so that we can Walk
	// deterministically and find the cycle.
	keys []Key
}

// NewGraph returns a new Graph for an any Value.
//
// The toKey function must convert a Value to a unique comparable key type.
// It is up to the caller to make sure that the key is unique on a per-value basis.
//
// This constructor must be used when initializing a Graph.
//
// TODO FUTURE: It really stinks that we have to use the constructor. We have what amounts
// to silent errors now below with functions that don't return an error. We should
// figure out a way to do this properly, or perhaps just panic if we don't use the constructor.
func NewGraph[Key comparable, Value any](toKey func(Value) Key) *Graph[Key, Value] {
	return &Graph[Key, Value]{
		getKeyForValue: toKey,
		keyToValue:     make(map[Key]Value),
		keyToNode:      make(map[Key]*node[Key]),
	}
}

// AddNode adds a node.
func (g *Graph[Key, Value]) AddNode(value Value) {
	if err := g.checkInit(); err != nil {
		return
	}
	g.getOrAddNode(value)
}

// AddEdge adds an edge.
func (g *Graph[Key, Value]) AddEdge(from Value, to Value) {
	if err := g.checkInit(); err != nil {
		return
	}
	fromNode := g.getOrAddNode(from)
	toNode := g.getOrAddNode(to)
	fromNode.addOutboundEdge(g.getKeyForValue(to))
	toNode.addInboundEdge(g.getKeyForValue(from))
}

// ContainsNode returns true if the graph contains the given node.
func (g *Graph[Key, Value]) ContainsNode(key Key) bool {
	if err := g.checkInit(); err != nil {
		return false
	}
	_, ok := g.keyToNode[key]
	return ok
}

// NumNodes returns the number of nodes in the graph.
func (g *Graph[Key, Value]) NumNodes() int {
	if err := g.checkInit(); err != nil {
		return 0
	}
	return len(g.keys)
}

// NumNodes returns the number of edges in the graph.
func (g *Graph[Key, Value]) NumEdges() int {
	if err := g.checkInit(); err != nil {
		return 0
	}
	var numEdges int
	for _, node := range g.keyToNode {
		numEdges += len(node.outboundEdges)
	}
	return numEdges
}

// InboundNodes returns the nodes that are inbound to the node for the key.
//
// Returns error if there is no node for the key
func (g *Graph[Key, Value]) InboundNodes(key Key) ([]Value, error) {
	if err := g.checkInit(); err != nil {
		return nil, err
	}
	node, ok := g.keyToNode[key]
	if !ok {
		return nil, fmt.Errorf("key not present: %v", key)
	}
	return g.getValuesForKeys(node.inboundEdges)
}

// OutboundNodes returns the nodes that are outbound from the node for the key.
//
// Returns error if there is no node for the key
func (g *Graph[Key, Value]) OutboundNodes(key Key) ([]Value, error) {
	if err := g.checkInit(); err != nil {
		return nil, err
	}
	node, ok := g.keyToNode[key]
	if !ok {
		return nil, fmt.Errorf("key not present: %v", key)
	}
	return g.getValuesForKeys(node.outboundEdges)
}

// WalkNodes visited each node in the Graph based on insertion order.
//
// f is called for each node. The first argument is the key for the node,
// the second argument is all inbound edges, the third argument
// is all outbound edges.
func (g *Graph[Key, Value]) WalkNodes(f func(Value, []Value, []Value) error) error {
	if err := g.checkInit(); err != nil {
		return err
	}
	for _, key := range g.keys {
		node, ok := g.keyToNode[key]
		if !ok {
			return fmt.Errorf("key not present: %v", key)
		}
		value, err := g.getValueForKey(key)
		if err != nil {
			return err
		}
		inboundValues, err := g.getValuesForKeys(node.inboundEdges)
		if err != nil {
			return err
		}
		outboundValues, err := g.getValuesForKeys(node.outboundEdges)
		if err != nil {
			return err
		}
		if err := f(value, inboundValues, outboundValues); err != nil {
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
func (g *Graph[Key, Value]) WalkEdges(f func(Value, Value) error) error {
	if err := g.checkInit(); err != nil {
		return err
	}
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
				func(Value, Value) error { return nil },
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
func (g *Graph[Key, Value]) TopoSort(start Key) ([]Value, error) {
	if err := g.checkInit(); err != nil {
		return nil, err
	}
	results := newOrderedSet[Key]()
	if err := g.topoVisit(start, results, newOrderedSet[Key]()); err != nil {
		return nil, err
	}
	return g.getValuesForKeys(results.keys)
}

// DOTString returns a DOT representation of the graph.
//
// valueToString is used to print out the label for each node.
//
// https://graphviz.org/doc/info/lang.html
func (g *Graph[Key, Value]) DOTString(valueToString func(Value) string) (string, error) {
	if err := g.checkInit(); err != nil {
		return "", err
	}
	var edgeStrings []string
	seenKeys := make(map[Key]struct{})
	if err := g.WalkEdges(
		func(from Value, to Value) error {
			seenKeys[g.getKeyForValue(from)] = struct{}{}
			seenKeys[g.getKeyForValue(to)] = struct{}{}
			fromName, err := xmlEscape(valueToString(from))
			if err != nil {
				return err
			}
			toName, err := xmlEscape(valueToString(to))
			if err != nil {
				return err
			}
			edgeStrings = append(edgeStrings, fmt.Sprintf("%q -> %q", fromName, toName))
			return nil
		},
	); err != nil {
		return "", err
	}
	// We also want to pick up any nodes that do not have edges, and display them.
	if err := g.WalkNodes(
		func(value Value, inboundEdges []Value, outboundEdges []Value) error {
			key := g.getKeyForValue(value)
			if _, ok := seenKeys[key]; ok {
				return nil
			}
			seenKeys[key] = struct{}{}
			if len(inboundEdges) == 0 && len(outboundEdges) == 0 {
				name, err := xmlEscape(valueToString(value))
				if err != nil {
					return err
				}
				edgeStrings = append(edgeStrings, fmt.Sprintf("%q", name))
				return nil
			}
			// This is a system error.
			return syserror.Newf(
				"got node %v with %d inbound edges and %d outbound edges, but this was not processed during WalkEdges",
				value,
				len(inboundEdges),
				len(outboundEdges),
			)
		},
	); err != nil {
		return "", err
	}
	if len(edgeStrings) == 0 {
		return "digraph {}", nil
	}
	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString("digraph {\n\n")
	for _, edgeString := range edgeStrings {
		_, _ = buffer.WriteString("  ")
		_, _ = buffer.WriteString(edgeString)
		_, _ = buffer.WriteString("\n")
	}
	_, _ = buffer.WriteString("\n}")
	return buffer.String(), nil
}

// *** PRIVATE ***

func (g *Graph[Key, Value]) checkInit() error {
	// We have to force usage of the constructor as there is no other clean way to get
	// c.getKeyForValue into the struct. Otherwise, we could use an init function for everything,
	// but c.getKeyForValue is required. There is no sensible default.
	//
	// We also do this with ComparableGraph, as otherwise, if we expose Graph as a public
	// inheritance, we can't guarantee that an init function will be called, as even if
	// we wrapped all the public functions specifically for ComparableGraph with init
	// and then c.Graph.Func(), we could not guarantee that these wrapped functions
	// would be called.
	if g.getKeyForValue == nil || g.keyToValue == nil || g.keyToNode == nil {
		return errors.New("graphs must be constructed with dag.NewGraph or dag.NewComparableGraph")
	}
	return nil
}

func (g *Graph[Key, Value]) getValuesForKeys(keys []Key) ([]Value, error) {
	return slicesext.MapError(keys, g.getValueForKey)
}

func (g *Graph[Key, Value]) getValueForKey(key Key) (Value, error) {
	value, ok := g.keyToValue[key]
	if !ok {
		// This should never happen.
		return value, fmt.Errorf("key not present: %v", key)
	}
	return value, nil
}

func (g *Graph[Key, Value]) getOrAddNode(value Value) *node[Key] {
	key := g.getKeyForValue(value)
	node, ok := g.keyToNode[key]
	if !ok {
		node = newNode[Key]()
		g.keyToValue[key] = value
		g.keyToNode[key] = node
		g.keys = append(g.keys, key)
	}
	return node
}

func (g *Graph[Key, Value]) getSourceKeys() ([]Key, error) {
	var sourceKeys []Key
	// need to get in deterministic order
	for _, key := range g.keys {
		node, ok := g.keyToNode[key]
		if !ok {
			return nil, fmt.Errorf("key not present: %v", key)
		}
		if len(node.inboundEdgeMap) == 0 {
			sourceKeys = append(sourceKeys, key)
		}
	}
	return sourceKeys, nil
}

func (g *Graph[Key, Value]) edgeVisit(
	from Key,
	f func(Value, Value) error,
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
	fromValue, err := g.getValueForKey(from)
	if err != nil {
		return err
	}
	for _, to := range fromNode.outboundEdges {
		toValue, err := g.getValueForKey(to)
		if err != nil {
			return err
		}
		if err := f(fromValue, toValue); err != nil {
			return err
		}
		if err := g.edgeVisit(to, f, thisSourceVisited.copy(), allSourcesVisited); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graph[Key, Value]) topoVisit(
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

func xmlEscape(s string) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := xml.EscapeText(buffer, []byte(s)); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
