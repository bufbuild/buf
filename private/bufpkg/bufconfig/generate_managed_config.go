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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// GenerateManagedConfig is a managed mode configuration.
type GenerateManagedConfig interface {
	// Enabled returns whether managed mode is enabled.
	Enabled() bool
	// Disables returns the disable rules in the configuration.
	Disables() []ManagedDisableRule
	// Overrides returns the override rules in the configuration.
	Overrides() []ManagedOverrideRule

	isGenerateManagedConfig()
}

// NewGenerateManagedConfig returns a new GenerateManagedConfig.
func NewGenerateManagedConfig(
	enabled bool,
	disables []ManagedDisableRule,
	overrides []ManagedOverrideRule,
) GenerateManagedConfig {
	return &generateManagedConfig{
		enabled:   enabled,
		disables:  disables,
		overrides: overrides,
	}
}

// ManagedDisableRule is a disable rule. A disable rule describes:
//
//   - The options to not modify. If not specified, it means all options (both
//     file options and field options) are not modified.
//   - The files/fields for which these options are not modified. If not specified,
//     it means for all files/fields the specified options are not modified.
//
// A ManagedDisableRule is guaranteed to specify at least one of the two aspects.
// i.e. At least one of Path, ModuleFullName, FieldName, FileOption and
// FieldOption is not empty. A rule can disable all options for certain files/fields,
// disable certains options for all files/fields, or disable certain options for
// certain files/fields. To disable all options for all files/fields, turn off managed mode.
type ManagedDisableRule interface {
	// Path returns the file path, relative to its module, to disable managed mode for.
	Path() string
	// ModuleFullName returns the full name string of the module to disable
	// managed mode for.
	ModuleFullName() string
	// FieldName returns the fully qualified name for the field to disable managed
	// mode for. This is guaranteed to be empty if FileOption is not empty.
	FieldName() string
	// FileOption returns the file option to disable managed mode for. This is
	// guaranteed to be empty if FieldName is not empty.
	FileOption() FileOption
	// FieldOption returns the field option to disalbe managed mode for.
	FieldOption() FieldOption

	isManagedDisableRule()
}

// NewManagedDisableRule returns a new ManagedDisableRule.
func NewManagedDisableRule(
	path string,
	moduleFullName string,
	fieldName string,
	fileOption FileOption,
	fieldOption FieldOption,
) (ManagedDisableRule, error) {
	return newDisableRule(
		path,
		moduleFullName,
		fieldName,
		fileOption,
		fieldOption,
	)
}

// ManagedOverrideRule is an override rule. An override describes:
//
//   - The options to modify. Exactly one of FileOption and FieldOption is not empty.
//   - The value to modify these options with.
//   - The files/fields for which the options are modified. If all of Path, ModuleFullName
//   - or FieldName are empty, all files/fields are modified. Otherwise, only
//     file/fields that match the specified Path, ModuleFullName and FieldName
//     is modified.
type ManagedOverrideRule interface {
	// Path is the file path, relative to its module, to disable managed mode for.
	Path() string
	// ModuleFullName is the full name string of the module to disable
	// managed mode for.
	ModuleFullName() string
	// FieldName is the fully qualified name for the field to disable managed
	// mode for. This is guaranteed to be empty is FileOption is not empty.
	FieldName() string
	// FileOption returns the file option to disable managed mode for. This is
	// guaranteed to be empty (FileOptionUnspecified) if FieldName is empty.
	FileOption() FileOption
	// FieldOption returns the field option to disable managed mode for.
	FieldOption() FieldOption
	// Value returns the override value.
	Value() interface{}

	isManagedOverrideRule()
}

// NewFieldOptionOverrideRule returns a new ManagedOverrideRule for a file option.
func NewManagedOverrideRuleForFileOption(
	path string,
	moduleFullName string,
	fileOption FileOption,
	value interface{},
) (*managedOverrideRule, error) {
	return newFileOptionOverrideRule(
		path,
		moduleFullName,
		fileOption,
		value,
	)
}

