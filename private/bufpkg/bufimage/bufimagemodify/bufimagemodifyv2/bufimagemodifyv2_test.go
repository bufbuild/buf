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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifySingleOption(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description    string
		subDir         string
		file           string
		modifyFunc     func(Marker, bufimage.ImageFile, ...ModifyOption) error
		fileOptionPath []int32
		modifyOptions  []ModifyOption
		expectedValue  interface{}
		assertFunc     func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
		shouldNotMark  bool
	}{
		{
			description:    "Modify Java Package with prefix on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefix(t, "prefix"),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with value on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "com.example"),
			expectedValue:  "com.example",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix", "suffix"),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package without override on file without a proto package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  make([]ModifyOption, 0),
			expectedValue:  "",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with override value the same as java package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
		{
			description:    "Modify Java Package with value on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "bar"),
			expectedValue:  "bar",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package without override on a file with a proto package",
			subDir:         "javaemptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  make([]ModifyOption, 0),
			expectedValue:  "foo",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefix(t, "prefix.override"),
			expectedValue:  "prefix.override.acme.weather",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix.override", "override.suffix"),
			expectedValue:  "prefix.override.acme.weather.override.suffix",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with suffix on file with a proto package",
			subDir:         "javaoptions",
			file:           "java_file.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithSuffix(t, "override.suffix"),
			expectedValue:  "acme.weather.override.suffix",
			assertFunc:     assertJavaPackage,
		},
		{
			description:    "Modify Java Package with prefix and suffix on a wkt file",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithPrefixAndSuffix(t, "prefix.override", "override.suffix"),
			expectedValue:  "com.google.protobuf",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
		{
			description:    "Modify Java Package with value on a wkt file",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaPackage,
			fileOptionPath: internal.JavaPackagePath,
			modifyOptions:  modifyWithStringValue(t, "value"),
			expectedValue:  "com.google.protobuf",
			assertFunc:     assertJavaPackage,
			shouldNotMark:  true,
		},
		{
			description:    "java outer classname without override on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaOuterClassname,
			fileOptionPath: internal.JavaOuterClassnamePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "AProto",
			assertFunc:     assertJavaOuterClassName,
		},
		{
			description:    "java outer classname with override on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaOuterClassname,
			fileOptionPath: internal.JavaOuterClassnamePath,
			modifyOptions:  modifyWithStringValue(t, "OverrideProto"),
			expectedValue:  "OverrideProto",
			assertFunc:     assertJavaOuterClassName,
		},
		{
			description:    "java outer classname with override on a file with this option equal to the same value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaOuterClassname,
			fileOptionPath: internal.JavaOuterClassnamePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertJavaOuterClassName,
		},
		{
			description:    "java outer classname with override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaOuterClassname,
			fileOptionPath: internal.JavaOuterClassnamePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "TimestampProto",
			shouldNotMark:  true,
			assertFunc:     assertJavaOuterClassName,
		},
		{
			description:    "objc class prefix without override on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc class prefix without override on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc class prefix without override on a file with this option and a package",
			subDir:         filepath.Join("objcoptions", "double"),
			file:           "objc.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "AWX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc namespace without override on a file without this option but with a package",
			subDir:         "wktimport",
			file:           "a.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "AWX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc class prefix with override on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  modifyWithStringValue(t, "OPX"),
			expectedValue:  "OPX",
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc class prefix with override on a file with this option equal to the same value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "objc class prefix with override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyObjcClassPrefix,
			fileOptionPath: internal.ObjcClassPrefixPath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "GPB",
			shouldNotMark:  true,
			assertFunc:     assertObjcClassPrefix,
		},
		{
			description:    "csharp namespace without override on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace without override on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace without override on a file with this option and a package",
			subDir:         filepath.Join("csharpoptions", "triple"),
			file:           "csharp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "Acme.Weather.Data.V1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace without override on a file without this option but with a package",
			subDir:         "wktimport",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "Acme.Weather.V1alpha1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace with value override",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "Override.Value"),
			expectedValue:  "Override.Value",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace prefix on a file without package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithPrefix(t, "Override.Prefix"),
			expectedValue:  "",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace prefix on a file with package",
			subDir:         filepath.Join("csharpoptions", "double"),
			file:           "csharp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithPrefix(t, "Override.Prefix"),
			expectedValue:  "Override.Prefix.Acme.Weather.V1",
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace with value override equal to the same value from file",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace with value override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "Google.Protobuf.WellKnownTypes",
			shouldNotMark:  true,
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "csharp namespace prefix on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyCsharpNamespace,
			fileOptionPath: internal.CsharpNamespacePath,
			modifyOptions:  modifyWithPrefix(t, "foo"),
			expectedValue:  "Google.Protobuf.WellKnownTypes",
			shouldNotMark:  true,
			assertFunc:     assertCsharpNamespace,
		},
		{
			description:    "php namespace without override on a file without this option and without a package",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, `Foo\Bar`),
			expectedValue:  `Foo\Bar`,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php namespace without override on a file with this option but without a package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php namespace without override on a file with this option and a package",
			subDir:         filepath.Join("phpoptions", "underscore"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  `Acme\Weather\FooBar\V1`,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php namespace with value override",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, `Override\Value`),
			expectedValue:  `Override\Value`,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php namespace with value equal to the same value from file",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php namespace with value override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyPhpNamespace,
			fileOptionPath: internal.PhpNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "",
			shouldNotMark:  true,
			assertFunc:     assertPhpNamespace,
		},
		{
			description:    "php metadata namespace without override on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace without override on a file with this option and a package",
			subDir:         filepath.Join("phpoptions", "underscore"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  `Acme\Weather\FooBar\V1`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace with value override on a file with this option and a package",
			subDir:         filepath.Join("phpoptions", "underscore"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "Override"),
			expectedValue:  "Override",
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace with suffix override on a file with this option and a package",
			subDir:         filepath.Join("phpoptions", "underscore"),
			file:           "php.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithSuffix(t, "Metadata"),
			expectedValue:  `Acme\Weather\FooBar\V1\Metadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace with suffix override on a file without this option but with a package",
			subDir:         "wktimport",
			file:           "a.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithSuffix(t, "SpecialMetadata"),
			expectedValue:  `Acme\Weather\V1alpha1\SpecialMetadata`,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace with value override equal to the same value from file",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace with value override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "",
			shouldNotMark:  true,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "php metadata namespace suffix on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyPhpMetadataNamespace,
			fileOptionPath: internal.PhpMetadataNamespacePath,
			modifyOptions:  modifyWithSuffix(t, "foo"),
			expectedValue:  "",
			shouldNotMark:  true,
			assertFunc:     assertPhpMetadataNamespace,
		},
		{
			description:    "ruby package without override on a file without this option and without a package",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "",
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package without override on a file with this option and a package",
			subDir:         filepath.Join("rubyoptions", "underscore"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  []ModifyOption{},
			expectedValue:  "Acme::Weather::FooBar::V1",
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package with value override on a file with this option and a package",
			subDir:         filepath.Join("rubyoptions", "underscore"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithStringValue(t, "Override"),
			expectedValue:  "Override",
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package with suffix override on a file with this option and a package",
			subDir:         filepath.Join("rubyoptions", "underscore"),
			file:           "ruby.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithSuffix(t, "Protos"),
			expectedValue:  "Acme::Weather::FooBar::V1::Protos",
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package with suffix override on a file with this option but without a package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithSuffix(t, "Protos"),
			expectedValue:  "",
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package with value equal to the same value from file",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "foo",
			shouldNotMark:  true,
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package value on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithStringValue(t, "foo"),
			expectedValue:  "",
			shouldNotMark:  true,
			assertFunc:     assertRubyPackage,
		},
		{
			description:    "ruby package suffix on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyRubyPackage,
			fileOptionPath: internal.RubyPackagePath,
			modifyOptions:  modifyWithSuffix(t, "foo"),
			expectedValue:  "",
			shouldNotMark:  true,
			assertFunc:     assertRubyPackage,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			// Get image with source code info.
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join(baseDir, test.subDir),
				true,
			)
			markSweeper := newMarkSweeper(image)
			require.NotNil(t, markSweeper)
			imageFile := image.GetFile(test.file)
			require.NotNil(t, imageFile)
			_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
			require.False(t, ok)
			err := test.modifyFunc(
				markSweeper,
				imageFile,
				test.modifyOptions...,
			)
			require.NoError(t, err)
			err = markSweeper.Sweep()
			require.NoError(t, err)
			require.NotNil(t, imageFile.Proto())
			test.assertFunc(t, test.expectedValue, imageFile.Proto())
			fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
			if test.shouldNotMark {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				_, ok = fileKeys[internal.GetPathKey(test.fileOptionPath)]
				require.True(t, ok)
			}
		})
	}
}

func TestModifySingleOptionWithOptionalOverride(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description    string
		subDir         string
		file           string
		modifyFunc     func(Marker, bufimage.ImageFile, Override) error
		fileOptionPath []int32
		override       Override
		expectedValue  interface{}
		assertFunc     func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
		shouldNotMark  bool
	}{
		{
			description:    "java multiple files on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaMultipleFiles,
		},
		{
			description:    "java multiple files on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaMultipleFiles,
		},
		{
			description:    "java multiple files with override on a file with this option equal to the same value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(false),
			expectedValue:  false,
			shouldNotMark:  true,
			assertFunc:     assertJavaMultipleFiles,
		},
		{
			description:    "java multiple files with override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaMultipleFiles,
			fileOptionPath: internal.JavaMultipleFilesPath,
			override:       NewValueOverride(false),
			expectedValue:  true,
			shouldNotMark:  true,
			assertFunc:     assertJavaMultipleFiles,
		},
		{
			description:    "java string check utf8 on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaStringCheckUtf8,
			fileOptionPath: internal.JavaStringCheckUtf8Path,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaStringCheckUTF8,
		},
		{
			description:    "java string check utf8 on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaStringCheckUtf8,
			fileOptionPath: internal.JavaStringCheckUtf8Path,
			override:       NewValueOverride(true),
			expectedValue:  true,
			assertFunc:     assertJavaStringCheckUTF8,
		},
		{
			description:    "java string check utf8 with override on a file with this option equal to the same value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyJavaStringCheckUtf8,
			fileOptionPath: internal.JavaStringCheckUtf8Path,
			override:       NewValueOverride(false),
			expectedValue:  false,
			shouldNotMark:  true,
			assertFunc:     assertJavaStringCheckUTF8,
		},
		{
			description:    "java string check utf8 with override on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyJavaStringCheckUtf8,
			fileOptionPath: internal.JavaStringCheckUtf8Path,
			override:       NewValueOverride(true),
			expectedValue:  false,
			shouldNotMark:  true,
			assertFunc:     assertJavaStringCheckUTF8,
		},
		{
			description:    "go package prefix on a file without this option without a package",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewPrefixOverride("override/prefix"),
			expectedValue:  "override/prefix",
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package prefix on a file with this option without a package",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewPrefixOverride("override/prefix"),
			expectedValue:  "override/prefix",
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package prefix on a file without this option but with a package",
			subDir:         "wktimport",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewPrefixOverride("override/prefix"),
			expectedValue:  "override/prefix;weatherv1alpha1",
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package prefix on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       newPrefixOverride("override/prefix"),
			expectedValue:  "google.golang.org/protobuf/types/known/timestamppb",
			shouldNotMark:  true,
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewValueOverride("override/value"),
			expectedValue:  "override/value",
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package on a file with this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewValueOverride("override/value"),
			expectedValue:  "override/value",
			assertFunc:     assertGoPackage,
		},
		{
			description:    "go package on a file with this option with equal value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       NewValueOverride("foo"),
			expectedValue:  "foo",
			assertFunc:     assertGoPackage,
			shouldNotMark:  true,
		},
		{
			description:    "go package on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyGoPackage,
			fileOptionPath: internal.GoPackagePath,
			override:       newValueOverride("override/value"),
			expectedValue:  "google.golang.org/protobuf/types/known/timestamppb",
			shouldNotMark:  true,
			assertFunc:     assertGoPackage,
		},
		{
			description:    "optimize for on a file without this option",
			subDir:         "emptyoptions",
			file:           "a.proto",
			modifyFunc:     ModifyOptimizeFor,
			fileOptionPath: internal.OptimizeForPath,
			override:       NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:  descriptorpb.FileOptions_CODE_SIZE,
			assertFunc:     assertOptimizeFor,
		},
		{
			description:    "optimize for on a file without this option",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyOptimizeFor,
			fileOptionPath: internal.OptimizeForPath,
			override:       NewValueOverride(descriptorpb.FileOptions_CODE_SIZE),
			expectedValue:  descriptorpb.FileOptions_CODE_SIZE,
			assertFunc:     assertOptimizeFor,
		},
		{
			description:    "optimize for on a file with this option with equal value",
			subDir:         "alloptions",
			file:           "a.proto",
			modifyFunc:     ModifyOptimizeFor,
			fileOptionPath: internal.OptimizeForPath,
			override:       NewValueOverride(descriptorpb.FileOptions_SPEED),
			expectedValue:  descriptorpb.FileOptions_SPEED,
			assertFunc:     assertOptimizeFor,
			shouldNotMark:  true,
		},
		{
			description:    "optmize for on a wkt",
			subDir:         "wktimport",
			file:           "google/protobuf/timestamp.proto",
			modifyFunc:     ModifyOptimizeFor,
			fileOptionPath: internal.OptimizeForPath,
			override:       NewValueOverride(descriptorpb.FileOptions_LITE_RUNTIME),
			expectedValue:  descriptorpb.FileOptions_SPEED,
			shouldNotMark:  true,
			assertFunc:     assertOptimizeFor,
		},
	}
	for _, test := range tests {
		test := test
		for _, includeSourceCodeInfo := range []bool{true, false} {
			includeSourceCodeInfo := includeSourceCodeInfo
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				image := bufimagemodifytesting.GetTestImage(
					t,
					filepath.Join(baseDir, test.subDir),
					includeSourceCodeInfo,
				)
				markSweeper := newMarkSweeper(image)
				require.NotNil(t, markSweeper)
				imageFile := image.GetFile(test.file)
				require.NotNil(t, imageFile)
				_, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				require.False(t, ok)
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
				fileKeys, ok := markSweeper.sourceCodeInfoPaths[imageFile.Path()]
				if test.shouldNotMark {
					require.False(t, ok)
				} else {
					require.True(t, ok)
					_, ok = fileKeys[internal.GetPathKey(test.fileOptionPath)]
					require.True(t, ok)
				}
			})
		}
	}
}

func modifyWithStringValue(t *testing.T, value string) []ModifyOption {
	option, err := ModifyWithOverride(NewValueOverride(value))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithPrefix(t *testing.T, prefix string) []ModifyOption {
	option, err := ModifyWithOverride(NewPrefixOverride(prefix))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithSuffix(t *testing.T, prefix string) []ModifyOption {
	option, err := ModifyWithOverride(NewSuffixOverride(prefix))
	require.NoError(t, err)
	return []ModifyOption{option}
}

func modifyWithPrefixAndSuffix(t *testing.T, prefix string, suffix string) []ModifyOption {
	option, err := ModifyWithOverride(
		NewPrefixSuffixOverride(
			prefix,
			suffix,
		),
	)
	require.NoError(t, err)
	return []ModifyOption{option}
}

func assertJavaPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaPackage())
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
