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

package bufimageutil

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/protoplugin/protopluginutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	anyFullName = "google.protobuf.Any"

	messageRangeInclusiveMax = 536870911
)

var (
	// ErrImageFilterTypeNotFound is returned from FilterImage when
	// a specified type cannot be found in an image.
	ErrImageFilterTypeNotFound = errors.New("not found")

	// ErrImageFilterTypeIsImport is returned from FilterImage when
	// a specified type name is declared in a module dependency.
	ErrImageFilterTypeIsImport = errors.New("type declared in imported module")
)

// FreeMessageRangeStrings gets the free MessageRange strings for the target files.
//
// Recursive.
func FreeMessageRangeStrings(
	ctx context.Context,
	filePaths []string,
	image bufimage.Image,
) ([]string, error) {
	var s []string
	for _, filePath := range filePaths {
		imageFile := image.GetFile(filePath)
		if imageFile == nil {
			return nil, fmt.Errorf("unexpected nil image file: %q", filePath)
		}
		prefix := imageFile.FileDescriptorProto().GetPackage()
		if prefix != "" {
			prefix += "."
		}
		for _, message := range imageFile.FileDescriptorProto().GetMessageType() {
			s = freeMessageRangeStringsRec(s, prefix+message.GetName(), message)
		}
	}
	return s, nil
}

// ImageFilterOption is an option that can be passed to FilterImage.
type ImageFilterOption func(*imageFilterOptions)

// WithExcludeCustomOptions returns an option that will cause an image filtered via
// FilterImage to *not* include custom options unless they are
// explicitly named in the list of filter types.
func WithExcludeCustomOptions() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.includeCustomOptions = false
	}
}

// WithExcludeKnownExtensions returns an option that will cause an image filtered via
// FilterImage to *not* include the known extensions for included
// extendable messages unless they are explicitly named in the list of filter types.
func WithExcludeKnownExtensions() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.includeKnownExtensions = false
	}
}

// WithAllowIncludeOfImportedType returns an option for FilterImage
// that allows a named included type to be in an imported file or module. Without this
// option, only types defined directly in the image to be filtered are allowed.
// Excluded types are always allowed to be in imported files or modules.
func WithAllowIncludeOfImportedType() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.allowImportedTypes = true
	}
}

// WithIncludeTypes returns an option for FilterImage that specifies
// the set of types that should be included in the filtered image.
//
// May be provided multiple times. The type names should be fully qualified.
// For example, "google.protobuf.Any" or "buf.validate". Types may be nested,
// and can be any package, message, enum, extension, service or method name.
//
// If the  type does not exist in the image, an error wrapping
// [ErrImageFilterTypeNotFound] will be returned.
func WithIncludeTypes(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if len(typeNames) > 0 && opts.includeTypes == nil {
			opts.includeTypes = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.includeTypes[typeName] = struct{}{}
		}
	}
}

// WithExcludeTypes returns an option for FilterImage that
// specifies the set of types that should be excluded from the filtered image.
//
// May be provided multiple times. The type names should be fully qualified.
// For example, "google.protobuf.Any" or "buf.validate". Types may be nested,
// and can be any package, message, enum, extension, service or method name.
//
// If the  type does not exist in the image, an error wrapping
// [ErrImageFilterTypeNotFound] will be returned.
func WithExcludeTypes(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if len(typeNames) > 0 && opts.excludeTypes == nil {
			opts.excludeTypes = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.excludeTypes[typeName] = struct{}{}
		}
	}
}

// WithMutateInPlace returns an option for FilterImage that specifies
// that the filtered image should be mutated in place. This option is useful when the
// unfiltered image is no longer needed and the caller wants to avoid the overhead of
// copying the image.
func WithMutateInPlace() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.mutateInPlace = true
	}
}

