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

package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

// GetCheckersForCategories filters the given Checkers to the categories.
//
// allKnownCategories is all known categories.
func GetCheckersForCategories(checkers []bufcheck.Checker, allKnownCategories []string, categories []string) ([]bufcheck.Checker, error) {
	if len(categories) == 0 {
		return nil, nil
	}
	categoriesMap := stringutil.SliceToMap(categories)
	if err := checkCategories(allKnownCategories, categoriesMap); err != nil {
		return nil, err
	}
	resultCheckers := make([]bufcheck.Checker, 0, len(checkers))
	for _, checker := range checkers {
		if checkerInCategories(checker, categoriesMap) {
			resultCheckers = append(resultCheckers, checker)
		}
	}
	return resultCheckers, nil
}

func checkCategories(knownCategories []string, categoriesMap map[string]struct{}) error {
	if len(categoriesMap) == 0 {
		return nil
	}
	knownCategoriesMap := stringutil.SliceToMap(knownCategories)
	var unknownCategories []string
	for category := range categoriesMap {
		if _, ok := knownCategoriesMap[category]; !ok {
			unknownCategories = append(unknownCategories, category)
		}
	}
	switch len(unknownCategories) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("%q is not a known category", unknownCategories[0])
	default:
		sort.Strings(unknownCategories)
		return fmt.Errorf("%q are not known categories", strings.Join(unknownCategories, ", "))
	}
}

func checkerInCategories(checker bufcheck.Checker, categoriesMap map[string]struct{}) bool {
	if len(categoriesMap) == 0 {
		return true
	}
	for _, category := range checker.Categories() {
		if _, ok := categoriesMap[category]; ok {
			return true
		}
	}
	return false
}
