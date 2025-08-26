// Copyright 2020-2025 Buf Technologies, Inc.
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
	"log/slog"
	"sort"
	"strings"

	"buf.build/go/bufplugin/check"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

func rulesConfigForCheckConfig(
	checkConfig bufconfig.CheckConfig,
	allRules []Rule,
	allCategories []Category,
	ruleType check.RuleType,
	relatedCheckConfigs []bufconfig.CheckConfig,
) (*rulesConfig, error) {
	return newRulesConfig(
		checkConfig.UseIDsAndCategories(),
		checkConfig.ExceptIDsAndCategories(),
		checkConfig.IgnorePaths(),
		checkConfig.IgnoreIDOrCategoryToPaths(),
		allRules,
		allCategories,
		ruleType,
		relatedCheckConfigs,
	)
}

func logRulesConfig(logger *slog.Logger, configName string, rulesConfig *rulesConfig, hasPolicyConfigs bool) {
	logger.Debug("rulesConfig", slog.Any("ruleIDs", rulesConfig.RuleIDs))
	if len(rulesConfig.RuleIDs) == 0 && !hasPolicyConfigs {
		logger.Warn("No " + rulesConfig.RuleType.String() + " rules are configured in " + configName + ".")
	}
	warnReferencedDeprecatedIDs(logger, configName, rulesConfig)
	warnUnusedPlugins(logger, configName, rulesConfig)
}

