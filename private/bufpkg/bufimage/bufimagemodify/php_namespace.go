// Copyright 2020-2022 Buf Technologies, Inc.
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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// PhpNamespaceID is the ID of the php_namespace modifier.
const PhpNamespaceID = "PHP_NAMESPACE"

var (
	// phpNamespacePath is the SourceCodeInfo path for the php_namespace option.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/61689226c0e3ec88287eaed66164614d9c4f2bf7/src/google/protobuf/descriptor.proto#L443
	phpNamespacePath = []int32{8, 41}

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

func phpNamespace(
	logger *zap.Logger,
	sweeper Sweeper,
	except []bufmoduleref.ModuleIdentity,
	moduleOverrides map[bufmoduleref.ModuleIdentity]string,
	overrides map[string]string,
) Modifier {
	// Convert the bufmoduleref.ModuleIdentity types into
	// strings so that they're comparable.
	exceptModuleIdentityStrings := make(map[string]struct{}, len(except))
	for _, moduleIdentity := range except {
		exceptModuleIdentityStrings[moduleIdentity.IdentityString()] = struct{}{}
	}
	overrideModuleIdentityStrings := make(map[string]string, len(moduleOverrides))
	for moduleIdentity, csharpNamespace := range moduleOverrides {
		overrideModuleIdentityStrings[moduleIdentity.IdentityString()] = csharpNamespace
	}
	return ModifierFunc(
		func(ctx context.Context, image bufimage.Image) error {
			seenModuleIdentityStrings := make(map[string]struct{}, len(overrideModuleIdentityStrings))
			seenOverrideFiles := make(map[string]struct{}, len(overrides))
			for _, imageFile := range image.Files() {
				phpNamespaceValue := phpNamespaceValue(imageFile)
				if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
					moduleIdentityString := moduleIdentity.IdentityString()
					if moduleNamespaceOverride, ok := overrideModuleIdentityStrings[moduleIdentityString]; ok {
						seenModuleIdentityStrings[moduleIdentityString] = struct{}{}
						phpNamespaceValue = moduleNamespaceOverride
					}
				}
				if overrideValue, ok := overrides[imageFile.Path()]; ok {
					phpNamespaceValue = overrideValue
					seenOverrideFiles[imageFile.Path()] = struct{}{}
				}
				if err := phpNamespaceForFile(
					ctx,
					sweeper,
					imageFile,
					phpNamespaceValue,
					exceptModuleIdentityStrings,
				); err != nil {
					return err
				}
			}
			for moduleIdentityString := range overrideModuleIdentityStrings {
				if _, ok := seenModuleIdentityStrings[moduleIdentityString]; !ok {
					logger.Sugar().Warnf("php_namespace override for %q was unused", moduleIdentityString)
				}
			}
			for overrideFile := range overrides {
				if _, ok := seenOverrideFiles[overrideFile]; !ok {
					logger.Sugar().Warnf("%s override for %q was unused", PhpNamespaceID, overrideFile)
				}
			}
			return nil
		},
	)
}

func phpNamespaceForFile(
	ctx context.Context,
	sweeper Sweeper,
	imageFile bufimage.ImageFile,
	phpNamespaceValue string,
	exceptModuleIdentityStrings map[string]struct{},
) error {
	descriptor := imageFile.Proto()
	if isWellKnownType(ctx, imageFile) || phpNamespaceValue == "" {
		// This is a well-known type or we could not resolve a non-empty php_namespace
		// value, so this is a no-op.
		return nil
	}
	if moduleIdentity := imageFile.ModuleIdentity(); moduleIdentity != nil {
		if _, ok := exceptModuleIdentityStrings[moduleIdentity.IdentityString()]; ok {
			return nil
		}
	}
	if descriptor.Options == nil {
		descriptor.Options = &descriptorpb.FileOptions{}
	}
	descriptor.Options.PhpNamespace = proto.String(phpNamespaceValue)
	if sweeper != nil {
		sweeper.mark(imageFile.Path(), phpNamespacePath)
	}
	return nil
}

// phpNamespaceValue returns the php_namespace for the given ImageFile based on its
// package declaration. If the image file doesn't have a package declaration, an
// empty string is returned.
func phpNamespaceValue(imageFile bufimage.ImageFile) string {
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
