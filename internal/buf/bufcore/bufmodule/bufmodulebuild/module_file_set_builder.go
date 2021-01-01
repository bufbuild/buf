// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmodulebuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"go.uber.org/zap"
)

type moduleFileSetBuilder struct {
	logger       *zap.Logger
	moduleReader bufmodule.ModuleReader
}

func newModuleFileSetBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) *moduleFileSetBuilder {
	return &moduleFileSetBuilder{
		logger:       logger,
		moduleReader: moduleReader,
	}
}
func (m *moduleFileSetBuilder) Build(
	ctx context.Context,
	module bufmodule.Module,
) (bufmodule.ModuleFileSet, error) {
	var dependencyModules []bufmodule.Module
	// we know these are unique by remote, owner, repository
	// these also contain all transitive dependencies
	for _, dependnecyModulePin := range module.DependencyModulePins() {
		dependencyModule, err := m.moduleReader.GetModule(ctx, dependnecyModulePin)
		if err != nil {
			return nil, err
		}
		dependencyModules = append(dependencyModules, dependencyModule)
	}
	return bufmodule.NewModuleFileSet(module, dependencyModules), nil
}
