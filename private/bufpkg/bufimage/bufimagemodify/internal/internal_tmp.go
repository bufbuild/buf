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
	"path"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

var // Keywords and classes that could be produced by our heuristic.
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
