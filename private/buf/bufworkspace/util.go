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

package bufworkspace

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

func applyRootsToTargetPath(roots []string, path string, pathType normalpath.PathType) (string, error) {
	var matchingRoots []string
	for _, root := range roots {
		if normalpath.ContainsPath(root, path, pathType) {
			matchingRoots = append(matchingRoots, root)
		}
	}
	switch len(matchingRoots) {
	case 0:
		// this is a user error and will likely happen often
		return "", fmt.Errorf(
			"path %q is not contained within any of roots %s - note that specified paths "+
				"cannot be roots, but must be contained within roots",
			path,
			stringutil.SliceToHumanStringQuoted(roots),
		)
	case 1:
		targetPath, err := normalpath.Rel(matchingRoots[0], path)
		if err != nil {
			return "", err
		}
		// just in case
		return normalpath.NormalizeAndValidate(targetPath)
	default:
		// this should never happen
		return "", fmt.Errorf("%q is contained in multiple roots %s", path, stringutil.SliceToHumanStringQuoted(roots))
	}
}

// normalizeAndAbsolutePaths verifies that:
//
//   - No paths are empty.
//   - All paths are normalized.
//   - All paths are unique.
//   - No path contains another path.
//
// Normalizes, absolutes, and sorts the paths.
func normalizeAndAbsolutePaths(paths []string, name string) ([]string, error) {
	if len(paths) == 0 {
		return paths, nil
	}
	outputs := make([]string, len(paths))
	for i, path := range paths {
		if path == "" {
			return nil, fmt.Errorf("%s contained an empty path", name)
		}
		output, err := normalpath.NormalizeAndAbsolute(path)
		if err != nil {
			// user error
			return nil, err
		}
		outputs[i] = output
	}
	sort.Strings(outputs)
	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			output1 := outputs[i]
			output2 := outputs[j]

			if output1 == output2 {
				return nil, fmt.Errorf("duplicate %s %q", name, output1)
			}
			if normalpath.EqualsOrContainsPath(output2, output1, normalpath.Absolute) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output1, name, output2)
			}
			if normalpath.EqualsOrContainsPath(output1, output2, normalpath.Absolute) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output2, name, output1)
			}
		}
	}
	return outputs, nil
}