// NewManagedOverrideRuleForFieldOption returns a new ManagedOverrideRule for a field option.
func NewManagedOverrideRuleForFieldOption(
	path string,
	moduleFullName string,
	fieldName string,
	fieldOption FieldOption,
	value interface{},
) (ManagedOverrideRule, error) {
	return newFieldOptionOverrideRule(
		path,
		moduleFullName,
		fieldName,
		fieldOption,
		value,
	)
}

// *** PRIVATE ***

type generateManagedConfig struct {
	enabled   bool
	disables  []ManagedDisableRule
	overrides []ManagedOverrideRule
}

func newManagedConfigFromExternalV1Beta1(
	enabled bool,
	externalConfig externalGenerateManagedConfigV1Beta1,
) (GenerateManagedConfig, error) {
	var (
		overrides []ManagedOverrideRule
	)
	if externalCCEnableArenas := externalConfig.CcEnableArenas; externalCCEnableArenas != nil {
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionCcEnableArenas,
			*externalCCEnableArenas,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if externalJavaMultipleFiles := externalConfig.JavaMultipleFiles; externalJavaMultipleFiles != nil {
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionJavaMultipleFiles,
			*externalJavaMultipleFiles,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if externalOptimizeFor := externalConfig.OptimizeFor; externalOptimizeFor != "" {
		defaultOverride, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionOptimizeFor,
			externalOptimizeFor,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
	}
	return &generateManagedConfig{
		enabled:   enabled,
		overrides: overrides,
	}, nil
}

func newManagedConfigFromExternalV1(
	externalConfig externalGenerateManagedConfigV1,
) (GenerateManagedConfig, error) {
	var (
		disables  []ManagedDisableRule
		overrides []ManagedOverrideRule
	)
	if externalCCEnableArenas := externalConfig.CcEnableArenas; externalCCEnableArenas != nil {
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionCcEnableArenas,
			*externalCCEnableArenas,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if externalJavaMultipleFiles := externalConfig.JavaMultipleFiles; externalJavaMultipleFiles != nil {
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionJavaMultipleFiles,
			*externalJavaMultipleFiles,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if externalJavaStringCheckUtf8 := externalConfig.JavaStringCheckUtf8; externalJavaStringCheckUtf8 != nil {
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionJavaStringCheckUtf8,
			*externalJavaStringCheckUtf8,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	if externalJavaPackagePrefix := externalConfig.JavaPackagePrefix; !externalJavaPackagePrefix.isEmpty() {
		if externalJavaPackagePrefix.Default == "" {
			return nil, errors.New("java_package_prefix requires a default value")
		}
		defaultOverride, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionJavaPackagePrefix,
			externalJavaPackagePrefix.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		javaPackagePrefixDisables, javaPackagePrefixOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionJavaPackage,
			externalJavaPackagePrefix.Except,
			FileOptionJavaPackagePrefix,
			externalJavaPackagePrefix.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, javaPackagePrefixDisables...)
		overrides = append(overrides, javaPackagePrefixOverrides...)
	}
	if externalCsharpNamespace := externalConfig.CsharpNamespace; !externalCsharpNamespace.isEmpty() {
		csharpNamespaceDisables, csharpNamespaceOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionCsharpNamespace,
			externalCsharpNamespace.Except,
			FileOptionCsharpNamespace,
			externalCsharpNamespace.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, csharpNamespaceDisables...)
		overrides = append(overrides, csharpNamespaceOverrides...)
	}
	if externalOptimizeFor := externalConfig.OptimizeFor; !externalOptimizeFor.isEmpty() {
		if externalOptimizeFor.Default == "" {
			return nil, errors.New("optimize_for requires a default value")
		}
		defaultOverride, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionOptimizeFor,
			externalOptimizeFor.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		optimizeForDisables, optimizeForOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionOptimizeFor,
			externalOptimizeFor.Except,
			FileOptionOptimizeFor,
			externalOptimizeFor.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, optimizeForDisables...)
		overrides = append(overrides, optimizeForOverrides...)
	}
	if externalGoPackagePrefix := externalConfig.GoPackagePrefix; !externalGoPackagePrefix.isEmpty() {
		if externalGoPackagePrefix.Default == "" {
			return nil, errors.New("go_package_prefix requires a default value")
		}
		defaultOverride, err := NewManagedOverrideRuleForFileOption(
			"",
			"",
			FileOptionGoPackagePrefix,
			externalGoPackagePrefix.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		goPackagePrefixDisables, goPackagePrefixOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionGoPackage,
			externalGoPackagePrefix.Except,
			FileOptionGoPackagePrefix,
			externalGoPackagePrefix.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, goPackagePrefixDisables...)
		overrides = append(overrides, goPackagePrefixOverrides...)
	}
	if externalObjcClassPrefix := externalConfig.ObjcClassPrefix; !externalObjcClassPrefix.isEmpty() {
		if externalObjcClassPrefix.Default != "" {
			// objc class prefix allows empty default
			defaultOverride, err := NewManagedOverrideRuleForFileOption(
				"",
				"",
				FileOptionObjcClassPrefix,
				externalObjcClassPrefix.Default,
			)
			if err != nil {
				return nil, err
			}
			overrides = append(overrides, defaultOverride)
		}
		objcClassPrefixDisables, objcClassPrefixOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionObjcClassPrefix,
			externalObjcClassPrefix.Except,
			FileOptionObjcClassPrefix,
			externalObjcClassPrefix.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, objcClassPrefixDisables...)
		overrides = append(overrides, objcClassPrefixOverrides...)
	}
	if externalRubyPackage := externalConfig.RubyPackage; !externalRubyPackage.isEmpty() {
		rubyPackageDisables, rubyPackageOverrides, err := disablesAndOverridesFromExceptAndOverrideV1(
			FileOptionRubyPackage,
			externalRubyPackage.Except,
			FileOptionRubyPackage,
			externalRubyPackage.Override,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, rubyPackageDisables...)
		overrides = append(overrides, rubyPackageOverrides...)
	}
	perFileOverrides, err := overrideRulesForPerFileOverridesV1(externalConfig.Override)
	if err != nil {
		return nil, err
	}
	overrides = append(overrides, perFileOverrides...)
	return &generateManagedConfig{
		enabled:   externalConfig.Enabled,
		disables:  disables,
		overrides: overrides,
	}, nil
}

func newManagedConfigFromExternalV2(
	externalConfig externalGenerateManagedConfigV2,
) (GenerateManagedConfig, error) {
	var disables []ManagedDisableRule
	var overrides []ManagedOverrideRule
	for _, externalDisableConfig := range externalConfig.Disable {
		var (
			fileOption  FileOption
			fieldOption FieldOption
			err         error
		)
		if externalDisableConfig.FileOption != "" {
			fileOption, err = parseFileOption(externalDisableConfig.FileOption)
			if err != nil {
				return nil, err
			}
		}
		if externalDisableConfig.FieldOption != "" {
			fieldOption, err = parseFieldOption(externalDisableConfig.FieldOption)
			if err != nil {
				return nil, err
			}
		}
		disable, err := newDisableRule(
			externalDisableConfig.Path,
			externalDisableConfig.Module,
			externalDisableConfig.Field,
			fileOption,
			fieldOption,
		)
		if err != nil {
			return nil, err
		}
		disables = append(disables, disable)
	}
	for _, externalOverrideConfig := range externalConfig.Override {
		if externalOverrideConfig.FileOption == "" && externalOverrideConfig.FieldOption == "" {
			return nil, errors.New("must set file_option or field_option for an override")
		}
		if externalOverrideConfig.FileOption != "" && externalOverrideConfig.FieldOption != "" {
			return nil, errors.New("exactly one of file_option and field_option must be set for an override")
		}
		if externalOverrideConfig.Value == nil {
			return nil, errors.New("must set value for an override")
		}
		if externalOverrideConfig.FieldOption != "" {
			fieldOption, err := parseFieldOption(externalOverrideConfig.FieldOption)
			if err != nil {
				return nil, err
			}
			override, err := NewManagedOverrideRuleForFieldOption(
				externalOverrideConfig.Path,
				externalOverrideConfig.Module,
				externalOverrideConfig.Field,
				fieldOption,
				externalOverrideConfig.Value,
			)
			if err != nil {
				return nil, err
			}
			overrides = append(overrides, override)
			continue
		}
		if externalOverrideConfig.Field != "" {
			return nil, errors.New("must not set field for a file_option override")
		}
		fileOption, err := parseFileOption(externalOverrideConfig.FileOption)
		if err != nil {
			return nil, err
		}
		override, err := NewManagedOverrideRuleForFileOption(
			externalOverrideConfig.Path,
			externalOverrideConfig.Module,
			fileOption,
			externalOverrideConfig.Value,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, override)
	}
	return &generateManagedConfig{
		enabled:   externalConfig.Enabled,
		disables:  disables,
		overrides: overrides,
	}, nil
}

func (g *generateManagedConfig) Enabled() bool {
	return g.enabled
}

func (g *generateManagedConfig) Disables() []ManagedDisableRule {
	return g.disables
}

func (g *generateManagedConfig) Overrides() []ManagedOverrideRule {
	return g.overrides
}

func (g *generateManagedConfig) isGenerateManagedConfig() {}

type managedDisableRule struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     FileOption
	fieldOption    FieldOption
}