// FilterImage returns a minimal image containing only the descriptors
// required to define the set of types provided by the filter options. If no
// filter options are provided, the original image is returned.
//
// The filtered image will contain only the files that contain the definitions of
// the specified types, and their transitive dependencies. If a file is no longer
// required, it will be removed from the image. Only the minimal set of types
// required to define the specified types will be included in the filtered image.
//
// Excluded types and options are not included in the filtered image. If an
// included type transitively depends on the excluded type, the descriptor will
// be altered to remove the dependency.
//
// This returns a new [bufimage.Image] that is a shallow copy of the underlying
// [descriptorpb.FileDescriptorProto]s of the original. The new image may
// therefore share state with the original image, so it should not be modified.
// If the original image is no longer needed, it should be discarded. To mutate
// the original image, use the [WithMutateInPlace] option. Otherwise, to avoid
// sharing of state clone the image before filtering.
//
// A descriptor is said to require another descriptor if the dependent
// descriptor is needed to accurately and completely describe that descriptor.
// For the following types that includes:
//
//	Messages
//	 - messages & enums referenced in fields
//	 - proto2 extension declarations for this field
//	 - custom options for the message, its fields, and the file in which the
//	   message is defined
//	 - the parent message if this message is a nested definition
//
//	Enums
//	 - Custom options used in the enum, enum values, and the file
//	   in which the message is defined
//	 - the parent message if this message is a nested definition
//
//	Services
//	 - request & response types referenced in methods
//	 - custom options for the service, its methods, and the file
//	   in which the message is defined
//
// As an example, consider the following proto structure:
//
//	--- foo.proto ---
//	package pkg;
//	message Foo {
//	  optional Bar bar = 1;
//	  extensions 2 to 3;
//	}
//	message Bar { ... }
//	message Baz {
//	  other.Qux qux = 1 [(other.my_option).field = "buf"];
//	}
//	--- baz.proto ---
//	package other;
//	extend Foo {
//	  optional Qux baz = 2;
//	}
//	message Qux{ ... }
//	message Quux{ ... }
//	extend google.protobuf.FieldOptions {
//	  optional Quux my_option = 51234;
//	}
//
// A filtered image for type `pkg.Foo` would include
//
//	files:      [foo.proto, bar.proto]
//	messages:   [pkg.Foo, pkg.Bar, other.Qux]
//	extensions: [other.baz]
//
// A filtered image for type `pkg.Bar` would include
//
//	 files:      [foo.proto]
//	 messages:   [pkg.Bar]
//
//	A filtered image for type `pkg.Baz` would include
//	 files:      [foo.proto, bar.proto]
//	 messages:   [pkg.Baz, other.Quux, other.Qux]
//	 extensions: [other.my_option]
func FilterImage(image bufimage.Image, options ...ImageFilterOption) (bufimage.Image, error) {
	if len(options) == 0 {
		return image, nil
	}
	filterOptions := newImageFilterOptions()
	for _, option := range options {
		option(filterOptions)
	}
	// Check for defaults that would result in no filtering.
	if len(filterOptions.excludeTypes) == 0 &&
		len(filterOptions.includeTypes) == 0 {
		return image, nil
	}
	return filterImage(image, filterOptions)
}

// StripSourceRetentionOptions strips any options with a retention of "source" from
// the descriptors in the given image. The image is not mutated but instead a new
// image is returned. The returned image may share state with the original.
func StripSourceRetentionOptions(image bufimage.Image) (bufimage.Image, error) {
	updatedFiles := make([]bufimage.ImageFile, len(image.Files()))
	for i, inputFile := range image.Files() {
		updatedFile, err := stripSourceRetentionOptionsFromFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to strip source-retention options from file %q: %w", inputFile.Path(), err)
		}
		updatedFiles[i] = updatedFile
	}
	return bufimage.NewImage(updatedFiles)
}

// transitiveClosure accumulates the elements for a subset of an image.
// When an element is added to the closure, all of its dependencies are
// recursively added.
type transitiveClosure struct {
	// The elements included in the transitive closure.
	elements map[namedDescriptor]closureInclusionMode
}

type closureInclusionMode int

const (
	// Element is explicitly excluded from the closure.
	inclusionModeExcluded = closureInclusionMode(iota - 1)
	// Element is not yet known to be included or excluded.
	inclusionModeUnknown
	// Element is included in closure because it is directly reachable from a root.
	inclusionModeExplicit
	// Element is included in closure because it is a message or service that
	// *contains* an explicitly included element but is not itself directly
	// reachable.
	inclusionModeEnclosing
	// Element is included in closure because it is implied by the presence of a
	// custom option. For example, a field element with a custom option implies
	// the presence of google.protobuf.FieldOptions. An option type could instead be
	// explicitly included if it is also directly reachable (i.e. some type in the
	// graph explicitly refers to the option type).
	inclusionModeImplicit
)

func newTransitiveClosure() *transitiveClosure {
	return &transitiveClosure{
		elements: map[namedDescriptor]closureInclusionMode{},
	}
}

func (t *transitiveClosure) hasType(
	descriptor namedDescriptor,
	options *imageFilterOptions,
) (isIncluded bool) {
	switch mode := t.elements[descriptor]; mode {
	case inclusionModeExplicit, inclusionModeImplicit, inclusionModeEnclosing:
		return true
	case inclusionModeExcluded:
		return false
	case inclusionModeUnknown:
		// True if no includes are specified.
		return options.includeTypes == nil
	default:
		return false
	}
}

func (t *transitiveClosure) hasOption(
	fieldDescriptor protoreflect.FieldDescriptor,
	imageIndex *imageIndex,
	options *imageFilterOptions,
) (isIncluded bool) {
	if !fieldDescriptor.IsExtension() {
		return true
	}
	if !options.includeCustomOptions {
		return false
	}
	fullName := fieldDescriptor.FullName()
	descriptor := imageIndex.ByName[fullName].element
	switch mode := t.elements[descriptor]; mode {
	case inclusionModeExplicit, inclusionModeImplicit, inclusionModeEnclosing:
		return true
	case inclusionModeExcluded:
		return false
	case inclusionModeUnknown:
		// True as option type is not explicitly excluded.
		// Occurs on first traversal when adding included types.
		return true
	default:
		return false
	}
}

