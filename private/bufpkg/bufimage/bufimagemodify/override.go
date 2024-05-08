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
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Keywords and classes that could be produced by our heuristic.
// They must not be used in a php_namespace.
// Ref: https://www.php.net/manual/en/reserved.php
var phpReservedKeywords = map[string]struct{}{
	// Reserved classes as per above.
	"directory":           {},
	"exception":           {},
	"errorexception":      {},
	"closure":             {},
	"generator":           {},
	"arithmeticerror":     {},
	"assertionerror":      {},
	"divisionbyzeroerror": {},
	"error":               {},
	"throwable":           {},
	"parseerror":          {},
	"typeerror":           {},
	// Keywords avoided by protoc.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/66d749188ff2a2e30e932110222d58da7c6a8d49/src/google/protobuf/compiler/php/php_generator.cc#L50-L66
	"abstract":     {},
	"and":          {},
	"array":        {},
	"as":           {},
	"break":        {},
	"callable":     {},
	"case":         {},
	"catch":        {},
	"class":        {},
	"clone":        {},
	"const":        {},
	"continue":     {},
	"declare":      {},
	"default":      {},
	"die":          {},
	"do":           {},
	"echo":         {},
	"else":         {},
	"elseif":       {},
	"empty":        {},
	"enddeclare":   {},
	"endfor":       {},
	"endforeach":   {},
	"endif":        {},
	"endswitch":    {},
	"endwhile":     {},
	"eval":         {},
	"exit":         {},
	"extends":      {},
	"final":        {},
	"finally":      {},
	"fn":           {},
	"for":          {},
	"foreach":      {},
	"function":     {},
	"global":       {},
	"goto":         {},
	"if":           {},
	"implements":   {},
	"include":      {},
	"include_once": {},
	"instanceof":   {},
	"insteadof":    {},
	"interface":    {},
	"isset":        {},
	"list":         {},
	"match":        {},
	"namespace":    {},
	"new":          {},
	"or":           {},
	"print":        {},
	"private":      {},
	"protected":    {},
	"public":       {},
	"require":      {},
	"require_once": {},
	"return":       {},
	"static":       {},
	"switch":       {},
	"throw":        {},
	"trait":        {},
	"try":          {},
	"unset":        {},
	"use":          {},
	"var":          {},
	"while":        {},
	"xor":          {},
	"yield":        {},
	"int":          {},
	"float":        {},
	"bool":         {},
	"string":       {},
	"true":         {},
	"false":        {},
	"null":         {},
	"void":         {},
	"iterable":     {},
}

type stringOverrideOptions struct {
	value  string
	prefix string
	suffix string
}

func stringOverrideFromConfig(
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	defaultOverrideOptions stringOverrideOptions,
	valueFileOption bufconfig.FileOption,
	prefixFileOption bufconfig.FileOption,
	suffixFileOption bufconfig.FileOption,
) (stringOverrideOptions, error) {
	if isFileOptionDisabledForFile(
		imageFile,
		valueFileOption,
		config,
	) {
		return stringOverrideOptions{}, nil
	}
	overrideOptions := defaultOverrideOptions
	ignorePrefix := prefixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, prefixFileOption, config)
	ignoreSuffix := suffixFileOption == bufconfig.FileOptionUnspecified || isFileOptionDisabledForFile(imageFile, suffixFileOption, config)
	if ignorePrefix {
		overrideOptions.prefix = ""
	}
	if ignoreSuffix {
		overrideOptions.suffix = ""
	}
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		switch overrideRule.FileOption() {
		case valueFileOption:
			valueString, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", valueFileOption, overrideRule.Value())
			}
			// If the latest override matched is a value override (java_package as opposed to java_package_prefix), use the value.
			overrideOptions = stringOverrideOptions{value: valueString}
		case prefixFileOption:
			if ignorePrefix {
				continue
			}
			prefix, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", prefixFileOption, overrideRule.Value())
			}
			// Keep the suffix if the last two overrides are suffix and prefix.
			overrideOptions = stringOverrideOptions{
				prefix: prefix,
				suffix: overrideOptions.suffix,
			}
		case suffixFileOption:
			if ignoreSuffix {
				continue
			}
			suffix, ok := overrideRule.Value().(string)
			if !ok {
				// This should never happen, since the override rule has been validated.
				return stringOverrideOptions{}, fmt.Errorf("invalid value type for %v override: %T", suffixFileOption, overrideRule.Value())
			}
			// Keep the prefix if the last two overrides are suffix and prefix.
			overrideOptions = stringOverrideOptions{
				prefix: overrideOptions.prefix,
				suffix: suffix,
			}
		}
	}
	return overrideOptions, nil
}

// returns the override value and whether managed mode is DISABLED for this file for this file option.
func overrideFromConfig[T bool | descriptorpb.FileOptions_OptimizeMode](
	imageFile bufimage.ImageFile,
	config bufconfig.GenerateManagedConfig,
	fileOption bufconfig.FileOption,
) (*T, error) {
	var override *T
	for _, overrideRule := range config.Overrides() {
		if !fileMatchConfig(imageFile, overrideRule.Path(), overrideRule.ModuleFullName()) {
			continue
		}
		if overrideRule.FileOption() != fileOption {
			continue
		}
		value, ok := overrideRule.Value().(T)
		if !ok {
			// This should never happen, since the override rule has been validated.
			return nil, fmt.Errorf("invalid value type for %v override: %T", fileOption, overrideRule.Value())
		}
		override = &value
	}
	return override, nil
}

