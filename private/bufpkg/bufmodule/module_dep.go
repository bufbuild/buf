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

package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// ModuleDep is the dependency of a Module.
//
// It's just a Module as well as whether or not the dependency is direct.
type ModuleDep interface {
	Module

	// Parent returns the Module that this ModuleDep is a dependency of.
	//
	// Note this is not recursive - this points ot the top-level Module that dependencies
	// were created for. That is, if a -> b -> c, then a will have ModuleDeps b and c, both
	// of which have a as a parent.
	Parent() Module
	// IsDirect returns true if the Module is a direct dependency of this Module.
	IsDirect() bool

	isModuleDep()
}

// *** PRIVATE ***

type moduleDep struct {
	Module

	parent   Module
	isDirect bool
}

func newModuleDep(
	module Module,
	parent Module,
	isDirect bool,
) *moduleDep {
	return &moduleDep{
		Module:   module,
		parent:   parent,
		isDirect: isDirect,
	}
}

func (m *moduleDep) Parent() Module {
	return m.parent
}

func (m *moduleDep) IsDirect() bool {
	return m.isDirect
}

func (*moduleDep) isModuleDep() {}

// getModuleDeps gets the actual dependencies for the Module.
func getModuleDeps(
	ctx context.Context,
	logger *zap.Logger,
	module Module,
) ([]ModuleDep, error) {
	depOpaqueIDToModuleDep := make(map[string]ModuleDep)
	if err := getModuleDepsRec(
		ctx,
		logger,
		module,
		module,
		make(map[string]struct{}),
		depOpaqueIDToModuleDep,
		true,
	); err != nil {
		return nil, err
	}
	moduleDeps := make([]ModuleDep, 0, len(depOpaqueIDToModuleDep))
	for _, moduleDep := range depOpaqueIDToModuleDep {
		moduleDeps = append(moduleDeps, moduleDep)
	}
	// Sorting by at least Opaque ID to get a consistent return order for a given call.
	sort.Slice(
		moduleDeps,
		func(i int, j int) bool {
			return moduleDeps[i].OpaqueID() < moduleDeps[j].OpaqueID()
		},
	)
	return moduleDeps, nil
}

func getModuleDepsRec(
	ctx context.Context,
	logger *zap.Logger,
	module Module,
	parentModule Module,
	visitedOpaqueIDs map[string]struct{},
	// already discovered deps
	depOpaqueIDToModuleDep map[string]ModuleDep,
	isDirect bool,
) error {
	opaqueID := module.OpaqueID()
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		// TODO: detect cycles, this is just making sure we don't recurse
		return nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	moduleSet := module.ModuleSet()
	if moduleSet == nil {
		// This should never happen.
		return syserror.New("moduleSet never set on module")
	}
	// Doing this BFS so we add all the direct deps to the map first, then if we
	// see a dep later, it will still be a direct dep in the map, but will be ignored
	// on recursive calls.
	var newModuleDeps []ModuleDep
	if err := module.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if fileInfo.FileType() != FileTypeProto {
				return nil
			}
			fastscanResult, err := module.getFastscanResultForPath(ctx, fileInfo.Path())
			if err != nil {
				var fileAnnotationSet bufanalysis.FileAnnotationSet
				if errors.As(err, &fileAnnotationSet) {
					// If a FileAnnotationSet, the error already contains path information, just return directly.
					//
					// We also specially handle FileAnnotationSets for exit code 100.
					// TODO: Should we just warn?
					return fileAnnotationSet
				}
				if errors.Is(err, fs.ErrNotExist) {
					// Strip any PathError and just get to the point.
					err = fs.ErrNotExist
				}
				return fmt.Errorf("%s: %w", fileInfo.Path(), err)
			}
			for _, imp := range fastscanResult.Imports {
				potentialModuleDep, err := moduleSet.getModuleForFilePath(ctx, imp.Path)
				if err != nil {
					if errors.Is(err, errIsWKT) {
						// Do not include as a dependency.
						continue
					}
					// We don't fail if we can't find an import, but we do provide a warning.
					// If we fail, we can't be compatible with commands that did pass in the pre-buf-refactor
					// world. This can happen in cases where you filter with --path and then do a ModuleDeps()
					// call via say ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles via lint, and
					// the --path specified is fine, but something else in the ModuleSet is not.
					//
					// Return the error and see what happens in integration testing for more details.
					//
					// Not great. There's other architecture decisions we could make that are wholesale
					// different here, and likely involve not using imports to derive dependencies.
					//
					// Keeping the error version of this commented out below.
					//
					// We may want to actually remove the warning here. It'll result a warning and
					// an error if somet cases.
					if errors.Is(err, fs.ErrNotExist) {
						logger.Sugar().Warnf("%s: import %q was not found.", fileInfo.Path(), imp.Path)
						continue
						//// Strip any PathError and just get to the point.
						//err = fs.ErrNotExist
						//return fmt.Errorf("%s: error on import %q: %w", fileInfo.Path(), imp, err)
					}
				}
				potentialDepOpaqueID := potentialModuleDep.OpaqueID()
				// If this is in the same module, it's not a dep
				if potentialDepOpaqueID != opaqueID {
					// No longer just potential, now real dep.
					if _, ok := depOpaqueIDToModuleDep[potentialDepOpaqueID]; !ok {
						moduleDep := newModuleDep(
							potentialModuleDep,
							parentModule,
							isDirect,
						)
						depOpaqueIDToModuleDep[potentialDepOpaqueID] = moduleDep
						newModuleDeps = append(newModuleDeps, moduleDep)
					}
				}
			}
			return nil
		},
	); err != nil {
		return err
	}
	for _, newModuleDep := range newModuleDeps {
		if err := getModuleDepsRec(
			ctx,
			logger,
			newModuleDep,
			parentModule,
			visitedOpaqueIDs,
			depOpaqueIDToModuleDep,
			// Always not direct on recursive calls.
			// We've already added all the direct deps.
			false,
		); err != nil {
			return err
		}
	}
	return nil
}
