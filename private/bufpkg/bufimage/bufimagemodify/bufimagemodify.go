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

package bufimagemodify

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
)

// Modify modifies the image according to the managed config.
//
// The CLI should use this function instead of ModifyXYZ.
func Modify(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImage(
		image,
		config,
		[]func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig, ...ModifyOption) error{
			modifyCcEnableArenas,
			modifyCsharpNamespace,
			modifyGoPackage,
			modifyJavaMultipleFiles,
			modifyJavaOuterClass,
			modifyJavaPackage,
			modifyJavaStringCheckUtf8,
			modifyObjcClassPrefix,
			modifyOptmizeFor,
			modifyPhpMetadataNamespace,
			modifyPhpNamespace,
			modifyRubyPackage,
			modifyJsType,
		},
		options...,
	)
}

// ModifyJavaOuterClass modifies the java_outer_class file option.
func ModifyJavaOuterClass(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyJavaOuterClass,
		options...,
	)
}

// ModifyJavaPackage modifies the java_package file option.
func ModifyJavaPackage(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyJavaPackage,
		options...,
	)
}

// ModifyGoPackage modifies the go_package file option.
func ModifyGoPackage(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyGoPackage,
		options...,
	)
}

// ModifyObjcClassPrefix modifies the objc_class_prefix file option.
func ModifyObjcClassPrefix(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyObjcClassPrefix,
		options...,
	)
}

// ModifyCsharpNamespace modifies the csharp_namespace file option.
func ModifyCsharpNamespace(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyCsharpNamespace,
		options...,
	)
}

// ModifyPhpNamespace modifies the php_namespace file option.
func ModifyPhpNamespace(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyPhpNamespace,
		options...,
	)
}

// ModifyPhpMetadataNamespace modifies the php_metadata_namespace file option.
func ModifyPhpMetadataNamespace(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyPhpMetadataNamespace,
		options...,
	)
}

// ModifyRubyPackage modifies the ruby_package file option.
func ModifyRubyPackage(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyRubyPackage,
		options...,
	)
}

// ModifyCcEnableArenas modifies the cc_enable_arenas file option.
func ModifyCcEnableArenas(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyCcEnableArenas,
		options...,
	)
}

// ModifyJavaMultipleFiles modifies the java_multiple_files file option.
func ModifyJavaMultipleFiles(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyJavaMultipleFiles,
		options...,
	)
}

// ModifyJavaStringCheckUtf8 modifies the java_string_check_utf8 file option.
func ModifyJavaStringCheckUtf8(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyJavaStringCheckUtf8,
		options...,
	)
}

// ModifyOptmizeFor modifies the optimize_for file option.
func ModifyOptmizeFor(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyOptmizeFor,
		options...,
	)
}

// ModifyJsType modifies the js_type field option.
func ModifyJsType(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	options ...ModifyOption,
) error {
	return modifyImageForSingleOption(
		image,
		config,
		modifyJsType,
		options...,
	)
}

// ModifyOption is an option for Modify.
type ModifyOption func(*modifyOptions)

// ModifyPreserveExisting only modifies an option if it is not defined in the file.
//
// Do not use this option in the CLI.
func ModifyPreserveExisting() ModifyOption {
	return func(modifyOptions *modifyOptions) {
		modifyOptions.preserveExisting = true
	}
}

// *** PRIVATE ***

type modifyOptions struct {
	preserveExisting bool
}

func newModifyOptions() *modifyOptions {
	return &modifyOptions{}
}

func modifyImage(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	modifyFuncs []func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig, ...ModifyOption) error,
	options ...ModifyOption,
) error {
	if !config.Enabled() {
		return nil
	}
	sweeper := internal.NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if datawkt.Exists(imageFile.Path()) {
			continue
		}
		for _, modifyFunc := range modifyFuncs {
			if err := modifyFunc(sweeper, imageFile, config, options...); err != nil {
				return err
			}
		}
	}
	return sweeper.Sweep()
}

func modifyImageForSingleOption(
	image bufimage.Image,
	config bufconfig.GenerateManagedConfig,
	modifyFunc func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig, ...ModifyOption) error,
	options ...ModifyOption,
) error {
	return modifyImage(
		image,
		config,
		[]func(internal.MarkSweeper, bufimage.ImageFile, bufconfig.GenerateManagedConfig, ...ModifyOption) error{modifyFunc},
		options...,
	)
}
