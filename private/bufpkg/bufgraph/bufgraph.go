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

package bufgraph

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/dag"
	"go.uber.org/zap"
)

// Node is a node in a dependency graph.
//
// This is a struct because this needs to be comparable for the *dag.Graph.
//
// TODO: Don't have the duplication across Node and ImageModuleDependency.
type Node struct {
	// Required,
	Remote string
	// Required.
	Owner string
	// Required.
	Repository string
	// Optional. Will not bet set for modules read from workspaces.
	Commit string
}

// IdentityString prints remote/owner/repository.
func (n *Node) IdentityString() string {
	return n.Remote + "/" + n.Owner + "/" + n.Repository
}

// String prints remote/owner/repository[:commit].
func (n *Node) String() string {
	s := n.IdentityString()
	if n.Commit != "" {
		return s + ":" + n.Commit
	}
	return s
}

// Builder builds dependency graphs.
type Builder interface {
	// Build builds the dependency graph.
	Build(
		ctx context.Context,
		modules []bufmodule.Module,
		options ...BuildOption,
	) (*dag.Graph[Node], []bufanalysis.FileAnnotation, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(
	logger *zap.Logger,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) Builder {
	return newBuilder(
		logger,
		moduleResolver,
		moduleReader,
	)
}

// BuildOption is an option for Build.
type BuildOption func(*buildOptions)

// BuildWithWorkspace returns a new BuildOption that specifies a workspace
// that is being operated on.
func BuildWithWorkspace(workspace bufmodule.Workspace) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.workspace = workspace
	}
}
