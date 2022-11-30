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
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// ErrImageFilterTypeNotFound is returned from ImageFilteredByTypes when
	// a specified type cannot be found in an image.
	ErrImageFilterTypeNotFound = errors.New("not found")

	// ErrImageFilterTypeIsImport is returned from ImageFilteredByTypes when
	// a specified type name is declared in a module dependency.
	ErrImageFilterTypeIsImport = errors.New("type declared in imported module")
)

// NewInputFiles converts the ImageFiles to InputFiles.
//
// Since protosource is a pkg package, it cannot depend on bufmoduleref, which has the
// definition for bufmoduleref.ModuleIdentity, so we have our own interfaces for this
// in protosource. Given Go's type system, we need to do a conversion here.
func NewInputFiles(imageFiles []bufimage.ImageFile) []protosource.InputFile {
	inputFiles := make([]protosource.InputFile, len(imageFiles))
	for i, imageFile := range imageFiles {
		inputFiles[i] = newInputFile(imageFile)
	}
	return inputFiles
}

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
		file, err := protosource.NewFile(newInputFile(imageFile))
		if err != nil {
			return nil, err
		}
		for _, message := range file.Messages() {
			s = freeMessageRangeStringsRec(s, message)
		}
	}
	return s, nil
}

// ImageFilterOption is an option that can be passed to ImageFilteredByTypesWithOptions.
type ImageFilterOption func(*imageFilterOptions)

// WithExcludeCustomOptions returns an option that will cause an image filtered via
// ImageFilteredByTypesWithOptions to *not* include custom options unless they are
// explicitly named in the list of filter types.
func WithExcludeCustomOptions() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.includeCustomOptions = false
	}
}

// WithExcludeKnownExtensions returns an option that will cause an image filtered via
// ImageFilteredByTypesWithOptions to *not* include the known extensions for included
// extendable messages unless they are explicitly named in the list of filter types.
func WithExcludeKnownExtensions() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.includeKnownExtensions = false
	}
}

// WithAllowFilterByImportedType returns an option for ImageFilteredByTypesWithOptions
// that allows a named filter type to be in an imported file or module. Without this
// option, only types defined directly in the image to be filtered are allowed.
func WithAllowFilterByImportedType() ImageFilterOption {
	return func(opts *imageFilterOptions) {
		opts.allowImportedTypes = true
	}
}

// ImageFilteredByTypes returns a minimal image containing only the descriptors
// required to define those types. The resulting contains only files in which
// those descriptors and their transitive closure of required descriptors, with
// each file only contains the minimal required types and imports.
//
// Although this returns a new bufimage.Image, it mutates the original image's
// underlying file's `descriptorpb.FileDescriptorProto` and the old image should
// not continue to be used.
//
// A descriptor is said to require another descriptor if the dependent
// descriptor is needed to accurately and completely describe that descriptor.
// For the follwing types that includes:
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
func ImageFilteredByTypes(image bufimage.Image, types ...string) (bufimage.Image, error) {
	return ImageFilteredByTypesWithOptions(image, types)
}

