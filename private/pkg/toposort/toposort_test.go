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
	"strings"
	"testing"
)

func TestTopoSort(t *testing.T) {
	t.Parallel()

	graph := &Graph[string]{}

	// a -> b -> c
	err := graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("b", "c")
	if err != nil {
		t.Error(err)
		return
	}

	results, err := graph.TopoSort("a")
	if err != nil {
		t.Error(err)
		return
	}
	if results[0] != "c" || results[1] != "b" || results[2] != "a" {
		t.Errorf("Wrong sort order: %v", results)
	}
}

func TestTopoSort2(t *testing.T) {
	t.Parallel()

	graph := &Graph[string]{}

	// a -> c
	// a -> b
	// b -> c
	err := graph.AddEdge("a", "c")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("b", "c")
	if err != nil {
		t.Error(err)
		return
	}

	results, err := graph.TopoSort("a")
	if err != nil {
		t.Error(err)
		return
	}
	if results[0] != "c" || results[1] != "b" || results[2] != "a" {
		t.Errorf("Wrong sort order: %v", results)
	}
}

func TestTopoSort3(t *testing.T) {
	t.Parallel()

	graph := &Graph[string]{}

	// a -> b
	// a -> d
	// d -> c
	// c -> b
	err := graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("a", "d")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("d", "c")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("c", "b")
	if err != nil {
		t.Error(err)
		return
	}

	results, err := graph.TopoSort("a")
	if err != nil {
		t.Error(err)
		return
	}
	if len(results) != 4 {
		t.Errorf("Wrong number of results: %v", results)
		return
	}
	expected := [4]string{"b", "c", "d", "a"}
	for i := 0; i < 4; i++ {
		if results[i] != expected[i] {
			t.Errorf("Wrong sort order: %v", results)
			break
		}
	}
}

func TestTopoSortCycleError(t *testing.T) {
	t.Parallel()

	graph := &Graph[string]{}

	// a -> b
	// b -> a
	err := graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("b", "a")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = graph.TopoSort("a")
	if err == nil {
		t.Errorf("Expected cycle error")
		return
	}
	if !strings.Contains(err.Error(), "a -> b -> a") {
		t.Errorf("Error doesn't print cycle: %q", err)
	}
}

func TestTopoSortCycleError2(t *testing.T) {
	graph := &Graph[string]{}

	// a -> b
	// b -> c
	// c -> a
	err := graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("b", "c")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("c", "a")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = graph.TopoSort("a")
	if err == nil {
		t.Errorf("Expected cycle error")
		return
	}
	if !strings.Contains(err.Error(), "a -> b -> c -> a") {
		t.Errorf("Error doesn't print cycle: %q", err)
	}
}

func TestTopoSortCycleError3(t *testing.T) {
	t.Parallel()

	graph := &Graph[string]{}

	// a -> b
	// b -> c
	// c -> b
	err := graph.AddEdge("a", "b")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("b", "c")
	if err != nil {
		t.Error(err)
		return
	}
	err = graph.AddEdge("c", "b")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = graph.TopoSort("a")
	if err == nil {
		t.Errorf("Expected cycle error")
		return
	}
	if !strings.Contains(err.Error(), "b -> c -> b") {
		t.Errorf("Error doesn't print cycle: %q", err)
	}
}
