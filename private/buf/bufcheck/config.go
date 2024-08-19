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

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckopt"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
)

const lintCommentIgnorePrefix = "buf:lint:ignore"

// config is the check config.
//
// This should only be built via a configSpec. If we were exposing this API publicly, we would
// enforce this.
type config struct {
	// RuleIDs contains the specific RuleIDs to use.
	//
	// If not set, all default Rules should be used.
	//
	// Note that ignoreAnnotation does not need to take this into account as the plugins
	// themselves will only return RuleIDs in this list TODO make sure bufplugin-go
	// validates this and that this is documented.
	RuleIDs []string
	// DefaultOptions are the options that should be passed to the default check.Client.
	//
	// Do not pass these to plugin check.Clients. Use options from checkClientSpecs instead.
	// Will never be nil.
	DefaultOptions check.Options

	IgnoreRootPaths     map[string]struct{}
	IgnoreIDToRootPaths map[string]map[string]struct{}

	AllowCommentIgnores    bool
	IgnoreUnstablePackages bool

	CommentIgnorePrefix string
	ExcludeImports      bool
}

// Only RuleIDs, IgnoreRootPaths,  IgnoreIDToRootPaths will be set. Options has no meaning.
func configForCheckConfig(checkConfig bufconfig.CheckConfig, allRules []check.Rule) (*config, error) {
	return configSpecForCheckConfig(checkConfig).newConfig(allRules)
}

func configForLintConfig(lintConfig bufconfig.LintConfig, allRules []check.Rule) (*config, error) {
	return configSpecForLintConfig(lintConfig).newConfig(allRules)
}

func configForBreakingConfig(breakingConfig bufconfig.BreakingConfig, allRules []check.Rule, excludeImports bool) (*config, error) {
	return configSpecForBreakingConfig(breakingConfig, excludeImports).newConfig(allRules)
}

// *** BELOW THIS LINE SHOULD ONLY BE USED BY THIS FILE ***

// configSpec is a config spec.
type configSpec struct {
	// May contain deprecated IDs.
	Use []string
	// May contain deprecated IDs.
	Except []string

	IgnoreRootPaths []string
	// May contain deprecated IDs.
	IgnoreIDOrCategoryToRootPaths map[string][]string

	AllowCommentIgnores    bool
	IgnoreUnstablePackages bool

	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string

	CommentIgnorePrefix string
	ExcludeImports      bool
}

func configSpecForCheckConfig(checkConfig bufconfig.CheckConfig) *configSpec {
	return &configSpec{
		Use:                                  checkConfig.UseIDsAndCategories(),
		Except:                               checkConfig.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      checkConfig.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        checkConfig.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  false,
		IgnoreUnstablePackages:               false,
		EnumZeroValueSuffix:                  "",
		RPCAllowSameRequestResponse:          false,
		RPCAllowGoogleProtobufEmptyRequests:  false,
		RPCAllowGoogleProtobufEmptyResponses: false,
		ServiceSuffix:                        "",
		CommentIgnorePrefix:                  "",
		ExcludeImports:                       false,
	}
}

func configSpecForLintConfig(lintConfig bufconfig.LintConfig) *configSpec {
	return &configSpec{
		Use:                                  lintConfig.UseIDsAndCategories(),
		Except:                               lintConfig.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      lintConfig.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        lintConfig.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  lintConfig.AllowCommentIgnores(),
		IgnoreUnstablePackages:               false,
		EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
		RPCAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
		RPCAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
		RPCAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyResponses(),
		ServiceSuffix:                        lintConfig.ServiceSuffix(),
		CommentIgnorePrefix:                  lintCommentIgnorePrefix,
		ExcludeImports:                       false,
	}
}

func configSpecForBreakingConfig(breakingConfig bufconfig.BreakingConfig, excludeImports bool) *configSpec {
	return &configSpec{
		Use:                                  breakingConfig.UseIDsAndCategories(),
		Except:                               breakingConfig.ExceptIDsAndCategories(),
		IgnoreRootPaths:                      breakingConfig.IgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        breakingConfig.IgnoreIDOrCategoryToPaths(),
		AllowCommentIgnores:                  false,
		IgnoreUnstablePackages:               breakingConfig.IgnoreUnstablePackages(),
		EnumZeroValueSuffix:                  "",
		RPCAllowSameRequestResponse:          false,
		RPCAllowGoogleProtobufEmptyRequests:  false,
		RPCAllowGoogleProtobufEmptyResponses: false,
		ServiceSuffix:                        "",
		CommentIgnorePrefix:                  "",
		ExcludeImports:                       excludeImports,
	}
}

