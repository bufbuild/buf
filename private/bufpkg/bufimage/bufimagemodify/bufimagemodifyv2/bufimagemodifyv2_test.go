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

package bufimagemodifyv2

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifySingleOption(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description             string
		subDir                  string
		file                    string
		fileHasNoSourceCodeInfo bool
		modifyFunc              func(Marker, bufimage.ImageFile, Override) error
		fileOptionPath          []int32
		override                Override
		expectedValue           interface{}
		// This should be set to true when an override has no effect,
		// i.e. override is the same as defined in proto file.
		shouldKeepSourceCodeInfo bool
		assertFunc               func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
	}{
		{
			description:             "Modify Java Package with value on file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewValueOverride("valueoverride"),
			expectedValue:           "valueoverride",
			assertFunc:              assertJavaPackage,
		},
		{
			description:             "Modify Java Package with prefix on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewPrefixOverride("prefixoverride"),
			// emptyoptions/a.proto does not have a proto package, thus the result is an empty string
			expectedValue: "",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride("prefixoverride"),
			// all/options/a.proto does not have a proto package, thus the result is an empty string
			expectedValue: "",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewValueOverride("alloverride"),
			expectedValue:  "alloverride",
			assertFunc:     assertJavaPackage,
		},
		{
			description:             "Modify Java Package with value on file with empty options and a proto package",
			subDir:                  "javaemptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewPrefixOverride("override.pre"),
			expectedValue:           "override.pre.foo",
			assertFunc:              assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride("prefix"),
			expectedValue:  "prefix.acme.weather",
			assertFunc:     assertJavaPackage,
		},
		{
			description:              "Modify Java Package with override value the same as java package",
			subDir:                   "javaoptions",
			file:                     "java_file.proto",
			modifyFunc:               ModifyJavaPackage,
			fileOptionPath:           internal.JavaPackagePath,
			override:                 NewValueOverride("foo"),
			expectedValue:            "foo",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaPackage,
		},
		{
			description:             "Modify Java Package with wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaPackage,
			fileOptionPath:          internal.JavaPackagePath,
			override:                NewValueOverride("override.value"),
			expectedValue:           "override.value",
			assertFunc:              assertJavaPackage,
		},
		{
			description:              "Modify Java Package with wkt",
			subDir:                   "wktimport",
			file:                     "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo:  true,
			modifyFunc:               ModifyJavaPackage,
			fileOptionPath:           internal.JavaPackagePath,
			override:                 NewValueOverride("override.value"),
			expectedValue:            "com.google.protobuf",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaPackage,
		},
		{
			description:    "Modify Java Package with empty prefix on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewPrefixOverride(""),
			// use the package name when prefix is empty
			expectedValue: "acme.weather",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with nil override on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       nil,
			// prepend the default prefix "com" to the package name
			expectedValue: "com.acme.weather",
			assertFunc:    assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file with java options and a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			override:       NewValueOverride("pkg.pkg"),
			expectedValue:  "pkg.pkg",
			assertFunc:     assertJavaPackage,
		},
		{
			description:             "Modify CC Enable Arenas to true on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCcEnableArenas,
			fileOptionPath:          internal.CCEnableArenasPath,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertCcEnableArenas,
		},
		{
			description:             "Modify CC Enable Arenas to false on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCcEnableArenas,
			fileOptionPath:          internal.CCEnableArenasPath,
			override:                NewValueOverride(false),
			expectedValue:           false,
			assertFunc:              assertCcEnableArenas,
		},
		{
			description:    "Modify CC Enable Arenas to true on a file with all options",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCcEnableArenas,
			fileOptionPath: internal.CCEnableArenasPath,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertCcEnableArenas,
		},
		{
			description:             "Modify CC Enable Arenas with nil override",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCcEnableArenas,
			fileOptionPath:          internal.CCEnableArenasPath,
			override:                nil,
			expectedValue:           true,
			assertFunc:              assertCcEnableArenas,
		},
		{
			description:              "Modify CC Enable Arenas to false on a file with all options",
			subDir:                   "alloptions",
			file:                     "a.proto",
			modifyFunc:               ModifyCcEnableArenas,
			fileOptionPath:           internal.CCEnableArenasPath,
			override:                 NewValueOverride(false),
			expectedValue:            false,
			shouldKeepSourceCodeInfo: true, // option already set to true in a.proto
			assertFunc:               assertCcEnableArenas,
		},
		{
			description:    "Modify CC Enable Arenas to true on a file with cc options",
			subDir:         "ccoptions",
			file:           "a.proto",
			modifyFunc:     ModifyCcEnableArenas,
			fileOptionPath: internal.CCEnableArenasPath,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertCcEnableArenas,
		},
		{
			description:              "Modify CC Enable Arenas to false on a file with cc options",
			subDir:                   "ccoptions",
			file:                     "a.proto",
			modifyFunc:               ModifyCcEnableArenas,
			fileOptionPath:           internal.CCEnableArenasPath,
			override:                 NewValueOverride(false),
			expectedValue:            false,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertCcEnableArenas,
		},
		{
			description:             "Modify CC Enable Arenas to true with wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCcEnableArenas,
			fileOptionPath:          internal.CCEnableArenasPath,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertCcEnableArenas,
		},
		{
			description:             "Modify CC Enable Arenas to false with wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCcEnableArenas,
			fileOptionPath:          internal.CCEnableArenasPath,
			override:                NewValueOverride(false),
			expectedValue:           false,
			assertFunc:              assertCcEnableArenas,
		},
		{
			description:             "Modify Csharp Namespace with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCsharpNamespace,
			fileOptionPath:          internal.CsharpNamespacePath,
			override:                NewValueOverride("csharp"),
			expectedValue:           "csharp",
			assertFunc:              assertCsharpNamespace,
		},
		{
			description:             "Modify Csharp Namespace with nil override and empty package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCsharpNamespace,
			fileOptionPath:          internal.CsharpNamespacePath,
			override:                nil,
			expectedValue:           "",
			assertFunc:              assertCsharpNamespace,
		},
		{
			description:    "Modify Csharp Namespace with nil override with a two-part package name",
			subDir:         filepath.Join("csharpoptions", "single"),
			file:           "csharp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			override:       nil,
			expectedValue:  "Acme.V1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "Modify Csharp Namespace with nil override with a three-part package name",
			subDir:         filepath.Join("csharpoptions", "double"),
			file:           "csharp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			override:       nil,
			expectedValue:  "Acme.Weather.V1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "Modify Csharp Namespace with nil override with a four-part package name",
			subDir:         filepath.Join("csharpoptions", "triple"),
			file:           "csharp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			override:       nil,
			expectedValue:  "Acme.Weather.Data.V1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "Modify Csharp Namespace with all options",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			override:       NewValueOverride("csharp"),
			expectedValue:  "csharp",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:             "Modify Csharp Namespace on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyCsharpNamespace,
			fileOptionPath:          internal.CsharpNamespacePath,
			override:                NewValueOverride("override.value"),
			expectedValue:           "override.value",
			assertFunc:              assertCsharpNamespace,
		},
		{
			description:              "Modify Csharp Namespace on a wkt file",
			subDir:                   "wktimport",
			file:                     "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo:  true,
			modifyFunc:               ModifyCsharpNamespace,
			fileOptionPath:           internal.CsharpNamespacePath,
			override:                 NewValueOverride("override.value"),
			expectedValue:            "Google.Protobuf.WellKnownTypes",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertCsharpNamespace,
		},
		{
			description:             "Modify Go Package with value on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyGoPackage,
			fileOptionPath:          internal.GoPackagePath,
			override:                NewValueOverride("valueoverride"),
			expectedValue:           "valueoverride",
			assertFunc:              assertGoPackage,
		},
		{
			description:             "Modify Go Package with prefix on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyGoPackage,
			fileOptionPath:          internal.GoPackagePath,
			override:                NewPrefixOverride("prefixoverride"),
			// a.proto is in top-level
			expectedValue: "prefixoverride",
			assertFunc:    assertGoPackage,
		},
		{
			description:             "Modify Go Package with nil on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyGoPackage,
			fileOptionPath:          internal.GoPackagePath,
			override:                nil,
			expectedValue:           "",
			assertFunc:              assertGoPackage,
		},
		{
			description:    "Modify Go Package with prefix on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewPrefixOverride("prefixoverride"),
			// a.proto is in top level
			expectedValue: "prefixoverride",
			assertFunc:    assertGoPackage,
		},
		{
			description:    "Modify Go Package with prefix on file with go options and empty proto package",
			subDir:         "gooptions",
			file:           filepath.Join("somedir", "a.proto"),
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewPrefixOverride("prefixoverride"),
			// a.proto is in somedir
			expectedValue: "prefixoverride/somedir",
			assertFunc:    assertGoPackage,
		},
		{
			description:    "Modify Go Package with nil on file with go options and empty proto package",
			subDir:         "gooptions",
			file:           filepath.Join("somedir", "a.proto"),
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       nil,
			// a.proto is in somedir
			expectedValue:            "foo",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertGoPackage,
		},
		{
			description:    "Modify Go Package with value on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewValueOverride("alloverride"),
			expectedValue:  "alloverride",
			assertFunc:     assertGoPackage,
		},
		{
			description:             "Modify Go Package on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyGoPackage,
			fileOptionPath:          internal.GoPackagePath,
			override:                NewValueOverride("override.value"),
			expectedValue:           "override.value",
			assertFunc:              assertGoPackage,
		},
		{
			description:              "Modify Go Package on a wkt file",
			subDir:                   "wktimport",
			file:                     "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo:  true,
			modifyFunc:               ModifyGoPackage,
			fileOptionPath:           internal.GoPackagePath,
			override:                 NewValueOverride("override.value"),
			expectedValue:            "google.golang.org/protobuf/types/known/timestamppb",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertGoPackage,
		},
		{
			description:             "Modify Java Multiple Files with true on file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaMultipleFiles,
			fileOptionPath:          internal.JavaMultipleFilesPath,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertJavaMultipleFiles,
		},
		{
			description:             "Modify Java Multiple Files with false on file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaMultipleFiles,
			fileOptionPath:          internal.JavaMultipleFilesPath,
			override:                NewValueOverride(false),
			expectedValue:           false,
			assertFunc:              assertJavaMultipleFiles,
		},
		{
			description:             "Modify Java Multiple Files with nil on file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaMultipleFiles,
			fileOptionPath:          internal.JavaMultipleFilesPath,
			override:                nil,
			// use default, which is true
			expectedValue: true,
			assertFunc:    assertJavaMultipleFiles,
		},
		{
			description:    "Modify Java Multiple Files with true on file with all options",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaMultipleFiles,
		},
		{
			description:              "Modify Java Multiple Files with true on file with all options",
			subDir:                   "alloptions",
			file:                     "a.proto",
			modifyFunc:               ModifyJavaMultipleFiles,
			fileOptionPath:           internal.JavaMultipleFilesPath,
			override:                 NewValueOverride(false),
			expectedValue:            false,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaMultipleFiles,
		},
		{
			description:             "Modify Java Multiple Files on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaMultipleFiles,
			fileOptionPath:          internal.JavaMultipleFilesPath,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertJavaMultipleFiles,
		},
		{
			description:    "Modify Java Multiple Files with wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(false),
			// should take no effect
			expectedValue:            true,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaMultipleFiles,
		},
		{
			description:             "Modify Java Outer Class Name on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaOuterClassname,
			fileOptionPath:          internal.JavaOuterClassnamePath,
			override:                NewValueOverride("OverrideClass"),
			expectedValue:           "OverrideClass",
			assertFunc:              assertJavaOuterClassName,
		},
		{
			description:             "Modify Java Outer Class Name with nil on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaOuterClassname,
			fileOptionPath:          internal.JavaOuterClassnamePath,
			override:                nil,
			expectedValue:           "AProto",
			assertFunc:              assertJavaOuterClassName,
		},
		{
			description:    "Modify Java Outer Class Name with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaOuterClassname,
			fileOptionPath: internal.JavaOuterClassnamePath,
			override:       NewValueOverride("OverrideOuter"),
			expectedValue:  "OverrideOuter",
			assertFunc:     assertJavaOuterClassName,
		},
		{
			description:             "Modify Java Outer Class Name on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaOuterClassname,
			fileOptionPath:          internal.JavaOuterClassnamePath,
			override:                NewValueOverride("OverrideValue"),
			expectedValue:           "OverrideValue",
			assertFunc:              assertJavaOuterClassName,
		},
		{
			description:              "Modify Java Outer Class Name on a wkt file",
			subDir:                   "wktimport",
			file:                     "google/protobuf/timestamp.proto",
			modifyFunc:               ModifyJavaOuterClassname,
			fileOptionPath:           internal.JavaOuterClassnamePath,
			override:                 NewValueOverride("OverrideValue"),
			expectedValue:            "TimestampProto",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaOuterClassName,
		},
		{
			description:             "Modify Java String Check UTF8 to true on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaStringCheckUTF8,
			fileOptionPath:          internal.JavaStringCheckUtf8Path,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertJavaStringCheckUTF8,
		},
		{
			description:             "Modify Java String Check UTF8 to false on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaStringCheckUTF8,
			fileOptionPath:          internal.JavaStringCheckUtf8Path,
			override:                NewValueOverride(false),
			expectedValue:           false,
			assertFunc:              assertJavaStringCheckUTF8,
		},
		{
			description:             "Modify Java String Check UTF8 with nil on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaStringCheckUTF8,
			fileOptionPath:          internal.JavaStringCheckUtf8Path,
			override:                nil,
			expectedValue:           false,
			assertFunc:              assertJavaStringCheckUTF8,
		},
		{
			description:    "Modify Java String Check UTF8 on file with all options and empty proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaStringCheckUTF8,
			fileOptionPath: internal.JavaStringCheckUtf8Path,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaStringCheckUTF8,
		},
		{
			description:              "Modify Java String Check UTF8 with nil on file with all options and empty proto package",
			subDir:                   "alloptions",
			file:                     "a.proto",
			modifyFunc:               ModifyJavaStringCheckUTF8,
			fileOptionPath:           internal.JavaStringCheckUtf8Path,
			override:                 nil,
			expectedValue:            false,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaStringCheckUTF8,
		},
		{
			description:              "Modify Java String Check UTF8 with false on file with all options and empty proto package",
			subDir:                   "alloptions",
			file:                     "a.proto",
			modifyFunc:               ModifyJavaStringCheckUTF8,
			fileOptionPath:           internal.JavaStringCheckUtf8Path,
			override:                 NewValueOverride(false),
			expectedValue:            false,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertJavaStringCheckUTF8,
		},
		{
			description:             "Modify Java String Check UTF8 on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaStringCheckUTF8,
			fileOptionPath:          internal.JavaStringCheckUtf8Path,
			override:                NewValueOverride(true),
			expectedValue:           true,
			assertFunc:              assertJavaStringCheckUTF8,
		},
		{
			description:             "Modify Java String Check UTF8 on a wkt file",
			subDir:                  "wktimport",
			file:                    "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyJavaStringCheckUTF8,
			fileOptionPath:          internal.JavaStringCheckUtf8Path,
			override:                NewValueOverride(true),
			expectedValue:           false,
			assertFunc:              assertJavaStringCheckUTF8,
		},
		{
			description:             "Modify Objc Class Prefix on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyObjcClassPrefix,
			fileOptionPath:          internal.ObjcClassPrefixPath,
			override:                NewValueOverride("Override"),
			expectedValue:           "Override",
			assertFunc:              assertObjcClassPrefix,
		},
		{
			description:             "Modify Objc Class Prefix with nil on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyObjcClassPrefix,
			fileOptionPath:          internal.ObjcClassPrefixPath,
			override:                nil,
			expectedValue:           "",
			assertFunc:              assertObjcClassPrefix,
		},
		{
			description:    "Modify Objc Class Prefix with nil on package name with one part and version",
			subDir:         filepath.Join("objcoptions", "single"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			override:       nil,
			expectedValue:  "AXX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "Modify Objc Class Prefix with nil on package name with two parts and version",
			subDir:         filepath.Join("objcoptions", "double"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			override:       nil,
			expectedValue:  "AWX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "Modify Objc Class Prefix with nil on package name with three parts and a version",
			subDir:         filepath.Join("objcoptions", "triple"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			override:       nil,
			expectedValue:  "AWD",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "Modify Objc Class Prefix with nil on package name with three parts without version",
			subDir:         filepath.Join("objcoptions", "unversioned"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			override:       nil,
			expectedValue:  "AWD",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "Modify Objc Class Prefix with nil on google protobuf file",
			subDir:         filepath.Join("objcoptions", "gpb"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			override:       nil,
			expectedValue:  "GPX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:              "Modify Objc Class Prefix on a wkt file",
			subDir:                   "wktimport",
			file:                     "google/protobuf/timestamp.proto",
			modifyFunc:               ModifyObjcClassPrefix,
			fileOptionPath:           internal.ObjcClassPrefixPath,
			override:                 NewValueOverride("Override"),
			expectedValue:            "GPB",
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertObjcClassPrefix,
		},
		{
			description:             "Modify Optimize For to SPEED on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                NewValueOverride(descriptorpb.FileOptions_SPEED),
			expectedValue:           descriptorpb.FileOptions_SPEED,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:             "Modify Optimize For to CODE_SIZE on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:           descriptorpb.FileOptions_CODE_SIZE,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:             "Modify Optimize For to LITE_RUNTIME on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                NewValueOverride(descriptorpb.FileOptions_LITE_RUNTIME),
			expectedValue:           descriptorpb.FileOptions_LITE_RUNTIME,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:             "Modify Optimize For with nil on a file with empty options",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                nil,
			expectedValue:           descriptorpb.FileOptions_SPEED,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:    "Modify Optimize For to CODE_SIZE on a file with all options",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyOptimizeFor,
			fileOptionPath: internal.OptimizeForPath,
			override:       NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:  descriptorpb.FileOptions_CODE_SIZE,
			assertFunc:     assertOptimizeFor,
		},
		{
			description:              "Modify Optimize For to SPEED on a file with all options",
			subDir:                   "alloptions",
			file:                     "a.proto",
			modifyFunc:               ModifyOptimizeFor,
			fileOptionPath:           internal.OptimizeForPath,
			override:                 NewValueOverride(descriptorpb.FileOptions_SPEED),
			expectedValue:            descriptorpb.FileOptions_SPEED,
			shouldKeepSourceCodeInfo: true,
			assertFunc:               assertOptimizeFor,
		},
		{
			description:             "Modify Optimize For to CODE_SIZE on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:           descriptorpb.FileOptions_CODE_SIZE,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:             "Modify Optimize For to CODE_SIZE on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyOptimizeFor,
			fileOptionPath:          internal.OptimizeForPath,
			override:                NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:           descriptorpb.FileOptions_SPEED,
			assertFunc:              assertOptimizeFor,
		},
		{
			description:             "Modify Php metadata namespace with value on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpMetadataNamespace,
			fileOptionPath:          internal.PhpMetadataNamespacePath,
			override:                NewValueOverride("valueoverride"),
			expectedValue:           "valueoverride",
			assertFunc:              assertPhpMetadataNamespace,
		},
		{
			description:             "Modify Php metadata namespace with nil on file with empty options and empty proto package",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpMetadataNamespace,
			fileOptionPath:          internal.PhpMetadataNamespacePath,
			override:                nil,
			expectedValue:           "",
			assertFunc:              assertPhpMetadataNamespace,
		},
		{
			description:    "Modify Php metadata namespace with nil on file with package name with one part",
			subDir:         filepath.Join("phpoptions", "single"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			override:       nil,
			expectedValue:  `Acme\V1\GPBMetadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "Modify Php metadata namespace with nil on file with package name with two parts",
			subDir:         filepath.Join("phpoptions", "double"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			override:       nil,
			expectedValue:  `Acme\Weather\V1\GPBMetadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "Modify Php metadata namespace with nil on file with package name with three parts",
			subDir:         filepath.Join("phpoptions", "triple"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			override:       nil,
			expectedValue:  `Acme\Weather\Data\V1\GPBMetadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "Modify Php metadata namespace with nil on file with package name with a reserved keyword",
			subDir:         filepath.Join("phpoptions", "reserved"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			override:       nil,
			expectedValue:  `Acme\Error_\V1\GPBMetadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:             "Modify Php metadata namespace on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpMetadataNamespace,
			fileOptionPath:          internal.PhpMetadataNamespacePath,
			override:                NewValueOverride("Override"),
			expectedValue:           "Override",
			assertFunc:              assertPhpMetadataNamespace,
		},
		{
			description:             "Modify Php metadata namespace on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpMetadataNamespace,
			fileOptionPath:          internal.PhpMetadataNamespacePath,
			override:                NewValueOverride("Override"),
			expectedValue:           "",
			assertFunc:              assertPhpMetadataNamespace,
		},
		{
			description:    "Modify Php Namespace on a file with empty options",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			override:       NewValueOverride("ValueOverride"),
			expectedValue:  "ValueOverride",
			assertFunc:     assertPhpNamespace,
		},
		{
			description:             "Modify Php Namespace on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpNamespace,
			fileOptionPath:          internal.PhpNamespacePath,
			override:                NewValueOverride("ValueOverride"),
			expectedValue:           "ValueOverride",
			assertFunc:              assertPhpNamespace,
		},
		{
			description:    "Modify Php Namespace default value",
			subDir:         filepath.Join("phpoptions", "double"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			override:       nil,
			expectedValue:  `Acme\Weather\V1`,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:             "Modify Php Namespace on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyPhpNamespace,
			fileOptionPath:          internal.PhpNamespacePath,
			override:                NewValueOverride("ValueOverride"),
			expectedValue:           "",
			assertFunc:              assertPhpNamespace,
		},
		{
			description:             "Modify Ruby package on a file with empty options and empty packages",
			subDir:                  "emptyoptions",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyRubyPackage,
			fileOptionPath:          internal.RubyPackagePath,
			override:                NewValueOverride("valueoverride"),
			expectedValue:           "valueoverride",
			assertFunc:              assertRubyPackage,
		},
		{
			description:    "Modify Ruby package with nil on a file with a package name with one part",
			subDir:         filepath.Join("rubyoptions", "single"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			override:       nil,
			expectedValue:  `Acme::V1`,
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "Modify Ruby package with nil on a file with a package name with two parts",
			subDir:         filepath.Join("rubyoptions", "double"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			override:       nil,
			expectedValue:  `Acme::Weather::V1`,
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "Modify Ruby package with nil on a file with a package name with three parts",
			subDir:         filepath.Join("rubyoptions", "triple"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			override:       nil,
			expectedValue:  `Acme::Weather::Data::V1`,
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "Modify Ruby package with nil on a file with a package name with underscore",
			subDir:         filepath.Join("rubyoptions", "underscore"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			override:       nil,
			expectedValue:  `Acme::Weather::FooBar::V1`,
			assertFunc:     assertRubyPackage,
		},
		{
			description:             "Modify Ruby package on a file that imports wkt",
			subDir:                  "wktimport",
			file:                    "a.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyRubyPackage,
			fileOptionPath:          internal.RubyPackagePath,
			override:                NewValueOverride("Override"),
			expectedValue:           "Override",
			assertFunc:              assertRubyPackage,
		},
		{
			description:             "Modify Ruby package on a wkt file",
			subDir:                  "wktimport",
			file:                    "google/protobuf/timestamp.proto",
			fileHasNoSourceCodeInfo: true,
			modifyFunc:              ModifyRubyPackage,
			fileOptionPath:          internal.RubyPackagePath,
			override:                NewValueOverride("Override"),
			expectedValue:           "",
			assertFunc:              assertRubyPackage,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			{
				// Get image with source code info.
				image := bufimagemodifytesting.GetTestImage(
					t,
					filepath.Join(baseDir, test.subDir),
					true,
				)
				markSweeper := NewMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				if test.fileHasNoSourceCodeInfo {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
						t,
						image,
						test.fileOptionPath,
						true,
						bufimagemodifytesting.AssertSourceCodeInfoWithIgnoreWKT(),
					)
				} else {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmptyForFile(
						t,
						imageFile,
						test.fileOptionPath,
					)
				}
				err := test.modifyFunc(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				test.assertFunc(t, test.expectedValue, imageFile.Proto())
				if test.shouldKeepSourceCodeInfo {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmptyForFile(
						t,
						imageFile,
						test.fileOptionPath,
					)
				} else {
					bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmptyForFile(
						t,
						imageFile,
						test.fileOptionPath,
						true,
					)
				}
			}
			{
				// Get image without source code info.
				image := bufimagemodifytesting.GetTestImage(
					t,
					filepath.Join(baseDir, test.subDir),
					false,
				)
				bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
					t,
					image,
					test.fileOptionPath,
					false,
				)
				markSweeper := NewMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				err := test.modifyFunc(
					markSweeper,
					imageFile,
					test.override,
				)
				require.NoError(t, err)
				err = markSweeper.Sweep()
				require.NoError(t, err)
				require.NotNil(t, imageFile.Proto())
				test.assertFunc(t, test.expectedValue, imageFile.Proto())
				bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(
					t,
					image,
					test.fileOptionPath,
					false,
				)
			}
		})
	}
}

func TestModifyError(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description        string
		subDir             string
		file               string
		modifyFunc         func(Marker, bufimage.ImageFile, Override) error
		override           Override
		expectedErrMessage string
	}{
		{
			description:        "Test bool override for java package",
			subDir:             "javaoptions",
			file:               "java_file.proto",
			modifyFunc:         ModifyJavaPackage,
			override:           NewValueOverride(true),
			expectedErrMessage: "a valid override is required for java_package",
		},
		{
			description:        "Test optimize mode override for java package",
			subDir:             "javaoptions",
			file:               "java_file.proto",
			modifyFunc:         ModifyJavaPackage,
			override:           NewValueOverride[descriptorpb.FileOptions_OptimizeMode](descriptorpb.FileOptions_CODE_SIZE),
			expectedErrMessage: "a valid override is required for java_package",
		},
		{
			description:        "Test string override for CC Enable Arenas",
			subDir:             "ccoptions",
			file:               "a.proto",
			modifyFunc:         ModifyCcEnableArenas,
			override:           NewValueOverride("string"),
			expectedErrMessage: "a valid override is required for cc_enable_arenas",
		},
		{
			description:        "Test prefix override for CC Enable Arenas",
			subDir:             "ccoptions",
			file:               "a.proto",
			modifyFunc:         ModifyCcEnableArenas,
			override:           NewPrefixOverride("string"),
			expectedErrMessage: "a valid override is required for cc_enable_arenas",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join(baseDir, test.subDir),
				true,
			)
			markSweeper := NewMarkSweeper(image)
			require.NotNil(t, markSweeper)
			imageFile := image.GetFile(test.file)
			require.NotNil(t, imageFile)
			err := test.modifyFunc(
				markSweeper,
				imageFile,
				test.override,
			)
			require.ErrorContains(t, err, test.expectedErrMessage)
		})
	}
}

func assertJavaPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaPackage())
}

func assertCcEnableArenas(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetCcEnableArenas())
}

func assertCsharpNamespace(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetCsharpNamespace())
}

func assertGoPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetGoPackage())
}

func assertJavaMultipleFiles(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaMultipleFiles())
}

func assertJavaOuterClassName(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaOuterClassname())
}

func assertJavaStringCheckUTF8(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaStringCheckUtf8())
}

func assertObjcClassPrefix(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetObjcClassPrefix())
}

func assertOptimizeFor(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetOptimizeFor())
}

func assertPhpMetadataNamespace(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetPhpMetadataNamespace())
}

func assertPhpNamespace(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetPhpNamespace())
}

func assertRubyPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetRubyPackage())
}
