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

package internal

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// DefaultJavaMultipleFilesValue is the default value for the java_multiple_files modifier.
	DefaultJavaMultipleFilesValue = true
	// DefaultJavaPackagePrefix is the default java_package prefix used in the java_package modifier.
	DefaultJavaPackagePrefix = "com"
)

var (
	// CCEnableArenas is the SourceCodeInfo path for the cc_enable_arenas option.
	// https://github.com/protocolbuffers/protobuf/blob/29152fbc064921ca982d64a3a9eae1daa8f979bb/src/google/protobuf/descriptor.proto#L420
	CCEnableArenasPath = []int32{8, 31}
	// CsharpNamespacePath is the SourceCodeInfo path for the csharp_namespace option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L428
	CsharpNamespacePath = []int32{8, 37}
	// GoPackagePath is the SourceCodeInfo path for the go_package option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L392
	GoPackagePath = []int32{8, 11}
	// JavaMultipleFilesPath is the SourceCodeInfo path for the java_multiple_files option.
	// https://github.com/protocolbuffers/protobuf/blob/ee04809540c098718121e092107fbc0abc231725/src/google/protobuf/descriptor.proto#L364
	JavaMultipleFilesPath = []int32{8, 10}
	// JavaOuterClassnamePath is the SourceCodeInfo path for the java_outer_classname option.
	// https://github.com/protocolbuffers/protobuf/blob/87d140f851131fb8a6e8a80449cf08e73e568259/src/google/protobuf/descriptor.proto#L356
	JavaOuterClassnamePath = []int32{8, 8}
	// JavaPackagePath is the SourceCodeInfo path for the java_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L348
	JavaPackagePath = []int32{8, 1}
	// JavaStringCheckUtf8Path is the SourceCodeInfo path for the java_string_check_utf8 option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L375
	JavaStringCheckUtf8Path = []int32{8, 27}
	// ObjcClassPrefixPath is the SourceCodeInfo path for the objc_class_prefix option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L425
	ObjcClassPrefixPath = []int32{8, 36}
	// optimizeFor is the SourceCodeInfo path for the optimize_for option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L385
	OptimizeForPath = []int32{8, 9}
	// PhpMetadataNamespacePath is the SourceCodeInfo path for the php_metadata_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L448
	PhpMetadataNamespacePath = []int32{8, 44}
	// PhpNamespacePath is the SourceCodeInfo path for the php_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L443
	PhpNamespacePath = []int32{8, 41}
	// RubyPackagePath is the SourceCodeInfo path for the ruby_package option.
	// https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L453
	RubyPackagePath = []int32{8, 45}
	// fileOptionPath is the path prefix used for FileOptions.
	// All file option locations are preceded by a location
	// with a path set to the fileOptionPath.
	// https://github.com/protocolbuffers/protobuf/blob/053966b4959bdd21e4a24e657bcb97cb9de9e8a4/src/google/protobuf/descriptor.proto#L80
	fileOptionPath = []int32{8}
	// Keywords and classes that could be produced by our heuristic.
	// They must not be used in a php_namespace.
	// Ref: https://www.php.net/manual/en/reserved.php
	phpReservedKeywords = map[string]struct{}{
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
)

// RemoveLocationsFromSourceCodeInfo removes from source code info the locations
// that match the paths provided.
func RemoveLocationsFromSourceCodeInfo(
	sourceCodeInfo *descriptorpb.SourceCodeInfo,
	paths map[string]struct{},
) error {
	// We can't just match on an exact path match because the target
	// file option's parent path elements would remain (i.e [8]).
	// Instead, we perform an initial pass to validate that the paths
	// are structured as expect, and collect all of the indices that
	// we need to delete.
	indices := make(map[int]struct{}, len(paths)*2)
	for i, location := range sourceCodeInfo.Location {
		if _, ok := paths[GetPathKey(location.Path)]; !ok {
			continue
		}
		if i == 0 {
			return fmt.Errorf("path %v must have a preceding parent path", location.Path)
		}
		if !Int32SliceIsEqual(sourceCodeInfo.Location[i-1].Path, fileOptionPath) {
			return fmt.Errorf("path %v must have a preceding parent path equal to %v", location.Path, fileOptionPath)
		}
		// Add the target path and its parent.
		indices[i-1] = struct{}{}
		indices[i] = struct{}{}
	}
	// Now that we know exactly which indices to exclude, we can
	// filter the SourceCodeInfo_Locations as needed.
	locations := make(
		[]*descriptorpb.SourceCodeInfo_Location,
		0,
		len(sourceCodeInfo.Location)-len(indices),
	)
	for i, location := range sourceCodeInfo.Location {
		if _, ok := indices[i]; ok {
			continue
		}
		locations = append(locations, location)
	}
	sourceCodeInfo.Location = locations
	return nil
}

// Int32SliceIsEqual returns true if x and y contain the same elements.
func Int32SliceIsEqual(x []int32, y []int32) bool {
	if len(x) != len(y) {
		return false
	}
	for i, elem := range x {
		if elem != y[i] {
			return false
		}
	}
	return true
}

// GetPathKey returns a unique key for the given path.
func GetPathKey(path []int32) string {
	key := make([]byte, len(path)*4)
	j := 0
	for _, elem := range path {
		key[j] = byte(elem)
		key[j+1] = byte(elem >> 8)
		key[j+2] = byte(elem >> 16)
		key[j+3] = byte(elem >> 24)
		j += 4
	}
	return string(key)
}

// IsWellKnownType returns true if the given path is one of the well-known types.
func IsWellKnownType(imageFile bufimage.ImageFile) bool {
	return datawkt.Exists(imageFile.Path())
}

// DefaultCsharpNamespace returns the csharp_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func DefaultCsharpNamespace(imageFile bufimage.ImageFile) string {
	pkg := imageFile.Proto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, ".")
}

