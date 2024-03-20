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

	"google.golang.org/protobuf/types/descriptorpb"
)

// FileOption is a file option.
type FileOption int

const (
	// FileOptionUnspecified is an unspecified file option.
	FileOptionUnspecified FileOption = iota
	// FileOptionJavaPackage is the file option java_package.
	FileOptionJavaPackage
	// FileOptionJavaPackagePrefix is the file option java_package_prefix.
	FileOptionJavaPackagePrefix
	// FileOptionJavaPackageSuffix is the file option java_package_suffix.
	FileOptionJavaPackageSuffix
	// FileOptionJavaOuterClassname is the file option java_outer_classname.
	FileOptionJavaOuterClassname
	// FileOptionJavaMultipleFiles is the file option java_multiple_files.
	FileOptionJavaMultipleFiles
	// FileOptionJavaStringCheckUtf8 is the file option java_string_check_utf8.
	FileOptionJavaStringCheckUtf8
	// FileOptionOptimizeFor is the file option optimize_for.
	FileOptionOptimizeFor
	// FileOptionGoPackage is the file option go_package.
	FileOptionGoPackage
	// FileOptionGoPackagePrefix is the file option go_package_prefix.
	FileOptionGoPackagePrefix
	// FileOptionCcEnableArenas is the file option cc_enable_arenas.
	FileOptionCcEnableArenas
	// FileOptionObjcClassPrefix is the file option objc_class_prefix.
	FileOptionObjcClassPrefix
	// FileOptionCsharpNamespace is the file option csharp_namespace.
	FileOptionCsharpNamespace
	// FileOptionCsharpNamespacePrefix is the file option csharp_namespace_prefix.
	FileOptionCsharpNamespacePrefix
	// FileOptionPhpNamespace is the file option php_namespace.
	FileOptionPhpNamespace
	// FileOptionPhpMetadataNamespace is the file option php_metadata_namespace.
	FileOptionPhpMetadataNamespace
	// FileOptionPhpMetadataNamespaceSuffix is the file option php_metadata_namespace_suffix.
	FileOptionPhpMetadataNamespaceSuffix
	// FileOptionRubyPackage is the file option ruby_package.
	FileOptionRubyPackage
	// FileOptionRubyPackageSuffix is the file option ruby_package_suffix.
	FileOptionRubyPackageSuffix
)

// String implements fmt.Stringer.
func (f FileOption) String() string {
	s, ok := fileOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// FieldOption is a field option.
type FieldOption int

const (
	// FieldOptionUnspecified is an unspecified field option.
	FieldOptionUnspecified FieldOption = iota
	// FieldOptionJSType is the field option js_type.
	FieldOptionJSType
)

// String implements fmt.Stringer.
func (f FieldOption) String() string {
	s, ok := fieldOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// *** PRIVATE ***

var (
	fileOptionToString = map[FileOption]string{
		FileOptionJavaPackage:                "java_package",
		FileOptionJavaPackagePrefix:          "java_package_prefix",
		FileOptionJavaPackageSuffix:          "java_package_suffix",
		FileOptionJavaOuterClassname:         "java_outer_classname",
		FileOptionJavaMultipleFiles:          "java_multiple_files",
		FileOptionJavaStringCheckUtf8:        "java_string_check_utf8",
		FileOptionOptimizeFor:                "optimize_for",
		FileOptionGoPackage:                  "go_package",
		FileOptionGoPackagePrefix:            "go_package_prefix",
		FileOptionCcEnableArenas:             "cc_enable_arenas",
		FileOptionObjcClassPrefix:            "objc_class_prefix",
		FileOptionCsharpNamespace:            "csharp_namespace",
		FileOptionCsharpNamespacePrefix:      "csharp_namespace_prefix",
		FileOptionPhpNamespace:               "php_namespace",
		FileOptionPhpMetadataNamespace:       "php_metadata_namespace",
		FileOptionPhpMetadataNamespaceSuffix: "php_metadata_namespace_suffix",
		FileOptionRubyPackage:                "ruby_package",
		FileOptionRubyPackageSuffix:          "ruby_package_suffix",
	}
	stringToFileOption = map[string]FileOption{
		"java_package":                  FileOptionJavaPackage,
		"java_package_prefix":           FileOptionJavaPackagePrefix,
		"java_package_suffix":           FileOptionJavaPackageSuffix,
		"java_outer_classname":          FileOptionJavaOuterClassname,
		"java_multiple_files":           FileOptionJavaMultipleFiles,
		"java_string_check_utf8":        FileOptionJavaStringCheckUtf8,
		"optimize_for":                  FileOptionOptimizeFor,
		"go_package":                    FileOptionGoPackage,
		"go_package_prefix":             FileOptionGoPackagePrefix,
		"cc_enable_arenas":              FileOptionCcEnableArenas,
		"objc_class_prefix":             FileOptionObjcClassPrefix,
		"csharp_namespace":              FileOptionCsharpNamespace,
		"csharp_namespace_prefix":       FileOptionCsharpNamespacePrefix,
		"php_namespace":                 FileOptionPhpNamespace,
		"php_metadata_namespace":        FileOptionPhpMetadataNamespace,
		"php_metadata_namespace_suffix": FileOptionPhpMetadataNamespaceSuffix,
		"ruby_package":                  FileOptionRubyPackage,
		"ruby_package_suffix":           FileOptionRubyPackageSuffix,
	}
	fileOptionToParseOverrideValueFunc = map[FileOption]func(interface{}) (interface{}, error){
		FileOptionJavaPackage:                parseOverrideValue[string],
		FileOptionJavaPackagePrefix:          parseOverrideValue[string],
		FileOptionJavaPackageSuffix:          parseOverrideValue[string],
		FileOptionOptimizeFor:                parseOverrideValueOptimizeMode,
		FileOptionJavaOuterClassname:         parseOverrideValue[string],
		FileOptionJavaMultipleFiles:          parseOverrideValue[bool],
		FileOptionJavaStringCheckUtf8:        parseOverrideValue[bool],
		FileOptionGoPackage:                  parseOverrideValue[string],
		FileOptionGoPackagePrefix:            parseOverrideValue[string],
		FileOptionCcEnableArenas:             parseOverrideValue[bool],
		FileOptionObjcClassPrefix:            parseOverrideValue[string], // objc_class_prefix is in descriptor.proto
		FileOptionCsharpNamespace:            parseOverrideValue[string],
		FileOptionCsharpNamespacePrefix:      parseOverrideValue[string],
		FileOptionPhpNamespace:               parseOverrideValue[string],
		FileOptionPhpMetadataNamespace:       parseOverrideValue[string],
		FileOptionPhpMetadataNamespaceSuffix: parseOverrideValue[string],
		FileOptionRubyPackage:                parseOverrideValue[string],
		FileOptionRubyPackageSuffix:          parseOverrideValue[string],
	}
	fieldOptionToString = map[FieldOption]string{
		FieldOptionJSType: "jstype",
	}
	stringToFieldOption = map[string]FieldOption{
		"jstype": FieldOptionJSType,
	}
	fieldOptionToParseOverrideValueFunc = map[FieldOption]func(interface{}) (interface{}, error){
		FieldOptionJSType: parseOverrideValueJSType,
	}
)

func parseFileOption(s string) (FileOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty file_option")
	}
	f, ok := stringToFileOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown file_option: %q", s)
}

func parseFieldOption(s string) (FieldOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty field_option")
	}
	f, ok := stringToFieldOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown field_option: %q", s)
}