func newDisableRule(
	path string,
	moduleFullName string,
	fieldName string,
	fileOption FileOption,
	fieldOption FieldOption,
) (ManagedDisableRule, error) {
	if path == "" && moduleFullName == "" && fieldName == "" && fileOption == FileOptionUnspecified && fieldOption == FieldOptionUnspecified {
		return nil, errors.New("empty disable rule is not allowed")
	}
	if fieldName != "" && fileOption != FileOptionUnspecified {
		return nil, errors.New("cannot disable a file option for a field")
	}
	if fileOption != FileOptionUnspecified && fieldOption != FieldOptionUnspecified {
		return nil, errors.New("at most one of file_option and field_option can be specified")
	}
	if path != "" {
		if err := validatePath(path); err != nil {
			return nil, fmt.Errorf("invalid path for disable rule: %w", err)
		}
	}
	if moduleFullName != "" {
		if _, err := bufmodule.ParseModuleFullName(moduleFullName); err != nil {
			return nil, err
		}
	}
	return &managedDisableRule{
		path:           path,
		moduleFullName: moduleFullName,
		fieldName:      fieldName,
		fileOption:     fileOption,
		fieldOption:    fieldOption,
	}, nil
}

func (m *managedDisableRule) Path() string {
	return m.path
}

func (m *managedDisableRule) ModuleFullName() string {
	return m.moduleFullName
}

