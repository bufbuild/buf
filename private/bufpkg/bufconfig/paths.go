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

package bufconfig

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// normalizeAndCheckPaths verifies that:
//
//   - No paths are empty.
//   - All paths are normalized and validated.
//   - All paths are unique.
//   - No path contains another path.
//
// Normalizes and sorts the paths.
func normalizeAndCheckPaths(paths []string, name string) ([]string, error) {
	if len(paths) == 0 {
		return paths, nil
	}
	outputs := make([]string, len(paths))
	for i, path := range paths {
		if path == "" {
			return nil, fmt.Errorf("%s contained an empty path", name)
		}
		output, err := normalpath.NormalizeAndValidate(path)
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
			if normalpath.EqualsOrContainsPath(output2, output1, normalpath.Relative) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output1, name, output2)
			}
			if normalpath.EqualsOrContainsPath(output1, output2, normalpath.Relative) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output2, name, output1)
			}
		}
	}
	return outputs, nil
}