func (t *transitiveClosure) includeType(
	typeName protoreflect.FullName,
	imageIndex *imageIndex,
	options *imageFilterOptions,
) error {
	descriptorInfo, ok := imageIndex.ByName[typeName]
	if ok {
		// It's a type name
		if !options.allowImportedTypes && descriptorInfo.file.IsImport() {
			return fmt.Errorf("inclusion of type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
		// Check if the type is already excluded.
		if mode := t.elements[descriptorInfo.element]; mode == inclusionModeExcluded {
			return fmt.Errorf("inclusion of excluded type %q", typeName)
		}
		// If an extension field, check if the extendee is excluded.
		if field, ok := descriptorInfo.element.(*descriptorpb.FieldDescriptorProto); ok && field.Extendee != nil {
			extendeeName := protoreflect.FullName(strings.TrimPrefix(field.GetExtendee(), "."))
			extendeeInfo, ok := imageIndex.ByName[extendeeName]
			if !ok {
				return fmt.Errorf("missing %q", extendeeName)
			}
			if mode := t.elements[extendeeInfo.element]; mode == inclusionModeExcluded {
				return fmt.Errorf("cannot include extension field %q as the extendee type %q is excluded", typeName, extendeeName)
			}
		}
		if err := t.addElement(descriptorInfo.element, false, imageIndex, options); err != nil {
			return fmt.Errorf("inclusion of type %q: %w", typeName, err)
		}
		return nil
	}
	// It could be a package name
	pkg, ok := imageIndex.Packages[string(typeName)]
	if !ok {
		// but it's not...
		return fmt.Errorf("inclusion of type %q: %w", typeName, ErrImageFilterTypeNotFound)
	}
	if !options.allowImportedTypes {
		// if package includes only imported files, then reject
		onlyImported := true
		for _, file := range pkg.files {
			if !file.IsImport() {
				onlyImported = false
				break
			}
		}
		if onlyImported {
			return fmt.Errorf("inclusion of type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
	}
	for _, file := range pkg.files {
		fileDescriptor := file.FileDescriptorProto()
		if mode := t.elements[fileDescriptor]; mode == inclusionModeExcluded {
			return fmt.Errorf("inclusion of excluded package %q", typeName)
		}
		if err := t.addElement(fileDescriptor, false, imageIndex, options); err != nil {
			return fmt.Errorf("inclusion of type %q: %w", typeName, err)
		}
	}
	return nil
}

func (t *transitiveClosure) addElement(
	descriptor namedDescriptor,
	impliedByCustomOption bool,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	if existingMode, ok := t.elements[descriptor]; ok && existingMode != inclusionModeEnclosing {
		if existingMode == inclusionModeExcluded {
			return nil // already excluded
		}
		if existingMode == inclusionModeImplicit && !impliedByCustomOption {
			// upgrade from implied to explicitly part of closure
			t.elements[descriptor] = inclusionModeExplicit
		}
		return nil // already added this element
	}
	descriptorInfo := imageIndex.ByDescriptor[descriptor]
	if impliedByCustomOption {
		t.elements[descriptor] = inclusionModeImplicit
	} else {
		t.elements[descriptor] = inclusionModeExplicit
	}

	switch typedDescriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		// If the file we are attempting to include is empty (has no types defined), it is still
		// valid for inclusion.
		typeNames := imageIndex.FileTypes[typedDescriptor.GetName()]
		// A file includes all elements. The types are resolved in the image index
		// to ensure all nested types are included.
		for _, typeName := range typeNames {
			typeInfo := imageIndex.ByName[typeName]
			if err := t.addElement(typeInfo.element, false, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.DescriptorProto:
		oneofFieldCounts := make([]int, len(typedDescriptor.GetOneofDecl()))
		// Options and types for all fields
		for _, field := range typedDescriptor.GetField() {
			isIncluded, err := t.addFieldType(field, imageIndex, opts)
			if err != nil {
				return err
			}
			if !isIncluded {
				continue
			}
			if index := field.OneofIndex; index != nil {
				index := *index
				if index < 0 || int(index) >= len(oneofFieldCounts) {
					return fmt.Errorf("invalid oneof index %d for field %q", index, field.GetName())
				}
				oneofFieldCounts[index]++
			}
			if err := t.exploreCustomOptions(field, imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all oneofs in this message
		for index, oneOfDescriptor := range typedDescriptor.GetOneofDecl() {
			if oneofFieldCounts[index] == 0 {
				// An empty oneof is not a valid protobuf construct, so we can
				// safely exclude it.
				t.elements[oneOfDescriptor] = inclusionModeExcluded
				continue
			}
			if err := t.exploreCustomOptions(oneOfDescriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all extension ranges in this message
		for _, extRange := range typedDescriptor.GetExtensionRange() {
			if err := t.exploreCustomOptions(extRange, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.EnumDescriptorProto:
		for _, enumValue := range typedDescriptor.GetValue() {
			if err := t.exploreCustomOptions(enumValue, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.ServiceDescriptorProto:
		for _, method := range typedDescriptor.GetMethod() {
			inputName := protoreflect.FullName(strings.TrimPrefix(method.GetInputType(), "."))
			inputInfo, ok := imageIndex.ByName[inputName]
			if !ok {
				return fmt.Errorf("missing %q", inputName)
			}
			outputName := protoreflect.FullName(strings.TrimPrefix(method.GetOutputType(), "."))
			outputInfo, ok := imageIndex.ByName[outputName]
			if !ok {
				return fmt.Errorf("missing %q", outputName)
			}
			inputMode, outputMode := t.elements[inputInfo.element], t.elements[outputInfo.element]
			if inputMode == inclusionModeExcluded || outputMode == inclusionModeExcluded {
				// The input or output is excluded, so this method is also excluded.
				t.elements[inputInfo.element] = inclusionModeExcluded
				continue
			}
			if err := t.addElement(method, false, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.MethodDescriptorProto:
		inputName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetInputType(), "."))
		inputInfo, ok := imageIndex.ByName[inputName]
		if !ok {
			return fmt.Errorf("missing %q", inputName)
		}
		outputName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetOutputType(), "."))
		outputInfo, ok := imageIndex.ByName[outputName]
		if !ok {
			return fmt.Errorf("missing %q", outputName)
		}
		if inputMode := t.elements[inputInfo.element]; inputMode == inclusionModeExcluded {
			// The input is excluded, it's an error to include the method.
			return fmt.Errorf("cannot include method %q as the input type %q is excluded", descriptorInfo.fullName, inputInfo.fullName)
		}
		if outputMode := t.elements[outputInfo.element]; outputMode == inclusionModeExcluded {
			// The output is excluded, it's an error to include the method.
			return fmt.Errorf("cannot include method %q as the output type %q is excluded", descriptorInfo.fullName, outputInfo.fullName)
		}
		if err := t.addElement(inputInfo.element, false, imageIndex, opts); err != nil {
			return err
		}
		if err := t.addElement(outputInfo.element, false, imageIndex, opts); err != nil {
			return err
		}

	case *descriptorpb.FieldDescriptorProto:
		// Regular fields are handled above in message descriptor case.
		// We should only find our way here for extensions.
		if typedDescriptor.Extendee == nil {
			return errorUnsupportedFilterType(descriptor, descriptorInfo.fullName)
		}
		if typedDescriptor.GetExtendee() == "" {
			return fmt.Errorf("expected extendee for field %q to not be empty", descriptorInfo.fullName)
		}
		extendeeName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetExtendee(), "."))
		extendeeInfo, ok := imageIndex.ByName[extendeeName]
		if !ok {
			return fmt.Errorf("missing %q", extendeeName)
		}
		if mode := t.elements[extendeeInfo.element]; mode == inclusionModeExcluded {
			// The extendee is excluded, so this extension is also excluded.
			t.elements[descriptor] = inclusionModeExcluded
			return nil
		}
		if err := t.addElement(extendeeInfo.element, impliedByCustomOption, imageIndex, opts); err != nil {
			return err
		}
		isIncluded, err := t.addFieldType(typedDescriptor, imageIndex, opts)
		if err != nil {
			return err
		}
		if !isIncluded {
			t.elements[descriptor] = inclusionModeExcluded
			return nil
		}

	default:
		return errorUnsupportedFilterType(descriptor, descriptorInfo.fullName)
	}

	// if this type is enclosed inside another, add enclosing types
	if err := t.addEnclosing(descriptorInfo.parent, imageIndex, opts); err != nil {
		return err
	}
	// add any custom options and their dependencies
	if err := t.exploreCustomOptions(descriptor, imageIndex, opts); err != nil {
		return err
	}

	return nil
}

func (t *transitiveClosure) excludeType(
	typeName protoreflect.FullName,
	imageIndex *imageIndex,
	options *imageFilterOptions,
) error {
	descriptorInfo, ok := imageIndex.ByName[typeName]
	if ok {
		return t.excludeElement(descriptorInfo.element, imageIndex, options)
	}
	// It could be a package name
	pkg, ok := imageIndex.Packages[string(typeName)]
	if !ok {
		// but it's not...
		return fmt.Errorf("exclusion of type %q: %w", typeName, ErrImageFilterTypeNotFound)
	}
	// Exclude the package and all of its files.
	for _, file := range pkg.files {
		fileDescriptor := file.FileDescriptorProto()
		if err := t.excludeElement(fileDescriptor, imageIndex, options); err != nil {
			return err
		}
	}
	return nil
}

func (t *transitiveClosure) excludeElement(
	descriptor namedDescriptor,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	descriptorInfo := imageIndex.ByDescriptor[descriptor]
	if existingMode, ok := t.elements[descriptor]; ok {
		if existingMode != inclusionModeExcluded {
			return fmt.Errorf("type %q is already included", descriptorInfo.fullName)
		}
		return nil
	}
	t.elements[descriptor] = inclusionModeExcluded
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		for _, descriptor := range descriptor.GetMessageType() {
			if err := t.excludeElement(descriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		for _, descriptor := range descriptor.GetEnumType() {
			if err := t.excludeElement(descriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		for _, descriptor := range descriptor.GetService() {
			if err := t.excludeElement(descriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		for _, extension := range descriptor.GetExtension() {
			if err := t.excludeElement(extension, imageIndex, opts); err != nil {
				return err
			}
		}
	case *descriptorpb.DescriptorProto:
		// Exclude all sub-elements
		for _, descriptor := range descriptor.GetNestedType() {
			if err := t.excludeElement(descriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		for _, enumDescriptor := range descriptor.GetEnumType() {
			if err := t.excludeElement(enumDescriptor, imageIndex, opts); err != nil {
				return err
			}
		}
		for _, extensionDescriptor := range descriptor.GetExtension() {
			if err := t.excludeElement(extensionDescriptor, imageIndex, opts); err != nil {
				return err
			}
		}
	case *descriptorpb.EnumDescriptorProto:
		// Enum values are not included in the closure, so nothing to do here.
	case *descriptorpb.ServiceDescriptorProto:
		for _, descriptor := range descriptor.GetMethod() {
			if err := t.excludeElement(descriptor, imageIndex, opts); err != nil {
				return err
			}
		}
	case *descriptorpb.MethodDescriptorProto:
	case *descriptorpb.FieldDescriptorProto:
		// Only extension fields can be excluded.
		if descriptor.Extendee == nil {
			return errorUnsupportedFilterType(descriptor, descriptorInfo.fullName)
		}
		if descriptor.GetExtendee() == "" {
			return fmt.Errorf("expected extendee for field %q to not be empty", descriptorInfo.fullName)
		}
	default:
		return errorUnsupportedFilterType(descriptor, descriptorInfo.fullName)
	}
	return nil
}

func errorUnsupportedFilterType(descriptor namedDescriptor, fullName protoreflect.FullName) error {
	var descriptorType string
	switch d := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		descriptorType = "file"
	case *descriptorpb.DescriptorProto:
		descriptorType = "message"
	case *descriptorpb.FieldDescriptorProto:
		if d.Extendee != nil {
			descriptorType = "extension field"
		} else {
			descriptorType = "non-extension field"
		}
	case *descriptorpb.OneofDescriptorProto:
		descriptorType = "oneof"
	case *descriptorpb.EnumDescriptorProto:
		descriptorType = "enum"
	case *descriptorpb.EnumValueDescriptorProto:
		descriptorType = "enum value"
	case *descriptorpb.ServiceDescriptorProto:
		descriptorType = "service"
	case *descriptorpb.MethodDescriptorProto:
		descriptorType = "method"
	default:
		descriptorType = fmt.Sprintf("%T", d)
	}
	return fmt.Errorf("%s is unsupported filter type: %s", fullName, descriptorType)
}

func (t *transitiveClosure) addEnclosing(descriptor namedDescriptor, imageIndex *imageIndex, opts *imageFilterOptions) error {
	// loop through all enclosing parents since nesting level
	// could be arbitrarily deep
	for descriptor != nil {
		_, isMsg := descriptor.(*descriptorpb.DescriptorProto)
		_, isSvc := descriptor.(*descriptorpb.ServiceDescriptorProto)
		_, isFile := descriptor.(*descriptorpb.FileDescriptorProto)
		if !isMsg && !isSvc && !isFile {
			break // not an enclosing type
		}
		if _, ok := t.elements[descriptor]; ok {
			break // already in closure
		}
		t.elements[descriptor] = inclusionModeEnclosing
		if err := t.exploreCustomOptions(descriptor, imageIndex, opts); err != nil {
			return err
		}
		// now move into this element's parent
		descriptor = imageIndex.ByDescriptor[descriptor].parent
	}
	return nil
}

func (t *transitiveClosure) addFieldType(field *descriptorpb.FieldDescriptorProto, imageIndex *imageIndex, opts *imageFilterOptions) (bool, error) {
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
		info, ok := imageIndex.ByName[typeName]
		if !ok {
			return false, fmt.Errorf("missing %q", typeName)
		}
		if mode := t.elements[info.element]; mode == inclusionModeExcluded {
			// The field's type is excluded, so this field is also excluded.
			return false, nil
		}
		err := t.addElement(info.element, false, imageIndex, opts)
		if err != nil {
			return false, err
		}
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64:
	// nothing to follow, custom options handled below.
	default:
		return false, fmt.Errorf("unknown field type %d", field.GetType())
	}
	return true, nil
}

// completeFileContents ensures that import files retained only as
// namespace containers (inclusionModeEnclosing) have their types
// walked when those types reference files not yet in the closure.
//
// In exclude-only mode, types not in the closure are implicitly
// included. If an enclosing import file has declared dependencies
// on files that have no elements in the closure, those files would
// be silently dropped from the output. This method fixes that by
// fully traversing such files.
//
// This loops until stable because walking a file's types may pull
// in additional enclosing files.
func (t *transitiveClosure) completeFileContents(
	image bufimage.Image,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	for {
		added := false
		// Collect files that currently have elements in the closure.
		filesInClosure := make(map[string]bool)
		for descriptor, mode := range t.elements {
			switch mode {
			case inclusionModeExcluded, inclusionModeUnknown:
				continue
			}
			info := imageIndex.ByDescriptor[descriptor]
			filesInClosure[info.file.Path()] = true
		}

		for _, file := range image.Files() {
			if !file.IsImport() {
				continue
			}
			fileDescriptorProto := file.FileDescriptorProto()
			mode, ok := t.elements[fileDescriptorProto]
			if !ok || mode != inclusionModeEnclosing {
				continue
			}
			// Check if this enclosing file has declared dependencies
			// on files not yet in the closure. If so, traverse the
			// file to discover those dependencies.
			needsTraversal := false
			for _, dep := range fileDescriptorProto.GetDependency() {
				if !filesInClosure[dep] {
					needsTraversal = true
					break
				}
			}
			if !needsTraversal {
				continue
			}
			if err := t.addElement(fileDescriptorProto, false, imageIndex, opts); err != nil {
				return err
			}
			added = true
		}
		if !added {
			break
		}
	}
	return nil
}

func (t *transitiveClosure) addExtensions(
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	if !opts.includeKnownExtensions {
		return nil // nothing to do
	}
	for e, mode := range t.elements {
		if mode != inclusionModeExplicit {
			// we only collect extensions for messages that are directly reachable/referenced.
			continue
		}
		msgDescriptor, ok := e.(*descriptorpb.DescriptorProto)
		if !ok {
			// not a message, nothing to do
			continue
		}
		descriptorInfo := imageIndex.ByDescriptor[msgDescriptor]
		for _, extendsDescriptor := range imageIndex.NameToExtensions[descriptorInfo.fullName] {
			if mode := t.elements[extendsDescriptor]; mode == inclusionModeExcluded {
				// This extension field is excluded.
				continue
			}
			if err := t.addElement(extendsDescriptor, false, imageIndex, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// computeImports derives the file-to-file import map from the closure's
// element set. For each included element, it inspects type references
// (field types, method input/output, extension extendees, custom options)
// to determine which files must import which other files.
//
// Returns a map of file path to the set of file paths it needs to import.
// Every file that contains at least one included element will have an entry
// in the map (possibly with an empty/nil import set).
func (t *transitiveClosure) computeImports(
	imageIndex *imageIndex,
	options *imageFilterOptions,
) map[string]map[string]struct{} {
	imports := make(map[string]map[string]struct{})

	addImport := func(fromPath, toPath string) {
		if fromPath == toPath {
			return
		}
		imps := imports[fromPath]
		if imps == nil {
			imps = make(map[string]struct{}, 2)
			imports[fromPath] = imps
		}
		imps[toPath] = struct{}{}
	}

	addTypeImport := func(fromPath string, typeName protoreflect.FullName) {
		info, ok := imageIndex.ByName[typeName]
		if !ok {
			return
		}
		if !t.hasType(info.element, options) {
			return
		}
		addImport(fromPath, info.file.Path())
	}

	for descriptor, mode := range t.elements {
		switch mode {
		case inclusionModeExcluded, inclusionModeUnknown:
			continue
		}
		info := imageIndex.ByDescriptor[descriptor]
		filePath := info.file.Path()
		// Ensure the file has an entry in the map.
		if _, ok := imports[filePath]; !ok {
			imports[filePath] = nil
		}

		switch typedDescriptor := descriptor.(type) {
		case *descriptorpb.FileDescriptorProto:
			// File options may reference option extension types.
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)

		case *descriptorpb.DescriptorProto:
			if mode == inclusionModeEnclosing {
				// Enclosing messages only provide namespace, no field references.
				continue
			}
			// Field type references.
			for _, field := range typedDescriptor.GetField() {
				switch field.GetType() {
				case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
					descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
					descriptorpb.FieldDescriptorProto_TYPE_GROUP:
					typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
					addTypeImport(filePath, typeName)
				}
				t.addOptionImports(field.GetOptions(), filePath, imageIndex, options, addImport)
			}
			// Oneof options.
			for _, oneof := range typedDescriptor.GetOneofDecl() {
				t.addOptionImports(oneof.GetOptions(), filePath, imageIndex, options, addImport)
			}
			// Extension range options.
			for _, extRange := range typedDescriptor.GetExtensionRange() {
				t.addOptionImports(extRange.GetOptions(), filePath, imageIndex, options, addImport)
			}
			// Message options.
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)

		case *descriptorpb.EnumDescriptorProto:
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)
			for _, enumValue := range typedDescriptor.GetValue() {
				t.addOptionImports(enumValue.GetOptions(), filePath, imageIndex, options, addImport)
			}

		case *descriptorpb.ServiceDescriptorProto:
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)
			// Method references are handled by MethodDescriptorProto case.

		case *descriptorpb.MethodDescriptorProto:
			addTypeImport(filePath, protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetInputType(), ".")))
			addTypeImport(filePath, protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetOutputType(), ".")))
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)

		case *descriptorpb.FieldDescriptorProto:
			// Only extension fields appear as top-level elements.
			if typedDescriptor.Extendee != nil {
				extendeeName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetExtendee(), "."))
				addTypeImport(filePath, extendeeName)
				switch typedDescriptor.GetType() {
				case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
					descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
					descriptorpb.FieldDescriptorProto_TYPE_GROUP:
					typeName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetTypeName(), "."))
					addTypeImport(filePath, typeName)
				}
			}
			t.addOptionImports(typedDescriptor.GetOptions(), filePath, imageIndex, options, addImport)
		}
	}
	return imports
}

// addOptionImports adds file imports for custom option extensions and
// Any-referenced types used in the given options message.
func (t *transitiveClosure) addOptionImports(
	optionsMsg proto.Message,
	filePath string,
	imageIndex *imageIndex,
	options *imageFilterOptions,
	addImport func(string, string),
) {
	if optionsMsg == nil || !options.includeCustomOptions {
		return
	}
	optionsRef := optionsMsg.ProtoReflect()
	if !optionsRef.IsValid() {
		return
	}
	optionsName := optionsRef.Descriptor().FullName()
	optionsRef.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if !fd.IsExtension() {
			return true
		}
		if !t.hasOption(fd, imageIndex, options) {
			return true
		}
		optionsByNumber := imageIndex.NameToOptions[optionsName]
		field, ok := optionsByNumber[int32(fd.Number())]
		if !ok {
			return true
		}
		fieldInfo := imageIndex.ByDescriptor[field]
		addImport(filePath, fieldInfo.file.Path())
		// Walk the option value for Any-referenced types.
		t.addAnyImports(fd, val, filePath, imageIndex, addImport)
		return true
	})
}

// addAnyImports walks an option value looking for google.protobuf.Any
// messages and adds file imports for the referenced types.
func (t *transitiveClosure) addAnyImports(
	fd protoreflect.FieldDescriptor,
	val protoreflect.Value,
	filePath string,
	imageIndex *imageIndex,
	addImport func(string, string),
) {
	switch {
	case fd.IsMap():
		if isMessageKind(fd.MapValue().Kind()) {
			val.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				t.addAnySingularImport(v.Message(), filePath, imageIndex, addImport)
				return true
			})
		}
	case isMessageKind(fd.Kind()):
		if fd.IsList() {
			listVal := val.List()
			for i := range listVal.Len() {
				t.addAnySingularImport(listVal.Get(i).Message(), filePath, imageIndex, addImport)
			}
		} else {
			t.addAnySingularImport(val.Message(), filePath, imageIndex, addImport)
		}
	}
}

func (t *transitiveClosure) addAnySingularImport(
	msg protoreflect.Message,
	filePath string,
	imageIndex *imageIndex,
	addImport func(string, string),
) {
	md := msg.Descriptor()
	if md.FullName() == anyFullName {
		typeURLFd := md.Fields().ByNumber(1)
		if typeURLFd.Kind() != protoreflect.StringKind || typeURLFd.IsList() {
			return
		}
		typeURL := msg.Get(typeURLFd).String()
		pos := strings.LastIndexByte(typeURL, '/')
		msgType := protoreflect.FullName(typeURL[pos+1:])
		if info, ok := imageIndex.ByName[msgType]; ok {
			addImport(filePath, info.file.Path())
		}
		return
	}
	// Recurse into nested messages.
	msg.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		t.addAnyImports(fd, val, filePath, imageIndex, addImport)
		return true
	})
}

