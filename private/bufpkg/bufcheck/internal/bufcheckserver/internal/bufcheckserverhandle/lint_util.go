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

package bufcheckserverhandle

import (
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

func fieldToLowerSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return stringutil.ToLowerSnakeCase(s, stringutil.SnakeCaseWithNewWordOnDigits())
	return stringutil.ToLowerSnakeCase(s)
}

func fieldToUpperSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return stringutil.ToUpperSnakeCase(s, stringutil.SnakeCaseWithNewWordOnDigits())
	return stringutil.ToUpperSnakeCase(s)
}

// validLeadingComment returns true if comment has at least one line that isn't empty
// and doesn't start with one of the comment excludes.
func validLeadingComment(commentExcludes []string, comment string) bool {
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		for _, commentExclude := range commentExcludes {
			if line != "" && !strings.HasPrefix(line, commentExclude) {
				return true
			}
		}
	}
	return false
}

// Returns the usedPackageList if there is an import cycle.
//
// Note this stops on the first import cycle detected, it doesn't attempt to get all of them - not perfect.
func getImportCycleIfExists(
	// Should never be ""
	pkg string,
	packageToDirectlyImportedPackageToFileImports map[string]map[string][]bufprotosource.FileImport,
	usedPackageMap map[string]struct{},
	usedPackageList []string,
) []string {
	// Append before checking so that the returned import cycle is actually a cycle
	usedPackageList = append(usedPackageList, pkg)
	if _, ok := usedPackageMap[pkg]; ok {
		// We have an import cycle, but if the first package in the list does not
		// equal the last, do not return as an import cycle unless the first
		// element equals the last - we do DFS from each package so this will
		// be picked up separately
		if usedPackageList[0] == usedPackageList[len(usedPackageList)-1] {
			return usedPackageList
		}
		return nil
	}
	usedPackageMap[pkg] = struct{}{}
	// Will never equal pkg
	for directlyImportedPackage := range packageToDirectlyImportedPackageToFileImports[pkg] {
		// Can equal "" per the function signature of PackageToDirectlyImportedPackageToFileImports
		if directlyImportedPackage == "" {
			continue
		}
		if importCycle := getImportCycleIfExists(
			directlyImportedPackage,
			packageToDirectlyImportedPackageToFileImports,
			usedPackageMap,
			usedPackageList,
		); len(importCycle) != 0 {
			return importCycle
		}
	}
	delete(usedPackageMap, pkg)
	return nil
}
