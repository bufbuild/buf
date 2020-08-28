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

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

const (
	defaultEnumZeroValueSuffix = "_UNSPECIFIED"
	defaultServiceSuffix       = "Service"
)

// Config is the check config.
type Config struct {
	// Checkers are the checkers to run.
	//
	// Checkers will be sorted by first categories, then id when Configs are
	// created from this package, i.e. created wth ConfigBuilder.NewConfig.
	Checkers []*Checker

	IgnoreRootPaths     map[string]struct{}
	IgnoreIDToRootPaths map[string]map[string]struct{}

	AllowCommentIgnores bool
}

// ConfigBuilder is a config builder.
type ConfigBuilder struct {
	Use    []string
	Except []string

	IgnoreRootPaths               []string
	IgnoreIDOrCategoryToRootPaths map[string][]string

	AllowCommentIgnores bool

	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string
}

// NewConfig returns a new Config.
func (b ConfigBuilder) NewConfig(
	checkerBuilders []*CheckerBuilder,
	idToCategories map[string][]string,
	defaultCategories []string,
) (*Config, error) {
	return newConfig(
		b,
		checkerBuilders,
		idToCategories,
		defaultCategories,
	)
}

func newConfig(
	configBuilder ConfigBuilder,
	checkerBuilders []*CheckerBuilder,
	idToCategories map[string][]string,
	defaultCategories []string,
) (*Config, error) {
	configBuilder.Use = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(configBuilder.Use)
	configBuilder.Except = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(configBuilder.Except)
	if len(configBuilder.Use) == 0 {
		// default behavior
		configBuilder.Use = defaultCategories
	}
	if configBuilder.EnumZeroValueSuffix == "" {
		configBuilder.EnumZeroValueSuffix = defaultEnumZeroValueSuffix
	}
	if configBuilder.ServiceSuffix == "" {
		configBuilder.ServiceSuffix = defaultServiceSuffix
	}
	return newConfigForCheckerBuilders(
		configBuilder,
		checkerBuilders,
		idToCategories,
	)
}

// revisionCheckerBuilders is a var such as Revision1CheckerBuilders
func newConfigForCheckerBuilders(
	configBuilder ConfigBuilder,
	checkerBuilders []*CheckerBuilder,
	idToCategories map[string][]string,
) (*Config, error) {
	// this checks that there are not duplicate IDs for a given revision
	// which would be a system error
	idToCheckerBuilder, err := getIDToCheckerBuilder(checkerBuilders)
	if err != nil {
		return nil, err
	}
	categoryToIDs := getCategoryToIDs(idToCategories)
	useIDMap, err := transformToIDMap(configBuilder.Use, idToCategories, categoryToIDs)
	if err != nil {
		return nil, err
	}
	exceptIDMap, err := transformToIDMap(configBuilder.Except, idToCategories, categoryToIDs)
	if err != nil {
		return nil, err
	}

	// this removes duplicates
	// we already know that a given checker with the same ID is equivalent
	resultIDToCheckerBuilder := make(map[string]*CheckerBuilder)

	for id := range useIDMap {
		checkerBuilder, ok := idToCheckerBuilder[id]
		if !ok {
			return nil, fmt.Errorf("%q is not a known id after verification", id)
		}
		resultIDToCheckerBuilder[checkerBuilder.id] = checkerBuilder
	}
	for id := range exceptIDMap {
		if _, ok := idToCheckerBuilder[id]; !ok {
			return nil, fmt.Errorf("%q is not a known id after verification", id)
		}
		delete(resultIDToCheckerBuilder, id)
	}

	resultCheckerBuilders := make([]*CheckerBuilder, 0, len(resultIDToCheckerBuilder))
	for _, checkerBuilder := range resultIDToCheckerBuilder {
		resultCheckerBuilders = append(resultCheckerBuilders, checkerBuilder)
	}
	resultCheckers := make([]*Checker, 0, len(resultCheckerBuilders))
	for _, checkerBuilder := range resultCheckerBuilders {
		categories, err := getCheckerBuilderCategories(checkerBuilder, idToCategories)
		if err != nil {
			return nil, err
		}
		checker, err := checkerBuilder.NewChecker(configBuilder, categories)
		if err != nil {
			return nil, err
		}
		resultCheckers = append(resultCheckers, checker)
	}
	sortCheckers(resultCheckers)

	ignoreIDToRootPathsUnnormalized, err := transformToIDToListMap(configBuilder.IgnoreIDOrCategoryToRootPaths, idToCategories, categoryToIDs)
	if err != nil {
		return nil, err
	}
	ignoreIDToRootPaths := make(map[string]map[string]struct{})
	for id, rootPaths := range ignoreIDToRootPathsUnnormalized {
		for rootPath := range rootPaths {
			if rootPath == "" {
				continue
			}
			rootPath, err := normalpath.NormalizeAndValidate(rootPath)
			if err != nil {
				return nil, err
			}
			if rootPath == "." {
				return nil, fmt.Errorf("cannot specify %q as an ignore path", rootPath)
			}
			resultRootPathMap, ok := ignoreIDToRootPaths[id]
			if !ok {
				resultRootPathMap = make(map[string]struct{})
				ignoreIDToRootPaths[id] = resultRootPathMap
			}
			resultRootPathMap[rootPath] = struct{}{}
		}
	}

	ignoreRootPaths := make(map[string]struct{}, len(configBuilder.IgnoreRootPaths))
	for _, rootPath := range configBuilder.IgnoreRootPaths {
		if rootPath == "" {
			continue
		}
		rootPath, err := normalpath.NormalizeAndValidate(rootPath)
		if err != nil {
			return nil, err
		}
		if rootPath == "." {
			return nil, fmt.Errorf("cannot specify %q as an ignore path", rootPath)
		}
		ignoreRootPaths[rootPath] = struct{}{}
	}

	return &Config{
		Checkers:            resultCheckers,
		IgnoreIDToRootPaths: ignoreIDToRootPaths,
		IgnoreRootPaths:     ignoreRootPaths,
		AllowCommentIgnores: configBuilder.AllowCommentIgnores,
	}, nil
}