func (t *transitiveClosure) exploreCustomOptions(
	descriptor proto.Message,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	if !opts.includeCustomOptions {
		return nil
	}

	var options protoreflect.Message
	switch descriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.DescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.FieldDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.OneofDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.EnumDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.EnumValueDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.ServiceDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.MethodDescriptorProto:
		options = descriptor.GetOptions().ProtoReflect()
	case *descriptorpb.DescriptorProto_ExtensionRange:
		options = descriptor.GetOptions().ProtoReflect()
	default:
		return fmt.Errorf("unexpected type for exploring options %T", descriptor)
	}

	optionsName := options.Descriptor().FullName()
	var err error
	options.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if !t.hasOption(fd, imageIndex, opts) {
			return true
		}
		// If the value contains an Any message, we should add the message type
		// therein to the closure.
		if err = t.exploreOptionValueForAny(fd, val, imageIndex, opts); err != nil {
			return false
		}

		// Also include custom option definitions (e.g. extensions)
		if !fd.IsExtension() {
			return true
		}
		optionsByNumber := imageIndex.NameToOptions[optionsName]
		field, ok := optionsByNumber[int32(fd.Number())]
		if !ok {
			// Option is unrecognized, ignore it.
			return true
		}
		err = t.addElement(field, true, imageIndex, opts)
		return err == nil
	})
	return err
}

