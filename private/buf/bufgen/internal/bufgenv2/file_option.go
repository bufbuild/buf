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
	// fileOptionCcEnableArenas is the file option cc_enable_arenas.
	fileOptionCcEnableArenas
	// fileOptionObjcClassPrefix is the file option objc_class_prefix.
	fileOptionObjcClassPrefix
	// fileOptionCsharpNamespace is the file option csharp_namespace.
	fileOptionCsharpNamespace
	// fileOptionPhpNamespace is the file option php_namespace.
	fileOptionPhpNamespace
	// fileOptionPhpMetadataNamespace is the file option php_metadata_namespace.
	fileOptionPhpMetadataNamespace
	// fileOptionRubyPackage is the file option ruby_package.
	fileOptionRubyPackage
	// managedOptionJavaPackage is the managed mode option java_package.
	managedOptionJavaPackage managedOption = iota + 1
	// managedOptionJavaPackagePrefix is the managed mode option java_package_prefix.
	managedOptionJavaPackagePrefix
	// managedOptionJavaPackageSuffix is the managed mode option java_package_suffix.
	managedOptionJavaPackageSuffix
	// managedOptionOptimizeFor is the managed mode option optimize_for.
	managedOptionOptimizeFor
	// TODO: add the rest
)

var (
	// allfileOptions are all fileOptions.
	allFileOptions = []fileOption{
		fileOptionJavaPackage,
		fileOptionJavaOuterClassname,
		fileOptionJavaMultipleFiles,
		fileOptionJavaStringCheckUtf8,
		fileOptionOptimizeFor,
		fileOptionGoPackage,
		fileOptionCcEnableArenas,
		fileOptionObjcClassPrefix,
		fileOptionCsharpNamespace,
		fileOptionPhpNamespace,
		fileOptionPhpMetadataNamespace,
		fileOptionRubyPackage,
	}
	fileOptionToString = map[fileOption]string{
		fileOptionJavaPackage:          "java_package",
		fileOptionJavaOuterClassname:   "java_outer_classname",
		fileOptionJavaMultipleFiles:    "java_multiple_files",
		fileOptionJavaStringCheckUtf8:  "java_string_check_utf8",
		fileOptionOptimizeFor:          "optimize_for",
		fileOptionGoPackage:            "go_package",
		fileOptionCcEnableArenas:       "cc_enable_arenas",
		fileOptionObjcClassPrefix:      "objc_class_prefix",
		fileOptionCsharpNamespace:      "csharp_namespace",
		fileOptionPhpNamespace:         "php_namespace",
		fileOptionPhpMetadataNamespace: "php_metadata_namespace",
		fileOptionRubyPackage:          "ruby_package",
	}
	stringToFileOption = map[string]fileOption{
		"java_package":           fileOptionJavaPackage,
		"java_outer_classname":   fileOptionJavaOuterClassname,
		"java_multiple_files":    fileOptionJavaMultipleFiles,
		"java_string_check_utf8": fileOptionJavaStringCheckUtf8,
		"optimize_for":           fileOptionOptimizeFor,
		"go_package":             fileOptionGoPackage,
		"cc_enable_arenas":       fileOptionCcEnableArenas,
		"objc_class_prefix":      fileOptionObjcClassPrefix,
		"csharp_namespace":       fileOptionCsharpNamespace,
		"php_namespace":          fileOptionPhpNamespace,
		"php_metadata_namespace": fileOptionPhpMetadataNamespace,
		"ruby_package":           fileOptionRubyPackage,
	}
	// TODO: fill in the following lists
	stringToManagedOption = map[string]managedOption{
		"java_package":        managedOptionJavaPackage,
		"java_package_prefix": managedOptionJavaPackagePrefix,
		"java_package_suffix": managedOptionJavaPackageSuffix,
	}
	managedOptionToString = map[managedOption]string{
		managedOptionJavaPackage:       "java_package",
		managedOptionJavaPackagePrefix: "java_package_prefix",
		managedOptionJavaPackageSuffix: "java_package_suffix",
	}
	managedOptionToFileOption = map[managedOption]fileOption{
		managedOptionJavaPackage:       fileOptionJavaPackage,
		managedOptionJavaPackagePrefix: fileOptionJavaPackage,
		managedOptionJavaPackageSuffix: fileOptionJavaPackage,
	}
	managedOptionToOverrideParseFunc = map[managedOption]func(interface{}, managedOption) (bufimagemodifyv2.Override, error){
		managedOptionJavaPackage:       parseValueOverride[string],
		managedOptionJavaPackagePrefix: parsePrefixOverride,
		managedOptionJavaPackageSuffix: parseSuffixOverride,
		managedOptionOptimizeFor:       parseValueOverrideOptmizeMode,
	}
)

// fileOption is a descriptor.proto file option that can be managed.
type fileOption int

// String implements fmt.Stringer.
func (f fileOption) String() string {
	s, ok := fileOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// managedOption is an option that modifies a fileOption.
type managedOption int

// String implements fmt.Stringer.
func (m managedOption) String() string {
	s, ok := managedOptionToString[m]
	if !ok {
		return strconv.Itoa(int(m))
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

// parseManagedOption parses the managedOption.
//
// The empty string is an error.
func parseManagedOption(s string) (managedOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty manageOption")
	}
	f, ok := stringToManagedOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown managedOption: %q", s)
}

// Pass type T to construct a function that only accepts type T and creates an override from it.
func parseValueOverride[T string | bool](value interface{}, managedOption managedOption) (bufimagemodifyv2.Override, error) {
	overrideValue, ok := value.(T)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", managedOption)
	}
	return bufimagemodifyv2.NewValueOverride[T](overrideValue), nil
}

func parsePrefixOverride(value interface{}, managedOption managedOption) (bufimagemodifyv2.Override, error) {
	prefix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", managedOption)
	}
	return bufimagemodifyv2.NewPrefixOverride(prefix), nil
}

func parseSuffixOverride(value interface{}, managedOption managedOption) (bufimagemodifyv2.Override, error) {
	suffix, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", managedOption)
	}
	return bufimagemodifyv2.NewSuffixOverride(suffix), nil
}

func parseValueOverrideOptmizeMode(override interface{}, managedOption managedOption) (bufimagemodifyv2.Override, error) {
	optimizeModeName, ok := override.(string)
	if !ok {
		return nil, fmt.Errorf("invalid value for %v", managedOption)
	}
	optimizeModeEnum, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeModeName]
	if !ok {
		return nil, fmt.Errorf("%v: %s is not a valid optmize_for value, valid values are SPEED, CODE_SIZE and LITE_RUNTIME", managedOption, optimizeModeName)
	}
	optimizeMode := descriptorpb.FileOptions_OptimizeMode(optimizeModeEnum)
	return bufimagemodifyv2.NewValueOverride[descriptorpb.FileOptions_OptimizeMode](optimizeMode), nil
}