func (m *managedDisableRule) FieldName() string {
	return m.fieldName
}

func (m *managedDisableRule) FileOption() FileOption {
	return m.fileOption
}

func (m *managedDisableRule) FieldOption() FieldOption {
	return m.fieldOption
}

func (m *managedDisableRule) isManagedDisableRule() {}

type managedOverrideRule struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     FileOption
	fieldOption    FieldOption
	value          interface{}
}

func newFileOptionOverrideRule(
	path string,
	moduleFullName string,
	fileOption FileOption,
	value interface{},
) (*managedOverrideRule, error) {
	// All valid file options have a parse func. This lookup implicitly validates the option.
	parseOverrideValueFunc, ok := fileOptionToParseOverrideValueFunc[fileOption]
	if !ok {
		return nil, fmt.Errorf("invalid fileOption: %v", fileOption)
	}
	if value == nil {
		return nil, fmt.Errorf("value must be specified for override")
	}
	parsedValue, err := parseOverrideValueFunc(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value %v for %v: %w", value, fileOption, err)
	}
	if moduleFullName != "" {
		if _, err := bufmodule.ParseModuleFullName(moduleFullName); err != nil {
			return nil, fmt.Errorf("invalid module name for %v override: %w", fileOption, err)
		}
	}
	if path != "" {
		if err := validatePath(path); err != nil {
			return nil, fmt.Errorf("invalid path for %v override: %w", fileOption, err)
		}
	}
	return &managedOverrideRule{
		path:           path,
		moduleFullName: moduleFullName,
		fileOption:     fileOption,
		value:          parsedValue,
	}, nil
}

