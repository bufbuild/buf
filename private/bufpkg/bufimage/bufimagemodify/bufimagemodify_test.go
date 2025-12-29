// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufimagemodify

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagetesting"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifyImage(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description               string
		dirPathToFullName         map[string]string
		config                    bufconfig.GenerateManagedConfig
		filePathToExpectedOptions map[string]*descriptorpb.FileOptions
	}{
		{
			description: "nil_config",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(false, nil, nil),
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"foo_empty/with_package.proto": nil,
				"bar_all/with_package.proto": {
					CcEnableArenas:       proto.Bool(false),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(false),
					JavaOuterClassname:   proto.String("bar"),
					JavaPackage:          proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String("bar"),
					PhpNamespace:         proto.String("bar"),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
			},
		},
		{
			description: "empty_config",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{},
			),
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"foo_empty/with_package.proto": {
					// CcEnableArena's default value is true
					CsharpNamespace: proto.String("Foo.Empty"),
					// GoPackage is not modified by default
					JavaMultipleFiles:  proto.Bool(true),
					JavaOuterClassname: proto.String("WithPackageProto"),
					JavaPackage:        proto.String("com.foo.empty"),
					// JavaStringCheckUtf8 is not modified by default
					ObjcClassPrefix: proto.String("FEX"),
					// OptimizeFor tries to modify this value to SPEED, which is already the default
					// Empty is a keyword in php
					PhpMetadataNamespace: proto.String(`Foo\Empty_\GPBMetadata`),
					PhpNamespace:         proto.String(`Foo\Empty_`),
					RubyPackage:          proto.String("Foo::Empty"),
				},
				"foo_empty/without_package.proto": {
					// CcEnableArena's default value is true
					// GoPackage is not modified by default
					JavaMultipleFiles:  proto.Bool(true),
					JavaOuterClassname: proto.String("WithoutPackageProto"),
					// JavaStringCheckUtf8 is not modified by default
					// OptimizeFor tries to modify this value to SPEED, which is already the default
				},
				"bar_all/with_package.proto": {
					CcEnableArenas:       proto.Bool(true),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("Bar.All"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(true),
					JavaOuterClassname:   proto.String("WithPackageProto"),
					JavaPackage:          proto.String("com.bar.all"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("BAX"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String(`Bar\All\GPBMetadata`),
					PhpNamespace:         proto.String(`Bar\All`),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("Bar::All"),
					SwiftPrefix:          proto.String("bar"),
				},
				"bar_all/without_package.proto": {
					CcEnableArenas:       proto.Bool(true),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(true),
					JavaOuterClassname:   proto.String("WithoutPackageProto"),
					JavaPackage:          proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String(`bar`),
					PhpNamespace:         proto.String(`bar`),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
			},
		},
	}
	for _, testcase := range testcases {
		for _, includeSourceInfo := range []bool{true, false} {
			t.Run(testcase.description, func(t *testing.T) {
				t.Parallel()
				image := testGetImageFromDirs(t, testcase.dirPathToFullName, includeSourceInfo)
				err := Modify(
					image,
					testcase.config,
				)
				require.NoError(t, err)
				for filePath, expectedOptions := range testcase.filePathToExpectedOptions {
					imageFile := image.GetFile(filePath)
					require.NotNil(t, imageFile)
					require.Empty(
						t,
						cmp.Diff(expectedOptions, imageFile.FileDescriptorProto().GetOptions(), protocmp.Transform()),
						imageFile.FileDescriptorProto().GetOptions(),
					)
				}
			})
		}
	}
}

func TestModifyImageFile(
	t *testing.T,
) {
	t.Parallel()
	testcases := []struct {
		description                           string
		dirPathToFullName                     map[string]string
		config                                bufconfig.GenerateManagedConfig
		modifyFunc                            func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig, ...ModifyOption) error
		filePathToExpectedOptions             map[string]*descriptorpb.FileOptions
		filePathToExpectedMarkedLocationPaths map[string][][]int32
	}{
		{
			description: "cc_enable_arena",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/bar", bufconfig.FileOptionCcEnableArenas, false),
				},
			),
			modifyFunc: modifyCcEnableArenas,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"foo_empty/without_package.proto": nil,
				"bar_empty/without_package.proto": {
					CcEnableArenas: proto.Bool(false),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"bar_empty/without_package.proto": {ccEnableArenasPath},
			},
		},
		{
			description: "csharp_namespace",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "foo_empty/with_package.proto", "", "", bufconfig.FileOptionCsharpNamespacePrefix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "bar_empty", "buf.build/acme/bar", bufconfig.FileOptionCsharpNamespacePrefix, "BarPrefix"),
					newTestFileOptionOverrideRule(t, "bar_empty/without_package.proto", "buf.build/acme/bar", bufconfig.FileOptionCsharpNamespace, "BarValue"),
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/foo", bufconfig.FileOptionCsharpNamespace, "FooValue"),
					newTestFileOptionOverrideRule(t, "foo_empty", "buf.build/acme/foo", bufconfig.FileOptionCsharpNamespacePrefix, "FooPrefix"),
				},
			),
			modifyFunc: modifyCsharpNamespace,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"bar_empty/with_package.proto": {
					CsharpNamespace: proto.String("BarPrefix.Bar.Empty"),
				},
				"bar_empty/without_package.proto": {
					CsharpNamespace: proto.String("BarValue"),
				},
				"foo_empty/with_package.proto": {
					CsharpNamespace: proto.String("FooValue"),
				},
				"foo_empty/without_package.proto": nil,
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"bar_empty/with_package.proto":    {csharpNamespacePath},
				"bar_empty/without_package.proto": {csharpNamespacePath},
				"foo_empty/with_package.proto":    {csharpNamespacePath},
			},
		},
		{
			description: "go_package",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "foo_empty/with_package.proto", "", "", bufconfig.FileOptionGoPackagePrefix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "bar_empty", "buf.build/acme/bar", bufconfig.FileOptionGoPackagePrefix, "barprefix"),
					newTestFileOptionOverrideRule(t, "bar_empty/without_package.proto", "buf.build/acme/bar", bufconfig.FileOptionGoPackage, "barvalue"),
					newTestFileOptionOverrideRule(t, "foo_empty/with_package.proto", "buf.build/acme/foo", bufconfig.FileOptionGoPackage, "foovalue"),
					newTestFileOptionOverrideRule(t, "foo_empty", "buf.build/acme/foo", bufconfig.FileOptionGoPackagePrefix, "fooprefix"),
				},
			),
			modifyFunc: modifyGoPackage,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"bar_empty/with_package.proto": {
					GoPackage: proto.String("barprefix/bar_empty"),
				},
				"bar_empty/without_package.proto": {
					GoPackage: proto.String("barvalue"),
				},
				"foo_empty/with_package.proto": {
					GoPackage: proto.String("foovalue"),
				},
				"foo_empty/without_package.proto": {
					GoPackage: proto.String("fooprefix/foo_empty"),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"bar_empty/with_package.proto":    {goPackagePath},
				"bar_empty/without_package.proto": {goPackagePath},
				"foo_empty/with_package.proto":    {goPackagePath},
				"foo_empty/without_package.proto": {goPackagePath},
			},
		},
		{
			description: "java_package_prefix",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "bar_empty", "", "", bufconfig.FileOptionJavaPackagePrefix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/bar", bufconfig.FileOptionJavaPackagePrefix, "barprefix"),
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/foo", bufconfig.FileOptionJavaPackageSuffix, "foosuffix"),
				},
			),
			modifyFunc: modifyJavaPackage,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"foo_empty/with_package.proto": {
					// default prefix and override suffix
					JavaPackage: proto.String("com.foo.empty.foosuffix"),
				},
				// prefix is disabled
				"bar_empty/with_package.proto": nil,
				// prefix is overridden
				"bar_all/with_package.proto": {
					JavaPackage: proto.String("barprefix.bar.all"),
					// below this point are the values from the file
					CcEnableArenas:       proto.Bool(false),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(false),
					JavaOuterClassname:   proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String("bar"),
					PhpNamespace:         proto.String("bar"),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
				// not modified because it doesn't have a package
				"foo_empty/without_package.proto": nil,
				"bar_empty/without_package.proto": nil,
				"foo_all/without_package.proto": {
					// values are from the file
					CcEnableArenas:       proto.Bool(true),
					CcGenericServices:    proto.Bool(true),
					CsharpNamespace:      proto.String("foo"),
					GoPackage:            proto.String("foo"),
					JavaGenericServices:  proto.Bool(true),
					JavaMultipleFiles:    proto.Bool(true),
					JavaOuterClassname:   proto.String("foo"),
					JavaPackage:          proto.String("foo"),
					JavaStringCheckUtf8:  proto.Bool(true),
					ObjcClassPrefix:      proto.String("foo"),
					OptimizeFor:          descriptorpb.FileOptions_CODE_SIZE.Enum(),
					PhpClassPrefix:       proto.String("foo"),
					PhpMetadataNamespace: proto.String("foo"),
					PhpNamespace:         proto.String("foo"),
					PyGenericServices:    proto.Bool(true),
					RubyPackage:          proto.String("foo"),
					SwiftPrefix:          proto.String("foo"),
				},
				"bar_all/without_package.proto": {
					// values are from the file
					CcEnableArenas:       proto.Bool(false),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(false),
					JavaOuterClassname:   proto.String("bar"),
					JavaPackage:          proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String("bar"),
					PhpNamespace:         proto.String("bar"),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"foo_empty/with_package.proto": {javaPackagePath},
				"bar_all/with_package.proto":   {javaPackagePath},
			},
		},
		{
			description: "java_package_suffix",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "bar_empty", "", "", bufconfig.FileOptionJavaPackageSuffix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
				},
			),
			modifyFunc: modifyJavaPackage,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"foo_empty/with_package.proto": {
					// only suffix matches, but apply both prefix and suffix
					JavaPackage: proto.String("com.foo.empty.suffix"),
				},
				"bar_empty/with_package.proto": {
					// only prefix because suffix is disabled
					JavaPackage: proto.String("com.bar.empty"),
				},
				"bar_all/with_package.proto": {
					JavaPackage: proto.String("com.bar.all.suffix"),
					// below this point are the values from the file
					CcEnableArenas:       proto.Bool(false),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(false),
					JavaOuterClassname:   proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String("bar"),
					PhpNamespace:         proto.String("bar"),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
				// not modified
				"foo_empty/without_package.proto": nil,
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"foo_empty/with_package.proto": {javaPackagePath},
				"bar_empty/with_package.proto": {javaPackagePath},
				"bar_all/with_package.proto":   {javaPackagePath},
			},
		},
		{
			description: "java_package",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "bar_empty", "", "", bufconfig.FileOptionJavaPackage, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/bar", bufconfig.FileOptionJavaPackage, "bar.value"),
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/foo", bufconfig.FileOptionJavaPackage, "foo.value"),
				},
			),
			modifyFunc: modifyJavaPackage,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				// bar_empty disabled
				"bar_empty/with_package.proto":    nil,
				"bar_empty/without_package.proto": nil,
				"bar_all/with_package.proto": {
					JavaPackage:          proto.String("bar.value"),
					CcEnableArenas:       proto.Bool(false),
					CcGenericServices:    proto.Bool(false),
					CsharpNamespace:      proto.String("bar"),
					GoPackage:            proto.String("bar"),
					JavaGenericServices:  proto.Bool(false),
					JavaMultipleFiles:    proto.Bool(false),
					JavaOuterClassname:   proto.String("bar"),
					JavaStringCheckUtf8:  proto.Bool(false),
					ObjcClassPrefix:      proto.String("bar"),
					OptimizeFor:          descriptorpb.FileOptions_SPEED.Enum(),
					PhpClassPrefix:       proto.String("bar"),
					PhpMetadataNamespace: proto.String("bar"),
					PhpNamespace:         proto.String("bar"),
					PyGenericServices:    proto.Bool(false),
					RubyPackage:          proto.String("bar"),
					SwiftPrefix:          proto.String("bar"),
				},
				"foo_empty/with_package.proto": {
					JavaPackage: proto.String("foo.value"),
				},
				"foo_empty/without_package.proto": {
					JavaPackage: proto.String("foo.value"),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"foo_empty/with_package.proto":    {javaPackagePath},
				"foo_empty/without_package.proto": {javaPackagePath},
				"bar_all/with_package.proto":      {javaPackagePath},
			},
		},
		{
			description: "objc_class_prefix",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "foo_empty/with_package.proto", "", "", bufconfig.FileOptionObjcClassPrefix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/bar", bufconfig.FileOptionObjcClassPrefix, "BAR"),
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/foo", bufconfig.FileOptionObjcClassPrefix, "FOO"),
					newTestFileOptionOverrideRule(t, "foo_all", "buf.build/acme/foo", bufconfig.FileOptionObjcClassPrefix, "FOOALL"),
				},
			),
			modifyFunc: modifyObjcClassPrefix,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"bar_empty/with_package.proto": {
					ObjcClassPrefix: proto.String("BAR"),
				},
				"bar_empty/without_package.proto": {
					ObjcClassPrefix: proto.String("BAR"),
				},
				// disabled
				"foo_empty/with_package.proto": nil,
				// no package
				"foo_empty/without_package.proto": {
					ObjcClassPrefix: proto.String("FOO"),
				},
				"foo_all/with_package.proto": {
					ObjcClassPrefix:      proto.String("FOOALL"),
					CcEnableArenas:       proto.Bool(true),
					CcGenericServices:    proto.Bool(true),
					CsharpNamespace:      proto.String("foo"),
					GoPackage:            proto.String("foo"),
					JavaGenericServices:  proto.Bool(true),
					JavaMultipleFiles:    proto.Bool(true),
					JavaOuterClassname:   proto.String("foo"),
					JavaPackage:          proto.String("foo"),
					JavaStringCheckUtf8:  proto.Bool(true),
					OptimizeFor:          descriptorpb.FileOptions_CODE_SIZE.Enum(),
					PhpClassPrefix:       proto.String("foo"),
					PhpMetadataNamespace: proto.String("foo"),
					PhpNamespace:         proto.String("foo"),
					PyGenericServices:    proto.Bool(true),
					RubyPackage:          proto.String("foo"),
					SwiftPrefix:          proto.String("foo"),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"bar_empty/with_package.proto":    {objcClassPrefixPath},
				"bar_empty/without_package.proto": {objcClassPrefixPath},
				"foo_empty/without_package.proto": {objcClassPrefixPath},
				"foo_all/without_package.proto":   {objcClassPrefixPath},
				"foo_all/with_package.proto":      {objcClassPrefixPath},
			},
		},
		{
			description: "swift_prefix",
			dirPathToFullName: map[string]string{
				filepath.Join("testdata", "foo"): "buf.build/acme/foo",
				filepath.Join("testdata", "bar"): "buf.build/acme/bar",
			},
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{
					newTestManagedDisableRule(t, "foo_empty/with_package.proto", "", "", bufconfig.FileOptionSwiftPrefix, bufconfig.FieldOptionUnspecified),
				},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/bar", bufconfig.FileOptionSwiftPrefix, "BAR"),
					newTestFileOptionOverrideRule(t, "", "buf.build/acme/foo", bufconfig.FileOptionSwiftPrefix, "FOO"),
					newTestFileOptionOverrideRule(t, "foo_all", "buf.build/acme/foo", bufconfig.FileOptionSwiftPrefix, "FOOALL"),
				},
			),
			modifyFunc: modifySwiftPrefix,
			filePathToExpectedOptions: map[string]*descriptorpb.FileOptions{
				"bar_empty/with_package.proto": {
					SwiftPrefix: proto.String("BAR"),
				},
				"bar_empty/without_package.proto": {
					SwiftPrefix: proto.String("BAR"),
				},
				// disabled
				"foo_empty/with_package.proto": nil,
				// no package
				"foo_empty/without_package.proto": {
					SwiftPrefix: proto.String("FOO"),
				},
				"foo_all/with_package.proto": {
					ObjcClassPrefix:      proto.String("foo"),
					CcEnableArenas:       proto.Bool(true),
					CcGenericServices:    proto.Bool(true),
					CsharpNamespace:      proto.String("foo"),
					GoPackage:            proto.String("foo"),
					JavaGenericServices:  proto.Bool(true),
					JavaMultipleFiles:    proto.Bool(true),
					JavaOuterClassname:   proto.String("foo"),
					JavaPackage:          proto.String("foo"),
					JavaStringCheckUtf8:  proto.Bool(true),
					OptimizeFor:          descriptorpb.FileOptions_CODE_SIZE.Enum(),
					PhpClassPrefix:       proto.String("foo"),
					PhpMetadataNamespace: proto.String("foo"),
					PhpNamespace:         proto.String("foo"),
					PyGenericServices:    proto.Bool(true),
					RubyPackage:          proto.String("foo"),
					SwiftPrefix:          proto.String("FOOALL"),
				},
			},
			filePathToExpectedMarkedLocationPaths: map[string][][]int32{
				"bar_empty/with_package.proto":    {swiftPrefixPath},
				"bar_empty/without_package.proto": {swiftPrefixPath},
				"foo_empty/without_package.proto": {swiftPrefixPath},
				"foo_all/without_package.proto":   {swiftPrefixPath},
				"foo_all/with_package.proto":      {swiftPrefixPath},
			},
		},
	}
	for _, testcase := range testcases {
		for _, includeSourceInfo := range []bool{true, false} {
			// TODO FUTURE: we are only testing sweep here, no need to test both include and exclude source info
			t.Run(testcase.description, func(t *testing.T) {
				t.Parallel()
				image := testGetImageFromDirs(t, testcase.dirPathToFullName, includeSourceInfo)
				sweeper := internal.NewMarkSweeper(image)
				// TODO FUTURE: check include source code info
				for filePath, expectedOptions := range testcase.filePathToExpectedOptions {
					imageFile := image.GetFile(filePath)
					require.NoError(
						t,
						testcase.modifyFunc(
							sweeper,
							imageFile,
							testcase.config,
						),
					)
					require.NotNil(t, imageFile)
					require.Empty(
						t,
						cmp.Diff(expectedOptions, imageFile.FileDescriptorProto().GetOptions(), protocmp.Transform()),
						"incorrect options result for %s",
						filePath,
					)
					// TODO FUTURE: sweep and check paths gone
				}
			})
		}
	}
}

