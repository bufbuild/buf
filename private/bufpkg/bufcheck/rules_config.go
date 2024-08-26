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

package bufcheck

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
)

func rulesConfigForCheckConfig(
	checkConfig bufconfig.CheckConfig,
	allRules []Rule,
	allCategories []Category,
	ruleType check.RuleType,
) (*rulesConfig, error) {
	return newRulesConfig(
		checkConfig.UseIDsAndCategories(),
		checkConfig.ExceptIDsAndCategories(),
		checkConfig.IgnorePaths(),
		checkConfig.IgnoreIDOrCategoryToPaths(),
		allRules,
		allCategories,
		ruleType,
	)
}

func warnReferencedDeprecatedIDs(logger *zap.Logger, rulesConfig *rulesConfig) {
	warnReferencedDeprecatedIDsForIDType(
		logger,
		rulesConfig.ReferencedDeprecatedRuleIDToReplacementIDs,
		"Rule",
		"rules",
	)
	warnReferencedDeprecatedIDsForIDType(
		logger,
		rulesConfig.ReferencedDeprecatedCategoryIDToReplacementIDs,
		"Category",
		"categories",
	)
}

type rulesConfig struct {
	// RuleIDs contains the specific RuleIDs to use.
	//
	// Will only contain non-deprecated RuleIDs.
	//
	// Will always be non-empty.
	//
	// If no specific RuleIDs were configured, this will return all default RuleIDs that were of
	// the specified RuleType.
	RuleIDs         []string
	IgnoreRootPaths map[string]struct{}
	// Will only contain non-deprecated RuleIDs.
	IgnoreRuleIDToRootPaths map[string]map[string]struct{}
	// ReferencedDeprecatedRuleIDToReplacementIDs contains a map from a Rule ID
	// that was used in the configuration, to a map of the IDs that
	// replace this Rule ID.
	//
	// This can be used for warning messages.
	ReferencedDeprecatedRuleIDToReplacementIDs map[string]map[string]struct{}
	// ReferencedDeprecatedCategoryIDToReplacementIDs contains a map from a Category ID
	// that was used in the configuration, to a map of the IDs that
	// replace this Category ID.
	//
	// This can be used for warning messages.
	ReferencedDeprecatedCategoryIDToReplacementIDs map[string]map[string]struct{}
}

