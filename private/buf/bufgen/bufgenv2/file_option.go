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
	fileOptionToParser = map[FileOption]parser{
		FileOptionJavaPackage: {
			allowPrefix:            true,
			valueOverrideParseFunc: parseValueOverride[string],
		},
		// TODO:
		FileOptionJavaOuterClassname: {},
		// TODO:
		FileOptionJavaMultipleFiles: {},
		// TODO:
		FileOptionJavaStringCheckUtf8: {},
		FileOptionOptimizeFor: {
			valueOverrideParseFunc: parseValueOverrideOptmizeMode,
		},
		// TODO:
		FileOptionGoPackage: {},
		FileOptionCcEnableArenas: {
			valueOverrideParseFunc: parseValueOverride[bool],
		},
		// TODO:
		FileOptionObjcClassPrefix: {},
		// TODO:
		FileOptionCsharpNamespace: {},
		// TODO:
		FileOptionPhpNamespace: {},
		// TODO:
		FileOptionPhpMetadataNamespace: {},
		// TODO:
		FileOptionRubyPackage: {},
	}
)

// FileOption is a descriptor.proto file option that can be managed.
type FileOption int

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

type parser struct {
	allowPrefix            bool
	valueOverrideParseFunc func(interface{}, FileOption) (bufimagemodifyv2.Override, error)
}

func (p parser) parse(prefix *string, value interface{}, fileOption FileOption) (bufimagemodifyv2.Override, error) {
	if prefix != nil && value != nil {
		return nil, fmt.Errorf("%v: only one of value and prefix can be set", fileOption)
	}
	if prefix == nil && value == nil {
		return nil, fmt.Errorf("%v: value or prefix must be set", fileOption)
	}
	if prefix != nil {
		if !p.allowPrefix {
			return nil, fmt.Errorf("%v: prefix is not allowed", fileOption)
		}
		return bufimagemodifyv2.NewPrefixOverride(*prefix), nil
	}
	return p.valueOverrideParseFunc(value, fileOption)
}

// Pass type T to construct a function that only accepts type T and creates an override from it.
func parseValueOverride[T string | bool](value interface{}, fileOption FileOption) (bufimagemodifyv2.Override, error) {
	overrideValue, ok := value.(T)
	if !ok {
		return nil, fmt.Errorf("invalid override for %v", fileOption)
	}
	return bufimagemodifyv2.NewValueOverride[T](overrideValue), nil
}

func parseValueOverrideOptmizeMode(override interface{}, fileOption FileOption) (bufimagemodifyv2.Override, error) {
	optimizeModeName, ok := override.(string)
	if !ok {
		return nil, fmt.Errorf("a valid optimize mode string is required for %v", fileOption)
	}
	optimizeModeEnum, ok := descriptorpb.FileOptions_OptimizeMode_value[optimizeModeName]
	if !ok {
		return nil, fmt.Errorf("invalid optimize mode %s set for %v", optimizeModeName, fileOption)
	}
	optimizeMode := descriptorpb.FileOptions_OptimizeMode(optimizeModeEnum)
	return bufimagemodifyv2.NewValueOverride[descriptorpb.FileOptions_OptimizeMode](optimizeMode), nil
}
