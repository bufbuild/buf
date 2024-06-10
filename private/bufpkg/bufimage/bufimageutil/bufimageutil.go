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

// ImageFilteredByTypes returns a minimal image containing only the descriptors
// required to define those types. The resulting contains only files in which
// those descriptors and their transitive closure of required descriptors, with
// each file only contains the minimal required types and imports.
//
// Although this returns a new [bufimage.Image], it mutates the original image's
// underlying file's [descriptorpb.FileDescriptorProto]. So the old image should
// not continue to be used.
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
func ImageFilteredByTypes(image bufimage.Image, types ...string) (bufimage.Image, error) {
	return ImageFilteredByTypesWithOptions(image, types)
}

// ImageFilteredByTypesWithOptions returns a minimal image containing only the descriptors
// required to define those types. See ImageFilteredByTypes for more details. This version
// allows for customizing the behavior with options.
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
	var startingPackages []*protoPackage
	for _, typeName := range types {
		// TODO: consider supporting a glob syntax of some kind, to do more advanced pattern
		//   matching, such as ability to get a package AND all of its sub-packages.
		startingDescriptor, ok := imageIndex.ByName[typeName]
		if ok {
			// It's a type name
			typeInfo := imageIndex.ByDescriptor[startingDescriptor]
			if image.GetFile(typeInfo.file).IsImport() && !options.allowImportedTypes {
				return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
			}
			startingDescriptors = append(startingDescriptors, startingDescriptor)
			continue
		}
		// It could be a package name
		pkg, ok := imageIndex.Packages[typeName]
		if !ok {
			// but it's not...
			return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeNotFound)
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
				return nil, fmt.Errorf("filtering by type %q: %w", typeName, ErrImageFilterTypeIsImport)
			}
		}
		startingPackages = append(startingPackages, pkg)
	}
	// Find all types to include in filtered image.
	closure := newTransitiveClosure()
	for _, startingPackage := range startingPackages {
		if err := closure.addPackage(startingPackage, imageIndex, options); err != nil {
			return nil, err
		}
	}
	for _, startingDescriptor := range startingDescriptors {
		if err := closure.addElement(startingDescriptor, "", false, imageIndex, options); err != nil {
			return nil, err
		}
	}
	// After all types are added, add their known extensions
	if err := closure.addExtensions(imageIndex, options); err != nil {
		return nil, err
	}
	// Create a new image with only the required descriptors.
	var includedFiles []bufimage.ImageFile
	for _, imageFile := range image.Files() {
		_, ok := closure.files[imageFile.Path()]
		if !ok {
			continue
		}
		includedFiles = append(includedFiles, imageFile)
		imageFileDescriptor := imageFile.FileDescriptorProto()

		importsRequired := closure.imports[imageFile.Path()]
		// If the file has source code info, we need to remap paths to correctly
		// update this info for the elements retained after filtering.
		var sourcePathRemapper *sourcePathsRemapTrie
		if len(imageFileDescriptor.SourceCodeInfo.GetLocation()) > 0 {
			sourcePathRemapper = &sourcePathsRemapTrie{}
		}
		// We track the source path as we go through the model, so that we can
		// mark paths as moved or deleted. Whenever an element is deleted, any
		// subsequent elements of the same type in the same scope have are "moved",
		// because their index is shifted down.
		basePath := make([]int32, 1, 16)
		basePath[0] = fileDependencyTag
		// While employing
		// https://github.com/golang/go/wiki/SliceTricks#filter-in-place,
		// also keep a record of which index moved where, so we can fixup
		// the file's WeakDependency field.
		indexFromTo := make(map[int32]int32)
		indexTo := 0
		for indexFrom, importPath := range imageFileDescriptor.GetDependency() {
			path := append(basePath, int32(indexFrom))
			if _, ok := importsRequired[importPath]; ok {
				sourcePathRemapper.markMoved(path, int32(indexTo))
				indexFromTo[int32(indexFrom)] = int32(indexTo)
				imageFileDescriptor.Dependency[indexTo] = importPath
				indexTo++
				// markDeleted them as we go, so we know which ones weren't in the list
				delete(importsRequired, importPath)
			} else {
				sourcePathRemapper.markDeleted(path)
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
		sourcePathRemapper.markDeleted([]int32{filePublicDependencyTag})

		basePath = basePath[:1]
		basePath[0] = fileWeakDependencyTag
		i := 0
		for _, indexFrom := range imageFileDescriptor.WeakDependency {
			path := append(basePath, indexFrom)
			if indexTo, ok := indexFromTo[indexFrom]; ok {
				sourcePathRemapper.markMoved(path, indexTo)
				imageFileDescriptor.WeakDependency[i] = indexTo
				i++
			} else {
				sourcePathRemapper.markDeleted(path)
			}
		}
		imageFileDescriptor.WeakDependency = imageFileDescriptor.WeakDependency[:i]

		if _, ok := closure.completeFiles[imageFile.Path()]; !ok {
			// if not keeping entire file, filter contents now
			basePath = basePath[:0]
			imageFileDescriptor.MessageType = trimMessageDescriptors(imageFileDescriptor.MessageType, closure.elements, sourcePathRemapper, append(basePath, fileMessagesTag))
			imageFileDescriptor.EnumType = trimSlice(imageFileDescriptor.EnumType, closure.elements, sourcePathRemapper, append(basePath, fileEnumsTag))
			// TODO: We could end up removing all extensions from a particular extend block
			// but we then don't mark that extend block's source code info for deletion. This
			// is because extend blocks don't have distinct paths -- we have to actually look
			// at the span information to determine which extensions correspond to which blocks
			// to decide which blocks to remove. That is possible, but non-trivial, and it's
			// unclear if the "juice is worth the squeeze", so we leave it. The best we do is
			// to remove comments for extend blocks when there are NO extensions.
			extsPath := append(basePath, fileExtensionsTag)
			imageFileDescriptor.Extension = trimSlice(imageFileDescriptor.Extension, closure.elements, sourcePathRemapper, extsPath)
			if len(imageFileDescriptor.Extension) == 0 {
				sourcePathRemapper.markDeleted(extsPath)
			}
			svcsPath := append(basePath, fileServicesTag)
			// We must iterate through the services *before* we trim the slice. That way the
			// index we see is for the "old path", which we need to know to mark elements as
			// moved or deleted with the sourcePathRemapper.
			for index, serviceDescriptor := range imageFileDescriptor.Service {
				if _, ok := closure.elements[serviceDescriptor]; !ok {
					continue
				}
				methodPath := append(svcsPath, int32(index), serviceMethodsTag)
				serviceDescriptor.Method = trimSlice(serviceDescriptor.Method, closure.elements, sourcePathRemapper, methodPath)
			}
			imageFileDescriptor.Service = trimSlice(imageFileDescriptor.Service, closure.elements, sourcePathRemapper, svcsPath)
		}

		if len(imageFileDescriptor.SourceCodeInfo.GetLocation()) > 0 {
			// Now the sourcePathRemapper has been fully populated for all of the deletions
			// and moves above. So we can use it to reconstruct the source code info slice
			// of locations.
			i := 0
			for _, location := range imageFileDescriptor.SourceCodeInfo.Location {
				// This function returns newPath==nil if the element at the given path
				// was marked for deletion (so this location should be omitted).
				newPath, noComment := sourcePathRemapper.newPath(location.Path)
				if newPath != nil {
					imageFileDescriptor.SourceCodeInfo.Location[i] = location
					location.Path = newPath
					if noComment {
						location.LeadingDetachedComments = nil
						location.LeadingComments = nil
						location.TrailingComments = nil
					}
					i++
				}
			}
			imageFileDescriptor.SourceCodeInfo.Location = imageFileDescriptor.SourceCodeInfo.Location[:i]
		}
	}
	return bufimage.NewImage(includedFiles)
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

// trimMessageDescriptors removes (nested) messages and nested enums from a slice
// of message descriptors if their type names are not found in the toKeep map.
func trimMessageDescriptors(
	in []*descriptorpb.DescriptorProto,
	toKeep map[namedDescriptor]closureInclusionMode,
	sourcePathRemapper *sourcePathsRemapTrie,
	pathSoFar []int32,
) []*descriptorpb.DescriptorProto {
	// We must iterate through the messages *before* we trim the slice. That way the
	// index we see is for the "old path", which we need to know to mark elements as
	// moved or deleted with the sourcePathRemapper.
	for index, messageDescriptor := range in {
		path := append(pathSoFar, int32(index))
		mode, ok := toKeep[messageDescriptor]
		if !ok {
			continue
		}
		if mode == inclusionModeEnclosing {
			// if this is just an enclosing element, we only care about it as a namespace for
			// other types and don't care about the rest of its contents
			messageDescriptor.Field = nil
			messageDescriptor.OneofDecl = nil
			messageDescriptor.ExtensionRange = nil
			messageDescriptor.ReservedRange = nil
			messageDescriptor.ReservedName = nil
			sourcePathRemapper.markNoComment(path)
			sourcePathRemapper.markDeleted(append(path, messageFieldsTag))
			sourcePathRemapper.markDeleted(append(path, messageOneofsTag))
			sourcePathRemapper.markDeleted(append(path, messageExtensionRangesTag))
			sourcePathRemapper.markDeleted(append(path, messageReservedRangesTag))
			sourcePathRemapper.markDeleted(append(path, messageReservedNamesTag))
		}
		messageDescriptor.NestedType = trimMessageDescriptors(messageDescriptor.NestedType, toKeep, sourcePathRemapper, append(path, messageNestedMessagesTag))
		messageDescriptor.EnumType = trimSlice(messageDescriptor.EnumType, toKeep, sourcePathRemapper, append(path, messageEnumsTag))
		// TODO: We could end up removing all extensions from a particular extend block
		// but we then don't mark that extend block's source code info for deletion. The
		// best we do is to remove comments for extend blocks when there are NO extensions.
		// See comment above for file extensions for more info.
		extsPath := append(path, messageExtensionsTag)
		messageDescriptor.Extension = trimSlice(messageDescriptor.Extension, toKeep, sourcePathRemapper, extsPath)
		if len(messageDescriptor.Extension) == 0 {
			sourcePathRemapper.markDeleted(extsPath)
		}
	}
	return trimSlice(in, toKeep, sourcePathRemapper, pathSoFar)
}

// trimSlice removes elements from a slice of descriptors if they are
// not present in the given map.
func trimSlice[T namedDescriptor](
	in []T,
	toKeep map[namedDescriptor]closureInclusionMode,
	sourcePathRemapper *sourcePathsRemapTrie,
	pathSoFar []int32,
) []T {
	i := 0
	for index, descriptor := range in {
		path := append(pathSoFar, int32(index))
		if _, ok := toKeep[descriptor]; ok {
			sourcePathRemapper.markMoved(path, int32(i))
			in[i] = descriptor
			i++
		} else {
			sourcePathRemapper.markDeleted(path)
		}
	}
	return in[:i]
}

// transitiveClosure accumulates the elements, files, and needed imports for a
// subset of an image. When an element is added to the closure, all of its
// dependencies are recursively added.
type transitiveClosure struct {
	// The elements included in the transitive closure.
	elements map[namedDescriptor]closureInclusionMode
	// The set of files that contain all items in elements.
	files map[string]struct{}
	// Any files that are part of the closure in their entirety (due to an
	// entire package being included). The above fields are used to filter the
	// contents of files. But files named in this set will not be filtered.
	completeFiles map[string]struct{}
	// The set of imports for each file. This allows for re-writing imports
	// for files whose contents have been pruned.
	imports map[string]map[string]struct{}
}

type closureInclusionMode int

const (
	// Element is included in closure because it is directly reachable from a root.
	inclusionModeExplicit = closureInclusionMode(iota)
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
		elements:      map[namedDescriptor]closureInclusionMode{},
		files:         map[string]struct{}{},
		completeFiles: map[string]struct{}{},
		imports:       map[string]map[string]struct{}{},
	}
}

func (t *transitiveClosure) addImport(fromPath, toPath string) {
	if fromPath == toPath {
		return // no need for a file to import itself
	}
	imps := t.imports[fromPath]
	if imps == nil {
		imps = map[string]struct{}{}
		t.imports[fromPath] = imps
	}
	imps[toPath] = struct{}{}
}

func (t *transitiveClosure) addFile(file string, imageIndex *imageIndex, opts *imageFilterOptions) error {
	if _, ok := t.files[file]; ok {
		return nil // already added
	}
	t.files[file] = struct{}{}
	return t.exploreCustomOptions(imageIndex.Files[file], file, imageIndex, opts)
}

func (t *transitiveClosure) addPackage(
	pkg *protoPackage,
	imageIndex *imageIndex,
	opts *imageFilterOptions,
) error {
	for _, file := range pkg.files {
		if err := t.addFile(file.Path(), imageIndex, opts); err != nil {
			return err
		}
		t.completeFiles[file.Path()] = struct{}{}
	}
	for _, descriptor := range pkg.elements {
		if err := t.addElement(descriptor, "", false, imageIndex, opts); err != nil {
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
	if err := t.addFile(descriptorInfo.file, imageIndex, opts); err != nil {
		return err
	}
	if referrerFile != "" {
		t.addImport(referrerFile, descriptorInfo.file)
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
	if err := t.addEnclosing(descriptorInfo.parent, descriptorInfo.file, imageIndex, opts); err != nil {
		return err
	}
	// add any custom options and their dependencies
	if err := t.exploreCustomOptions(descriptor, descriptorInfo.file, imageIndex, opts); err != nil {
		return err
	}

	switch typedDescriptor := descriptor.(type) {
	case *descriptorpb.DescriptorProto:
		// Options and types for all fields
		for _, field := range typedDescriptor.GetField() {
			if err := t.addFieldType(field, descriptorInfo.file, imageIndex, opts); err != nil {
				return err
			}
			if err := t.exploreCustomOptions(field, referrerFile, imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all oneofs in this message
		for _, oneOfDescriptor := range typedDescriptor.GetOneofDecl() {
			if err := t.exploreCustomOptions(oneOfDescriptor, descriptorInfo.file, imageIndex, opts); err != nil {
				return err
			}
		}
		// Options for all extension ranges in this message
		for _, extRange := range typedDescriptor.GetExtensionRange() {
			if err := t.exploreCustomOptions(extRange, descriptorInfo.file, imageIndex, opts); err != nil {
				return err
			}
		}

	case *descriptorpb.EnumDescriptorProto:
		for _, enumValue := range typedDescriptor.GetValue() {
			if err := t.exploreCustomOptions(enumValue, descriptorInfo.file, imageIndex, opts); err != nil {
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
		inputName := strings.TrimPrefix(typedDescriptor.GetInputType(), ".")
		inputDescriptor, ok := imageIndex.ByName[inputName]
		if !ok {
			return fmt.Errorf("missing %q", inputName)
		}
		if err := t.addElement(inputDescriptor, descriptorInfo.file, false, imageIndex, opts); err != nil {
			return err
		}

		outputName := strings.TrimPrefix(typedDescriptor.GetOutputType(), ".")
		outputDescriptor, ok := imageIndex.ByName[outputName]
		if !ok {
			return fmt.Errorf("missing %q", outputName)
		}
		if err := t.addElement(outputDescriptor, descriptorInfo.file, false, imageIndex, opts); err != nil {
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
		extendeeName := strings.TrimPrefix(typedDescriptor.GetExtendee(), ".")
		extendeeDescriptor, ok := imageIndex.ByName[extendeeName]
		if !ok {
			return fmt.Errorf("missing %q", extendeeName)
		}
		if err := t.addElement(extendeeDescriptor, descriptorInfo.file, impliedByCustomOption, imageIndex, opts); err != nil {
			return err
		}
		if err := t.addFieldType(typedDescriptor, descriptorInfo.file, imageIndex, opts); err != nil {
			return err
		}

	default:
		return errorUnsupportedFilterType(descriptor, descriptorInfo.fullName)
	}

	return nil
}

func errorUnsupportedFilterType(descriptor namedDescriptor, fullName string) error {
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
		if !isMsg && !isSvc {
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
		typeName := strings.TrimPrefix(field.GetTypeName(), ".")
		typeDescriptor, ok := imageIndex.ByName[typeName]
		if !ok {
			return fmt.Errorf("missing %q", typeName)
		}
		err := t.addElement(typeDescriptor, referrerFile, false, imageIndex, opts)
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
		msgDescriptor, ok := e.(*descriptorpb.DescriptorProto)
		if !ok {
			// not a message, nothing to do
			continue
		}
		descriptorInfo := imageIndex.ByDescriptor[msgDescriptor]
		for _, extendsDescriptor := range imageIndex.NameToExtensions[descriptorInfo.fullName] {
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

	optionsName := string(options.Descriptor().FullName())
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
		msgType := typeURL[pos+1:]
		d, _ := imageIndex.ByName[msgType].(*descriptorpb.DescriptorProto)
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
}

func newImageFilterOptions() *imageFilterOptions {
	return &imageFilterOptions{
		includeCustomOptions:   true,
		includeKnownExtensions: true,
		allowImportedTypes:     false,
	}
}

func stripSourceRetentionOptionsFromFile(imageFile bufimage.ImageFile) (bufimage.ImageFile, error) {
	updatedFileDescriptor, err := protopluginutil.StripSourceRetentionOptions(imageFile.FileDescriptorProto())
	if err != nil {
		return nil, err
	}
	return bufimage.NewImageFile(
		updatedFileDescriptor,
		imageFile.ModuleFullName(),
		imageFile.CommitID(),
		imageFile.ExternalPath(),
		imageFile.LocalPath(),
		imageFile.IsImport(),
		imageFile.IsSyntaxUnspecified(),
		imageFile.UnusedDependencyIndexes(),
	)
}
