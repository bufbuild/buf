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

package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
	module Module,
) ([]ModuleDep, error) {
	depOpaqueIDToModuleDep := make(map[string]ModuleDep)
	protoFileTracker := newProtoFileTracker()
	if err := getModuleDepsRec(
		ctx,
		module,
		make(map[string]struct{}),
		make(map[string]struct{}),
		nil,
		depOpaqueIDToModuleDep,
		protoFileTracker,
		true,
	); err != nil {
		return nil, err
	}
	if err := protoFileTracker.validate(); err != nil {
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
	module Module,
	visitedOpaqueIDs map[string]struct{},
	// Changes as we go down the stack.
	parentOpaqueIDs map[string]struct{},
	// Ordered version of parentOpaqueIDs so we can print a cycle error.
	orderedParentOpaqueIDs []string,
	// Already discovered deps.
	depOpaqueIDToModuleDep map[string]ModuleDep,
	protoFileTracker *protoFileTracker,
	isDirect bool,
) error {
	opaqueID := module.OpaqueID()
	if _, ok := parentOpaqueIDs[opaqueID]; ok {
		return &ModuleCycleError{OpaqueIDs: append(orderedParentOpaqueIDs, opaqueID)}
	}
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		return nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	moduleSet := module.ModuleSet()
	if moduleSet == nil {
		// This should never happen.
		return syserror.New("moduleSet never set on module")
	}

	protoFileTracker.trackModule(module)
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
			protoFileTracker.trackFileInfo(fileInfo)

			fastscanResult, err := module.getFastscanResultForPath(ctx, fileInfo.Path())
			if err != nil {
				var fileAnnotationSet bufanalysis.FileAnnotationSet
				if errors.As(err, &fileAnnotationSet) {
					// If a FileAnnotationSet, the error already contains path information, just return directly.
					//
					// We also specially handle FileAnnotationSets for exit code 100.
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
					if errors.Is(err, fs.ErrNotExist) {
						// It is OK to not have a module with the WKT specified as a dependency. In this case,
						// we do not include WKTs in our digest calculations. We've determined this is OK since
						// WKTs are not downloaded and not subject to supply-side attacks.
						//
						// It is also OK for a module to have the WKT, such as buf.build/protocolbuffers/wellknowntypes.
						// In this case, we want this to be recognized.
						//
						// Note that if someone checks in a WKT as part of their module, this will result in this module
						// being a dependency for all other modules that import that WKT.
						// This could result in unintended module cycles.
						if datawkt.Exists(imp.Path) {
							continue
						}
						// We specifically handle ImportNotExistErrors with exit code 100 in buf.go.
						//
						// We don't want to return a FileAnnotationSet here as we never have line
						// and column information, and the FileAnnotation will get printed out as 1:1.
						//
						// This isn't a FileAnnotation, it's a not exist error, semantically it's different.
						return &ImportNotExistError{
							fileInfo:   fileInfo,
							importPath: imp.Path,
						}
					}
					return err
				}
				potentialDepOpaqueID := potentialModuleDep.OpaqueID()
				// If this is in the same module, it's not a dep
				if potentialDepOpaqueID != opaqueID {
					// No longer just potential, now real dep.
					if _, ok := depOpaqueIDToModuleDep[potentialDepOpaqueID]; !ok {
						moduleDep := newModuleDep(
							potentialModuleDep,
							module,
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
	parentOpaqueIDs[opaqueID] = struct{}{}
	newOrderedParentOpaqueIDs := append(orderedParentOpaqueIDs, opaqueID)
	// Triple-check to make sure newModuleDeps order is deterministic. This is just defensive.
	sort.Slice(
		newModuleDeps,
		func(i int, j int) bool {
			return newModuleDeps[i].OpaqueID() < newModuleDeps[j].OpaqueID()
		},
	)
	for _, newModuleDep := range newModuleDeps {
		if err := getModuleDepsRec(
			ctx,
			newModuleDep,
			visitedOpaqueIDs,
			parentOpaqueIDs,
			newOrderedParentOpaqueIDs,
			depOpaqueIDToModuleDep,
			protoFileTracker,
			// Always not direct on recursive calls.
			// We've already added all the direct deps.
			false,
		); err != nil {
			return err
		}
	}
	delete(parentOpaqueIDs, opaqueID)
	return nil
}