func isFileOptionDisabledForFile(
	imageFile bufimage.ImageFile,
	fileOption bufconfig.FileOption,
	config bufconfig.GenerateManagedConfig,
) bool {
	for _, disableRule := range config.Disables() {
		if disableRule.FileOption() != bufconfig.FileOptionUnspecified && disableRule.FileOption() != fileOption {
			continue
		}
		if disableRule.FieldOption() != bufconfig.FieldOptionUnspecified {
			continue // FieldOption specified, not a matching rule.
		}
		if !fileMatchConfig(imageFile, disableRule.Path(), disableRule.ModuleFullName()) {
			continue
		}
		return true
	}
	return false
}

func fileMatchConfig(
	imageFile bufimage.ImageFile,
	requiredPath string,
	requiredModuleFullName string,
) bool {
	if requiredPath != "" && !normalpath.EqualsOrContainsPath(requiredPath, imageFile.Path(), normalpath.Relative) {
		return false
	}
	if requiredModuleFullName != "" && (imageFile.ModuleFullName() == nil || imageFile.ModuleFullName().String() != requiredModuleFullName) {
		return false
	}
	return true
}

// TODO FUTURE: unify naming of these helpers
func getJavaPackageValue(imageFile bufimage.ImageFile, stringOverrideOptions stringOverrideOptions) string {
	if pkg := imageFile.FileDescriptorProto().GetPackage(); pkg != "" {
		if stringOverrideOptions.prefix != "" {
			pkg = stringOverrideOptions.prefix + "." + pkg
		}
		if stringOverrideOptions.suffix != "" {
			pkg = pkg + "." + stringOverrideOptions.suffix
		}
		return pkg
	}
	return ""
}

func getCsharpNamespaceValue(imageFile bufimage.ImageFile, prefix string) string {
	namespace := csharpNamespaceValue(imageFile)
	if namespace == "" {
		return ""
	}
	if prefix == "" {
		return namespace
	}
	return prefix + "." + namespace
}

func getPhpMetadataNamespaceValue(imageFile bufimage.ImageFile, suffix string) string {
	namespace := phpNamespaceValue(imageFile)
	if namespace == "" {
		return ""
	}
	if suffix == "" {
		return namespace
	}
	return namespace + `\` + suffix
}

func getRubyPackageValue(imageFile bufimage.ImageFile, suffix string) string {
	rubyPackage := rubyPackageValue(imageFile)
	if rubyPackage == "" {
		return ""
	}
	if suffix == "" {
		return rubyPackage
	}
	return rubyPackage + "::" + suffix
}

// TODO FUTURE: is this needed?
// csharpNamespaceValue returns the csharp_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func csharpNamespaceValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, ".")
}

// goPackageImportPathForFile returns the go_package import path for the given
// ImageFile. If the package contains a version suffix, and if there are more
// than two components, concatenate the final two components. Otherwise, we
// exclude the ';' separator and adopt the default behavior from the import path.
//
// For example, an ImageFile with `package acme.weather.v1;` will include `;weatherv1`
// in the `go_package` declaration so that the generated package is named as such.
func goPackageImportPathForFile(imageFile bufimage.ImageFile, importPathPrefix string) string {
	goPackageImportPath := path.Join(importPathPrefix, path.Dir(imageFile.Path()))
	packageName := imageFile.FileDescriptorProto().GetPackage()
	if _, ok := protoversion.NewPackageVersionForPackage(packageName); ok {
		parts := strings.Split(packageName, ".")
		if len(parts) >= 2 {
			goPackageImportPath += ";" + parts[len(parts)-2] + parts[len(parts)-1]
		}
	}
	return goPackageImportPath
}

func javaOuterClassnameValue(imageFile bufimage.ImageFile) string {
	return stringutil.ToPascalCase(normalpath.Base(imageFile.Path()))
}

// objcClassPrefixValue returns the objc_class_prefix for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func objcClassPrefixValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	_, hasPackageVersion := protoversion.NewPackageVersionForPackage(pkg)
	packageParts := strings.Split(pkg, ".")
	var prefixParts []rune
	for i, part := range packageParts {
		// Check if last part is a version before appending.
		if i == len(packageParts)-1 && hasPackageVersion {
			continue
		}
		// Probably should never be a non-ASCII character,
		// but why not support it just in case?
		runeSlice := []rune(part)
		prefixParts = append(prefixParts, unicode.ToUpper(runeSlice[0]))
	}
	for len(prefixParts) < 3 {
		prefixParts = append(prefixParts, 'X')
	}
	prefix := string(prefixParts)
	if prefix == "GPB" {
		prefix = "GPX"
	}
	return prefix
}

// phpMetadataNamespaceValue returns the php_metadata_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func phpMetadataNamespaceValue(imageFile bufimage.ImageFile) string {
	phpNamespace := phpNamespaceValue(imageFile)
	if phpNamespace == "" {
		return ""
	}
	return phpNamespace + `\GPBMetadata`
}

// phpNamespaceValue returns the php_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func phpNamespaceValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packagePart := stringutil.ToPascalCase(part)
		if _, ok := phpReservedKeywords[strings.ToLower(part)]; ok {
			// Append _ to the package part if it is a reserved keyword.
			packagePart += "_"
		}
		packageParts[i] = packagePart
	}
	return strings.Join(packageParts, `\`)
}

// rubyPackageValue returns the ruby_package for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func rubyPackageValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.FileDescriptorProto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, "::")
}