func newRulesConfig(
	// May contain deprecated IDs.
	useRuleIDsAndCategoryIDs []string,
	// May contain deprecated IDs.
	exceptRuleIDsAndCategoryIDs []string,
	ignoreRootPaths []string,
	// May contain deprecated IDs.
	ignoreRuleIDOrCategoryIDToRootPaths map[string][]string,
	// Rules and Categories are guaranteed to be unique by ID at this point,
	// including across each other.
	allRules []Rule,
	allCategories []Category,
	ruleType check.RuleType,
) (*rulesConfig, error) {
	if len(allRules) == 0 {
		return nil, syserror.New("no rules configured")
	}
	// Transform to struct map values. We'll want to use this later
	// in the function instead of slices.
	ignoreRuleIDOrCategoryIDToRootPathMap := make(
		map[string]map[string]struct{},
		len(ignoreRuleIDOrCategoryIDToRootPaths),
	)
	for ruleIDOrCategoryID, rootPaths := range ignoreRuleIDOrCategoryIDToRootPaths {
		ignoreRuleIDOrCategoryIDToRootPathMap[ruleIDOrCategoryID] = slicesext.ToStructMap(rootPaths)
	}

	ruleIDToRule, err := getIDToRuleOrCategory(allRules)
	if err != nil {
		return nil, err
	}
	// Contains all rules, not referenced rules.
	deprecatedRuleIDToReplacementRuleIDs, err := GetDeprecatedIDToReplacementIDs(allRules)
	if err != nil {
		return nil, err
	}
	// Contains all categories, not referenced categories.
	deprecatedCategoryIDToReplacementCategoryIDs, err := GetDeprecatedIDToReplacementIDs(allCategories)
	if err != nil {
		return nil, err
	}

	// Gather all the referenced deprecated IDs into maps for the rulesConfig.
	referencedDeprecatedRuleIDToReplacementIDs := make(map[string]map[string]struct{})
	referencedDeprecatedCategoryIDToReplacementIDs := make(map[string]map[string]struct{})
	for _, ids := range [][]string{
		useRuleIDsAndCategoryIDs,
		exceptRuleIDsAndCategoryIDs,
		slicesext.MapKeysToSlice(ignoreRuleIDOrCategoryIDToRootPathMap),
	} {
		for _, id := range ids {
			replacementRuleIDs, ok := deprecatedRuleIDToReplacementRuleIDs[id]
			if ok {
				referencedIDMap, ok2 := referencedDeprecatedRuleIDToReplacementIDs[id]
				if !ok2 {
					referencedIDMap = make(map[string]struct{})
					referencedDeprecatedRuleIDToReplacementIDs[id] = referencedIDMap
				}
				for _, replacementRuleID := range replacementRuleIDs {
					referencedIDMap[replacementRuleID] = struct{}{}
				}
			}
			replacementCategoryIDs, ok := deprecatedCategoryIDToReplacementCategoryIDs[id]
			if ok {
				referencedIDMap, ok2 := referencedDeprecatedCategoryIDToReplacementIDs[id]
				if !ok2 {
					referencedIDMap = make(map[string]struct{})
					referencedDeprecatedCategoryIDToReplacementIDs[id] = referencedIDMap
				}
				for _, replacementCategoryID := range replacementCategoryIDs {
					referencedIDMap[replacementCategoryID] = struct{}{}
				}
			}
		}
	}

	// Sort and filter empty.
	useRuleIDsAndCategoryIDs = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(useRuleIDsAndCategoryIDs)
	if len(useRuleIDsAndCategoryIDs) == 0 {
		useRuleIDsAndCategoryIDs = slicesext.Map(slicesext.Filter(allRules, func(rule Rule) bool { return rule.IsDefault() }), Rule.ID)
	}
	exceptRuleIDsAndCategoryIDs = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(exceptRuleIDsAndCategoryIDs)
	if len(useRuleIDsAndCategoryIDs) == 0 && len(exceptRuleIDsAndCategoryIDs) == 0 {
		return nil, syserror.New("use and except should always be non-empty at this point")
	}

	// Transform from rules/categories to rules.
	ruleIDToCategoryIDs, err := getRuleIDToCategoryIDs(allRules)
	if err != nil {
		return nil, err
	}
	categoryIDToRuleIDs := getCategoryIDToRuleIDs(ruleIDToCategoryIDs)
	useRuleIDs, err := transformRuleOrCategoryIDsToRuleIDs(
		useRuleIDsAndCategoryIDs,
		ruleIDToCategoryIDs,
		categoryIDToRuleIDs,
	)
	if err != nil {
		return nil, err
	}
	exceptRuleIDs, err := transformRuleOrCategoryIDsToRuleIDs(
		exceptRuleIDsAndCategoryIDs,
		ruleIDToCategoryIDs,
		categoryIDToRuleIDs,
	)
	if err != nil {
		return nil, err
	}
	ignoreRuleIDToRootPathMap, err := transformRuleOrCategoryIDToIgnoreRootPathsToRuleIDs(
		ignoreRuleIDOrCategoryIDToRootPathMap,
		ruleIDToCategoryIDs,
		categoryIDToRuleIDs,
	)
	if err != nil {
		return nil, err
	}

	// Replace deprecated rules.
	useRuleIDs = transformRuleIDsToUndeprecated(
		useRuleIDs,
		deprecatedRuleIDToReplacementRuleIDs,
	)
	exceptRuleIDs = transformRuleIDsToUndeprecated(
		exceptRuleIDs,
		deprecatedRuleIDToReplacementRuleIDs,
	)
	ignoreRuleIDToRootPathMap = transformRuleIDToIgnoreRootPathsToUndeprecated(
		ignoreRuleIDToRootPathMap,
		deprecatedRuleIDToReplacementRuleIDs,
	)

	// Figure out result rules.
	resultRuleIDToRule := make(map[string]Rule)
	for _, ruleID := range useRuleIDs {
		rule, ok := ruleIDToRule[ruleID]
		if !ok {
			return nil, fmt.Errorf("%q is not a known rule ID after verification", ruleID)
		}
		resultRuleIDToRule[rule.ID()] = rule
	}
	for _, ruleID := range exceptRuleIDs {
		if _, ok := ruleIDToRule[ruleID]; !ok {
			return nil, fmt.Errorf("%q is not a known rule ID after verification", ruleID)
		}
		delete(resultRuleIDToRule, ruleID)
	}
	resultRuleIDs := slicesext.MapKeysToSortedSlice(resultRuleIDToRule)
	if len(resultRuleIDs) == 0 {
		return nil, syserror.New("resultRuleIDs was empty")
	}

	// Validate that no Rules are not of the given RuleType.
	for _, rule := range resultRuleIDToRule {
		if rule.Type() != ruleType {
			return nil, fmt.Errorf("%q was configured in a non-%s configuration section but is of type %s", rule.ID(), ruleType.String(), ruleType.String())
		}
	}

	// Normalize ignore paths.
	ignoreRootPaths, err = normalizeIgnoreRootPaths(ignoreRootPaths)
	if err != nil {
		return nil, err
	}
	ignoreRuleIDToRootPathMap, err = normalizeKeyToIgnoreRootPathMap(ignoreRuleIDToRootPathMap)
	if err != nil {
		return nil, err
	}

	return &rulesConfig{
		RuleIDs:                 resultRuleIDs,
		IgnoreRootPaths:         slicesext.ToStructMap(ignoreRootPaths),
		IgnoreRuleIDToRootPaths: ignoreRuleIDToRootPathMap,
		ReferencedDeprecatedRuleIDToReplacementIDs:     referencedDeprecatedRuleIDToReplacementIDs,
		ReferencedDeprecatedCategoryIDToReplacementIDs: referencedDeprecatedCategoryIDToReplacementIDs,
	}, nil
}

