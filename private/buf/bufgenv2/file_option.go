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
)

const (
	// FileOptionJavaPackage is the file option java_package.
	FileOptionJavaPackage FileOption = iota + 1
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
	// FileOptionCcEnableArenas is the file option cc_enable_arenas.
	FileOptionCcEnableArenas
	// FileOptionObjcClassPrefix is the file option objc_class_prefix.
	FileOptionObjcClassPrefix
	// FileOptionCsharpNamespace is the file option csharp_namespace.
	FileOptionCsharpNamespace
	// FileOptionPhpNamespace is the file option php_namespace.
	FileOptionPhpNamespace
	// FileOptionPhpMetadataNamespace is the file option php_metadata_namespace.
	FileOptionPhpMetadataNamespace
	// FileOptionRubyPackage is the file option ruby_package.
	FileOptionRubyPackage
)

var (
	// AllFileOptions are all FileOptions.
	AllFileOptions = []FileOption{
		FileOptionJavaPackage,
		FileOptionJavaOuterClassname,
		FileOptionJavaMultipleFiles,
		FileOptionJavaStringCheckUtf8,
		FileOptionOptimizeFor,
		FileOptionGoPackage,
		FileOptionCcEnableArenas,
		FileOptionObjcClassPrefix,
		FileOptionCsharpNamespace,
		FileOptionPhpNamespace,
		FileOptionPhpMetadataNamespace,
		FileOptionRubyPackage,
	}

	// TODO: this type vs prefix thing isn't great
	// fill it out if we want to use it
	fileOptionToFileOptionType = map[FileOption]FileOptionType{
		FileOptionJavaPackage:          FileOptionTypeValue,
		FileOptionJavaOuterClassname:   FileOptionTypeValue,
		FileOptionJavaMultipleFiles:    FileOptionTypeValue,
		FileOptionJavaStringCheckUtf8:  FileOptionTypeValue,
		FileOptionOptimizeFor:          FileOptionTypeValue,
		FileOptionGoPackage:            FileOptionTypeValue,
		FileOptionCcEnableArenas:       FileOptionTypeValue,
		FileOptionObjcClassPrefix:      FileOptionTypeValue,
		FileOptionCsharpNamespace:      FileOptionTypeValue,
		FileOptionPhpNamespace:         FileOptionTypeValue,
		FileOptionPhpMetadataNamespace: FileOptionTypeValue,
		FileOptionRubyPackage:          FileOptionTypeValue,
	}
	// TODO: double-check these
	// This might stay in bufimagemodify based on how the prototype has worked out
	fileOptionToSourceCodeInfoPath = map[FileOption][]int32{
		FileOptionJavaPackage:          []int32{8, 1},
		FileOptionJavaOuterClassname:   []int32{8, 8},
		FileOptionJavaMultipleFiles:    []int32{8, 10},
		FileOptionJavaStringCheckUtf8:  []int32{8, 27},
		FileOptionOptimizeFor:          []int32{8, 9},
		FileOptionGoPackage:            []int32{8, 11},
		FileOptionCcEnableArenas:       []int32{8, 31},
		FileOptionObjcClassPrefix:      []int32{8, 36},
		FileOptionCsharpNamespace:      []int32{8, 37},
		FileOptionPhpNamespace:         []int32{8, 41},
		FileOptionPhpMetadataNamespace: []int32{8, 44},
		FileOptionRubyPackage:          []int32{8, 45},
	}
	fileOptionToString = map[FileOption]string{
		FileOptionJavaPackage:          "java_package",
		FileOptionJavaOuterClassname:   "java_outer_classname",
		FileOptionJavaMultipleFiles:    "java_multiple_files",
		FileOptionJavaStringCheckUtf8:  "java_string_check_utf8",
		FileOptionOptimizeFor:          "optimize_for",
		FileOptionGoPackage:            "go_package",
		FileOptionCcEnableArenas:       "cc_enable_arenas",
		FileOptionObjcClassPrefix:      "objc_class_prefix",
		FileOptionCsharpNamespace:      "csharp_namespace",
		FileOptionPhpNamespace:         "php_namespace",
		FileOptionPhpMetadataNamespace: "php_metadata_namespace",
		FileOptionRubyPackage:          "ruby_package",
	}
	stringToFileOption = map[string]FileOption{
		"java_package":           FileOptionJavaPackage,
		"java_outer_classname":   FileOptionJavaOuterClassname,
		"java_multiple_files":    FileOptionJavaMultipleFiles,
		"java_string_check_utf8": FileOptionJavaStringCheckUtf8,
		"optimize_for":           FileOptionOptimizeFor,
		"go_package":             FileOptionGoPackage,
		"cc_enable_arenas":       FileOptionCcEnableArenas,
		"objc_class_prefix":      FileOptionObjcClassPrefix,
		"csharp_namespace":       FileOptionCsharpNamespace,
		"php_namespace":          FileOptionPhpNamespace,
		"php_metadata_namespace": FileOptionPhpMetadataNamespace,
		"ruby_package":           FileOptionRubyPackage,
	}
)

// FileOption is a descriptor.proto file option that can be managed.
type FileOption int

// Type returns the FileOptionType or 0 if unknown..
func (f FileOption) Type() FileOptionType {
	t, ok := fileOptionToFileOptionType[f]
	if !ok {
		return 0
	}
	return t
}

// SourceCodeInfoPath returns the SourceCodeInfo path, or nil if unknown.
//
// Does not return a copy! Do not modify the return value!
func (f FileOption) SourceCodeInfoPath() []int32 {
	return fileOptionToSourceCodeInfoPath[f]
}

// String implements fmt.Stringer.
func (f FileOption) String() string {
	s, ok := fileOptionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// ParseFileOption parses the FileOption.
//
// The empty string is an error.
func ParseFileOption(s string) (FileOption, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("empty FileOption")
	}
	f, ok := stringToFileOption[s]
	if ok {
		return f, nil
	}
	return 0, fmt.Errorf("unknown FileOption: %q", s)
}