func newFieldOptionOverrideRule(
	path string,
	moduleFullName string,
	fieldName string,
	fieldOption FieldOption,
	value interface{},
) (ManagedOverrideRule, error) {
	// All valid field options have a parse func. This lookup implicitly validates the option.
	parseOverrideValueFunc, ok := fieldOptionToParseOverrideValueFunc[fieldOption]
	if !ok {
		return nil, fmt.Errorf("invalid fieldOption: %v", fieldOption)
	}
	if value == nil {
		return nil, fmt.Errorf("value must be specified for override")
	}
	parsedValue, err := parseOverrideValueFunc(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value %v for %v: %w", value, fieldOption, err)
	}
	if moduleFullName != "" {
		if _, err := bufmodule.ParseModuleFullName(moduleFullName); err != nil {
			return nil, fmt.Errorf("invalid module name for %v override: %w", fieldOption, err)
		}
	}
	if path != "" {
		if err := validatePath(path); err != nil {
			return nil, fmt.Errorf("invalid path for %v override: %w", fieldOption, err)
		}
	}
	return &managedOverrideRule{
		path:           path,
		moduleFullName: moduleFullName,
		fieldName:      fieldName,
		fieldOption:    fieldOption,
		value:          parsedValue,
	}, nil
}

func (m *managedOverrideRule) Path() string {
	return m.path
}

func (m *managedOverrideRule) ModuleFullName() string {
	return m.moduleFullName
}

func (m *managedOverrideRule) FieldName() string {
	return m.fieldName
}

func (m *managedOverrideRule) FileOption() FileOption {
	return m.fileOption
}

func (m *managedOverrideRule) FieldOption() FieldOption {
	return m.fieldOption
}

func (m *managedOverrideRule) Value() interface{} {
	return m.value
}

func (m *managedOverrideRule) isManagedOverrideRule() {}

func disablesAndOverridesFromExceptAndOverrideV1(
	exceptFileOption FileOption,
	exceptModuleFullNames []string,
	overrideFileOption FileOption,
	moduleFullNameToOverride map[string]string,
) ([]ManagedDisableRule, []ManagedOverrideRule, error) {
	var (
		disables  []ManagedDisableRule
		overrides []ManagedOverrideRule
	)
	seenExceptModuleFullNames := make(map[string]struct{}, len(exceptModuleFullNames))
	for _, exceptModuleFullName := range exceptModuleFullNames {
		if _, err := bufmodule.ParseModuleFullName(exceptModuleFullName); err != nil {
			return nil, nil, err
		}
		if _, ok := seenExceptModuleFullNames[exceptModuleFullName]; ok {
			return nil, nil, fmt.Errorf("%q is defined multiple times in except", exceptModuleFullName)
		}
		seenExceptModuleFullNames[exceptModuleFullName] = struct{}{}
		disable, err := newDisableRule(
			"",
			exceptModuleFullName,
			"",
			exceptFileOption,
			FieldOptionUnspecified,
		)
		if err != nil {
			return nil, nil, err
		}
		disables = append(disables, disable)
	}
	// Sort by keys for deterministic order.
	sortedModuleFullNames := slicesext.MapKeysToSortedSlice(moduleFullNameToOverride)
	for _, overrideModuleFullName := range sortedModuleFullNames {
		if _, err := bufmodule.ParseModuleFullName(overrideModuleFullName); err != nil {
			return nil, nil, err
		}
		if _, ok := seenExceptModuleFullNames[overrideModuleFullName]; ok {
			return nil, nil, fmt.Errorf("override %q is already defined as an except", overrideModuleFullName)
		}
		override, err := NewManagedOverrideRuleForFileOption(
			"",
			overrideModuleFullName,
			overrideFileOption,
			moduleFullNameToOverride[overrideModuleFullName],
		)
		if err != nil {
			return nil, nil, err
		}
		overrides = append(overrides, override)
	}
	return disables, overrides, nil
}