// *** JUST USED WITHIN THIS FILE ***

func warnReferencedDeprecatedIDsForIDType(
	logger *zap.Logger,
	referencedDeprecatedIDToReplacementIDs map[string]map[string]struct{},
	capitalizedIDType string,
	pluralIDType string,
) {
	for _, deprecatedID := range slicesext.MapKeysToSortedSlice(referencedDeprecatedIDToReplacementIDs) {
		replacementIDs := slicesext.MapKeysToSortedSlice(referencedDeprecatedIDToReplacementIDs[deprecatedID])
		var replaceString string
		switch len(replacementIDs) {
		case 0:
		case 1:
			replaceString = fmt.Sprintf(" It has been replaced by %s %s.", strings.ToLower(capitalizedIDType), replacementIDs[0])
		default:
			replaceString = fmt.Sprintf(" It has been replaced by %s %s.", pluralIDType, strings.Join(replacementIDs, ", "))
		}
		logger.Warn(
			fmt.Sprintf(
				"%s %s referenced in your buf.yaml is deprecated.%s\n\t%s will continue to work. We recommend replacing %s in your buf.yaml, but no action is immediately necessary.",
				capitalizedIDType,
				deprecatedID,
				replaceString,
				deprecatedID,
				deprecatedID,
			),
		)
	}
}

func getIDToRuleOrCategory[R RuleOrCategory](ruleOrCategories []R) (map[string]R, error) {
	m := make(map[string]R)
	for _, ruleOrCategory := range ruleOrCategories {
		if _, ok := m[ruleOrCategory.ID()]; ok {
			return nil, syserror.Newf("duplicate rule or category ID: %q", ruleOrCategory.ID())
		}
		m[ruleOrCategory.ID()] = ruleOrCategory
	}
	return m, nil
}

func getRuleIDToCategoryIDs(rules []Rule) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, rule := range rules {
		if _, ok := m[rule.ID()]; ok {
			return nil, syserror.Newf("duplicate rule ID: %q", rule.ID())
		}
		m[rule.ID()] = slicesext.Map(rule.Categories(), check.Category.ID)
	}
	return m, nil
}

func getCategoryIDToRuleIDs(ruleIDToCategoryIDs map[string][]string) map[string][]string {
	categoryIDToRuleIDs := make(map[string][]string)
	for id, categoryIDs := range ruleIDToCategoryIDs {
		for _, categoryID := range categoryIDs {
			// handles empty category as well
			categoryIDToRuleIDs[categoryID] = append(categoryIDToRuleIDs[categoryID], id)
		}
	}
	return categoryIDToRuleIDs
}

func transformRuleOrCategoryIDsToRuleIDs(
	ruleOrCategoryIDs []string,
	ruleIDToCategoryIDs map[string][]string,
	categoryIDToRuleIDs map[string][]string,
) ([]string, error) {
	if len(ruleOrCategoryIDs) == 0 {
		return nil, nil
	}
	ruleIDMap := make(map[string]struct{}, len(ruleOrCategoryIDs))
	for _, ruleOrCategoryID := range ruleOrCategoryIDs {
		if ruleOrCategoryID == "" {
			continue
		}
		if _, ok := ruleIDToCategoryIDs[ruleOrCategoryID]; ok {
			ruleIDMap[ruleOrCategoryID] = struct{}{}
		} else if ruleIDs, ok := categoryIDToRuleIDs[ruleOrCategoryID]; ok {
			for _, ruleID := range ruleIDs {
				ruleIDMap[ruleID] = struct{}{}
			}
		} else {
			return nil, fmt.Errorf("%q is not a known rule or category ID", ruleOrCategoryID)
		}
	}
	return slicesext.MapKeysToSortedSlice(ruleIDMap), nil
}

