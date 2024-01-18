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

package bufimagebuild

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"go.uber.org/zap"
)

// Builder builds Protobuf files into Images.
type Builder interface {
	// Build runs compilation.
	//
	// The FileRefs are assumed to have been created by a FileRefProvider, that is
	// they are unique relative to the roots.
	//
	// If an error is returned, it is a system error.
	// Only one of Image and FileAnnotations will be returned.
	//
	// FileAnnotations will use external file paths.
	Build(
		ctx context.Context,
		module bufmodule.Module,
		options ...BuildOption,
	) (bufimage.Image, []bufanalysis.FileAnnotation, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(logger *zap.Logger, moduleReader bufmodule.ModuleReader) Builder {
	return newBuilder(logger, moduleReader)
}

// BuildOption is an option for Build.
type BuildOption func(*buildOptions)

// WithExcludeSourceCodeInfo returns a BuildOption that excludes sourceCodeInfo.
func WithExcludeSourceCodeInfo() BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.excludeSourceCodeInfo = true
	}
}

// WithExpectedDirectDependencies sets the module dependencies that are expected, usually because
// they are in a configuration file (buf.yaml). If the build detects that there are direct dependencies
// outside of this list, a warning will be printed.
func WithExpectedDirectDependencies(expectedDirectDependencies []bufmoduleref.ModuleReference) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.expectedDirectDependencies = expectedDirectDependencies
	}
}

// WithWorkspace sets the workspace to be read from instead of ModuleReader, and to not warn imports for.
//
// TODO: this can probably be dealt with by finding out if an ImageFile has a commit
// or not, although that is hacky, that's an implementation detail in practice, but perhaps
// we could justify it - transitive dependencies without commits don't make sense?
//
// TODO: shouldn't buf.yamls in workspaces have deps properly declared in them anyways? Why not warn?
func WithWorkspace(workspace bufmodule.Workspace) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.workspace = workspace
	}
}
