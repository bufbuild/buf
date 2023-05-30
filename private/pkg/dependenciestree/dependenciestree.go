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

// A dependencies tree implements helpers to better handle nodes depending on each other. For
// example, for a workspace module that wants to be pushed or synced, the order in which we iterate
// the modules is important:
//
// - module a depends on b
// - module b depends on c
//
// We should iterate in the order [c, b, a].
//
// This is also useful for managed modules sync jobs in the BSR. Some of the third-party modules
// depend on each other, so we need to make sure that the dependencies modules sync first, before
// syncing the parent modules.
//
// The dependencies tree is one level deep, and contain multiple root nodes. The implementation
// protects the tree against circular dependencies. Some examples:
//
// Valid tree:
// - a: [b, c, d]
// - b: [c, d]
// - c: none
// - d: none
//
// Invalid tree:
// - a: [a] // dependency on self, circular dependency: a.a
// - b: [c]
// - c: [d]
// - d: [b] // circular dependency: b.c.d.b

package dependenciestree

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/stringutil"
)

// DependenciesTree is a tree that holds nodes depending on other nodes. One level deep.
type DependenciesTree interface {
	// NewRootNode adds a new node at the root of the tree, with its dependencies. Both root node name
	// and dependencies names need to have values. Adding root nodes with invalid dependencies
	// (external or circular) is possible, tree should be checked against the Validate func.
	NewRootNode(rootNode string, dependencies map[string]struct{}) error
	// Validate checks that the tree nodes have valid dependencies, with no circular dependencies.
	Validate() error
	// Sort validates that the tree is valid, and then returns the tree in topological order, leafs
	// first. Useful for installation or update order.
	Sort() ([]string, error)
}

// New initializes a new dependencies tree, ready to have root nodes added to it.
func New(opts ...NewDependenciesTreeOption) DependenciesTree {
	var config newDependenciesTreeOptions
	for _, opt := range opts {
		opt(&config)
	}
	return &depsTree{
		tree:                      make(map[string]map[string]struct{}),
		allowExternalDependencies: config.allowExternalDependencies,
	}
}

// NewDependenciesTreeOption are options that can be passed when instantiating a new dependencies tree.
type NewDependenciesTreeOption func(*newDependenciesTreeOptions)

// NewDependenciesTreeWithAllowExternalDeps allows a root node in the tree to depend on external
// dependencies, not present in other root nodes.
func NewDependenciesTreeWithAllowExternalDeps() NewDependenciesTreeOption {
	return func(opts *newDependenciesTreeOptions) {
		opts.allowExternalDependencies = true
	}
}

func (t *depsTree) NewRootNode(rootNode string, dependencies map[string]struct{}) error {
	if rootNode == "" {
		return errors.New("empty root node")
	}
	if _, ok := t.tree[rootNode]; !ok {
		t.tree[rootNode] = make(map[string]struct{})
	}
	for dep := range dependencies {
		if dep == "" {
			return errors.New("empty dependency node")
		}
		t.tree[rootNode][dep] = struct{}{}
	}
	return nil
}

func (t *depsTree) Validate() error {
	for parent := range t.tree {
		if err := t.validateDependencies(parent, []string{parent}); err != nil {
			return err
		}
	}
	return nil
}

func (t *depsTree) Sort() ([]string, error) {
	if len(t.tree) == 0 {
		return nil, nil
	}
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("cannot sort, invalid tree: %w", err)
	}
	pendingRootNodes := make(map[string]struct{}, len(t.tree))
	for node := range t.tree {
		pendingRootNodes[node] = struct{}{}
	}
	return t.sort(pendingRootNodes, nil)
}

type newDependenciesTreeOptions struct {
	allowExternalDependencies bool
}

type depsTree struct {
	tree                      map[string]map[string]struct{}
	allowExternalDependencies bool
}

func (t *depsTree) validateDependencies(
	node string,
	parents []string,
) error {
	if node == "" {
		// this shouldn't happen, nodes are only added via NewRootNode which checks this, but let's
		// protect it here as well.
		return errors.New("empty node")
	}
	nodeDeps, nodeInRoot := t.tree[node]
	if !nodeInRoot && !t.allowExternalDependencies {
		return fmt.Errorf("node %s is not present in tree root nodes", strings.Join(parents, "."))
	}
	for dep := range nodeDeps {
		for _, parent := range parents {
			if parent == dep {
				return fmt.Errorf("circular dependency: %s.%s", strings.Join(parents, "."), dep)
			}
		}
		if err := t.validateDependencies(dep, append(parents, dep)); err != nil {
			return err
		}
	}
	return nil
}

func (t *depsTree) sort(
	pendingNodes map[string]struct{},
	sortedNodes []string,
) ([]string, error) {
	if len(pendingNodes) == 0 {
		// no more pending root nodes to sort
		return sortedNodes, nil
	}
	iterationNodes := make([]string, 0)
	for node, deps := range t.tree {
		if _, pending := pendingNodes[node]; !pending {
			// node was already sorted
			continue
		}
		if len(deps) == 0 {
			// node has no deps, it's a leaf, can be sorted now
			iterationNodes = append(iterationNodes, node)
			continue
		}
		var anyDependencyMissing bool
		for dep := range deps {
			if _, pending := pendingNodes[dep]; pending {
				anyDependencyMissing = true
				break
			}
		}
		if anyDependencyMissing {
			// node cannot be sorted this iteration, still has missing dependencies
			continue
		}
		// all node deps, have been sorted already, node can be sorted now
		iterationNodes = append(iterationNodes, node)
	}
	if len(iterationNodes) == 0 {
		// finished the iteration w/out sorting anything, this shouldn't happen, let's break out of
		// infinite loop
		pending := stringutil.MapToSlice(pendingNodes)
		sort.Strings(pending)
		return nil, fmt.Errorf(
			"cannot determine next node: sorted so far: [%s], pending [%s], complete tree: %v",
			strings.Join(sortedNodes, ", "), pending, t.tree,
		)
	}
	// now that the iteration is complete, we can clear those nodes from the pending map
	for _, iterationNode := range iterationNodes {
		delete(pendingNodes, iterationNode)
	}
	// sort iteration nodes lexicographically, so result is deterministic
	sort.Strings(iterationNodes)
	// add iteration nodes to all sorted nodes
	sortedNodes = append(sortedNodes, iterationNodes...)
	return t.sort(pendingNodes, sortedNodes)
}
