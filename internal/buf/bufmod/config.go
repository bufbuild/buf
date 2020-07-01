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

package bufmod

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

// all of this code can likely be simplified
func newConfig(externalConfig ExternalConfig) (*Config, error) {
	rootToExcludes := make(map[string][]string)

	roots := externalConfig.Roots
	// not yet relative to roots
	fullExcludes := externalConfig.Excludes

	if len(roots) == 0 {
		roots = []string{"."}
	}
	roots, err := normalizeAndValidateFileList(roots, "root")
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
		}, nil
	}

	// this also verifies that fullExcludes is unique
	fullExcludes, err = normalizeAndValidateFileList(fullExcludes, "exclude")
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
		switch matchingRoots := normalpath.MapAllEqualOrContainingPaths(rootMap, fullExclude); len(matchingRoots) {
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
	}, nil
}

// normalizeAndValidate verifies that:
//
//   - All inputs are normalized and validated.
//   - All inputs are unique.
//   - No input contains another input.
func normalizeAndValidateFileList(inputs []string, name string) ([]string, error) {
	if len(inputs) == 0 {
		return inputs, nil
	}

	var outputs []string
	for _, input := range inputs {
		if input == "" {
			return nil, fmt.Errorf("%s value is empty", name)
		}
		output, err := normalpath.NormalizeAndValidate(input)
		if err != nil {
			// user error
			return nil, err
		}
		outputs = append(outputs, output)
	}
	sort.Strings(outputs)

	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			output1 := outputs[i]
			output2 := outputs[j]

			if output1 == output2 {
				return nil, fmt.Errorf("duplicate %s %s", name, output1)
			}
			if normalpath.EqualsOrContainsPath(output2, output1) {
				return nil, fmt.Errorf("%s %s is within %s %s which is not allowed", name, output1, name, output2)
			}
			if normalpath.EqualsOrContainsPath(output1, output2) {
				return nil, fmt.Errorf("%s %s is within %s %s which is not allowed", name, output2, name, output1)
			}
		}
	}

	// already checked duplicates, but if there are multiple directories and we have ".", then the other
	// directories are within the output directory "."
	var notDotDir []string
	hasDotDir := false
	for _, output := range outputs {
		if output != "." {
			notDotDir = append(notDotDir, output)
		} else {
			hasDotDir = true
		}
	}
	if hasDotDir {
		if len(notDotDir) == 1 {
			return nil, fmt.Errorf("%s %s is within %s . which is not allowed", name, notDotDir[0], name)
		}
		if len(notDotDir) > 1 {
			return nil, fmt.Errorf("%ss %v are within %s . which is not allowed", name, notDotDir, name)
		}
	}

	return outputs, nil
}
