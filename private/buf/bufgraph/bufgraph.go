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

package bufgraph

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/dag"
	"go.uber.org/zap"
)

// Node is a node in a dependency graph.
//
// This is effectively the same as a ModuleIdentityOptionalCommit, however
// ModuleIdentityOptionalCommits are not comparable, and the current
// implementation of *dag.Graph requires comparable keys.
//
// TODO: Don't have the duplication across Node and ModuleIdentityOptionalCommit.
// TODO: deal with the case where there are differing commits for a given ModuleIdentity.
type Node struct {
	// required
	Remote string
	// required
	Owner string
	// required
	Repository string
	// optional
	Commit string
}

// String implements fmt.Stringer.
func (n *Node) String() string {
	s := n.Remote + "/" + n.Owner + "/" + n.Repository
	if n.Commit == "" {
		return s
	}
	return s + ":" + n.Commit
}

// Builder builds dependency graphs.
type Builder interface {
	Build(
		ctx context.Context,
		modules []bufmodule.Module,
		options ...BuildOption,
	) (*dag.Graph[Node], error)
}

// NewBuilder returns a new Builder.
func NewBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
	moduleResolver bufmodule.ModuleResolver,
) Builder {
	return newBuilder(
		logger,
		moduleReader,
		moduleResolver,
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