type rulesConfig struct {
	// RuleType is the RuleType that was passed when creating this rulesConfig.
	//
	// All of the Rule IDs will be for Rules of this type.
	RuleType check.RuleType
	// RuleIDs contains the specific RuleIDs to use.
	//
	// Will only contain non-deprecated RuleIDs.
	// This will only contain RuleIDs of the given RuleType.
	//
	// Will always be non-empty.
	//
	// If no specific RuleIDs were configured, this will return all default RuleIDs that were of
	// the specified RuleType.
	RuleIDs         []string
	IgnoreRootPaths map[string]struct{}
	// Will only contain non-deprecated RuleIDs.
	// This will only contain RuleIDs of the given RuleType.
	IgnoreRuleIDToRootPaths map[string]map[string]struct{}
	// ReferencedDeprecatedRuleIDToReplacementIDs contains a map from a Rule ID
	// that was used in the configuration, to a map of the IDs that
	// replace this Rule ID.
	//
	// This will only contain RuleIDs of the given RuleType.
	//
	// This can be used for warning messages.
	ReferencedDeprecatedRuleIDToReplacementIDs map[string]map[string]struct{}
	// ReferencedDeprecatedCategoryIDToReplacementIDs contains a map from a Category ID
	// that was used in the configuration, to a map of the IDs that
	// replace this Category ID.
	//
	// This will only contain RuleIDs of the given RuleType.
	//
	// This can be used for warning messages.
	ReferencedDeprecatedCategoryIDToReplacementIDs map[string]map[string]struct{}
	// UnusedPluginNameToRuleIDs contains a map from unused plugin name to the Rule IDs that
	// that plugin has.
	//
	// A plugin is unused if no rules from the plugin are configured.
	//
	// The caller can provide additional related check configs to check if the plugin has rules
	// configured. If the plugin has rules configured in any of the additional check configs
	// provided, then we don't warn.
	//
	// The Rule IDs will be sorted.
	// This will only contain RuleIDs of the given RuleType.
	// There will be no empty key for plugin name (which means the Rule is builtin), that is
	// builtin rules are not accounted for as unused.
	//
	// This can be used for warning messages.
	UnusedPluginNameToRuleIDs map[string][]string
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
	relatedCheckConfigs []bufconfig.CheckConfig,
) (*rulesConfig, error) {
	allRulesForType := rulesForType(allRules, ruleType)
	if len(allRulesForType) == 0 {
		// This can happen with i.e. disable_builtin pretty easily.
		//
		// We return here so that we can do some syserror checking below for expectations
		// that certain variables are non-empty at certain points.
		return &rulesConfig{
			RuleType:                ruleType,
			RuleIDs:                 make([]string, 0),
			IgnoreRootPaths:         make(map[string]struct{}),
			IgnoreRuleIDToRootPaths: make(map[string]map[string]struct{}),
			ReferencedDeprecatedRuleIDToReplacementIDs:     make(map[string]map[string]struct{}),
			ReferencedDeprecatedCategoryIDToReplacementIDs: make(map[string]map[string]struct{}),
			UnusedPluginNameToRuleIDs:                      make(map[string][]string),
		}, nil
	}

	// Transform to struct map values. We'll want to use this later
	// in the function instead of slices.
	ignoreRuleIDOrCategoryIDToRootPathMap := make(
		map[string]map[string]struct{},
		len(ignoreRuleIDOrCategoryIDToRootPaths),
	)
	for ruleIDOrCategoryID, rootPaths := range ignoreRuleIDOrCategoryIDToRootPaths {
		ignoreRuleIDOrCategoryIDToRootPathMap[ruleIDOrCategoryID] = xslices.ToStructMap(rootPaths)
	}

	ruleIDToRule, err := getIDToRuleOrCategory(allRulesForType)
	if err != nil {
		return nil, err
	}
	// Contains all rules, not referenced rules.
	deprecatedRuleIDToReplacementRuleIDs, err := GetDeprecatedIDToReplacementIDs(allRulesForType)
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
		xslices.MapKeysToSlice(ignoreRuleIDOrCategoryIDToRootPathMap),
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
	useRuleIDsAndCategoryIDs = xstrings.SliceToUniqueSortedSliceFilterEmptyStrings(useRuleIDsAndCategoryIDs)
	if len(useRuleIDsAndCategoryIDs) == 0 {
		useRuleIDsAndCategoryIDs = xslices.Map(xslices.Filter(allRulesForType, func(rule Rule) bool { return rule.Default() }), Rule.ID)
	}
	exceptRuleIDsAndCategoryIDs = xstrings.SliceToUniqueSortedSliceFilterEmptyStrings(exceptRuleIDsAndCategoryIDs)
	if len(useRuleIDsAndCategoryIDs) == 0 && len(exceptRuleIDsAndCategoryIDs) == 0 {
		return nil, syserror.New("use and except should always be non-empty at this point")
	}

	// Transform from rules/categories to rules.
	ruleIDToCategoryIDs, err := getRuleIDToCategoryIDs(allRulesForType)
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
	resultRules := xslices.MapValuesToSlice(resultRuleIDToRule)
	if len(resultRules) == 0 {
		return nil, syserror.New("resultRules was empty")
	}
	sort.Slice(
		resultRules,
		func(i int, j int) bool {
			return resultRules[i].ID() < resultRules[j].ID()
		},
	)

	// Normalize ignore paths.
	ignoreRootPaths, err = normalizeIgnoreRootPaths(ignoreRootPaths)
	if err != nil {
		return nil, err
	}
	ignoreRuleIDToRootPathMap, err = normalizeKeyToIgnoreRootPathMap(ignoreRuleIDToRootPathMap)
	if err != nil {
		return nil, err
	}

	// This map initially contains a map from plugin name to ALL Rule IDs, but we will
	// then delete the used plugin names, and then delete the plugins with other rule types.
	//
	// Note this will only contain RuleIDs for the given RuleType.
	unusedPluginNameToRuleIDs := getPluginNameToRuleOrCategoryIDs(allRulesForType)
	for _, rule := range resultRules {
		// If the rule is not from a builtin rule (i.e. PluginName() is not empty),
		// delete the plugin name from the map.
		if pluginName := rule.PluginName(); pluginName != "" {
			delete(unusedPluginNameToRuleIDs, pluginName)
		}
	}
	// We check additional check configs. If rules are set in the related check configs, then
	// the plugin is not considered unused. We check against all rules for all rule types.
	allRuleIDToRule, err := getIDToRuleOrCategory(allRules)
	if err != nil {
		return nil, err
	}
	allRuleIDToCategoryIDs, err := getRuleIDToCategoryIDs(allRules)
	if err != nil {
		return nil, err
	}
	allCategoryIDToRuleIDs := getCategoryIDToRuleIDs(allRuleIDToCategoryIDs)
	for _, checkConfig := range relatedCheckConfigs {
		checkConfigUseRuleIDs, err := transformRuleOrCategoryIDsToRuleIDs(
			checkConfig.UseIDsAndCategories(),
			allRuleIDToCategoryIDs,
			allCategoryIDToRuleIDs,
		)
		if err != nil {
			return nil, err
		}
		for _, checkConfigRuleID := range checkConfigUseRuleIDs {
			// If a rule from a non-builtin plugin is found, then we remove it from the unused plugins.
			if rule, ok := allRuleIDToRule[checkConfigRuleID]; rule.PluginName() != "" && ok {
				delete(unusedPluginNameToRuleIDs, rule.PluginName())
			}
		}
	}

	return &rulesConfig{
		RuleType:                ruleType,
		RuleIDs:                 xslices.Map(resultRules, Rule.ID),
		IgnoreRootPaths:         xslices.ToStructMap(ignoreRootPaths),
		IgnoreRuleIDToRootPaths: ignoreRuleIDToRootPathMap,
		ReferencedDeprecatedRuleIDToReplacementIDs:     referencedDeprecatedRuleIDToReplacementIDs,
		ReferencedDeprecatedCategoryIDToReplacementIDs: referencedDeprecatedCategoryIDToReplacementIDs,
		UnusedPluginNameToRuleIDs:                      unusedPluginNameToRuleIDs,
	}, nil
}

