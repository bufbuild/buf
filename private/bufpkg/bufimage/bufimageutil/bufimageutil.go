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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	startingDescriptors := make([]namedDescriptor, 0, len(types))
	for _, typeName := range types {
		startingDescriptor, ok := imageIndex.ByName[typeName]
		if !ok {
			return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeNotFound)
		}
		typeInfo := imageIndex.ByDescriptor[startingDescriptor]
		if image.GetFile(typeInfo.file).IsImport() && !options.allowImportedTypes {
			return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
		startingDescriptors = append(startingDescriptors, startingDescriptor)
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
		typeInfo := imageIndex.ByDescriptor[descriptor.Descriptor]
		descriptorsByFile[typeInfo.file] = append(descriptorsByFile[typeInfo.file], descriptor)
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
			typeInfo := imageIndex.ByDescriptor[descriptor.Descriptor]
			typesToKeep[typeInfo.fullName] = struct{}{}
			for _, importedDescriptor := range descriptor.Directs {
				importedInfo := imageIndex.ByDescriptor[importedDescriptor]
				if importedInfo.file != typeInfo.file {
					importsRequired[importedInfo.file] = struct{}{}
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
	Descriptor namedDescriptor
	Directs    []namedDescriptor
}

func descriptorTransitiveClosure(
	descriptor namedDescriptor,
	imageIndex *imageIndex,
	seen map[string]struct{},
	opts *imageFilterOptions,
) ([]descriptorAndDirects, error) {
	descriptorInfo := imageIndex.ByDescriptor[descriptor]
	if _, ok := seen[descriptorInfo.fullName]; ok {
		return nil, nil
	}
	seen[descriptorInfo.fullName] = struct{}{}

	var directDependencies []namedDescriptor
	var transitiveDependencies []descriptorAndDirects
	switch typedDescriptor := descriptor.(type) {
	case *descriptorpb.DescriptorProto:
		for _, field := range typedDescriptor.GetField() {
			switch field.GetType() {
			case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
				descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
				descriptorpb.FieldDescriptorProto_TYPE_GROUP:
				typeName := strings.TrimPrefix(field.GetTypeName(), ".")
				typeDescriptor, ok := imageIndex.ByName[typeName]
				if !ok {
					return nil, fmt.Errorf("missing %q", typeName)
				}
				directDependencies = append(directDependencies, typeDescriptor)
				recursiveDescriptors, err := descriptorTransitiveClosure(typeDescriptor, imageIndex, seen, opts)
				if err != nil {
					return nil, err
				}
				transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
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
				// nothing to explore for the field type, but
				// there might be custom field options
			default:
				return nil, fmt.Errorf("unknown field type %d", field.GetType())
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
		if opts.includeKnownExtensions && !isOptionsTypeName(descriptorInfo.fullName) {
			for _, extendsDescriptor := range imageIndex.NameToExtensions[descriptorInfo.fullName] {
				recursiveDescriptors, err := descriptorTransitiveClosure(extendsDescriptor, imageIndex, seen, opts)
				if err != nil {
					return nil, err
				}
				transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
			}
		}
		// Messages in which this message is nested
		if _, ok := descriptorInfo.parent.(*descriptorpb.DescriptorProto); ok {
			// TODO: we don't actually want or need the entire parent message unless it is actually
			//  referenced by other parts of the schema. As a reference from a nested message, we
			//  only care about it as a namespace (so all of its other elements, aside from needed
			//  nested types, could be omitted).
			directDependencies = append(directDependencies, descriptorInfo.parent)
			recursiveDescriptors, err := descriptorTransitiveClosure(descriptorInfo.parent, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		// Options for all oneofs in this message
		for _, oneOfDescriptor := range typedDescriptor.GetOneofDecl() {
			explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(oneOfDescriptor, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			directDependencies = append(directDependencies, explicitOptionDeps...)
			transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)
		}
		// Options for all extension ranges in this message
		for _, extRange := range typedDescriptor.GetExtensionRange() {
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
	case *descriptorpb.EnumDescriptorProto:
		// Parent messages
		if _, ok := descriptorInfo.parent.(*descriptorpb.DescriptorProto); ok {
			// TODO: ditto above: unless the parent message is used elsewhere, we don't need the
			//  entire message; we only need it as a placeholder for namespacing.
			directDependencies = append(directDependencies, descriptorInfo.parent)
			recursiveDescriptors, err := descriptorTransitiveClosure(descriptorInfo.parent, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		}
		for _, enumValue := range typedDescriptor.GetValue() {
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
	case *descriptorpb.ServiceDescriptorProto:
		for _, method := range typedDescriptor.GetMethod() {
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
	case *descriptorpb.MethodDescriptorProto:
		// in case method was directly named as a filter type, make sure we include parent service
		// TODO: if the service is not also named as a filter type, we could prune the service down
		//  to only the named methods and shrink the size of the filtered image further.
		recursiveDescriptors, err := descriptorTransitiveClosure(descriptorInfo.parent, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)

		inputName := strings.TrimPrefix(typedDescriptor.GetInputType(), ".")
		inputDescriptor, ok := imageIndex.ByName[inputName]
		if !ok {
			return nil, fmt.Errorf("missing %q", inputName)
		}
		recursiveDescriptorsIn, err := descriptorTransitiveClosure(inputDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptorsIn...)
		directDependencies = append(directDependencies, inputDescriptor)

		outputName := strings.TrimPrefix(typedDescriptor.GetOutputType(), ".")
		outputDescriptor, ok := imageIndex.ByName[outputName]
		if !ok {
			return nil, fmt.Errorf("missing %q", outputName)
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

	case *descriptorpb.FieldDescriptorProto:
		// Regular fields get handled by protosource.Message, only
		// protosource.Fields's for extends definitions should reach
		// here.
		if typedDescriptor.GetExtendee() == "" {
			return nil, fmt.Errorf("expected extendee for field %q to not be empty", descriptorInfo.fullName)
		}
		extendeeName := strings.TrimPrefix(typedDescriptor.GetExtendee(), ".")
		extendeeDescriptor, ok := imageIndex.ByName[extendeeName]
		if !ok {
			return nil, fmt.Errorf("missing %q", extendeeName)
		}
		directDependencies = append(directDependencies, extendeeDescriptor)
		recursiveDescriptors, err := descriptorTransitiveClosure(extendeeDescriptor, imageIndex, seen, opts)
		if err != nil {
			return nil, err
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)

		switch typedDescriptor.GetType() {
		case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
			descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
			descriptorpb.FieldDescriptorProto_TYPE_GROUP:
			typeName := strings.TrimPrefix(typedDescriptor.GetTypeName(), ".")
			typeDescriptor, ok := imageIndex.ByName[typeName]
			if !ok {
				return nil, fmt.Errorf("missing %q", typeName)
			}
			directDependencies = append(directDependencies, typeDescriptor)
			recursiveDescriptors, err := descriptorTransitiveClosure(typeDescriptor, imageIndex, seen, opts)
			if err != nil {
				return nil, err
			}
			transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
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
			return nil, fmt.Errorf("unknown field type %d", typedDescriptor.GetType())
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

	fileDescriptor := imageIndex.Files[descriptorInfo.file]
	explicitOptionDeps, recursedOptionDeps, err := exploreCustomOptions(fileDescriptor, imageIndex, seen, opts)
	if err != nil {
		return nil, err
	}
	directDependencies = append(directDependencies, explicitOptionDeps...)
	transitiveDependencies = append(transitiveDependencies, recursedOptionDeps...)

	return append(
		transitiveDependencies,
		descriptorAndDirects{Descriptor: descriptor, Directs: directDependencies},
	), nil
}

func exploreCustomOptions(
	descriptor proto.Message,
	imageIndex *imageIndex,
	seen map[string]struct{},
	opts *imageFilterOptions,
) ([]namedDescriptor, []descriptorAndDirects, error) {
	if !opts.includeCustomOptions {
		return nil, nil, nil
	}

	var directDependencies []namedDescriptor
	var transitiveDependencies []descriptorAndDirects

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
		return nil, nil, fmt.Errorf("unexpected type for exploring options %T", descriptor)
	}

	optionsName := string(options.Descriptor().FullName())
	var err error
	options.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if !fd.IsExtension() {
			return true
		}
		optionsByNumber := imageIndex.NameToOptions[optionsName]
		field, ok := optionsByNumber[int32(fd.Number())]
		if !ok {
			err = fmt.Errorf("cannot find ext no %d on %s", fd.Number(), optionsName)
			return false
		}
		directDependencies = append(directDependencies, field)
		recursiveDescriptors, recurseErr := descriptorTransitiveClosure(field, imageIndex, seen, opts)
		if recurseErr != nil {
			err = recurseErr
			return false
		}
		transitiveDependencies = append(transitiveDependencies, recursiveDescriptors...)
		return true
	})
	if err != nil {
		return nil, nil, err
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
