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

package bufmodule

import (
	"fmt"
	"sort"

	"github.com/bufbuild/buf/internal/buf/bufcore"
)

func sortFileInfos(fileInfos []bufcore.FileInfo) {
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
}

func sortModuleNames(moduleNames []ModuleName) {
	sort.Slice(moduleNames, func(i, j int) bool {
		return moduleNameLess(moduleNames[i], moduleNames[j])
	})
}

func moduleNameLess(a ModuleName, b ModuleName) bool {
	return moduleNameCompareTo(a, b) < 0
}

// return -1 if less
// return 1 if greater
// return 0 if equal
func moduleNameCompareTo(a ModuleName, b ModuleName) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil && b != nil {
		return -1
	}
	if a != nil && b == nil {
		return 1
	}
	if a.Server() < b.Server() {
		return -1
	}
	if a.Server() > b.Server() {
		return 1
	}
	if a.Owner() < b.Owner() {
		return -1
	}
	if a.Owner() > b.Owner() {
		return 1
	}
	if a.Repository() < b.Repository() {
		return -1
	}
	if a.Repository() > b.Repository() {
		return 1
	}
	if a.Version() < b.Version() {
		return -1
	}
	if a.Version() > b.Version() {
		return 1
	}
	if a.Digest() < b.Digest() {
		return -1
	}
	if a.Digest() > b.Digest() {
		return 1
	}
	return 0
}

func newInvalidModuleNameStringError(path string, reason string) error {
	return fmt.Errorf("invalid module name: %s: %s", reason, path)
}
