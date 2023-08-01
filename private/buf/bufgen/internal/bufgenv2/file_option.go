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

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifyv2"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// fileOptionJavaPackage is the file option java_package.
	fileOptionJavaPackage fileOption = iota + 1
	// fileOptionJavaPackagePrefix is the file option java_package_prefix.
	fileOptionJavaPackagePrefix
	// fileOptionJavaPackageSuffix is the file option java_package_suffix.
	fileOptionJavaPackageSuffix
	// fileOptionJavaOuterClassname is the file option java_outer_classname.
	fileOptionJavaOuterClassname
	// fileOptionJavaMultipleFiles is the file option java_multiple_files.
	fileOptionJavaMultipleFiles
	// fileOptionJavaStringCheckUtf8 is the file option java_string_check_utf8.
	fileOptionJavaStringCheckUtf8
	// fileOptionOptimizeFor is the file option optimize_for.
	fileOptionOptimizeFor
	// fileOptionGoPackage is the file option go_package.
	fileOptionGoPackage
	// fileOptionGoPackagePrefix is the file option go_package_prefix.
	fileOptionGoPackagePrefix
	// fileOptionCcEnableArenas is the file option cc_enable_arenas.
	fileOptionCcEnableArenas
	// fileOptionObjcClassPrefix is the file option objc_class_prefix.
	fileOptionObjcClassPrefix
	// fileOptionCsharpNamespace is the file option csharp_namespace.
	fileOptionCsharpNamespace
	// fileOptionCsharpNamespacePrefix is the file option csharp_namespace_prefix.
	fileOptionCsharpNamespacePrefix
	// fileOptionPhpNamespace is the file option php_namespace.
	fileOptionPhpNamespace
	// fileOptionPhpMetadataNamespace is the file option php_metadata_namespace.
	fileOptionPhpMetadataNamespace
	// fileOptionPhpMetadataNamespaceSuffix is the file option php_metadata_namespace_suffix.
	fileOptionPhpMetadataNamespaceSuffix
	// fileOptionRubyPackage is the file option ruby_package.
	fileOptionRubyPackage
	// fileOptionRubyPackageSuffix is the file option ruby_package_suffix.
	fileOptionRubyPackageSuffix
	// groupJavaPackage is the file option group that modifies java_package.
	groupJavaPackage fileOptionGroup = iota + 1
	groupJavaOuterClassname
	groupJavaMultipleFiles
	groupJavaStringCheckUtf8
	groupOptimizeFor
	groupGoPackage
	groupObjcClassPrefix
	groupCsharpNamespace
	groupPhpNamespace
	groupPhpMetadataNamespace
	groupRubyPackage
)

