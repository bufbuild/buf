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

package bufbuild

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

func normalizeAndValidateRoots(roots []string) ([]string, error) {
	if len(roots) == 0 {
		roots = []string{"."}
	}
	return normalizeAndValidateFileList(roots, "root")
}

func normalizeAndValidateRootsExcludes(roots []string, excludes []string) ([]string, []string, error) {
	roots, err := normalizeAndValidateRoots(roots)
	if err != nil {
		return nil, nil, err
	}

	if len(excludes) == 0 {
		return roots, nil, nil
	}

	excludes, err = normalizeAndValidateFileList(excludes, "exclude")
	if err != nil {
		return nil, nil, err
	}

	rootMap := stringutil.SliceToMap(roots)
	excludeMap := stringutil.SliceToMap(excludes)

	// verify that no exclude equals a root directly
	for exclude := range excludeMap {
		if _, ok := rootMap[exclude]; ok {
			return nil, nil, fmt.Errorf("%s is both a root and exclude, which means the entire root is excluded, which is not valid", exclude)
		}
	}
	// verify that all excludes are within a root
	for exclude := range excludeMap {
		if !normalpath.MapContainsMatch(rootMap, exclude) {
			return nil, nil, fmt.Errorf("exclude %s is not contained in any root, which is not valid", exclude)
		}
		if normalpath.Ext(exclude) == ".proto" {
			return nil, nil, fmt.Errorf("excludes can only be directories but file %s discovered", exclude)
		}
	}
	return roots, excludes, nil
}

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
			if normalpath.IsMatch(output2, output1) {
				return nil, fmt.Errorf("%s %s is within %s %s which is not allowed", name, output1, name, output2)
			}
			if normalpath.IsMatch(output1, output2) {
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