// TODO FUTURE: add default values
func TestGetStringOverrideFromConfig(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description            string
		config                 bufconfig.GenerateManagedConfig
		imageFile              bufimage.ImageFile
		defaultOverrideOptions stringOverrideOptions
		expectedOverride       stringOverrideOptions
		expectedDisable        bool
	}{
		{
			description: "only_value",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{value: "value"},
		},
		{
			description: "only_prefix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{prefix: "prefix"},
		},
		{
			description: "only_suffix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{suffix: "suffix"},
		},
		{
			description: "prefix_then_value",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{value: "value"},
		},
		{
			description: "value_then_prefix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{prefix: "prefix"},
		},
		{
			description: "prefix_then_suffix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
				},
			),
			imageFile: testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{
				prefix: "prefix",
				suffix: "suffix",
			},
		},
		{
			description: "value_prefix_then_suffix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
				},
			),
			imageFile: testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{
				prefix: "prefix",
				suffix: "suffix",
			},
		},
		{
			description: "prefix_value_then_suffix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{suffix: "suffix"},
		},
		{
			description: "prefix_then_prefix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackagePrefix, "prefix2"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{prefix: "prefix2"},
		},
		{
			description: "suffix_then_suffix",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackageSuffix, "suffix2"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{suffix: "suffix2"},
		},
		{
			description: "value_then_value",
			config: bufconfig.NewGenerateManagedConfig(
				true,
				[]bufconfig.ManagedDisableRule{},
				[]bufconfig.ManagedOverrideRule{
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value"),
					newTestFileOptionOverrideRule(t, "", "", bufconfig.FileOptionJavaPackage, "value2"),
				},
			),
			imageFile:        testGetImageFile(t, "a.proto", "buf.build/foo/bar"),
			expectedOverride: stringOverrideOptions{value: "value2"},
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			override, err := stringOverrideFromConfig(
				testcase.imageFile,
				testcase.config,
				testcase.defaultOverrideOptions,
				bufconfig.FileOptionJavaPackage,
				bufconfig.FileOptionJavaPackagePrefix,
				bufconfig.FileOptionJavaPackageSuffix,
			)
			require.NoError(t, err)
			require.Equal(t, testcase.expectedOverride, override)
		})
	}
}