var (
	fileOptionToString = map[fileOption]string{
		fileOptionJavaPackage:                "java_package",
		fileOptionJavaPackagePrefix:          "java_package_prefix",
		fileOptionJavaPackageSuffix:          "java_package_suffix",
		fileOptionJavaOuterClassname:         "java_outer_classname",
		fileOptionJavaMultipleFiles:          "java_multiple_files",
		fileOptionJavaStringCheckUtf8:        "java_string_check_utf8",
		fileOptionOptimizeFor:                "optimize_for",
		fileOptionGoPackage:                  "go_package",
		fileOptionGoPackagePrefix:            "go_package_prefix",
		fileOptionCcEnableArenas:             "cc_enable_arenas",
		fileOptionObjcClassPrefix:            "objc_class_prefix",
		fileOptionCsharpNamespace:            "csharp_namespace",
		fileOptionCsharpNamespacePrefix:      "csharp_namespace_prefix",
		fileOptionPhpNamespace:               "php_namespace",
		fileOptionPhpMetadataNamespace:       "php_metadata_namespace",
		fileOptionPhpMetadataNamespaceSuffix: "php_metadata_namespace_suffix",
		fileOptionRubyPackage:                "ruby_package",
		fileOptionRubyPackageSuffix:          "ruby_package_suffix",
	}
	stringToFileOption = map[string]fileOption{
		"java_package":                  fileOptionJavaPackage,
		"java_package_prefix":           fileOptionJavaPackagePrefix,
		"java_package_suffix":           fileOptionJavaPackageSuffix,
		"java_outer_classname":          fileOptionJavaOuterClassname,
		"java_multiple_files":           fileOptionJavaMultipleFiles,
		"java_string_check_utf8":        fileOptionJavaStringCheckUtf8,
		"optimize_for":                  fileOptionOptimizeFor,
		"go_package":                    fileOptionGoPackage,
		"go_package_prefix":             fileOptionGoPackagePrefix,
		"cc_enable_arenas":              fileOptionCcEnableArenas,
		"objc_class_prefix":             fileOptionObjcClassPrefix,
		"csharp_namespace":              fileOptionCsharpNamespace,
		"csharp_namespace_prefix":       fileOptionCsharpNamespacePrefix,
		"php_namespace":                 fileOptionPhpNamespace,
		"php_metadata_namespace":        fileOptionPhpMetadataNamespace,
		"php_metadata_namespace_suffix": fileOptionPhpMetadataNamespaceSuffix,
		"ruby_package":                  fileOptionRubyPackage,
		"ruby_package_suffix":           fileOptionRubyPackageSuffix,
	}
	fileOptionToOverrideParseFunc = map[fileOption]func(interface{}, fileOption) (bufimagemodifyv2.Override, error){
		fileOptionJavaPackage:                parseValueOverride[string],
		fileOptionJavaPackagePrefix:          parsePrefixOverride,
		fileOptionJavaPackageSuffix:          parseSuffixOverride,
		fileOptionOptimizeFor:                parseValueOverrideOptmizeMode,
		fileOptionJavaOuterClassname:         parseValueOverride[string],
		fileOptionJavaMultipleFiles:          parseValueOverride[bool],
		fileOptionJavaStringCheckUtf8:        parseValueOverride[bool],
		fileOptionGoPackage:                  parseValueOverride[string],
		fileOptionGoPackagePrefix:            parsePrefixOverride,
		fileOptionObjcClassPrefix:            parseValueOverride[string], // objc_class_prefix is in descriptor.proto
		fileOptionCsharpNamespace:            parseValueOverride[string],
		fileOptionCsharpNamespacePrefix:      parsePrefixOverride,
		fileOptionPhpNamespace:               parseValueOverride[string],
		fileOptionPhpMetadataNamespace:       parseValueOverride[string],
		fileOptionPhpMetadataNamespaceSuffix: parseSuffixOverride,
		fileOptionRubyPackage:                parseValueOverride[string],
		fileOptionRubyPackageSuffix:          parseSuffixOverride,
	}
	fileOptionToGroup = map[fileOption]fileOptionGroup{
		fileOptionJavaPackage:                groupJavaPackage,
		fileOptionJavaPackagePrefix:          groupJavaPackage,
		fileOptionJavaPackageSuffix:          groupJavaPackage,
		fileOptionJavaOuterClassname:         groupJavaOuterClassname,
		fileOptionJavaMultipleFiles:          groupJavaMultipleFiles,
		fileOptionJavaStringCheckUtf8:        groupJavaStringCheckUtf8,
		fileOptionOptimizeFor:                groupOptimizeFor,
		fileOptionGoPackage:                  groupGoPackage,
		fileOptionGoPackagePrefix:            groupGoPackage,
		fileOptionObjcClassPrefix:            groupObjcClassPrefix,
		fileOptionCsharpNamespace:            groupCsharpNamespace,
		fileOptionCsharpNamespacePrefix:      groupCsharpNamespace,
		fileOptionPhpNamespace:               groupPhpNamespace,
		fileOptionPhpMetadataNamespace:       groupPhpMetadataNamespace,
		fileOptionPhpMetadataNamespaceSuffix: groupPhpMetadataNamespace,
		fileOptionRubyPackage:                groupRubyPackage,
		fileOptionRubyPackageSuffix:          groupRubyPackage,
	}
	allFileOptionGroups = []fileOptionGroup{
		groupJavaPackage,
		groupJavaOuterClassname,
		groupJavaMultipleFiles,
		groupJavaStringCheckUtf8,
		groupOptimizeFor,
		groupGoPackage,
		groupObjcClassPrefix,
		groupCsharpNamespace,
		groupPhpNamespace,
		groupPhpMetadataNamespace,
		groupRubyPackage,
	}
)

// fileOption is a file option in managed mode
type fileOption int

// fileOptionGroup is a group of file options, and file options modify the same
// proto file option if they belong to the same group
type fileOptionGroup int

// String implements fmt.Stringer.
func (f fileOption) String() string {
	s, ok := fileOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// parseFileOption parses the fileOption.
//
// The empty string is an error.
func parseFileOption(s string) (fileOption, error) {
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
func parseValueOverride[T string | bool](value interface{}, fileOption fileOption) (bufimagemodifyv2.Override, error) {
	overrideValue, ok := value.(T)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return bufimagemodifyv2.NewValueOverride(overrideValue), nil
}

func parsePrefixOverride(value interface{}, fileOption fileOption) (bufimagemodifyv2.Override, error) {
	prefix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return bufimagemodifyv2.NewPrefixOverride(prefix), nil
}

func parseSuffixOverride(value interface{}, fileOption fileOption) (bufimagemodifyv2.Override, error) {
	suffix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	return bufimagemodifyv2.NewSuffixOverride(suffix), nil
}

func parseValueOverrideOptmizeMode(override interface{}, fileOption fileOption) (bufimagemodifyv2.Override, error) {
	optimizeModeName, ok := override.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", fileOption)
	}
	optimizeModeEnum, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeModeName]
	if !ok {
		return nil, fmt.Errorf("%v: %s is not a valid optmize_for value, must be one of SPEED, CODE_SIZE and LITE_RUNTIME", fileOption, optimizeModeName)
	}
	optimizeMode := descriptorpb.FileOptions_OptimizeMode(optimizeModeEnum)
	return bufimagemodifyv2.NewValueOverride(optimizeMode), nil
}
