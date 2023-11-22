// Copyright 2020-2023 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

// GenerateManagedConfig is a managed mode configuration.
type GenerateManagedConfig interface {
	// Disables returns the disable rules in the configuration.
	Disables() []ManagedDisableRule
	// Overrides returns the override rules in the configuration.
	Overrides() []ManagedOverrideRule

	isGenerateManagedConfig()
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
	// mode for. This is guranteed to be empty is FileOption is not empty.
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

type generateManagedConfig struct {
	disables  []ManagedDisableRule
	overrides []ManagedOverrideRule
}

func newManagedOverrideRuleFromExternalV1(
	externalConfig externalGenerateManagedConfigV1,
) (GenerateManagedConfig, error) {
	if externalConfig.isEmpty() || !externalConfig.Enabled {
		return nil, nil
	}
	var (
		disables  []ManagedDisableRule
		overrides []ManagedOverrideRule
	)
	if externalCCEnableArenas := externalConfig.CcEnableArenas; externalCCEnableArenas != nil {
		override, err := newFileOptionOverrideRule(
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
		override, err := newFileOptionOverrideRule(
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
		override, err := newFileOptionOverrideRule(
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
			// TODO: resolve this: this message has been updated, compared to the one in bufgen/config.go:
			// "java_package_prefix setting requires a default value"
			return nil, errors.New("java_package_prefix must have a default value")
		}
		defaultOverride, err := newFileOptionOverrideRule(
			"",
			"",
			FileOptionJavaPackagePrefix,
			externalJavaPackagePrefix.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		javaPackagePrefixDisables, javaPackagePrefixOverrides, err := getDisablesAndOverrides(
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
		csharpNamespaceDisables, csharpNamespaceOverrides, err := getDisablesAndOverrides(
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
			return nil, errors.New("optimize_for must have a default value")
		}
		defaultOverride, err := newFileOptionOverrideRule(
			"",
			"",
			FileOptionOptimizeFor,
			externalOptimizeFor.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		optimizeForDisables, optimizeForOverrides, err := getDisablesAndOverrides(
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
		if externalGoPackagePrefix.Default != "" {
			return nil, errors.New("go_package_prefix must have a default value")
		}
		defaultOverride, err := newFileOptionOverrideRule(
			"",
			"",
			FileOptionGoPackagePrefix,
			externalGoPackagePrefix.Default,
		)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, defaultOverride)
		goPackagePrefixDisables, goPackagePrefixOverrides, err := getDisablesAndOverrides(
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
			defaultOverride, err := newFileOptionOverrideRule(
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
		objcClassPrefixDisables, objcClassPrefixOverrides, err := getDisablesAndOverrides(
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
		rubyPackageDisables, rubyPackageOverrides, err := getDisablesAndOverrides(
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
	for upperCaseFileOption, fileToOverride := range externalConfig.Override {
		lowerCaseFileOption := strings.ToLower(upperCaseFileOption)
		fileOption, ok := stringToFileOption[lowerCaseFileOption]
		if !ok {
			return nil, fmt.Errorf("%q is not a valid file option", upperCaseFileOption)
		}
		for filePath, override := range fileToOverride {
			normalizedFilePath, err := normalpath.NormalizeAndValidate(filePath)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to normalize import path: %s provided for override: %s",
					filePath,
					upperCaseFileOption,
				)
			}
			if filePath != normalizedFilePath {
				return nil, fmt.Errorf(
					"override can only take normalized import paths, invalid import path: %s provided for override: %s",
					filePath,
					upperCaseFileOption,
				)
			}
			var overrideValue interface{} = override
			switch fileOption {
			case FileOptionCcEnableArenas, FileOptionJavaMultipleFiles, FileOptionJavaStringCheckUtf8:
				parseOverrideValue, err := strconv.ParseBool(override)
				if err != nil {
					return nil, fmt.Errorf("")
				}
				overrideValue = parseOverrideValue
			}
			overrideRule, err := newFileOptionOverrideRule(
				filePath,
				"",
				fileOption,
				overrideValue,
			)
			if err != nil {
				return nil, err
			}
			overrides = append(overrides, overrideRule)
		}
	}
	return &generateManagedConfig{
		disables:  disables,
		overrides: overrides,
	}, nil
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
) (*managedDisableRule, error) {
	if path == "" && moduleFullName == "" && fieldName == "" && fileOption == FileOptionUnspecified && fieldOption == FieldOptionUnspecified {
		// This should never happen to parsing configs from provided by users.
		return nil, errors.New("empty disable rule is not allowed")
	}
	if fileOption != FileOptionUnspecified && fieldOption != FieldOptionUnspecified {
		return nil, errors.New("at most one of file_option and field_option can be specified")
	}
	if fieldName != "" && fileOption != FileOptionUnspecified {
		return nil, errors.New("cannot disable a file option for a field")
	}
	// TODO: validate path here? Was it validated in v1/main?
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
	// TODO: validate path here? Was it validated in v1/main?
	if moduleFullName != "" {
		if _, err := bufmodule.ParseModuleFullName(moduleFullName); err != nil {
			return nil, err
		}
	}
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
) (*managedOverrideRule, error) {
	// TODO: validate path here? Was it validated in v1/main?
	if moduleFullName != "" {
		if _, err := bufmodule.ParseModuleFullName(moduleFullName); err != nil {
			return nil, err
		}
	}
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

func getDisablesAndOverrides(
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
	for overrideModuleFullName, overrideValue := range moduleFullNameToOverride {
		if _, err := bufmodule.ParseModuleFullName(overrideModuleFullName); err != nil {
			return nil, nil, err
		}
		if _, ok := seenExceptModuleFullNames[overrideModuleFullName]; ok {
			return nil, nil, fmt.Errorf("override %q is already defined as an except", overrideModuleFullName)
		}
		override, err := newFileOptionOverrideRule(
			"",
			overrideModuleFullName,
			overrideFileOption,
			overrideValue,
		)
		if err != nil {
			return nil, nil, err
		}
		overrides = append(overrides, override)
	}
	return disables, overrides, nil
}
