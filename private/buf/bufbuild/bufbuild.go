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

// Package bufbuild implements convenience functionality that takes a Module
// and turns it into an Image. This contains logic that used to be in bufwire
// but is abstracted out in a cleaner manner so we can start deprecating bufwire
// over time.
//
// TODO: we could argue that this should replace bufimagebuild.Builder altogether.
package bufbuild

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"go.uber.org/zap"
)

// Builder builds Images from Modules.
type Builder interface {
	Build(
		ctx context.Context,
		modules bufmodule.Module,
		options ...BuildOption,
	) (bufimage.Image, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) Builder {
	return newBuilder(
		logger,
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