// *** JUST USED WITHIN THIS FILE ***

func warnReferencedDeprecatedIDs(logger *slog.Logger, configName string, rulesConfig *rulesConfig) {
	warnReferencedDeprecatedIDsForIDType(
		logger,
		configName,
		rulesConfig.ReferencedDeprecatedRuleIDToReplacementIDs,
		"Rule",
		"rules",
	)
	warnReferencedDeprecatedIDsForIDType(
		logger,
		configName,
		rulesConfig.ReferencedDeprecatedCategoryIDToReplacementIDs,
		"Category",
		"categories",
	)
}

func warnUnusedPlugins(logger *slog.Logger, configName string, rulesConfig *rulesConfig) {
	if len(rulesConfig.UnusedPluginNameToRuleIDs) == 0 {
		return
	}
	unusedPluginNames := xslices.MapKeysToSortedSlice(rulesConfig.UnusedPluginNameToRuleIDs)
	var sb strings.Builder
	_, _ = sb.WriteString("Your " + configName + " has plugins added which have no rules configured:\n\n")
	for _, unusedPluginName := range unusedPluginNames {
		_, _ = sb.WriteString("\t  - ")
		_, _ = sb.WriteString(unusedPluginName)
		_, _ = sb.WriteString(" (available rules: ")
		_, _ = sb.WriteString(strings.Join(rulesConfig.UnusedPluginNameToRuleIDs[unusedPluginName], ","))
		_, _ = sb.WriteString(")\n")
	}
	_, _ = sb.WriteString("\n\tThis is usually a configuration error. You must specify the rules or categories you want to use from this plugin.\n")
	_, _ = sb.WriteString("\tFor example (selecting one rule from each plugin):\n\n\t")
	_, _ = sb.WriteString(rulesConfig.RuleType.String())
	_, _ = sb.WriteString("\n\t  use:\n")
	for _, unusedPluginName := range unusedPluginNames {
		_, _ = sb.WriteString("\t    - ")
		// We assume that all values have at least one element given how we constructed this.
		// We know that the rule IDs are sorted, so this is deterministic.
		_, _ = sb.WriteString(rulesConfig.UnusedPluginNameToRuleIDs[unusedPluginName][0])
		_, _ = sb.WriteString("\n")
	}
	_, _ = sb.WriteString("\n\tIf you do not want to use these plugins, we recommend removing them from your configuration.")
	logger.Warn(sb.String())
}