func isMessageKind(k protoreflect.Kind) bool {
	return k == protoreflect.MessageKind || k == protoreflect.GroupKind
}

func (t *transitiveClosure) exploreOptionValueForAny(
	fd protoreflect.FieldDescriptor,
	val protoreflect.Value,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	switch {
	case fd.IsMap():
		if isMessageKind(fd.MapValue().Kind()) {
			var err error
			val.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				if err = t.exploreOptionSingularValueForAny(v.Message(), imageIndex, opts); err != nil {
					return false
				}
				return true
			})
			return err
		}
	case isMessageKind(fd.Kind()):
		if fd.IsList() {
			listVal := val.List()
			for i := range listVal.Len() {
				if err := t.exploreOptionSingularValueForAny(listVal.Get(i).Message(), imageIndex, opts); err != nil {
					return err
				}
			}
		} else {
			return t.exploreOptionSingularValueForAny(val.Message(), imageIndex, opts)
		}
	}
	return nil
}

func (t *transitiveClosure) exploreOptionSingularValueForAny(
	msg protoreflect.Message,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	md := msg.Descriptor()
	if md.FullName() == anyFullName {
		// Found one!
		typeURLFd := md.Fields().ByNumber(1)
		if typeURLFd.Kind() != protoreflect.StringKind || typeURLFd.IsList() {
			// should not be possible...
			return nil
		}
		typeURL := msg.Get(typeURLFd).String()
		pos := strings.LastIndexByte(typeURL, '/')
		msgType := protoreflect.FullName(typeURL[pos+1:])
		d, _ := imageIndex.ByName[msgType].element.(*descriptorpb.DescriptorProto)
		if d != nil {
			if err := t.addElement(d, false, imageIndex, opts); err != nil {
				return err
			}
		}
		// TODO: unmarshal the bytes to see if there are any nested Any messages
		return nil
	}
	// keep digging
	var err error
	msg.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		err = t.exploreOptionValueForAny(fd, val, imageIndex, opts)
		return err == nil
	})
	return err
}