func ImageFilteredByTypesWithOptions(image bufimage.Image, types []string, opts ...ImageFilterOption) (bufimage.Image, error) {
	options := newImageFilterOptions()
	for _, o := range opts {
		o(options)
	}

	imageIndex, err := newImageIndexForImage(image, options)
	if err != nil {
		return nil, err
	}
	// Check types exist
	startingDescriptors := make([]protosource.NamedDescriptor, 0, len(types))
	for _, typeName := range types {
		descriptor, ok := imageIndex.NameToDescriptor[typeName]
		if !ok {
			return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeNotFound)
		}
		if image.GetFile(descriptor.File().Path()).IsImport() && !options.allowImportedTypes {
			return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
		startingDescriptors = append(startingDescriptors, descriptor)
	}
	// Find all types to include in filtered image.
	seen := make(map[string]struct{})
	neededDescriptors := []descriptorAndDirects{}
	for _, startingDescriptor := range startingDescriptors {
		closure, err := descriptorTransitiveClosure(startingDescriptor, imageIndex, seen, options)
		if err != nil {
			return nil, err
		}
		neededDescriptors = append(neededDescriptors, closure...)
	}
	descriptorsByFile := make(map[string][]descriptorAndDirects)
	for _, descriptor := range neededDescriptors {
		descriptorsByFile[descriptor.Descriptor.File().Path()] = append(
			descriptorsByFile[descriptor.Descriptor.File().Path()],
			descriptor,
		)
	}
	// Create a new image with only the required descriptors.
	var includedFiles []bufimage.ImageFile
	for _, imageFile := range image.Files() {
		descriptors, ok := descriptorsByFile[imageFile.Path()]
		if !ok {
			continue
		}

		importsRequired := make(map[string]struct{})
		typesToKeep := make(map[string]struct{})
		for _, descriptor := range descriptors {
			typesToKeep[descriptor.Descriptor.FullName()] = struct{}{}
			for _, importedDescriptor := range descriptor.Directs {
				if importedDescriptor.File() != descriptor.Descriptor.File() {
					importsRequired[importedDescriptor.File().Path()] = struct{}{}
				}
			}
		}

		includedFiles = append(includedFiles, imageFile)
		imageFileDescriptor := imageFile.Proto()
		// While employing
		// https://github.com/golang/go/wiki/SliceTricks#filter-in-place,
		// also keep a record of which index moved where so we can fixup
		// the file's WeakDependency field.
		indexFromTo := make(map[int32]int32)
		indexTo := 0
		for indexFrom, importPath := range imageFileDescriptor.GetDependency() {
			if _, ok := importsRequired[importPath]; ok {
				indexFromTo[int32(indexFrom)] = int32(indexTo)
				imageFileDescriptor.Dependency[indexTo] = importPath
				indexTo++
				// delete them as we go, so we know which ones weren't in the list
				delete(importsRequired, importPath)
			}
		}
		imageFileDescriptor.Dependency = imageFileDescriptor.Dependency[:indexTo]

		// Add any other imports (which may not have been in the list because
		// they were picked up via a public import). The filtered files will not
		// use public imports.
		for importPath := range importsRequired {
			imageFileDescriptor.Dependency = append(imageFileDescriptor.Dependency, importPath)
		}
		imageFileDescriptor.PublicDependency = nil

		i := 0
		for _, indexFrom := range imageFileDescriptor.WeakDependency {
			if indexTo, ok := indexFromTo[indexFrom]; ok {
				imageFileDescriptor.WeakDependency[i] = indexTo
				i++
			}
		}
		imageFileDescriptor.WeakDependency = imageFileDescriptor.WeakDependency[:i]

		prefix := ""
		if imageFileDescriptor.Package != nil {
			prefix = imageFileDescriptor.GetPackage() + "."
		}
		trimMessages, err := trimMessageDescriptor(imageFileDescriptor.MessageType, prefix, typesToKeep)
		if err != nil {
			return nil, err
		}
		imageFileDescriptor.MessageType = trimMessages
		trimEnums, err := trimEnumDescriptor(imageFileDescriptor.EnumType, prefix, typesToKeep)
		if err != nil {
			return nil, err
		}
		imageFileDescriptor.EnumType = trimEnums
		trimExtensions, err := trimExtensionDescriptors(imageFileDescriptor.Extension, prefix, typesToKeep)
		if err != nil {
			return nil, err
		}
		imageFileDescriptor.Extension = trimExtensions
		i = 0
		for _, serviceDescriptor := range imageFileDescriptor.Service {
			name := prefix + serviceDescriptor.GetName()
			if _, ok := typesToKeep[name]; ok {
				imageFileDescriptor.Service[i] = serviceDescriptor
				i++
			}
		}
		imageFileDescriptor.Service = imageFileDescriptor.Service[:i]

		// TODO: With some from/to mappings, perhaps even sourcecodeinfo
		// isn't too bad.
		imageFileDescriptor.SourceCodeInfo = nil
	}
	return bufimage.NewImage(includedFiles)
}