func warnReferencedDeprecatedIDsForIDType(
	logger *slog.Logger,
	configName string,
	referencedDeprecatedIDToReplacementIDs map[string]map[string]struct{},
	capitalizedIDType string,
	pluralIDType string,
) {
	for _, deprecatedID := range xslices.MapKeysToSortedSlice(referencedDeprecatedIDToReplacementIDs) {
		replacementIDs := xslices.MapKeysToSortedSlice(referencedDeprecatedIDToReplacementIDs[deprecatedID])
		var replaceString string
		switch len(replacementIDs) {
		case 0:
		case 1:
			replaceString = fmt.Sprintf(" It has been replaced by %s %s.", strings.ToLower(capitalizedIDType), replacementIDs[0])
		default:
			replaceString = fmt.Sprintf(" It has been replaced by %s %s.", pluralIDType, strings.Join(replacementIDs, ", "))
		}
		var specialCallout string
		if deprecatedID == "DEFAULT" {
			specialCallout = fmt.Sprintf(`

	The concept of a default rule has been introduced. A default rule is a rule that will be run
	if no rules are explicitly configured in your %s. Run buf config ls-lint-rules or
	buf config ls-breaking-rules to see which rules are defaults. With this introduction, having a category
	also named DEFAULT is confusing, as while it happens that all the rules in the DEFAULT category
	are also default rules, the name has become overloaded.
`, configName)
		}
		logger.Warn(
			fmt.Sprintf(
				"%s %s referenced in your %s is deprecated.%s%s\n\tAs with all buf changes, this change is backwards-compatible: %s will continue to work.\n\tWe recommend replacing %s in your %s, but no action is immediately necessary.",
				capitalizedIDType,
				deprecatedID,
				configName,
				replaceString,
				specialCallout,
				deprecatedID,
				deprecatedID,
				configName,
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

func getPluginNameToRuleOrCategoryIDs[R RuleOrCategory](ruleOrCategories []R) map[string][]string {
	m := make(map[string][]string)
	for _, ruleOrCategory := range ruleOrCategories {
		if pluginName := ruleOrCategory.PluginName(); pluginName != "" {
			m[pluginName] = append(m[pluginName], ruleOrCategory.ID())
		}
	}
	for _, ruleOrCategoryIDs := range m {
		sort.Strings(ruleOrCategoryIDs)
	}
	return m
}

func getRuleIDToCategoryIDs(rules []Rule) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, rule := range rules {
		if _, ok := m[rule.ID()]; ok {
			return nil, syserror.Newf("duplicate rule ID: %q", rule.ID())
		}
		m[rule.ID()] = xslices.Map(rule.Categories(), check.Category.ID)
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
	return xslices.MapKeysToSortedSlice(ruleIDMap), nil
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
	return xslices.MapKeysToSortedSlice(undeprecatedRuleIDMap)
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
	return xslices.MapKeysToSortedSlice(rootPathMap), nil
}

func normalizeKeyToIgnoreRootPathMap[K comparable](
	keyToRootPaths map[K]map[string]struct{},
) (map[K]map[string]struct{}, error) {
	keyToNormalizedRootPathMap := make(map[K]map[string]struct{}, len(keyToRootPaths))
	for key, rootPathMap := range keyToRootPaths {
		rootPaths, err := normalizeIgnoreRootPaths(xslices.MapKeysToSortedSlice(rootPathMap))
		if err != nil {
			return nil, err
		}
		keyToNormalizedRootPathMap[key] = xslices.ToStructMap(rootPaths)
	}
	return keyToNormalizedRootPathMap, nil
}