// GoPackageImportPathForFile returns the go_package import path for the given
// ImageFile. If the package contains a version suffix, and if there are more
// than two components, concatenate the final two components. Otherwise, we
// exclude the ';' separator and adopt the default behavior from the import path.
//
// For example, an ImageFile with `package acme.weather.v1;` will include `;weatherv1`
// in the `go_package` declaration so that the generated package is named as such.
func GoPackageImportPathForFile(imageFile bufimage.ImageFile, importPathPrefix string) string {
	goPackageImportPath := path.Join(importPathPrefix, path.Dir(imageFile.Path()))
	packageName := imageFile.FileDescriptor().GetPackage()
	if _, ok := protoversion.NewPackageVersionForPackage(packageName); ok {
		parts := strings.Split(packageName, ".")
		if len(parts) >= 2 {
			goPackageImportPath += ";" + parts[len(parts)-2] + parts[len(parts)-1]
		}
	}
	return goPackageImportPath
}

// DefaultJavaOuterClassname returns the default outer class name for an image file.
func DefaultJavaOuterClassname(imageFile bufimage.ImageFile) string {
	return stringutil.ToPascalCase(normalpath.Base(imageFile.Path()))
}

// JavaPackageValue returns the java_package for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func JavaPackageValue(imageFile bufimage.ImageFile, packagePrefix string) string {
	if pkg := imageFile.Proto().GetPackage(); pkg != "" {
		if packagePrefix == "" {
			return pkg
		}
		return packagePrefix + "." + pkg
	}
	return ""
}

// DefaultObjcClassPrefixValue returns the objc_class_prefix for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func DefaultObjcClassPrefixValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.Proto().GetPackage()
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

// DefaultPhpMetadataNamespaceValue returns the php_metadata_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func DefaultPhpMetadataNamespaceValue(imageFile bufimage.ImageFile) string {
	phpNamespace := DefaultPhpNamespaceValue(imageFile)
	if phpNamespace == "" {
		return ""
	}
	return phpNamespace + `\GPBMetadata`
}

// DefaultPhpNamespaceValue returns the php_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func DefaultPhpNamespaceValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.Proto().GetPackage()
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

// DefaultRubyPackageValue returns the ruby_package for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func DefaultRubyPackageValue(imageFile bufimage.ImageFile) string {
	pkg := imageFile.Proto().GetPackage()
	if pkg == "" {
		return ""
	}
	packageParts := strings.Split(pkg, ".")
	for i, part := range packageParts {
		packageParts[i] = stringutil.ToPascalCase(part)
	}
	return strings.Join(packageParts, "::")
}
