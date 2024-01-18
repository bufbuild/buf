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

package bufwork

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

func newConfigV1(externalConfig ExternalConfigV1, workspaceID string) (*Config, error) {
	if len(externalConfig.Directories) == 0 {
		return nil, fmt.Errorf(
			`%s has no directories set. Please add "directories: [...]"`,
			workspaceID,
		)
	}
	directorySet := make(map[string]struct{}, len(externalConfig.Directories))
	for _, directory := range externalConfig.Directories {
		normalizedDirectory, err := normalpath.NormalizeAndValidate(directory)
		if err != nil {
			return nil, fmt.Errorf(`directory "%s" listed in %s is invalid: %w`, normalpath.Unnormalize(directory), workspaceID, err)
		}
		if _, ok := directorySet[normalizedDirectory]; ok {
			return nil, fmt.Errorf(
				`directory "%s" is listed more than once in %s`,
				normalpath.Unnormalize(normalizedDirectory),
				workspaceID,
			)
		}
		if normalizedDirectory == "." {
			return nil, fmt.Errorf(`directory "." listed in %s, it is not valid to have "." as a workspace directory, as this is no different than not having a workspace at all, see https://buf.build/docs/reference/workspaces/#directories for more details`, workspaceID)
		}
		directorySet[normalizedDirectory] = struct{}{}
	}
	// It's very important that we sort the directories here so that the
	// constructed modules and/or images are in a deterministic order.
	directories := slicesext.MapKeysToSlice(directorySet)
	sort.Slice(directories, func(i int, j int) bool {
		return directories[i] < directories[j]
	})
	if err := validateConfigurationOverlap(directories, workspaceID); err != nil {
		return nil, err
	}
	return &Config{
		Directories: directories,
	}, nil
}

// validateOverlap returns a non-nil error if any of the directories overlap
// with each other. The given directories are expected to be sorted.
func validateConfigurationOverlap(directories []string, workspaceID string) error {
	for i := 0; i < len(directories); i++ {
		for j := i + 1; j < len(directories); j++ {
			left := directories[i]
			right := directories[j]
			if normalpath.ContainsPath(left, right, normalpath.Relative) {
				return fmt.Errorf(
					`directory "%s" contains directory "%s" in %s`,
					normalpath.Unnormalize(left),
					normalpath.Unnormalize(right),
					workspaceID,
				)
			}
			if normalpath.ContainsPath(right, left, normalpath.Relative) {
				return fmt.Errorf(
					`directory "%s" contains directory "%s" in %s`,
					normalpath.Unnormalize(right),
					normalpath.Unnormalize(left),
					workspaceID,
				)
			}
		}
	}
	return nil
}