func transformRuleOrCategoryIDToIgnoreRootPathsToRuleIDs(
	ruleOrCategoryIDToIgnoreRootPaths map[string]map[string]struct{},
	ruleIDToCategoryIDs map[string][]string,
	categoryIDToRuleIDs map[string][]string,
) (map[string]map[string]struct{}, error) {
	if len(ruleOrCategoryIDToIgnoreRootPaths) == 0 {
		return nil, nil
	}
	ruleIDToIgnoreRootPaths := make(
		map[string]map[string]struct{},
		len(ruleOrCategoryIDToIgnoreRootPaths),
	)
	addRootPaths := func(ruleID string, rootPaths map[string]struct{}) {
		ignoreRootPathMap, ok := ruleIDToIgnoreRootPaths[ruleID]
		if !ok {
			ignoreRootPathMap = make(map[string]struct{})
			ruleIDToIgnoreRootPaths[ruleID] = ignoreRootPathMap
		}
		for rootPath := range rootPaths {
			ignoreRootPathMap[rootPath] = struct{}{}
		}
	}
	for ruleOrCategoryID, rootPaths := range ruleOrCategoryIDToIgnoreRootPaths {
		if ruleOrCategoryID == "" {
			continue
		}
		if _, ok := ruleIDToCategoryIDs[ruleOrCategoryID]; ok {
			addRootPaths(ruleOrCategoryID, rootPaths)
		} else if ruleIDs, ok := categoryIDToRuleIDs[ruleOrCategoryID]; ok {
			for _, ruleID := range ruleIDs {
				addRootPaths(ruleID, rootPaths)
			}
		} else {
			return nil, fmt.Errorf("%q is not a known rule or category ID", ruleOrCategoryID)
		}
	}
	return ruleIDToIgnoreRootPaths, nil
}

func transformRuleIDsToUndeprecated(
	ruleIDs []string,
	deprecatedRuleIDToReplacementIDs map[string][]string,
) []string {
	undeprecatedRuleIDMap := make(map[string]struct{}, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		replacementIDs, ok := deprecatedRuleIDToReplacementIDs[ruleID]
		if ok {
			// May iterate over empty.
			for _, replacementID := range replacementIDs {
				undeprecatedRuleIDMap[replacementID] = struct{}{}
			}
		} else {
			undeprecatedRuleIDMap[ruleID] = struct{}{}
		}
	}
	return slicesext.MapKeysToSortedSlice(undeprecatedRuleIDMap)
}

func transformRuleIDToIgnoreRootPathsToUndeprecated(
	ruleIDToIgnoreRootPaths map[string]map[string]struct{},
	deprecatedRuleIDToReplacementIDs map[string][]string,
) map[string]map[string]struct{} {
	undeprecatedRuleIDToIgnoreRootPaths := make(
		map[string]map[string]struct{},
		len(ruleIDToIgnoreRootPaths),
	)
	addRootPaths := func(ruleID string, rootPaths map[string]struct{}) {
		ignoreRootPathMap, ok := undeprecatedRuleIDToIgnoreRootPaths[ruleID]
		if !ok {
			ignoreRootPathMap = make(map[string]struct{})
			undeprecatedRuleIDToIgnoreRootPaths[ruleID] = ignoreRootPathMap
		}
		for rootPath := range rootPaths {
			ignoreRootPathMap[rootPath] = struct{}{}
		}
	}
	for ruleID, rootPaths := range ruleIDToIgnoreRootPaths {
		replacementIDs, ok := deprecatedRuleIDToReplacementIDs[ruleID]
		if ok {
			// May iterate over empty.
			for _, replacementID := range replacementIDs {
				addRootPaths(replacementID, rootPaths)
			}
		} else {
			addRootPaths(ruleID, rootPaths)
		}
	}
	return undeprecatedRuleIDToIgnoreRootPaths
}

func normalizeIgnoreRootPaths(rootPaths []string) ([]string, error) {
	rootPathMap := make(map[string]struct{}, len(rootPaths))
	for _, rootPath := range rootPaths {
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
		rootPathMap[rootPath] = struct{}{}
	}
	return slicesext.MapKeysToSortedSlice(rootPathMap), nil
}

func normalizeKeyToIgnoreRootPathMap[K comparable](
	keyToRootPaths map[K]map[string]struct{},
) (map[K]map[string]struct{}, error) {
	keyToNormalizedRootPathMap := make(map[K]map[string]struct{}, len(keyToRootPaths))
	for key, rootPathMap := range keyToRootPaths {
		rootPaths, err := normalizeIgnoreRootPaths(slicesext.MapKeysToSortedSlice(rootPathMap))
		if err != nil {
			return nil, err
		}
		keyToNormalizedRootPathMap[key] = slicesext.ToStructMap(rootPaths)
	}
	return keyToNormalizedRootPathMap, nil
}
