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

// ComparableGraph is a Graph that uses comparable Values.
type ComparableGraph[Value comparable] struct {
	graph *Graph[Value, Value]
}

// NewComparableGraph returns a new ComparableGraph for a comparable Value.
//
// It is safe to initialize a ComparableGraph with &dag.ComparableGraph[value]{}.
//
// Do not use interfaces as Values! If your Value type is an interface, use NewGraph.
func NewComparableGraph[Value comparable]() *ComparableGraph[Value] {
	comparableGraph := &ComparableGraph[Value]{}
	comparableGraph.init()
	return comparableGraph
}

// AddNode adds a node.
func (g *ComparableGraph[Value]) AddNode(value Value) {
	g.Graph().AddNode(value)
}

// AddEdge adds an edge.
func (g *ComparableGraph[Value]) AddEdge(from Value, to Value) {
	g.Graph().AddEdge(from, to)
}

// ContainsNode returns true if the graph contains the given node.
func (g *ComparableGraph[Value]) ContainsNode(key Value) bool {
	return g.Graph().ContainsNode(key)
}

// NumNodes returns the number of nodes in the graph.
func (g *ComparableGraph[Value]) NumNodes() int {
	return g.Graph().NumNodes()
}

// NumNodes returns the number of edges in the graph.
func (g *ComparableGraph[Value]) NumEdges() int {
	return g.Graph().NumEdges()
}

// OutboundNodes returns the nodes that are outbound from the node for the key.
//
// Returns error if there is no node for the key
func (g *ComparableGraph[Value]) OutboundNodes(key Value) ([]Value, error) {
	return g.Graph().OutboundNodes(key)
}

// InboundNodes returns the nodes that are inbound to the node for the key.
//
// Returns error if there is no node for the key
func (g *ComparableGraph[Value]) InboundNodes(key Value) ([]Value, error) {
	return g.Graph().InboundNodes(key)
}

// WalkNodes visited each node in the Graph based on insertion order.
//
// f is called for each node. The first argument is the key for the node,
// the second argument is all inbound edges, the third argument
// is all outbound edges.
func (g *ComparableGraph[Value]) WalkNodes(f func(Value, []Value, []Value) error) error {
	return g.Graph().WalkNodes(f)
}

// WalkEdges visits each edge in the Graph starting at the source keys.
//
// f is called for each directed edge. The first argument is the source
// node, the second is the destination node.
//
// Returns a *CycleError if there is a cycle in the graph.
func (g *ComparableGraph[Value]) WalkEdges(f func(Value, Value) error) error {
	return g.Graph().WalkEdges(f)
}

// TopoSort topologically sorts the nodes in the Graph starting at the given key.
//
// Returns a *CycleError if there is a cycle in the graph.
func (g *ComparableGraph[Value]) TopoSort(start Value) ([]Value, error) {
	return g.Graph().TopoSort(start)
}

// DOTString returns a DOT representation of the graph.
//
// valueToString is used to print out the label for each node.
//
// https://graphviz.org/doc/info/lang.html
func (g *ComparableGraph[Value]) DOTString(valueToString func(Value) string) (string, error) {
	return g.Graph().DOTString(valueToString)
}

// Graph returns the underlying Graph that backs the ComparableGraph.
//
// Used for functions that need a Graph instead of a ComparableGraph.
func (g *ComparableGraph[Value]) Graph() *Graph[Value, Value] {
	g.init()
	return g.graph
}

// *** PRIVATE ***

func (g *ComparableGraph[Value]) init() {
	if g.graph == nil {
		g.graph = NewGraph[Value, Value](func(value Value) Value { return value })
	}
}