type int32Range struct {
	start, end int32 // both inclusive
}

func freeMessageRangeStringsRec(
	s []string,
	fullName string,
	message *descriptorpb.DescriptorProto,
) []string {
	for _, nestedMessage := range message.GetNestedType() {
		s = freeMessageRangeStringsRec(s, fullName+"."+nestedMessage.GetName(), nestedMessage)
	}
	freeRanges := freeMessageRanges(message)
	if len(freeRanges) == 0 {
		return s
	}
	suffixes := make([]string, len(freeRanges))
	for i, freeRange := range freeRanges {
		start := freeRange.start
		end := freeRange.end
		var suffix string
		switch {
		case start == end:
			suffix = fmt.Sprintf("%d", start)
		case freeRange.end == messageRangeInclusiveMax:
			suffix = fmt.Sprintf("%d-INF", start)
		default:
			suffix = fmt.Sprintf("%d-%d", start, end)
		}
		suffixes[i] = suffix
	}
	return append(s, fmt.Sprintf(
		"%- 35s free: %s",
		fullName,
		strings.Join(suffixes, " "),
	))
}

// freeMessageRanges returns the free message ranges for the given message.
//
// Not recursive.
func freeMessageRanges(message *descriptorpb.DescriptorProto) []int32Range {
	used := make([]int32Range, 0, len(message.GetReservedRange())+len(message.GetExtensionRange())+len(message.GetField()))
	for _, reservedRange := range message.GetReservedRange() {
		// we subtract one because ranges in the proto have exclusive end
		used = append(used, int32Range{start: reservedRange.GetStart(), end: reservedRange.GetEnd() - 1})
	}
	for _, extensionRange := range message.GetExtensionRange() {
		// we subtract one because ranges in the proto have exclusive end
		used = append(used, int32Range{start: extensionRange.GetStart(), end: extensionRange.GetEnd() - 1})
	}
	for _, field := range message.GetField() {
		used = append(used, int32Range{start: field.GetNumber(), end: field.GetNumber()})
	}
	sort.Slice(used, func(i, j int) bool {
		return used[i].start < used[j].start
	})
	// now compute the inverse (unused ranges)
	var unused []int32Range
	var last int32
	for _, r := range used {
		if r.start <= last+1 {
			last = r.end
			continue
		}
		unused = append(unused, int32Range{start: last + 1, end: r.start - 1})
		last = r.end
	}
	if last < messageRangeInclusiveMax {
		unused = append(unused, int32Range{start: last + 1, end: messageRangeInclusiveMax})
	}
	return unused
}

type imageFilterOptions struct {
	includeCustomOptions   bool
	includeKnownExtensions bool
	allowImportedTypes     bool
	mutateInPlace          bool
	includeTypes           map[string]struct{}
	excludeTypes           map[string]struct{}
}

func newImageFilterOptions() *imageFilterOptions {
	return &imageFilterOptions{
		includeCustomOptions:   true,
		includeKnownExtensions: true,
		allowImportedTypes:     false,
	}
}

func stripSourceRetentionOptionsFromFile(imageFile bufimage.ImageFile) (bufimage.ImageFile, error) {
	fileDescriptor := imageFile.FileDescriptorProto()
	updatedFileDescriptor, err := protopluginutil.StripSourceRetentionOptions(fileDescriptor)
	if err != nil {
		return nil, err
	}
	if updatedFileDescriptor == fileDescriptor {
		return imageFile, nil
	}
	return bufimage.NewImageFile(
		updatedFileDescriptor,
		imageFile.FullName(),
		imageFile.CommitID(),
		imageFile.ExternalPath(),
		imageFile.LocalPath(),
		imageFile.IsImport(),
		imageFile.IsSyntaxUnspecified(),
		imageFile.UnusedDependencyIndexes(),
	)
}
