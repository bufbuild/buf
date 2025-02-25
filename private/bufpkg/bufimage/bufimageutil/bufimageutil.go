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

	includeTypes   map[string]struct{}
	excludeTypes   map[string]struct{}
	includeOptions map[string]struct{}
	excludeOptions map[string]struct{}
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