// newConfig returns a new Config.
func (b *configSpec) newConfig(allRules []check.Rule) (*config, error) {
	// transformDeprecated should always be true if building a Config for a Runner.
	// TODO: Evaluate whether we still need this after the refactor. Keeping logic
	// around for now
	transformDeprecated := true

	// this checks that there are not duplicate IDs for a given revision
	// which would be a system error
	idToRule, err := getIDToRule(allRules)
	if err != nil {
		return nil, err
	}
	deprecatedIDToReplacementIDs, err := getDeprecatedIDToReplacementIDs(idToRule)
	if err != nil {
		return nil, err
	}
	idToCategories, err := getIDToCategories(allRules)
	if err != nil {
		return nil, err
	}
	categoryToIDs := getCategoryToIDs(idToCategories)

	// These may both be empty, and that's OK.
	b.Use = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(b.Use)
	b.Except = stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(b.Except)
	// If Use is empty but Except is not, we need to populate Use with the default Rule IDs
	// so that we can then remove the exceptions.
	//
	// If both Use and Except are empty, we want Use to remain empty, so that RuleIDs
	// is empty, which results in default rules being used for the builtin CheckClient.
	if len(b.Use) == 0 && len(b.Except) > 0 {
		b.Use = slicesext.Map(slicesext.Filter(allRules, check.Rule.IsDefault), check.Rule.ID)
	}

	// Will be empty if Use and Except are both empty.
	var resultRules []check.Rule
	if len(b.Use) > 0 || len(b.Except) > 0 {
		useIDMap, err := transformToIDMap(b.Use, idToCategories, categoryToIDs)
		if err != nil {
			return nil, err
		}
		if transformDeprecated {
			useIDMap = transformIDsToUndeprecated(useIDMap, deprecatedIDToReplacementIDs)
		}
		exceptIDMap, err := transformToIDMap(b.Except, idToCategories, categoryToIDs)
		if err != nil {
			return nil, err
		}
		if transformDeprecated {
			exceptIDMap = transformIDsToUndeprecated(exceptIDMap, deprecatedIDToReplacementIDs)
		}

		// this removes duplicates
		// we already know that a given rule with the same ID is equivalent
		resultIDToRule := make(map[string]check.Rule)

		for id := range useIDMap {
			rule, ok := idToRule[id]
			if !ok {
				return nil, fmt.Errorf("%q is not a known id after verification", id)
			}
			resultIDToRule[rule.ID()] = rule
		}
		for id := range exceptIDMap {
			if _, ok := idToRule[id]; !ok {
				return nil, fmt.Errorf("%q is not a known id after verification", id)
			}
			delete(resultIDToRule, id)
		}

		resultRules = make([]check.Rule, 0, len(resultIDToRule))
		for _, rule := range resultIDToRule {
			resultRules = append(resultRules, rule)
		}
	}

	ignoreIDToRootPathsUnnormalized, err := transformToIDToListMap(
		b.IgnoreIDOrCategoryToRootPaths,
		idToCategories,
		categoryToIDs,
	)
	if err != nil {
		return nil, err
	}
	if transformDeprecated {
		ignoreIDToRootPathsUnnormalized = transformIDsToUndeprecated(
			ignoreIDToRootPathsUnnormalized,
			deprecatedIDToReplacementIDs,
		)
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

	ignoreRootPaths := make(map[string]struct{}, len(b.IgnoreRootPaths))
	for _, rootPath := range b.IgnoreRootPaths {
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

	optionsSpec := &bufcheckopt.OptionsSpec{
		EnumZeroValueSuffix:                  b.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          b.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  b.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: b.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        b.ServiceSuffix,
	}
	if b.CommentIgnorePrefix != "" {
		optionsSpec.CommentExcludes = []string{b.CommentIgnorePrefix}
	}
	options, err := optionsSpec.ToOptions()
	if err != nil {
		return nil, err
	}

	return &config{
		RuleIDs:                slicesext.Map(resultRules, check.Rule.ID),
		DefaultOptions:         options,
		IgnoreIDToRootPaths:    ignoreIDToRootPaths,
		IgnoreRootPaths:        ignoreRootPaths,
		AllowCommentIgnores:    b.AllowCommentIgnores,
		IgnoreUnstablePackages: b.IgnoreUnstablePackages,
		CommentIgnorePrefix:    b.CommentIgnorePrefix,
		ExcludeImports:         b.ExcludeImports,
	}, nil
}

func transformIDsToUndeprecated[T any](idToValue map[string]T, deprecatedIDToReplacementIDs map[string][]string) map[string]T {
	undeprecatedIDToValue := make(map[string]T, len(idToValue))
	for id, value := range idToValue {
		replacementIDs, ok := deprecatedIDToReplacementIDs[id]
		if ok {
			// May iterate over empty.
			for _, replacementID := range replacementIDs {
				undeprecatedIDToValue[replacementID] = value
			}
		} else {
			undeprecatedIDToValue[id] = value
		}
	}
	return undeprecatedIDToValue
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
			// handles empty category as well
			categoryToIDs[category] = append(categoryToIDs[category], id)
		}
	}
	return categoryToIDs
}

// []string{} as a value represents that the ID is deprecated but has no replacements.
func getDeprecatedIDToReplacementIDs(idToRule map[string]check.Rule) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, rule := range idToRule {
		if rule.Deprecated() {
			replacementIDs := rule.ReplacementIDs()
			if replacementIDs == nil {
				replacementIDs = []string{}
			}
			for _, replacementID := range replacementIDs {
				if _, ok := idToRule[replacementID]; !ok {
					return nil, syserror.Newf("unknown rule given as a replacement ID: %q", replacementID)
				}
			}
			m[rule.ID()] = replacementIDs
		}
	}
	return m, nil
}

func getIDToRule(rules []check.Rule) (map[string]check.Rule, error) {
	m := make(map[string]check.Rule)
	for _, rule := range rules {
		if _, ok := m[rule.ID()]; ok {
			return nil, syserror.Newf("duplicate rule ID: %q", rule.ID())
		}
		m[rule.ID()] = rule
	}
	return m, nil
}

func getIDToCategories(rules []check.Rule) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, rule := range rules {
		if _, ok := m[rule.ID()]; ok {
			return nil, syserror.Newf("duplicate rule ID: %q", rule.ID())
		}
		m[rule.ID()] = rule.Categories()
	}
	return m, nil
}
