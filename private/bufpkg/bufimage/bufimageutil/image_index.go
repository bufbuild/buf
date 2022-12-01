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

package bufimageutil

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protosource"
)

// imageIndex holds an index of fully qualified type names to various
// protosource descriptors.
type imageIndex struct {
	// NameToDescriptor maps fully qualified type names to a NamedDescriptor
	// and can be used to lookup Descriptors referenced by other
	// descriptor's fields like `Extendee`, `Parent`, `InputType`, etc.
	NameToDescriptor map[string]protosource.NamedDescriptor

	// NameToExtensions maps fully qualified type names to all known
	// extension definitions for a type name.
	NameToExtensions map[string][]protosource.Field

	// NameToOptions maps `google.protobuf.*Options` type names to their
	// known extensions by field tag.
	NameToOptions map[string]map[int32]protosource.Field
}

// newImageIndexForImage builds an imageIndex for a given image.
func newImageIndexForImage(image bufimage.Image, opts *imageFilterOptions) (*imageIndex, error) {
	index := &imageIndex{
		NameToDescriptor: make(map[string]protosource.NamedDescriptor),
	}
	if opts.includeCustomOptions {
		index.NameToOptions = make(map[string]map[int32]protosource.Field)
	}
	if opts.includeKnownExtensions {
		index.NameToExtensions = make(map[string][]protosource.Field)
	}

	for _, file := range image.Files() {
		// TODO: protosource.File is a heavyweight representation, and NewFile is not a cheap
		//  operation. We only really need to gather fully-qualified names for each element
		//  and to have some minimal level of upward references (e.g. access a descriptor's
		//  parent element), which could be done with a far more efficient representation.
		protosourceFile, err := protosource.NewFile(newInputFile(file))
		if err != nil {
			return nil, err
		}
		if err := protosource.ForEachMessage(func(message protosource.Message) error {
			if storedDescriptor, ok := index.NameToDescriptor[message.FullName()]; ok && storedDescriptor != message {
				return fmt.Errorf("duplicate for %q: %#v != %#v", message.FullName(), storedDescriptor, message)
			}
			index.NameToDescriptor[message.FullName()] = message
			if !opts.includeKnownExtensions && !opts.includeCustomOptions {
				return nil
			}
			for _, field := range message.Extensions() {
				index.NameToDescriptor[field.FullName()] = field
				extendeeName := field.Extendee()
				if opts.includeCustomOptions && isOptionsTypeName(extendeeName) {
					if _, ok := index.NameToOptions[extendeeName]; !ok {
						index.NameToOptions[extendeeName] = make(map[int32]protosource.Field)
					}
					index.NameToOptions[extendeeName][int32(field.Number())] = field
				}
				if opts.includeKnownExtensions {
					index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], field)
				}
			}
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		if err = protosource.ForEachEnum(func(enum protosource.Enum) error {
			if storedDescriptor, ok := index.NameToDescriptor[enum.FullName()]; ok {
				return fmt.Errorf("duplicate for %q: %#v != %#v", enum.FullName(), storedDescriptor, enum)
			}
			index.NameToDescriptor[enum.FullName()] = enum
			return nil
		}, protosourceFile); err != nil {
			return nil, err
		}
		for _, service := range protosourceFile.Services() {
			if storedDescriptor, ok := index.NameToDescriptor[service.FullName()]; ok {
				return nil, fmt.Errorf("duplicate for %q: %#v != %#v", service.FullName(), storedDescriptor, service)
			}
			index.NameToDescriptor[service.FullName()] = service
			for _, method := range service.Methods() {
				if storedDescriptor, ok := index.NameToDescriptor[method.FullName()]; ok {
					return nil, fmt.Errorf("duplicate for %q: %#v != %#v", method.FullName(), storedDescriptor, method)
				}
				index.NameToDescriptor[method.FullName()] = method
			}
		}
		if !opts.includeKnownExtensions && !opts.includeCustomOptions {
			continue
		}
		for _, field := range protosourceFile.Extensions() {
			index.NameToDescriptor[field.FullName()] = field
			extendeeName := field.Extendee()
			if opts.includeCustomOptions && isOptionsTypeName(extendeeName) {
				if _, ok := index.NameToOptions[extendeeName]; !ok {
					index.NameToOptions[extendeeName] = make(map[int32]protosource.Field)
				}
				index.NameToOptions[extendeeName][int32(field.Number())] = field
			}
			if opts.includeKnownExtensions {
				index.NameToExtensions[extendeeName] = append(index.NameToExtensions[extendeeName], field)
			}
		}
	}
	return index, nil
}

func isOptionsTypeName(typeName string) bool {
	switch typeName {
	case "google.protobuf.FileOptions",
		"google.protobuf.MessageOptions",
		"google.protobuf.FieldOptions",
		"google.protobuf.OneofOptions",
		"google.protobuf.EnumOptions",
		"google.protobuf.EnumValueOptions",
		"google.protobuf.ServiceOptions",
		"google.protobuf.MethodOptions":
		return true
	default:
		return false
	}
}
