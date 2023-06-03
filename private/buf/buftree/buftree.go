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

package buftree

import (
	"context"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/dag"
)

// DependencyGraph builds the dependency graph.
func DependencyGraph(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageConfigReader bufwire.ImageConfigReader,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
) (*dag.Graph[string], error) {
	return nil, nil
}
