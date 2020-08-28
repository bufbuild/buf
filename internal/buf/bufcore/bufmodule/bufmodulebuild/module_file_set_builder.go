// Copyright 2020 Buf Technologies, Inc.
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
	var dependencies []bufmodule.Module
	if len(module.Dependencies()) > 0 {
		var err error
		dependencies, err = m.getModules(ctx, make(map[string]struct{}), module.Dependencies())
		if err != nil {
			return nil, err
		}
	}
	return bufmodule.NewModuleFileSet(module, dependencies), nil
}

func (m *moduleFileSetBuilder) getModules(
	ctx context.Context,
	seenModules map[string]struct{},
	moduleNames []bufmodule.ModuleName,
) ([]bufmodule.Module, error) {
	var modules []bufmodule.Module
	for _, moduleName := range moduleNames {
		// Avoid pulling module more than once
		if _, ok := seenModules[moduleName.String()]; ok {
			continue
		}
		seenModules[moduleName.String()] = struct{}{}
		module, err := m.moduleReader.GetModule(ctx, moduleName)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
		if len(module.Dependencies()) > 0 {
			dependencies, err := m.getModules(ctx, seenModules, module.Dependencies())
			if err != nil {
				return nil, err
			}
			modules = append(modules, dependencies...)
		}
	}
	return modules, nil
}
