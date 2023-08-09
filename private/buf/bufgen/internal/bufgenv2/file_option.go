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

package bufgenv2

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// FileOptionJavaPackage is the file option java_package.
	FileOptionJavaPackage FileOption = iota + 1
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
	// groupJavaPackage is the file option group that modifies java_package.
	groupJavaPackage fileOptionGroup = iota + 1
	groupJavaOuterClassname
	groupJavaMultipleFiles
	groupJavaStringCheckUtf8
	groupOptimizeFor
	groupGoPackage
	groupCcEnableArenas
	groupObjcClassPrefix
	groupCsharpNamespace
	groupPhpNamespace
	groupPhpMetadataNamespace
	groupRubyPackage
)

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
	fileOptionToOverrideParseFunc = map[FileOption]func(interface{}, FileOption) (override, error){
		FileOptionJavaPackage:                parseValueOverride[string],
		FileOptionJavaPackagePrefix:          parsePrefixOverride,
		FileOptionJavaPackageSuffix:          parseSuffixOverride,
		FileOptionOptimizeFor:                parseValueOverrideOptmizeMode,
		FileOptionJavaOuterClassname:         parseValueOverride[string],
		FileOptionJavaMultipleFiles:          parseValueOverride[bool],
		FileOptionJavaStringCheckUtf8:        parseValueOverride[bool],
		FileOptionGoPackage:                  parseValueOverride[string],
		FileOptionGoPackagePrefix:            parsePrefixOverride,
		FileOptionCcEnableArenas:             parseValueOverride[bool],
		FileOptionObjcClassPrefix:            parseValueOverride[string], // objc_class_prefix is in descriptor.proto
		FileOptionCsharpNamespace:            parseValueOverride[string],
		FileOptionCsharpNamespacePrefix:      parsePrefixOverride,
		FileOptionPhpNamespace:               parseValueOverride[string],
		FileOptionPhpMetadataNamespace:       parseValueOverride[string],
		FileOptionPhpMetadataNamespaceSuffix: parseSuffixOverride,
		FileOptionRubyPackage:                parseValueOverride[string],
		FileOptionRubyPackageSuffix:          parseSuffixOverride,
	}
	fileOptionToGroup = map[FileOption]fileOptionGroup{
		FileOptionJavaPackage:                groupJavaPackage,
		FileOptionJavaPackagePrefix:          groupJavaPackage,
		FileOptionJavaPackageSuffix:          groupJavaPackage,
		FileOptionJavaOuterClassname:         groupJavaOuterClassname,
		FileOptionJavaMultipleFiles:          groupJavaMultipleFiles,
		FileOptionJavaStringCheckUtf8:        groupJavaStringCheckUtf8,
		FileOptionOptimizeFor:                groupOptimizeFor,
		FileOptionGoPackage:                  groupGoPackage,
		FileOptionGoPackagePrefix:            groupGoPackage,
		FileOptionCcEnableArenas:             groupCcEnableArenas,
		FileOptionObjcClassPrefix:            groupObjcClassPrefix,
		FileOptionCsharpNamespace:            groupCsharpNamespace,
		FileOptionCsharpNamespacePrefix:      groupCsharpNamespace,
		FileOptionPhpNamespace:               groupPhpNamespace,
		FileOptionPhpMetadataNamespace:       groupPhpMetadataNamespace,
		FileOptionPhpMetadataNamespaceSuffix: groupPhpMetadataNamespace,
		FileOptionRubyPackage:                groupRubyPackage,
		FileOptionRubyPackageSuffix:          groupRubyPackage,
	}
	allFileOptionGroups = []fileOptionGroup{
		groupJavaPackage,
		groupJavaOuterClassname,
		groupJavaMultipleFiles,
		groupJavaStringCheckUtf8,
		groupOptimizeFor,
		groupGoPackage,
		groupCcEnableArenas,
		groupObjcClassPrefix,
		groupCsharpNamespace,
		groupPhpNamespace,
		groupPhpMetadataNamespace,
		groupRubyPackage,
	}
)

// FileOption is a file option in managed mode
type FileOption int

// fileOptionGroup is a group of file options, and file options modify the same
// proto file option if they belong to the same group
type fileOptionGroup int

// String implements fmt.Stringer.
func (f FileOption) String() string {
	s, ok := fileOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// parseFileOption parses the fileOption.
//
// The empty string is an error.
func parseFileOption(s string) (FileOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty fileOption")
	}
	f, ok := stringToFileOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown fileOption: %q", s)
}

// Pass type T to construct a function that only accepts type T and creates an override from it.
func parseValueOverride[T string | bool](value interface{}, fileOption FileOption) (override, error) {
	overrideValue, ok := value.(T)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return newValueOverride(overrideValue), nil
}

func parsePrefixOverride(value interface{}, fileOption FileOption) (override, error) {
	prefix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return newPrefixOverride(prefix), nil
}

func parseSuffixOverride(value interface{}, fileOption FileOption) (override, error) {
	suffix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return newSuffixOverride(suffix), nil
}

func parseValueOverrideOptmizeMode(override interface{}, fileOption FileOption) (override, error) {
	optimizeModeName, ok := override.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	optimizeModeEnum, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeModeName]
	if !ok {
		return nil, fmt.Errorf("%v: %s is not a valid optmize_for value, must be one of SPEED, CODE_SIZE and LITE_RUNTIME", fileOption, optimizeModeName)
	}
	optimizeMode := descriptorpb.FileOptions_OptimizeMode(optimizeModeEnum)
	return newValueOverride(optimizeMode), nil
}
