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

package dependenciestree_test

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/dependenciestree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootNode(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                    string
		rootNode                string
		deps                    map[string]struct{}
		expectedErrMessageExact string
	}
	testCases := []testCase{
		{
			name:     "no_deps",
			rootNode: "foo",
		},
		{
			name:     "single_dep",
			rootNode: "foo",
			deps:     map[string]struct{}{"bar": {}},
		},
		{
			name:     "multiple_deps",
			rootNode: "foo",
			deps: map[string]struct{}{
				"bar": {},
				"baz": {},
			},
		},
		{
			name:                    "empty_root_node",
			expectedErrMessageExact: "empty root node",
		},
		{
			name:                    "empty_deps",
			rootNode:                "foo",
			deps:                    map[string]struct{}{"": {}},
			expectedErrMessageExact: "empty dependency node",
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tree := dependenciestree.New()
				err := tree.NewRootNode(tc.rootNode, tc.deps)
				if tc.expectedErrMessageExact == "" {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, tc.expectedErrMessageExact)
				}
			})
		}(tc)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                       string
		tree                       map[string]map[string]struct{}
		allowExternalDependencies  bool
		expectedErrMessageExact    string
		expectedErrMessageContains []string
	}
	testCases := []testCase{
		{
			name: "nil",
		},
		{
			name: "empty",
			tree: make(map[string]map[string]struct{}),
		},
		{
			name: "single_node",
			tree: map[string]map[string]struct{}{
				"foo": {},
			},
		},
		{
			name: "multiple_nodes_no_deps",
			tree: map[string]map[string]struct{}{
				"foo": {},
				"bar": {},
				"baz": {},
			},
		},
		{
			name: "single_dep",
			tree: map[string]map[string]struct{}{
				"foo": {"bar": {}},
				"bar": {},
			},
		},
		{
			name: "multilevel_deps",
			tree: map[string]map[string]struct{}{
				"foo": {"bar": {}},
				"bar": {"baz": {}},
				"baz": {},
			},
		},
		{
			name: "external_deps_not_allowed",
			tree: map[string]map[string]struct{}{
				"foo": {"external_dep": {}},
			},
			expectedErrMessageExact: "node foo.external_dep is not present in tree root nodes",
		},
		{
			name: "external_deps_allowed",
			tree: map[string]map[string]struct{}{
				"foo": {"external_dep": {}},
			},
			allowExternalDependencies: true,
		},
		{
			name: "depend_on_self",
			tree: map[string]map[string]struct{}{
				"foo": {"foo": {}},
			},
			expectedErrMessageExact: "circular dependency: foo.foo",
		},
		{
			name: "circular_dependency",
			tree: map[string]map[string]struct{}{
				"foo": {"bar": {}},
				"bar": {"foo": {}},
			},
			expectedErrMessageContains: []string{ // map sorting is not deterministic
				"circular dependency: ",
				"foo.bar",
				"bar.foo",
			},
		},
		{
			name: "multilevel_circular_dependency",
			tree: map[string]map[string]struct{}{
				"foo": {"bar": {}},
				"bar": {"baz": {}},
				"baz": {"foo": {}},
			},
			expectedErrMessageContains: []string{ // map sorting is not deterministic
				"circular dependency: ",
				"foo.bar",
				"bar.baz",
				"baz.foo",
			},
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				newTreeDepsOpts := make([]dependenciestree.NewDependenciesTreeOption, 0)
				if tc.allowExternalDependencies {
					newTreeDepsOpts = append(newTreeDepsOpts, dependenciestree.NewDependenciesTreeWithAllowExternalDeps())
				}
				tree := dependenciestree.New(newTreeDepsOpts...)
				for node, deps := range tc.tree {
					require.NoError(t, tree.NewRootNode(node, deps))
				}
				validateErr := tree.Validate()
				var expectErr bool
				if tc.expectedErrMessageExact != "" {
					expectErr = true
					assert.EqualError(t, validateErr, tc.expectedErrMessageExact)
				}
				if len(tc.expectedErrMessageContains) > 0 {
					expectErr = true
					for _, partialErrMsg := range tc.expectedErrMessageContains {
						assert.Contains(t, validateErr.Error(), partialErrMsg)
					}
				}
				if expectErr {
					assert.Error(t, validateErr)
				} else {
					assert.NoError(t, validateErr)
				}
			})
		}(tc)
	}
}

func TestSort(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                      string
		tree                      map[string]map[string]struct{}
		expectedSortedNodes       []string
		allowExternalDependencies bool
	}
	testCases := []testCase{
		{
			name: "nil",
		},
		{
			name: "empty",
			tree: make(map[string]map[string]struct{}),
		},
		{
			name: "single_node",
			tree: map[string]map[string]struct{}{
				"foo": {},
			},
			expectedSortedNodes: []string{"foo"},
		},
		{
			name: "multiple_nodes_no_deps",
			tree: map[string]map[string]struct{}{
				"foo": {},
				"bar": {},
				"baz": {},
			},
			expectedSortedNodes: []string{
				"bar",
				"baz",
				"foo",
			},
		},
		{
			name: "single_dep",
			tree: map[string]map[string]struct{}{
				"foo": {"bar": {}},
				"bar": {},
			},
			expectedSortedNodes: []string{
				"bar",
				"foo", // depends on bar
			},
		},
		{
			name: "external_deps",
			tree: map[string]map[string]struct{}{
				"foo": {"external_dep": {}},
			},
			allowExternalDependencies: true,
			expectedSortedNodes:       []string{"foo"}, // does not include external deps
		},
		{
			name: "multilevel_deps",
			tree: map[string]map[string]struct{}{
				// leafs
				"z1": {},
				"z2": {"external_dep": {}},
				"z3": {},
				// depends on z
				"y1": {"z1": {}},
				"y2": {
					"z1":           {},
					"z2":           {},
					"external_dep": {},
				},
				// depends on y
				"x1": {
					"y1": {},
					"z3": {},
				},
				// depends on x
				"w1": {
					"x1":           {},
					"y1":           {},
					"external_dep": {},
				},
			},
			allowExternalDependencies: true,
			expectedSortedNodes: []string{
				"z1",
				"z2",
				"z3",
				"y1",
				"y2",
				"x1",
				"w1",
			},
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				newTreeDepsOpts := make([]dependenciestree.NewDependenciesTreeOption, 0)
				if tc.allowExternalDependencies {
					newTreeDepsOpts = append(newTreeDepsOpts, dependenciestree.NewDependenciesTreeWithAllowExternalDeps())
				}
				tree := dependenciestree.New(newTreeDepsOpts...)
				for node, deps := range tc.tree {
					require.NoError(t, tree.NewRootNode(node, deps))
				}
				sortedNodes, err := tree.Sort()
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSortedNodes, sortedNodes)
			})
		}(tc)
	}
}
