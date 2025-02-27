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
	// ErrImageFilterTypeNotFound is returned from ImageFilteredByTypes when
	// a specified type cannot be found in an image.
	ErrImageFilterTypeNotFound = errors.New("not found")

	// ErrImageFilterTypeIsImport is returned from ImageFilteredByTypes when
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

// WithIncludeTypes returns an option for ImageFilteredByTypesWithOptions that specifies
// the set of types that should be included in the filtered image.
func WithIncludeTypes(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if opts.includeTypes == nil {
			opts.includeTypes = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.includeTypes[typeName] = struct{}{}
		}
	}
}

// WithExcludeTypes returns an option for ImageFilteredByTypesWithOptions that specifies
// the set of types that should be excluded from the filtered image.
func WithExcludeTypes(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if opts.excludeTypes == nil {
			opts.excludeTypes = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.excludeTypes[typeName] = struct{}{}
		}
	}
}

// WithIncludeOptions returns an option for ImageFilteredByTypesWithOptions that specifies
// the set of options that should be included in the filtered image.
func WithIncludeOptions(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if opts.includeOptions == nil {
			opts.includeOptions = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.includeOptions[typeName] = struct{}{}
		}
	}
}

// WithExcludeOptions returns an option for ImageFilteredByTypesWithOptions that specifies
// the set of options that should be excluded from the filtered image.
//
// May be provided multiple times.
func WithExcludeOptions(typeNames ...string) ImageFilterOption {
	return func(opts *imageFilterOptions) {
		if opts.excludeOptions == nil {
			opts.excludeOptions = make(map[string]struct{}, len(typeNames))
		}
		for _, typeName := range typeNames {
			opts.excludeOptions[typeName] = struct{}{}
		}
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
// included type transitively depens on the excluded type, the descriptor will
// be altered to remove the dependency.
//
// This returns a new [bufimage.Image] that is a shallow copy of the underlying
// [descriptorpb.FileDescriptorProto]s of the original. The new image may therefore
// share state with the original image, so it should not be modified.
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

// transitiveClosure accumulates the elements, files, and needed imports for a
// subset of an image. When an element is added to the closure, all of its
// dependencies are recursively added.
type transitiveClosure struct {
	// The elements included in the transitive closure.
	elements map[namedDescriptor]closureInclusionMode
	// The ordered set of imports for each file. This allows for re-writing imports
	// for files whose contents have been pruned.
	imports map[string]*orderedImports
}

type closureInclusionMode int

const (
	// Element is explicitly excluded from the closure.
	inclusionModeExcluded = closureInclusionMode(iota - 1) // -1
	// Element is not yet known to be included or excluded.
	inclusionModeUnknown // 0
	// Element is included in closure because it is directly reachable from a root.
	inclusionModeExplicit // 1
	// Element is included in closure because it is a message or service that
	// *contains* an explicitly included element but is not itself directly
	// reachable.
	inclusionModeEnclosing // 2
	// Element is included in closure because it is implied by the presence of a
	// custom option. For example, a field element with a custom option implies
	// the presence of google.protobuf.FieldOptions. An option type could instead be
	// explicitly included if it is also directly reachable (i.e. some type in the
	// graph explicitly refers to the option type).
	inclusionModeImplicit // 3
)

func newTransitiveClosure() *transitiveClosure {
	return &transitiveClosure{
		elements: map[namedDescriptor]closureInclusionMode{},
		imports:  map[string]*orderedImports{},
	}
}

func (t *transitiveClosure) hasType(
	descriptor namedDescriptor,
	options *imageFilterOptions,
) (isIncluded bool) {
	defer func() { fmt.Println("\thasType", descriptor.GetName(), isIncluded) }()
	if options == nil {
		return true // no filter
	}
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
	options *imageFilterOptions,
) (isIncluded bool) {
	defer func() { fmt.Println("\thasOption", fieldDescriptor.FullName(), isIncluded) }()
	fullName := fieldDescriptor.FullName()
	defer fmt.Println("\thasOption", fullName, isIncluded)
	if options == nil {
		return true // no filter
	}
	if options.excludeTypes != nil {
		if _, ok := options.excludeTypes[string(fullName)]; ok {
			return false
		}
	}
	if fieldDescriptor.IsExtension() && !options.includeCustomOptions {
		return false
	}
	if options.includeOptions != nil {
		_, isIncluded = options.includeOptions[string(fullName)]
		return isIncluded
	}
	return true
}

func (t *transitiveClosure) excludeElement(
	descriptor namedDescriptor,
	imageIndex *imageIndex,
	options *imageFilterOptions,
) error {
	_ = descriptor
	// TODO: implement
	return nil
}

func (t *transitiveClosure) addType(
	typeName protoreflect.FullName,
	imageIndex *imageIndex,
	options *imageFilterOptions,
) error {
	// TODO: consider supporting a glob syntax of some kind, to do more advanced pattern
	//   matching, such as ability to get a package AND all of its sub-packages.
	descriptorInfo, ok := imageIndex.ByName[typeName]
	if ok {
		// It's a type name
		if !options.allowImportedTypes && descriptorInfo.file.IsImport() {
			return fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
		return t.addElement(descriptorInfo.element, "", false, imageIndex, options)
	}
	// It could be a package name
	pkg, ok := imageIndex.Packages[string(typeName)]
	if !ok {
		// but it's not...
		return fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeNotFound)
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
			return fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
		}
	}
	return t.addPackage(pkg, imageIndex, options)
}

func (t *transitiveClosure) addImport(fromPath, toPath string) {
	if fromPath == toPath {
		return // no need for a file to import itself
	}
	imps := t.imports[fromPath]
	if imps == nil {
		imps = newOrderedImports()
		t.imports[fromPath] = imps
	}
	imps.add(toPath)
}

func (t *transitiveClosure) addPackage(
	pkg *packageInfo,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	fmt.Printf("ADD PACKAGE: %q\n", pkg.fullName)
	for _, file := range pkg.files {
		fmt.Println("\taddPackage", file.Path())
		fileDescriptor := file.FileDescriptorProto()
		if err := t.addElement(fileDescriptor, "", false, imageIndex, opts); err != nil {
			return err
		}
	}
	return nil
}

func (t *transitiveClosure) addElement(
	descriptor namedDescriptor,
	referrerFile string,
	impliedByCustomOption bool,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	descriptorInfo := imageIndex.ByDescriptor[descriptor]
	if referrerFile != "" {
		t.addImport(referrerFile, descriptorInfo.file.Path())
	}

	if existingMode, ok := t.elements[descriptor]; ok && existingMode != inclusionModeEnclosing {
		if existingMode == inclusionModeImplicit && !impliedByCustomOption {
			// upgrade from implied to explicitly part of closure
			t.elements[descriptor] = inclusionModeExplicit
		}
		return nil // already added this element
	}
	if impliedByCustomOption {
		t.elements[descriptor] = inclusionModeImplicit
	} else {
		t.elements[descriptor] = inclusionModeExplicit
	}

	// if this type is enclosed inside another, add enclosing types
	fmt.Println("--- ADDING ELEMENT", descriptorInfo.fullName, "=>", t.elements[descriptor])
	fmt.Printf("\t %s %T\n", descriptorInfo.file.Path(), descriptorInfo.parent)
	if err := t.addEnclosing(descriptorInfo.parent, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
		return err
	}
	// add any custom options and their dependencies
	if err := t.exploreCustomOptions(descriptor, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
		return err
	}

	switch typedDescriptor := descriptor.(type) {
	case *descriptorpb.FileDescriptorProto:
		typeNames, ok := imageIndex.FileTypes[typedDescriptor.GetName()]
		if !ok {
			return fmt.Errorf("missing %q", typedDescriptor.GetName())
		}
		// A file includes all elements. The types are resolved in the image index
		// to ensure all nested types are included.
		for _, typeName := range typeNames {
			typeInfo := imageIndex.ByName[typeName]
			if err := t.addElement(typeInfo.element, "", false, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.DescriptorProto:
		// Options and types for all fields
		for _, field := range typedDescriptor.GetField() {
			if err := t.addFieldType(field, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
				return err
			}
			if err := t.exploreCustomOptions(field, referrerFile, imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all oneofs in this message
		for _, oneOfDescriptor := range typedDescriptor.GetOneofDecl() {
			if err := t.exploreCustomOptions(oneOfDescriptor, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all extension ranges in this message
		for _, extRange := range typedDescriptor.GetExtensionRange() {
			if err := t.exploreCustomOptions(extRange, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.EnumDescriptorProto:
		for _, enumValue := range typedDescriptor.GetValue() {
			if err := t.exploreCustomOptions(enumValue, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.ServiceDescriptorProto:
		for _, method := range typedDescriptor.GetMethod() {
			if err := t.addElement(method, "", false, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.MethodDescriptorProto:
		inputName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetInputType(), "."))
		inputInfo, ok := imageIndex.ByName[inputName]
		if !ok {
			return fmt.Errorf("missing %q", inputName)
		}
		if err := t.addElement(inputInfo.element, descriptorInfo.file.Path(), false, imageIndex, opts); err != nil {
			return err
		}

		outputName := protoreflect.FullName(strings.TrimPrefix(typedDescriptor.GetOutputType(), "."))
		outputInfo, ok := imageIndex.ByName[outputName]
		if !ok {
			return fmt.Errorf("missing %q", outputName)
		}
		if err := t.addElement(outputInfo.element, descriptorInfo.file.Path(), false, imageIndex, opts); err != nil {
			return err
		}

	case *descriptorpb.FieldDescriptorProto:
		fmt.Println("ADDING EXTENSION", typedDescriptor.GetExtendee())
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
		if err := t.addElement(extendeeInfo.element, descriptorInfo.file.Path(), impliedByCustomOption, imageIndex, opts); err != nil {
			return err
		}
		if err := t.addFieldType(typedDescriptor, descriptorInfo.file.Path(), imageIndex, opts); err != nil {
			return err
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

func (t *transitiveClosure) addEnclosing(descriptor namedDescriptor, enclosingFile string, imageIndex *imageIndex, opts *imageFilterOptions) error {
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
		if err := t.exploreCustomOptions(descriptor, enclosingFile, imageIndex, opts); err != nil {
			return err
		}
		// now move into this element's parent
		descriptor = imageIndex.ByDescriptor[descriptor].parent
	}
	return nil
}

func (t *transitiveClosure) addFieldType(field *descriptorpb.FieldDescriptorProto, referrerFile string, imageIndex *imageIndex, opts *imageFilterOptions) error {
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		typeName := protoreflect.FullName(strings.TrimPrefix(field.GetTypeName(), "."))
		info, ok := imageIndex.ByName[typeName]
		if !ok {
			return fmt.Errorf("missing %q", typeName)
		}
		err := t.addElement(info.element, referrerFile, false, imageIndex, opts)
		if err != nil {
			return err
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
		return fmt.Errorf("unknown field type %d", field.GetType())
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
		fmt.Println("ADDING EXTENSIONS FOR", e.GetName())
		msgDescriptor, ok := e.(*descriptorpb.DescriptorProto)
		if !ok {
			// not a message, nothing to do
			continue
		}
		fmt.Println("\t", msgDescriptor.GetName())
		descriptorInfo := imageIndex.ByDescriptor[msgDescriptor]
		for _, extendsDescriptor := range imageIndex.NameToExtensions[descriptorInfo.fullName] {
			fmt.Println("\t\t", extendsDescriptor.GetName())
			if err := t.addElement(extendsDescriptor, "", false, imageIndex, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *transitiveClosure) exploreCustomOptions(
	descriptor proto.Message,
	referrerFile string,
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
		// If the value contains an Any message, we should add the message type
		// therein to the closure.
		if err = t.exploreOptionValueForAny(fd, val, referrerFile, imageIndex, opts); err != nil {
			return false
		}

		// Also include custom option definitions (e.g. extensions)
		if !fd.IsExtension() {
			return true
		}
		optionsByNumber := imageIndex.NameToOptions[optionsName]
		field, ok := optionsByNumber[int32(fd.Number())]
		if !ok {
			err = fmt.Errorf("cannot find ext no %d on %s", fd.Number(), optionsName)
			return false
		}
		err = t.addElement(field, referrerFile, true, imageIndex, opts)
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
	referrerFile string,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	switch {
	case fd.IsMap():
		if isMessageKind(fd.MapValue().Kind()) {
			var err error
			val.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				if err = t.exploreOptionSingularValueForAny(v.Message(), referrerFile, imageIndex, opts); err != nil {
					return false
				}
				return true
			})
			return err
		}
	case isMessageKind(fd.Kind()):
		if fd.IsList() {
			listVal := val.List()
			for i := 0; i < listVal.Len(); i++ {
				if err := t.exploreOptionSingularValueForAny(listVal.Get(i).Message(), referrerFile, imageIndex, opts); err != nil {
					return err
				}
			}
		} else {
			return t.exploreOptionSingularValueForAny(val.Message(), referrerFile, imageIndex, opts)
		}
	}
	return nil
}

func (t *transitiveClosure) exploreOptionSingularValueForAny(
	msg protoreflect.Message,
	referrerFile string,
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
			if err := t.addElement(d, referrerFile, false, imageIndex, opts); err != nil {
				return err
			}
		}
		// TODO: unmarshal the bytes to see if there are any nested Any messages
		return nil
	}
	// keep digging
	var err error
	msg.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		err = t.exploreOptionValueForAny(fd, val, referrerFile, imageIndex, opts)
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
	includeTypes           map[string]struct{}
	excludeTypes           map[string]struct{}
	includeOptions         map[string]struct{}
	excludeOptions         map[string]struct{}
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

// orderedImports is a structure to maintain an ordered set of imports. This is needed
// because we want to be able to iterate through imports in a deterministic way when filtering
// the image.
type orderedImports struct {
	pathToIndex map[string]int
	paths       []string
}

// newOrderedImports creates a new orderedImports structure.
func newOrderedImports() *orderedImports {
	return &orderedImports{
		pathToIndex: map[string]int{},
	}
}

// index returns the index for a given path. If the path does not exist in index map, -1
// is returned and should be considered deleted.
func (o *orderedImports) index(path string) int {
	if index, ok := o.pathToIndex[path]; ok {
		return index
	}
	return -1
}

// add appends a path to the paths list and the index in the map. If a key already exists,
// then this is a no-op.
func (o *orderedImports) add(path string) {
	if _, ok := o.pathToIndex[path]; !ok {
		o.pathToIndex[path] = len(o.paths)
		o.paths = append(o.paths, path)
	}
}

// delete removes a key from the index map of ordered imports. If a non-existent path is
// set for deletion, then this is a no-op.
// Note that the path is not removed from the paths list. If you want to iterate through
// the paths, use keys() to get all non-deleted keys.
func (o *orderedImports) delete(path string) {
	delete(o.pathToIndex, path)
}

// keys provides all non-deleted keys from the ordered imports.
func (o *orderedImports) keys() []string {
	keys := make([]string, 0, len(o.pathToIndex))
	for _, path := range o.paths {
		if _, ok := o.pathToIndex[path]; ok {
			keys = append(keys, path)
		}
	}
	return keys
}
