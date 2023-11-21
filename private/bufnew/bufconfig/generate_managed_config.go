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
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"google.golang.org/protobuf/types/descriptorpb"
)

// GenerateManagedConfig is a managed mode configuration.
type GenerateManagedConfig interface {
	// Enabled returns whether managed mode is enabled.
	Enabled() bool
	// The first return value is the value to modify the file's option to. For
	// example, if a config has set java_package_prefix to foo for a file with
	// package bar, this returns "foo.bar" for the first value.
	// The second value indicates whether this option should be modified.
	// Reasons not to modify:
	// 1. Some options are not supposed to be modified unless a value is specified,
	//    such as optimize_for.
	// 2. In the config the user has specified not to modify this option for this file.
	// 3. The file already has the desired file option value.
	// 4. The file is a WKT.
	// TODO: 3 and 4 are debatable, asking the question what is the responsibility
	// of a managed config.
	//
	// Note: this means a GenerateManagedConfig's interface does not deal with
	// options like java_package_prefix and java_package_suffix, because it returns
	// the final value to modify java_package to.
	ValueForFileOption(imageFile, fileOption) (interface{}, bool)

	isGenerateManagedConfig()
}

// bufimage.ImageFile implements this, but this also allows private implementation
// for testing.
type imageFile interface {
	ModuleFullName() bufmodule.ModuleFullName
	Path() string
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
}

type fileOption int

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
)