func transformToIDMap(idsOrCategories []string, idToCategories map[string][]string, categoryToIDs map[string][]string) (map[string]struct{}, error) {
	if len(idsOrCategories) == 0 {
		return nil, nil
	}
	idMap := make(map[string]struct{}, len(idsOrCategories))
	for _, idOrCategory := range idsOrCategories {
		if idOrCategory == "" {
			continue
		}
		if _, ok := idToCategories[idOrCategory]; ok {
			id := idOrCategory
			idMap[id] = struct{}{}
		} else if ids, ok := categoryToIDs[idOrCategory]; ok {
			for _, id := range ids {
				idMap[id] = struct{}{}
			}
		} else {
			return nil, fmt.Errorf("%q is not a known id or category", idOrCategory)
		}
	}
	return idMap, nil
}

func transformToIDToListMap(idOrCategoryToList map[string][]string, idToCategories map[string][]string, categoryToIDs map[string][]string) (map[string]map[string]struct{}, error) {
	if len(idOrCategoryToList) == 0 {
		return nil, nil
	}
	idToListMap := make(map[string]map[string]struct{}, len(idOrCategoryToList))
	for idOrCategory, list := range idOrCategoryToList {
		if idOrCategory == "" {
			continue
		}
		if _, ok := idToCategories[idOrCategory]; ok {
			id := idOrCategory
			if _, ok := idToListMap[id]; !ok {
				idToListMap[id] = make(map[string]struct{})
			}
			for _, elem := range list {
				idToListMap[id][elem] = struct{}{}
			}
		} else if ids, ok := categoryToIDs[idOrCategory]; ok {
			for _, id := range ids {
				if _, ok := idToListMap[id]; !ok {
					idToListMap[id] = make(map[string]struct{})
				}
				for _, elem := range list {
					idToListMap[id][elem] = struct{}{}
				}
			}
		} else {
			return nil, fmt.Errorf("%q is not a known id or category", idOrCategory)
		}
	}
	return idToListMap, nil
}

func getCategoryToIDs(idToCategories map[string][]string) map[string][]string {
	categoryToIDs := make(map[string][]string)
	for id, categories := range idToCategories {
		for _, category := range categories {
			categoryToIDs[category] = append(categoryToIDs[category], id)
		}
	}
	return categoryToIDs
}

func getIDToCheckerBuilder(checkerBuilders []*CheckerBuilder) (map[string]*CheckerBuilder, error) {
	m := make(map[string]*CheckerBuilder)
	for _, checkerBuilder := range checkerBuilders {
		if _, ok := m[checkerBuilder.id]; ok {
			return nil, fmt.Errorf("duplicate checker ID: %q", checkerBuilder.id)
		}
		m[checkerBuilder.id] = checkerBuilder
	}
	return m, nil
}

func getCheckerBuilderCategories(
	checkerBuilder *CheckerBuilder,
	idToCategories map[string][]string,
) ([]string, error) {
	categories, ok := idToCategories[checkerBuilder.id]
	if !ok {
		return nil, fmt.Errorf("%q is not configured for categories", checkerBuilder.id)
	}
	// it is ok for categories to be empty, however the map must contain an entry
	// or otherwise this is a system error
	return categories, nil
}

func sortCheckers(checkers []*Checker) {
	sort.Slice(
		checkers,
		func(i int, j int) bool {
			// categories are sorted at this point
			// so we know the first category is a top-level category if present
			one := checkers[i]
			two := checkers[j]
			oneCategories := one.Categories()
			twoCategories := two.Categories()
			if len(oneCategories) > 0 && len(twoCategories) > 0 {
				compare := categoryCompare(oneCategories[0], twoCategories[0])
				if compare < 0 {
					return true
				}
				if compare > 0 {
					return false
				}
			}
			oneCategoriesString := strings.Join(oneCategories, ",")
			twoCategoriesString := strings.Join(twoCategories, ",")
			if oneCategoriesString < twoCategoriesString {
				return true
			}
			if oneCategoriesString > twoCategoriesString {
				return false
			}
			return one.ID() < two.ID()
		},
	)
}