// trimMessageDescriptor removes (nested) messages and nested enums from a slice
// of message descriptors if their type names are not found in the toKeep map.
func trimMessageDescriptor(in []*descriptorpb.DescriptorProto, prefix string, toKeep map[string]struct{}) ([]*descriptorpb.DescriptorProto, error) {
	i := 0
	for _, messageDescriptor := range in {
		name := prefix + messageDescriptor.GetName()
		if _, ok := toKeep[name]; ok {
			trimMessages, err := trimMessageDescriptor(messageDescriptor.NestedType, name+".", toKeep)
			if err != nil {
				return nil, err
			}
			messageDescriptor.NestedType = trimMessages
			trimEnums, err := trimEnumDescriptor(messageDescriptor.EnumType, name+".", toKeep)
			if err != nil {
				return nil, err
			}
			messageDescriptor.EnumType = trimEnums
			trimExtensions, err := trimExtensionDescriptors(messageDescriptor.Extension, name+".", toKeep)
			if err != nil {
				return nil, err
			}
			messageDescriptor.Extension = trimExtensions
			in[i] = messageDescriptor
			i++
		}
	}
	return in[:i], nil
}

// trimEnumDescriptor removes enums from a slice of enum descriptors if their
// type names are not found in the toKeep map.
func trimEnumDescriptor(in []*descriptorpb.EnumDescriptorProto, prefix string, toKeep map[string]struct{}) ([]*descriptorpb.EnumDescriptorProto, error) {
	i := 0
	for _, enumDescriptor := range in {
		name := prefix + enumDescriptor.GetName()
		if _, ok := toKeep[name]; ok {
			in[i] = enumDescriptor
			i++
		}
	}
	return in[:i], nil
}

// trimExtensionDescriptors removes fields from a slice of field descriptors if their
// type names are not found in the toKeep map.
func trimExtensionDescriptors(in []*descriptorpb.FieldDescriptorProto, prefix string, toKeep map[string]struct{}) ([]*descriptorpb.FieldDescriptorProto, error) {
	i := 0
	for _, fieldDescriptor := range in {
		name := prefix + fieldDescriptor.GetName()
		if _, ok := toKeep[name]; ok {
			in[i] = fieldDescriptor
			i++
		}
	}
	return in[:i], nil
}

// descriptorAndDirects holds a protosource.NamedDescriptor and a list of all
// named descriptors it directly references. A directly referenced dependency is
// any type that if defined in a different file from the principal descriptor,
// an import statement would be required for the proto to compile.
type descriptorAndDirects struct {
	Descriptor protosource.NamedDescriptor
	Directs    []protosource.NamedDescriptor
}