func parseOverrideValue[T string | bool](overrideValue interface{}) (interface{}, error) {
	parsedValue, ok := overrideValue.(T)
	if !ok {
		return nil, fmt.Errorf("expected a %T, got %T", parsedValue, overrideValue)
	}
	return parsedValue, nil
}

func parseOverrideValueOptimizeMode(overrideValue interface{}) (interface{}, error) {
	optimizeModeName, ok := overrideValue.(string)
	if !ok {
		return nil, errors.New("must be one of SPEED, CODE_SIZE or LITE_RUNTIME")
	}
	optimizeMode, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeModeName]
	if !ok {
		return nil, errors.New("must be one of SPEED, CODE_SIZE or LITE_RUNTIME")
	}
	return descriptorpb.FileOptions_OptimizeMode(optimizeMode), nil
}

func parseOverrideValueJSType(override interface{}) (interface{}, error) {
	jsTypeName, ok := override.(string)
	if !ok {
		return nil, errors.New("must be one of JS_NORMAL, JS_STRING or JS_NUMBER")
	}
	jsTypeEnum, ok := descriptorpb.FieldOptions_JSType_value[jsTypeName]
	if !ok {
		return nil, errors.New("must be one of JS_NORMAL, JS_STRING or JS_NUMBER")
	}
	return descriptorpb.FieldOptions_JSType(jsTypeEnum), nil
}

// If the file or field option override value is one of the supported enum types,
// then we want to write out the string representation of the enum value, not
// the corresponding int32.
// Otherwise we just return the value.
func getOverrideValue(fileOptionName string, fieldOptionName string, value interface{}) (interface{}, error) {
	var optionName string
	if fileOptionName != "" && fieldOptionName != "" {
		return externalGenerateManagedConfigV2{}, fmt.Errorf("field option %s and file option %s set on the same override", fileOptionName, fieldOptionName)
	}
	if fileOptionName != "" {
		optionName = fileOptionName
		fileOption, err := parseFileOption(fileOptionName)
		if err != nil {
			return nil, err
		}
		switch fileOption {
		case
			FileOptionJavaPackage,
			FileOptionJavaPackagePrefix,
			FileOptionJavaPackageSuffix,
			FileOptionJavaOuterClassname,
			FileOptionJavaMultipleFiles,
			FileOptionJavaStringCheckUtf8,
			FileOptionGoPackage,
			FileOptionGoPackagePrefix,
			FileOptionCcEnableArenas,
			FileOptionObjcClassPrefix,
			FileOptionCsharpNamespace,
			FileOptionCsharpNamespacePrefix,
			FileOptionPhpNamespace,
			FileOptionPhpMetadataNamespace,
			FileOptionPhpMetadataNamespaceSuffix,
			FileOptionRubyPackage,
			FileOptionRubyPackageSuffix:
			return value, nil

		case FileOptionOptimizeFor:
			if optimizeModeValue, ok := value.(descriptorpb.FileOptions_OptimizeMode); ok {
				return optimizeModeValue.String(), nil
			}
		}
	}
	if fieldOptionName != "" {
		optionName = fieldOptionName
		fieldOption, err := parseFieldOption(fieldOptionName)
		if err != nil {
			return nil, err
		}
		switch fieldOption {
		case FieldOptionJSType:
			if jsTypeValue, ok := value.(descriptorpb.FieldOptions_JSType); ok {
				return jsTypeValue.String(), nil
			}
		}
	}
	return nil, fmt.Errorf("unable to get override value for %s: %v", optionName, value)
}