func overrideRulesForPerFileOverridesV1(
	fileOptionToFilePathToOverride map[string]map[string]string,
) ([]ManagedOverrideRule, error) {
	var overrideRules []ManagedOverrideRule
	sortedFileOptionStrings := slicesext.MapKeysToSortedSlice(fileOptionToFilePathToOverride)
	for _, fileOptionString := range sortedFileOptionStrings {
		fileOption, ok := stringToFileOption[strings.ToLower(fileOptionString)]
		if !ok {
			return nil, fmt.Errorf("%q is not a valid file option", fileOptionString)
		}
		filePathToOverride := fileOptionToFilePathToOverride[fileOptionString]
		sortedFilePaths := slicesext.MapKeysToSortedSlice(filePathToOverride)
		for _, filePath := range sortedFilePaths {
			err := validatePath(filePath)
			if err != nil {
				return nil, fmt.Errorf("invalid import path for override %s: %w", fileOptionString, err)
			}
			overrideString := filePathToOverride[filePath]
			var overrideValue interface{} = overrideString
			switch fileOption {
			case FileOptionCcEnableArenas, FileOptionJavaMultipleFiles, FileOptionJavaStringCheckUtf8:
				overrideValue, err = strconv.ParseBool(overrideString)
				if err != nil {
					return nil, fmt.Errorf("")
				}
			}
			overrideRule, err := NewManagedOverrideRuleForFileOption(
				filePath,
				"",
				fileOption,
				overrideValue,
			)
			if err != nil {
				return nil, err
			}
			overrideRules = append(overrideRules, overrideRule)
		}
	}
	return overrideRules, nil
}

func newExternalManagedConfigV2FromGenerateManagedConfig(
	managedConfig GenerateManagedConfig,
) (externalGenerateManagedConfigV2, error) {
	if managedConfig == nil {
		return externalGenerateManagedConfigV2{}, nil
	}
	var externalDisables []externalManagedDisableConfigV2
	for _, disable := range managedConfig.Disables() {
		var fileOptionName string
		if disable.FileOption() != FileOptionUnspecified {
			fileOptionName = disable.FileOption().String()
		}
		var fieldOptionName string
		if disable.FieldOption() != FieldOptionUnspecified {
			fieldOptionName = disable.FieldOption().String()
		}
		externalDisables = append(
			externalDisables,
			externalManagedDisableConfigV2{
				FileOption:  fileOptionName,
				FieldOption: fieldOptionName,
				Module:      disable.ModuleFullName(),
				Path:        disable.Path(),
				Field:       disable.FieldName(),
			},
		)
	}
	var externalOverrides []externalManagedOverrideConfigV2
	for _, override := range managedConfig.Overrides() {
		var fileOptionName string
		if override.FileOption() != FileOptionUnspecified {
			fileOptionName = override.FileOption().String()
		}
		var fieldOptionName string
		if override.FieldOption() != FieldOptionUnspecified {
			fieldOptionName = override.FieldOption().String()
		}
		value, err := getOverrideValue(fileOptionName, fieldOptionName, override.Value())
		if err != nil {
			return externalGenerateManagedConfigV2{}, err
		}
		externalOverrides = append(
			externalOverrides,
			externalManagedOverrideConfigV2{
				FileOption:  fileOptionName,
				FieldOption: fieldOptionName,
				Module:      override.ModuleFullName(),
				Path:        override.Path(),
				Field:       override.FieldName(),
				Value:       value,
			},
		)
	}
	return externalGenerateManagedConfigV2{
		Enabled:  managedConfig.Enabled(),
		Disable:  externalDisables,
		Override: externalOverrides,
	}, nil
}

func validatePath(path string) error {
	normalizedPath, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return err
	}
	if path != normalizedPath {
		return fmt.Errorf(
			"path %q in your configuration must be relative and use '/' as the path separator",
			path,
		)
	}
	return nil
}