// TODO FUTURE in v2
//func TestModifyFieldOption(t *testing.T) {
//t.Parallel()
//}

func testGetImageFile(
	t *testing.T,
	path string,
	moduleFullName string,
) bufimage.ImageFile {
	parsedFullName, err := bufparse.ParseFullName(moduleFullName)
	require.NoError(t, err)
	return bufimagetesting.NewImageFile(
		t,
		&descriptorpb.FileDescriptorProto{
			Name:   proto.String(path),
			Syntax: proto.String("proto3"),
		},
		parsedFullName,
		uuid.Nil,
		path,
		"",
		false,
		false,
		nil,
	)
}

func testGetImageFromDirs(
	t *testing.T,
	dirPathToFullName map[string]string,
	includeSourceInfo bool,
) bufimage.Image {
	moduleDatas := make([]bufmoduletesting.ModuleData, 0, len(dirPathToFullName))
	for dirPath, moduleFullName := range dirPathToFullName {
		moduleDatas = append(
			moduleDatas,
			bufmoduletesting.ModuleData{
				Name:    moduleFullName,
				DirPath: dirPath,
			},
		)
	}
	moduleSet, err := bufmoduletesting.NewModuleSet(moduleDatas...)
	require.NoError(t, err)
	var options []bufimage.BuildImageOption
	if !includeSourceInfo {
		options = []bufimage.BuildImageOption{bufimage.WithExcludeSourceCodeInfo()}
	}
	image, err := bufimage.BuildImage(
		context.Background(),
		slogtestext.NewLogger(t),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	require.NoError(t, err)
	return image
}

func newTestManagedDisableRule(
	t *testing.T,
	path string,
	moduleFullName string,
	fieldName string,
	fileOption bufconfig.FileOption,
	fieldOption bufconfig.FieldOption,
) bufconfig.ManagedDisableRule {
	disable, err := bufconfig.NewManagedDisableRule(
		path,
		moduleFullName,
		fieldName,
		fileOption,
		fieldOption,
	)
	require.NoError(t, err)
	return disable
}

func newTestFileOptionOverrideRule(
	t *testing.T,
	path string,
	moduleFullName string,
	fileOption bufconfig.FileOption,
	value any,
) bufconfig.ManagedOverrideRule {
	fileOptionOverride, err := bufconfig.NewManagedOverrideRuleForFileOption(
		path,
		moduleFullName,
		fileOption,
		value,
	)
	require.NoError(t, err)
	return fileOptionOverride
}
