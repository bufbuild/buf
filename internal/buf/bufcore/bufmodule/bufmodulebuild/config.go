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
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

// all of this code can likely be simplified
func newConfig(externalConfig ExternalConfig, deps ...string) (*Config, error) {
	dependencies, err := parseDependencies(deps...)
	if err != nil {
		return nil, err
	}

	rootToExcludes := make(map[string][]string)

	roots := externalConfig.Roots
	// not yet relative to roots
	fullExcludes := externalConfig.Excludes

	if len(roots) == 0 {
		roots = []string{"."}
	}
	roots, err = normalizeAndCheckPaths(roots, "root", normalpath.Relative, true)
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		// we already checked duplicates, but just in case
		if _, ok := rootToExcludes[root]; ok {
			return nil, fmt.Errorf("unexpected duplicate root: %q", root)
		}
		rootToExcludes[root] = make([]string, 0)
	}

	if len(fullExcludes) == 0 {
		return &Config{
			RootToExcludes: rootToExcludes,
			Deps:           dependencies,
		}, nil
	}

	// this also verifies that fullExcludes is unique
	fullExcludes, err = normalizeAndCheckPaths(fullExcludes, "exclude", normalpath.Relative, true)
	if err != nil {
		return nil, err
	}

	// verify that no exclude equals a root directly and only directories are specified
	for _, fullExclude := range fullExcludes {
		if normalpath.Ext(fullExclude) == ".proto" {
			return nil, fmt.Errorf("excludes can only be directories but file %s discovered", fullExclude)
		}
		if _, ok := rootToExcludes[fullExclude]; ok {
			return nil, fmt.Errorf("%s is both a root and exclude, which means the entire root is excluded, which is not valid", fullExclude)
		}
	}

	// verify that all excludes are within a root
	rootMap := stringutil.SliceToMap(roots)
	for _, fullExclude := range fullExcludes {
		switch matchingRoots := normalpath.MapAllEqualOrContainingPaths(rootMap, fullExclude, normalpath.Relative); len(matchingRoots) {
		case 0:
			return nil, fmt.Errorf("exclude %s is not contained in any root, which is not valid", fullExclude)
		case 1:
			root := matchingRoots[0]
			exclude, err := normalpath.Rel(root, fullExclude)
			if err != nil {
				return nil, err
			}
			// just in case
			exclude, err = normalpath.NormalizeAndValidate(exclude)
			if err != nil {
				return nil, err
			}
			rootToExcludes[root] = append(rootToExcludes[root], exclude)
		default:
			// this should never happen, but just in case
			return nil, fmt.Errorf("exclude %q was in multiple roots %v (system error)", fullExclude, matchingRoots)
		}
	}

	for root, excludes := range rootToExcludes {
		uniqueSortedExcludes := stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(excludes)
		if len(excludes) != len(uniqueSortedExcludes) {
			// this should never happen, but just in case
			return nil, fmt.Errorf("excludes %v are not unique (system error)", excludes)
		}
		rootToExcludes[root] = uniqueSortedExcludes
	}
	return &Config{
		RootToExcludes: rootToExcludes,
		Deps:           dependencies,
	}, nil
}

// parseDependencies parses the given dependencies into structured ModuleNames.
func parseDependencies(deps ...string) ([]bufmodule.ModuleName, error) {
	if len(deps) == 0 {
		return nil, nil
	}
	dependencies := make([]bufmodule.ModuleName, 0, len(deps))
	for _, dependency := range deps {
		dependency := strings.TrimSpace(dependency)
		moduleName, err := bufmodule.ModuleNameForString(dependency)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, moduleName)
	}
	if err := bufmodule.ValidateModuleNamesUnique(dependencies); err != nil {
		return nil, err
	}
	return dependencies, nil
}