func descriptorTransitiveClosure(
	namedDescriptor protosource.NamedDescriptor,
	imageIndex *imageIndex,
	seen map[string]struct{},
	opts *imageFilterOptions,
) ([]descriptorAndDirects, error) {
	if _, ok := seen[namedDescriptor.FullName()]; ok {
		return nil, nil
	}
	seen[namedDescriptor.FullName()] = struct{}{}

	directDependencies := []protosource.NamedDescriptor{}
	transitiveDependencies := []descriptorAndDirects{}
	switch typedDescriptor := namedDescriptor.(type) {
	case protosource.Message:
		for _, field := range typedDescriptor.Fields() {
			switch field.Type() {
			case protosource.FieldDescriptorProtoTypeEnum,
				protosource.FieldDescriptorProtoTypeMessage,
				protosource.FieldDescriptorProtoTypeGroup:
				inputDescriptor, ok := imageIndex.NameToDescriptor[field.TypeName()]
				if !ok {
					return nil, fmt.Errorf("missing %q", field.TypeName())
				}
				directDependencies = append(directDependencies, inputDescriptor)
				recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen, opts)
				if err != nil {
					return nil, err
				}
				transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
			case protosource.FieldDescriptorProtoTypeDouble,
				protosource.FieldDescriptorProtoTypeFloat,
				protosource.FieldDescriptorProtoTypeInt64,
				protosource.FieldDescriptorProtoTypeUint64,
				protosource.FieldDescriptorProtoTypeInt32,
				protosource.FieldDescriptorProtoTypeFixed64,
				protosource.FieldDescriptorProtoTypeFixed32,
				protosource.FieldDescriptorProtoTypeBool,
				protosource.FieldDescriptorProtoTypeString,
				protosource.FieldDescriptorProtoTypeBytes,
				protosource.FieldDescriptorProtoTypeUint32,
				protosource.FieldDescriptorProtoTypeSfixed32,
				protosource.FieldDescriptorProtoTypeSfixed64,
				protosource.FieldDescriptorProtoTypeSint32,
				protosource.FieldDescriptorProtoTypeSint64:
				// nothing to explore for the field type, but
				// there might be custom field options
			default:
				return nil, fmt.Errorf("unknown field type %d", field.Type())
			}
			// fieldoptions
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(field, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Extensions declared for this message.
		// TODO: We currently exclude all extensions for descriptor options types (and instead
		//  only gather the ones used in relevant custom options). But if the descriptor option
		//  type were a named type for filtering, we SHOULD include all of them.
		if opts.includeKnownExtensions && !isOptionsTypeName(namedDescriptor.FullName()) {
			for _, extendsDescriptor := range imageIndex.NameToExtensions[namedDescriptor.FullName()] {
				recursiveDescriptors, err := descriptorTransitiveClosure(extendsDescriptor, imageIndex, seen, opts)
				if err != nil {
					return nil, err
				}
				transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
			}
		}
		// Messages in which this message is nested
		if typedDescriptor.Parent() != nil {
			// TODO: we don't actually want or need the entire parent message unless it is actually
			//  referenced by other parts of the schema. As a reference from a nested message, we
			//  only care about it as a namespace (so all of its other elements, aside from needed
			//  nested types, could be omitted).
			directDependencies = append(directDependencies, typedDescriptor.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(typedDescriptor.Parent(), imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		// Options for all oneofs in this message
		for _, oneOfDescriptor := range typedDescriptor.Oneofs() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(oneOfDescriptor, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Options for all extension ranges in this message
		for _, extRange := range typedDescriptor.ExtensionRanges() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(extRange, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.Enum:
		// Parent messages
		if typedDescriptor.Parent() != nil {
			// TODO: ditto above: unless the parent message is used elsewhere, we don't need the
			//  entire message; we only need it as a placeholder for namespacing.
			directDependencies = append(directDependencies, typedDescriptor.Parent())
			recursiveDescriptors, err := descriptorTransitiveClosure(typedDescriptor.Parent(), imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		for _, enumValue := range typedDescriptor.Values() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(enumValue, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.Service:
		for _, method := range typedDescriptor.Methods() {
			directDependencies = append(directDependencies, method)
			recursiveDescriptors, err := descriptorTransitiveClosure(method, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		// Options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	case protosource.Method:
		// in case method was directly named as a filter type, make sure we include parent service
		// TODO: if the service is not also named as a filter type, we could prune the service down
		//  to only the named methods and shrink the size of the filtered image further.
		recursiveDescriptors, err := descriptorTransitiveClosure(typedDescriptor.Service(), imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)

		inputDescriptor, ok := imageIndex.NameToDescriptor[typedDescriptor.InputTypeName()]
		if !ok {
			return nil, fmt.Errorf("missing %q", typedDescriptor.InputTypeName())
		}
		recursiveDescriptorsIn, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptorsIn...)
		directDependencies = append(directDependencies, inputDescriptor)

		outputDescriptor, ok := imageIndex.NameToDescriptor[typedDescriptor.OutputTypeName()]
		if !ok {
			return nil, fmt.Errorf("missing %q", typedDescriptor.OutputTypeName())
		}
		recursiveDescriptorsOut, err := descriptorTransitiveClosure(outputDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptorsOut...)
		directDependencies = append(directDependencies, outputDescriptor)

		// options
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)

	case protosource.Field:
		// Regular fields get handled by protosource.Message, only
		// protosource.Fields's for extends definitions should reach
		// here.
		if typedDescriptor.Extendee() == "" {
			return nil, fmt.Errorf("expected extendee for field %q to not be empty", typedDescriptor.FullName())
		}
		extendeeDescriptor, ok := imageIndex.NameToDescriptor[typedDescriptor.Extendee()]
		if !ok {
			return nil, fmt.Errorf("missing %q", typedDescriptor.Extendee())
		}
		directDependencies = append(directDependencies, extendeeDescriptor)
		recursiveDescriptors, err := descriptorTransitiveClosure(extendeeDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)

		switch typedDescriptor.Type() {
		case protosource.FieldDescriptorProtoTypeEnum,
			protosource.FieldDescriptorProtoTypeMessage,
			protosource.FieldDescriptorProtoTypeGroup:
			inputDescriptor, ok := imageIndex.NameToDescriptor[typedDescriptor.TypeName()]
			if !ok {
				return nil, fmt.Errorf("missing %q", typedDescriptor.TypeName())
			}
			directDependencies = append(directDependencies, inputDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		case protosource.FieldDescriptorProtoTypeDouble,
			protosource.FieldDescriptorProtoTypeFloat,
			protosource.FieldDescriptorProtoTypeInt64,
			protosource.FieldDescriptorProtoTypeUint64,
			protosource.FieldDescriptorProtoTypeInt32,
			protosource.FieldDescriptorProtoTypeFixed64,
			protosource.FieldDescriptorProtoTypeFixed32,
			protosource.FieldDescriptorProtoTypeBool,
			protosource.FieldDescriptorProtoTypeString,
			protosource.FieldDescriptorProtoTypeBytes,
			protosource.FieldDescriptorProtoTypeUint32,
			protosource.FieldDescriptorProtoTypeSfixed32,
			protosource.FieldDescriptorProtoTypeSfixed64,
			protosource.FieldDescriptorProtoTypeSint32,
			protosource.FieldDescriptorProtoTypeSint64:
			// nothing to follow, custom options handled below.
		default:
			return nil, fmt.Errorf("unknown field type %d", typedDescriptor.Type())
		}
		explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(typedDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		directDependencies = append(directDependencies, explicitOptionDeps...)
		transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
	default:
		return nil, fmt.Errorf("unexpected protosource type %T", typedDescriptor)
	}

	explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(namedDescriptor.File(), imageIndex, seen, opts)
	if err != nil {
		return nil, err
	}
	directDependencies = append(directDependencies, explicitOptionDeps...)
	transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)

	return append(
		transitiveDependencies,
		descriptorAndDirects{Descriptor: namedDescriptor, Directs: directDependencies},
	), nil
}

func exploreCustomOptions(
	descriptor protosource.OptionExtensionDescriptor,
	imageIndex *imageIndex,
	seen map[string]struct{},
	opts *imageFilterOptions,
) ([]protosource.NamedDescriptor, []descriptorAndDirects, error) {
	if !opts.includeCustomOptions {
		return nil, nil, nil
	}

	directDependencies := []protosource.NamedDescriptor{}
	transitiveDependencies := []descriptorAndDirects{}

	var optionName string
	switch descriptor.(type) {
	case protosource.File:
		optionName = "google.protobuf.FileOptions"
	case protosource.Message:
		optionName = "google.protobuf.MessageOptions"
	case protosource.Field:
		optionName = "google.protobuf.FieldOptions"
	case protosource.Oneof:
		optionName = "google.protobuf.OneofOptions"
	case protosource.Enum:
		optionName = "google.protobuf.EnumOptions"
	case protosource.EnumValue:
		optionName = "google.protobuf.EnumValueOptions"
	case protosource.Service:
		optionName = "google.protobuf.ServiceOptions"
	case protosource.Method:
		optionName = "google.protobuf.MethodOptions"
	case protosource.ExtensionRange:
		optionName = "google.protobuf.ExtensionRangeOptions"
	default:
		return nil, nil, fmt.Errorf("unexpected type for exploring options %T", descriptor)
	}

	for _, n := range descriptor.PresentExtensionNumbers() {
		optionsByNumber := imageIndex.NameToOptions[optionName]
		field, ok := optionsByNumber[n]
		if !ok {
			return nil, nil, fmt.Errorf("cannot find ext no %d on %s", n, optionName)
		}
		directDependencies = append(directDependencies, field)
		recursiveDescriptors, err := descriptorTransitiveClosure(field, imageIndex, seen, opts)
		if err != nil {
			return nil, nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
	}
	return directDependencies, transitiveDependencies, nil
}

func freeMessageRangeStringsRec(
	s []string,
	message protosource.Message,
) []string {
	for _, nestedMessage := range message.Messages() {
		s = freeMessageRangeStringsRec(s, nestedMessage)
	}
	if e := protosource.FreeMessageRangeString(message); e != "" {
		return append(s, e)
	}
	return s
}

type imageFilterOptions struct {
	includeCustomOptions   bool
	includeKnownExtensions bool
	allowImportedTypes     bool
}

func newImageFilterOptions() *imageFilterOptions {
	return &imageFilterOptions{
		includeCustomOptions:   true,
		includeKnownExtensions: true,
		allowImportedTypes:     false,
	}
}
