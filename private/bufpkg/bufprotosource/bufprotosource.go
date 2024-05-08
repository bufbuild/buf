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

// Package bufprotosource defines minimal interfaces for Protobuf descriptor types.
//
// This is done so that the backing package can be swapped out easily.
//
// All values that return SourceLocation can be nil.
//
// Testing is currently implicitly done through the bufcheck packages, however
// if this were to be split out into a separate library, it would need a separate
// testing suite.
package bufprotosource

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	// SyntaxUnspecified represents no syntax being specified.
	//
	// This is functionally equivalent to SyntaxProto2.
	SyntaxUnspecified Syntax = iota + 1
	// SyntaxProto2 represents the proto2 syntax.
	SyntaxProto2
	// SyntaxProto3 represents the proto3 syntax.
	SyntaxProto3
	// SyntaxEditions represents the editions syntax.
	SyntaxEditions
)

// Syntax is the syntax of a file.
type Syntax int

// String returns the string representation of s
func (s Syntax) String() string {
	switch s {
	case SyntaxUnspecified:
		return "unspecified"
	case SyntaxProto2:
		return "proto2"
	case SyntaxProto3:
		return "proto3"
	case SyntaxEditions:
		return "editions"
	default:
		return strconv.Itoa(int(s))
	}
}

// Descriptor is the base interface for a descriptor type.
type Descriptor interface {
	// File returns the associated File.
	//
	// Always non-nil.
	File() File
}

// LocationDescriptor is the base interface for a descriptor type with a location.
type LocationDescriptor interface {
	Descriptor

	// Location returns the location of the entire descriptor.
	//
	// Can return nil, although will generally not be nil.
	Location() Location
}

// NamedDescriptor is the base interface for a named descriptor type.
type NamedDescriptor interface {
	LocationDescriptor

	// FullName returns the fully-qualified name, i.e. some.pkg.Nested.ParentMessage.FooEnum.ENUM_VALUE.
	//
	// Always non-empty.
	FullName() string
	// NestedName returns the full nested name without the package, i.e. Nested.ParentMessage.FooEnum
	// or Nested.ParentMessage.FooEnum.ENUM_VALUE.
	//
	// Always non-empty.
	NestedName() string
	// Name returns the short name, or the name of a value or field, i.e. FooEnum or ENUM_VALUE.
	//
	// Always non-empty.
	Name() string
	// NameLocation returns the location of the name of the descriptor.
	//
	// If the backing descriptor does not have name-level resolution, this will
	// attempt to return a location of the entire descriptor.
	//
	// If the backing descriptor has comments for the entire descriptor, these
	// will be added to the named location.
	//
	// Can return nil.
	NameLocation() Location
}

// ContainerDescriptor contains Enums and Messages.
type ContainerDescriptor interface {
	Enums() []Enum
	Messages() []Message
	Extensions() []Field
}

// OptionExtensionDescriptor contains options and option extensions.
type OptionExtensionDescriptor interface {
	// Features returns information about any features present in the
	// options. It only provides information about standard features,
	// not custom features (i.e. extensions of google.protobuf.FeatureSet).
	//
	// Never returns nil.
	Features() FeaturesDescriptor

	// OptionExtension returns the value for an options extension field.
	//
	// Returns false if the extension is not set.
	//
	// See https://pkg.go.dev/google.golang.org/protobuf/proto#HasExtension
	// See https://pkg.go.dev/google.golang.org/protobuf/proto#GetExtension
	OptionExtension(extensionType protoreflect.ExtensionType) (interface{}, bool)

	// OptionExtensionLocation returns the source location where the given extension
	// field value is defined. This is the same as OptionLocation, but specific to
	// extension fields.
	OptionExtensionLocation(extensionType protoreflect.ExtensionType, extraPath ...int32) Location
	// TODO: Should we just delete OptionExtensionLocation?

	// OptionLocation returns the source location where the given option field
	// value is defined. The extra path can be additional path elements, for getting
	// getting the location of specific elements inside the field, for message
	// and repeated values.
	//
	// If a precise location cannot be found, but a general one can be, the general
	// location will be returned. For example, if a specific field inside a message
	// field is requested but the source code info only includes information
	// about the message itself (and not that particular field), the location of the
	// message value is returned. Conversely, if a message location is requested but
	// the source code info only has information about specific fields inside that
	// message, the first such location is returned. Similarly, if multiple locations
	// are in source code info for the requested value, the first one is returned.
	//
	// If no relevant location is found in source code info, this returns nil.
	OptionLocation(field protoreflect.FieldDescriptor, extraPath ...int32) Location

	// PresentExtensionNumbers returns field numbers for all extensions/custom options
	// that have a set value on this descriptor.
	PresentExtensionNumbers() []int32

	// ForEachPresentOption iterates through all options that have a set value on this
	// descriptor, invoking fn for each present option.
	//
	// If fn returns false, the iteration is terminated and ForEachPresentOption
	// immediately returns.
	ForEachPresentOption(fn func(protoreflect.FieldDescriptor, protoreflect.Value) bool)
}

// FeaturesDescriptor contains information about features, which are
// special options in Protobuf Editions.
type FeaturesDescriptor interface {
	// FieldPresenceLocation returns the location for the field_presence
	// feature, if it is present in the options of the containing element.
	FieldPresenceLocation() Location
	// EnumTypeLocation returns the location for the enum_type
	// feature, if it is present in the options of the containing element.
	EnumTypeLocation() Location
	// RepeatedFieldEncodingLocation returns the location for the repeated_field_encoding
	// feature, if it is present in the options of the containing element.
	RepeatedFieldEncodingLocation() Location
	// UTF8ValidationLocation returns the location for the utf8_validation
	// feature, if it is present in the options of the containing element.
	UTF8ValidationLocation() Location
	// MessageEncodingLocation returns the location for the message_encoding
	// feature, if it is present in the options of the containing element.
	MessageEncodingLocation() Location
	// JSONFormatLocation returns the location for the json_format
	// feature, if it is present in the options of the containing element.
	JSONFormatLocation() Location
}

// Location defines source code info location information.
//
// May be extended in the future to include comments.
//
// Note that unlike SourceCodeInfo_Location, these are not zero-indexed.
type Location interface {
	StartLine() int
	StartColumn() int
	EndLine() int
	EndColumn() int
	LeadingComments() string
	TrailingComments() string
	// NOT a copy. Do not modify.
	LeadingDetachedComments() []string
}

// ModuleFullName is a module full name.
type ModuleFullName interface {
	Registry() string
	Owner() string
	Name() string
}

// FileInfo contains Protobuf file info.
type FileInfo interface {
	// Path is the path of the file relative to the root it is contained within.
	// This will be normalized, validated and never empty,
	// This will be unique within a given Image.
	Path() string
	// ExternalPath returns the path that identifies this file externally.
	//
	// This will be unnormalized.
	// Never empty. Falls back to Path if there is not an external path.
	//
	// Example:
	//	 Assume we had the input path /foo/bar which is a local directory.

	//   Path: one/one.proto
	//   RootDirPath: proto
	//   ExternalPath: /foo/bar/proto/one/one.proto
	ExternalPath() string
	// ModuleFullName is the module that this file came from.
	//
	// Note this *can* be nil if we did not build from a named module.
	// All code must assume this can be nil.
	// Note that nil checking should work since the backing type is always a pointer.
	ModuleFullName() bufmodule.ModuleFullName
	// CommitID is the commit for the module that this file came from.
	//
	// This will only be set if ModuleFullName is set, but may not be set
	// even if ModuleFullName is set, that is commit is optional information
	// even if we know what module this file came from.
	CommitID() uuid.UUID
	// IsImport returns true if this file is an import.
	IsImport() bool
}

// File is a file descriptor.
type File interface {
	Descriptor
	FileInfo

	// Top-level only.
	ContainerDescriptor
	OptionExtensionDescriptor

	Syntax() Syntax
	Package() string
	FileImports() []FileImport
	Services() []Service
	Extensions() []Field
	Edition() descriptorpb.Edition

	CsharpNamespace() string
	GoPackage() string
	JavaMultipleFiles() bool
	JavaOuterClassname() string
	JavaPackage() string
	JavaStringCheckUtf8() bool
	ObjcClassPrefix() string
	PhpClassPrefix() string
	PhpNamespace() string
	PhpMetadataNamespace() string
	RubyPackage() string
	SwiftPrefix() string
	Deprecated() bool

	OptimizeFor() descriptorpb.FileOptions_OptimizeMode
	CcGenericServices() bool
	JavaGenericServices() bool
	PyGenericServices() bool
	CcEnableArenas() bool

	SyntaxLocation() Location
	PackageLocation() Location
	CsharpNamespaceLocation() Location
	GoPackageLocation() Location
	JavaMultipleFilesLocation() Location
	JavaOuterClassnameLocation() Location
	JavaPackageLocation() Location
	JavaStringCheckUtf8Location() Location
	ObjcClassPrefixLocation() Location
	PhpClassPrefixLocation() Location
	PhpNamespaceLocation() Location
	PhpMetadataNamespaceLocation() Location
	RubyPackageLocation() Location
	SwiftPrefixLocation() Location

	OptimizeForLocation() Location
	CcGenericServicesLocation() Location
	JavaGenericServicesLocation() Location
	PyGenericServicesLocation() Location
	CcEnableArenasLocation() Location

	// FileDescriptor returns the backing FileDescriptor for this File.
	//
	// Users should prefer to use the core protosource API to read properties of the File as opposed
	// to using the FileDescriptor directly, however we needed to add this to be able to build
	// Resolvers, and did not want to rewrite the whole API.
	FileDescriptor() protodescriptor.FileDescriptor
}

// FileImport is a file import descriptor.
type FileImport interface {
	LocationDescriptor

	Import() string
	IsPublic() bool
	IsWeak() bool
	IsUnused() bool
}

// TagRange is a tag range from start to end.
type TagRange interface {
	LocationDescriptor

	// Start is the start of the range.
	Start() int
	// End is the end of the range.
	// Inclusive.
	End() int
	// Max says that the End is the max.
	Max() bool
}

// ReservedName is a reserved name for an enum or message.
type ReservedName interface {
	LocationDescriptor

	Value() string
}

// ReservedDescriptor has reserved ranges and names.
type ReservedDescriptor interface {
	ReservedTagRanges() []TagRange
	ReservedNames() []ReservedName
}

// EnumRange is a TagRange for Enums.
type EnumRange interface {
	TagRange

	Enum() Enum
}

// MessageRange is a TagRange for Messages.
type MessageRange interface {
	TagRange

	Message() Message
}

// ExtensionRange represents an extension range in Messages.
type ExtensionRange interface {
	MessageRange
	OptionExtensionDescriptor
}

// Enum is an enum descriptor.
type Enum interface {
	NamedDescriptor
	ReservedDescriptor
	OptionExtensionDescriptor

	Values() []EnumValue
	ReservedEnumRanges() []EnumRange

	AllowAlias() bool
	DeprecatedLegacyJSONFieldConflicts() bool
	Deprecated() bool
	AllowAliasLocation() Location

	// Will return nil if this is a top-level Enum
	Parent() Message

	// AsDescriptor returns a [protoreflect.Descriptor] that
	// corresponds to this message. This should only be needed
	// for reflection usages.
	AsDescriptor() (protoreflect.EnumDescriptor, error)
}

// EnumValue is an enum value descriptor.
type EnumValue interface {
	NamedDescriptor
	OptionExtensionDescriptor

	Enum() Enum
	Number() int

	Deprecated() bool
	NumberLocation() Location
}

// Message is a message descriptor.
type Message interface {
	NamedDescriptor
	// Only those directly nested under this message.
	ContainerDescriptor
	ReservedDescriptor
	OptionExtensionDescriptor

	// Includes fields in oneofs.
	Fields() []Field
	Extensions() []Field
	Oneofs() []Oneof
	ExtensionRanges() []ExtensionRange
	ExtensionMessageRanges() []MessageRange
	ReservedMessageRanges() []MessageRange

	// Will return nil if this is a top-level message
	Parent() Message
	IsMapEntry() bool

	MessageSetWireFormat() bool
	NoStandardDescriptorAccessor() bool
	DeprecatedLegacyJSONFieldConflicts() bool
	Deprecated() bool
	MessageSetWireFormatLocation() Location
	NoStandardDescriptorAccessorLocation() Location

	// AsDescriptor returns a [protoreflect.Descriptor] that
	// corresponds to this message. This should only be needed
	// for reflection usages.
	AsDescriptor() (protoreflect.MessageDescriptor, error)
}

// Field is a field descriptor.
type Field interface {
	NamedDescriptor
	OptionExtensionDescriptor

	// May be nil if this is attached to a file.
	ParentMessage() Message
	Number() int
	Label() descriptorpb.FieldDescriptorProto_Label
	Type() descriptorpb.FieldDescriptorProto_Type
	TypeName() string
	// may be nil
	Oneof() Oneof
	Proto3Optional() bool
	JSONName() string
	JSType() descriptorpb.FieldOptions_JSType
	CType() descriptorpb.FieldOptions_CType
	Retention() descriptorpb.FieldOptions_OptionRetention
	Targets() []descriptorpb.FieldOptions_OptionTargetType
	DebugRedact() bool
	// Set vs unset matters for packed
	// See the comments on descriptor.proto
	Packed() *bool
	Deprecated() bool
	// Default is the field's default value, encoded as a string.
	// Instead of trying to interpret or decode this string, it is
	// typically better to instead use AsDescriptor() and query the
	// Default() method of the resulting protoreflect.FieldDescriptor.
	// If empty, the default value is a zero value for the field's
	// type. Defaults cannot be set for repeated or message fields
	// (which also means it cannot be set for map fields).
	Default() string
	// Empty string unless the field is part of an extension
	Extendee() string

	NumberLocation() Location
	TypeLocation() Location
	TypeNameLocation() Location
	JSONNameLocation() Location
	JSTypeLocation() Location
	CTypeLocation() Location
	PackedLocation() Location
	DefaultLocation() Location
	ExtendeeLocation() Location

	// AsDescriptor returns a [protoreflect.Descriptor] that
	// corresponds to this message. This should only be needed
	// for reflection usages.
	AsDescriptor() (protoreflect.FieldDescriptor, error)
}

// Oneof is a oneof descriptor.
type Oneof interface {
	NamedDescriptor
	OptionExtensionDescriptor

	Message() Message
	Fields() []Field

	// AsDescriptor returns a [protoreflect.Descriptor] that
	// corresponds to this message. This should only be needed
	// for reflection usages.
	AsDescriptor() (protoreflect.OneofDescriptor, error)
}

// Service is a service descriptor.
type Service interface {
	NamedDescriptor
	OptionExtensionDescriptor

	Methods() []Method
	Deprecated() bool
}

// Method is a method descriptor.
type Method interface {
	NamedDescriptor
	OptionExtensionDescriptor

	Service() Service
	InputTypeName() string
	OutputTypeName() string
	ClientStreaming() bool
	ServerStreaming() bool
	InputTypeLocation() Location
	OutputTypeLocation() Location

	Deprecated() bool
	IdempotencyLevel() descriptorpb.MethodOptions_IdempotencyLevel
	IdempotencyLevelLocation() Location
}

// NewFiles converts the input Image into Files.
func NewFiles(ctx context.Context, image bufimage.Image) ([]File, error) {
	return newFiles(ctx, image)
}

// SortFiles sorts the Files by FilePath.
func SortFiles(files []File) {
	sort.Slice(files, func(i int, j int) bool { return files[i].Path() < files[j].Path() })
}

// FilePathToFile maps the Files to a map from Path() to File.
//
// Returns error if file paths are not unique.
func FilePathToFile(files ...File) (map[string]File, error) {
	filePathToFile := make(map[string]File, len(files))
	for _, file := range files {
		filePath := file.Path()
		if _, ok := filePathToFile[filePath]; ok {
			return nil, fmt.Errorf("duplicate filePath: %q", filePath)
		}
		filePathToFile[filePath] = file
	}
	return filePathToFile, nil
}

// DirPathToFiles maps the Files to a map from directory
// to the slice of Files in that directory.
//
// Returns error if file paths are not unique.
// Directories are normalized.
//
// Files will be sorted by FilePath.
func DirPathToFiles(files ...File) (map[string][]File, error) {
	return mapFiles(files, func(file File) string { return normalpath.Dir(file.Path()) })
}

// PackageToFiles maps the Files to a map from Protobuf package
// to the slice of Files in that package.
//
// Returns error if file paths are not unique.
//
// Files will be sorted by Path.
func PackageToFiles(files ...File) (map[string][]File, error) {
	// works for no package since "" is a valid map key
	return mapFiles(files, File.Package)
}

// ForEachEnum calls f on each Enum in the given ContainerDescriptor, including nested Enums.
//
// Returns error and stops iterating if f returns error.
// Never returns error unless f returns error.
func ForEachEnum(f func(Enum) error, containerDescriptor ContainerDescriptor) error {
	for _, enum := range containerDescriptor.Enums() {
		if err := f(enum); err != nil {
			return err
		}
	}
	for _, message := range containerDescriptor.Messages() {
		if err := ForEachEnum(f, message); err != nil {
			return err
		}
	}
	return nil
}

// ForEachExtension calls f on each extension Field in the given ContainerDescriptor,
// including nested extensions.
//
// Returns error and stops iterating if f returns error.
// Never returns error unless f returns error.
func ForEachExtension(f func(Field) error, containerDescriptor ContainerDescriptor) error {
	for _, extension := range containerDescriptor.Extensions() {
		if err := f(extension); err != nil {
			return err
		}
	}
	for _, message := range containerDescriptor.Messages() {
		if err := ForEachExtension(f, message); err != nil {
			return err
		}
	}
	return nil
}

// ForEachMessage calls f on each Message in the given ContainerDescriptor, including nested Messages.
//
// Returns error and stops iterating if f returns error
// Never returns error unless f returns error.
func ForEachMessage(f func(Message) error, containerDescriptor ContainerDescriptor) error {
	for _, message := range containerDescriptor.Messages() {
		if err := f(message); err != nil {
			return err
		}
		if err := ForEachMessage(f, message); err != nil {
			return err
		}
	}
	return nil
}

// NestedNameToEnum maps the Enums in the ContainerDescriptor to a map from
// nested name to Enum.
//
// Returns error if Enums do not have unique nested names within the ContainerDescriptor,
// which should generally never happen for properly-formed ContainerDescriptors.
func NestedNameToEnum(containerDescriptor ContainerDescriptor) (map[string]Enum, error) {
	nestedNameToEnum := make(map[string]Enum)
	if err := ForEachEnum(
		func(enum Enum) error {
			nestedName := enum.NestedName()
			if _, ok := nestedNameToEnum[nestedName]; ok {
				return fmt.Errorf("duplicate enum: %q", nestedName)
			}
			nestedNameToEnum[nestedName] = enum
			return nil
		},
		containerDescriptor,
	); err != nil {
		return nil, err
	}
	return nestedNameToEnum, nil
}

// NestedNameToExtension maps the Enums in the ContainerDescriptor to a map from
// nested name to extension Field.
//
// Returns error if extensions do not have unique nested names within the
// ContainerDescriptor, which should generally never happen for properly-formed
// ContainerDescriptors.
func NestedNameToExtension(containerDescriptor ContainerDescriptor) (map[string]Field, error) {
	nestedNameToExtension := make(map[string]Field)
	if err := ForEachExtension(
		func(extension Field) error {
			nestedName := extension.NestedName()
			if _, ok := nestedNameToExtension[nestedName]; ok {
				return fmt.Errorf("duplicate extension: %q", nestedName)
			}
			nestedNameToExtension[nestedName] = extension
			return nil
		},
		containerDescriptor,
	); err != nil {
		return nil, err
	}
	return nestedNameToExtension, nil
}

// FullNameToEnum maps the Enums in the Files to a map from full name to enum.
//
// Returns error if the Enums do not have unique full names within the Files,
// which should generally never happen for properly-formed Files.
func FullNameToEnum(files ...File) (map[string]Enum, error) {
	fullNameToEnum := make(map[string]Enum)
	for _, file := range files {
		if err := ForEachEnum(
			func(enum Enum) error {
				fullName := enum.FullName()
				if _, ok := fullNameToEnum[fullName]; ok {
					return fmt.Errorf("duplicate enum: %q", fullName)
				}
				fullNameToEnum[fullName] = enum
				return nil
			},
			file,
		); err != nil {
			return nil, err
		}
	}
	return fullNameToEnum, nil
}

// PackageToNestedNameToEnum maps the Enums in the Files to a map from
// package to nested name to Enum.
//
// Returns error if the Enums do not have unique nested names within the packages,
// which should generally never happen for properly-formed Files.
func PackageToNestedNameToEnum(files ...File) (map[string]map[string]Enum, error) {
	packageToNestedNameToEnum := make(map[string]map[string]Enum)
	for _, file := range files {
		if err := ForEachEnum(
			func(enum Enum) error {
				pkg := enum.File().Package()
				nestedName := enum.NestedName()
				nestedNameToEnum, ok := packageToNestedNameToEnum[pkg]
				if !ok {
					nestedNameToEnum = make(map[string]Enum)
					packageToNestedNameToEnum[pkg] = nestedNameToEnum
				}
				if _, ok := nestedNameToEnum[nestedName]; ok {
					return fmt.Errorf("duplicate enum in package %q: %q", pkg, nestedName)
				}
				nestedNameToEnum[nestedName] = enum
				return nil
			},
			file,
		); err != nil {
			return nil, err
		}
	}
	return packageToNestedNameToEnum, nil
}

// PackageToNestedNameToExtension maps the extension Fields in the Files to a map
// from package to nested name to Field.
//
// Returns error if the extension do not have unique nested names within the packages,
// which should generally never happen for properly-formed Files.
func PackageToNestedNameToExtension(files ...File) (map[string]map[string]Field, error) {
	packageToNestedNameToExtension := make(map[string]map[string]Field)
	for _, file := range files {
		if err := ForEachExtension(
			func(enum Field) error {
				pkg := enum.File().Package()
				nestedName := enum.NestedName()
				nestedNameToExtension, ok := packageToNestedNameToExtension[pkg]
				if !ok {
					nestedNameToExtension = make(map[string]Field)
					packageToNestedNameToExtension[pkg] = nestedNameToExtension
				}
				if _, ok := nestedNameToExtension[nestedName]; ok {
					return fmt.Errorf("duplicate extension in package %q: %q", pkg, nestedName)
				}
				nestedNameToExtension[nestedName] = enum
				return nil
			},
			file,
		); err != nil {
			return nil, err
		}
	}
	return packageToNestedNameToExtension, nil
}

// NameToEnumValue maps the EnumValues in the Enum to a map from name to EnumValue.
//
// Returns error if the EnumValues do not have unique names within the Enum,
// which should generally never happen for properly-formed Enums.
func NameToEnumValue(enum Enum) (map[string]EnumValue, error) {
	nameToEnumValue := make(map[string]EnumValue)
	for _, enumValue := range enum.Values() {
		name := enumValue.Name()
		if _, ok := nameToEnumValue[name]; ok {
			return nil, fmt.Errorf("duplicate enum value name for enum %q: %q", enum.NestedName(), name)
		}
		nameToEnumValue[name] = enumValue
	}
	return nameToEnumValue, nil
}

// NumberToNameToEnumValue maps the EnumValues in the Enum to a map from number to name to EnumValue.
//
// Duplicates by number may occur if allow_alias = true.
//
// Returns error if the EnumValues do not have unique names within the Enum for a given number,
// which should generally never happen for properly-formed Enums.
func NumberToNameToEnumValue(enum Enum) (map[int]map[string]EnumValue, error) {
	numberToNameToEnumValue := make(map[int]map[string]EnumValue)
	for _, enumValue := range enum.Values() {
		number := enumValue.Number()
		nameToEnumValue, ok := numberToNameToEnumValue[number]
		if !ok {
			nameToEnumValue = make(map[string]EnumValue)
			numberToNameToEnumValue[number] = nameToEnumValue
		}
		name := enumValue.Name()
		if _, ok := nameToEnumValue[name]; ok {
			return nil, fmt.Errorf("duplicate enum value name for enum %q: %q", enum.NestedName(), name)
		}
		nameToEnumValue[name] = enumValue
	}
	return numberToNameToEnumValue, nil
}

// NestedNameToMessage maps the Messages in the ContainerDescriptor to a map from
// nested name to Message.
//
// Returns error if Messages do not have unique nested names within the ContainerDescriptor,
// which should generally never happen for properly-formed files.
func NestedNameToMessage(containerDescriptor ContainerDescriptor) (map[string]Message, error) {
	nestedNameToMessage := make(map[string]Message)
	if err := ForEachMessage(
		func(message Message) error {
			nestedName := message.NestedName()
			if _, ok := nestedNameToMessage[nestedName]; ok {
				return fmt.Errorf("duplicate message: %q", nestedName)
			}
			nestedNameToMessage[nestedName] = message
			return nil
		},
		containerDescriptor,
	); err != nil {
		return nil, err
	}
	return nestedNameToMessage, nil
}

// FullNameToMessage maps the Messages in the Files to a map from full name to message.
//
// Returns error if the Messages do not have unique full names within the Files,
// which should generally never happen for properly-formed Files.
func FullNameToMessage(files ...File) (map[string]Message, error) {
	fullNameToMessage := make(map[string]Message)
	for _, file := range files {
		if err := ForEachMessage(
			func(message Message) error {
				fullName := message.FullName()
				if _, ok := fullNameToMessage[fullName]; ok {
					return fmt.Errorf("duplicate message: %q", fullName)
				}
				fullNameToMessage[fullName] = message
				return nil
			},
			file,
		); err != nil {
			return nil, err
		}
	}
	return fullNameToMessage, nil
}

// PackageToNestedNameToMessage maps the Messages in the Files to a map from
// package to nested name to Message.
//
// Returns error if the Messages do not have unique nested names within the packages,
// which should generally never happen for properly-formed Files.
func PackageToNestedNameToMessage(files ...File) (map[string]map[string]Message, error) {
	packageToNestedNameToMessage := make(map[string]map[string]Message)
	for _, file := range files {
		if err := ForEachMessage(
			func(message Message) error {
				pkg := message.File().Package()
				nestedName := message.NestedName()
				nestedNameToMessage, ok := packageToNestedNameToMessage[pkg]
				if !ok {
					nestedNameToMessage = make(map[string]Message)
					packageToNestedNameToMessage[pkg] = nestedNameToMessage
				}
				if _, ok := nestedNameToMessage[nestedName]; ok {
					return fmt.Errorf("duplicate message in package %q: %q", pkg, nestedName)
				}
				nestedNameToMessage[nestedName] = message
				return nil
			},
			file,
		); err != nil {
			return nil, err
		}
	}
	return packageToNestedNameToMessage, nil
}

// NumberToMessageField maps the Fields in the Message to a map from number to Field.
//
// Does not includes extensions.
//
// Returns error if the Fields do not have unique numbers within the Message,
// which should generally never happen for properly-formed Messages.
func NumberToMessageField(message Message) (map[int]Field, error) {
	numberToMessageField := make(map[int]Field)
	for _, messageField := range message.Fields() {
		number := messageField.Number()
		if _, ok := numberToMessageField[number]; ok {
			return nil, fmt.Errorf("duplicate message field: %d", number)
		}
		numberToMessageField[number] = messageField
	}
	return numberToMessageField, nil
}

// NumberToMessageFieldForLabel maps the Fields with the given label in the message
// to a map from number to Field.
//
// Does not includes extensions.
//
// Returns error if the Fields do not have unique numbers within the Message,
// which should generally never happen for properly-formed Messages.
func NumberToMessageFieldForLabel(message Message, label descriptorpb.FieldDescriptorProto_Label) (map[int]Field, error) {
	numberToField, err := NumberToMessageField(message)
	if err != nil {
		return nil, err
	}
	for number, field := range numberToField {
		if field.Label() != label {
			delete(numberToField, number)
		}
	}
	return numberToField, nil
}

// NameToMessageOneof maps the Oneofs in the Message to a map from name to Oneof.
//
// Returns error if the Oneofs do not have unique names within the Message,
// which should generally never happen for properly-formed Messages.
func NameToMessageOneof(message Message) (map[string]Oneof, error) {
	nameToMessageOneof := make(map[string]Oneof)
	for _, messageOneof := range message.Oneofs() {
		name := messageOneof.Name()
		if _, ok := nameToMessageOneof[name]; ok {
			return nil, fmt.Errorf("duplicate message oneof: %q", name)
		}
		nameToMessageOneof[name] = messageOneof
	}
	return nameToMessageOneof, nil
}

// NameToService maps the Services in the File to a map from name to Service.
//
// Returns error if Services do not have unique names within the File, which should
// generally never happen for properly-formed Files.
func NameToService(file File) (map[string]Service, error) {
	nameToService := make(map[string]Service)
	for _, service := range file.Services() {
		name := service.Name()
		if _, ok := nameToService[name]; ok {
			return nil, fmt.Errorf("duplicate service: %q", name)
		}
		nameToService[name] = service
	}
	return nameToService, nil
}

// FullNameToService maps the Services in the Files to a map from full name to Service.
//
// Returns error if Services do not have unique full names within the Files, which should
// generally never happen for properly-formed Files.
func FullNameToService(files ...File) (map[string]Service, error) {
	fullNameToService := make(map[string]Service)
	for _, file := range files {
		for _, service := range file.Services() {
			fullName := service.FullName()
			if _, ok := fullNameToService[fullName]; ok {
				return nil, fmt.Errorf("duplicate service: %q", fullName)
			}
			fullNameToService[fullName] = service
		}
	}
	return fullNameToService, nil
}

// PackageToNameToService maps the Services in the Files to a map from
// package to name to Service.
//
// Returns error if the Services do not have unique names within the packages,
// which should generally never happen for properly-formed Files.
func PackageToNameToService(files ...File) (map[string]map[string]Service, error) {
	packageToNameToService := make(map[string]map[string]Service)
	for _, file := range files {
		pkg := file.Package()
		nameToService, ok := packageToNameToService[pkg]
		if !ok {
			nameToService = make(map[string]Service)
			packageToNameToService[pkg] = nameToService
		}
		for _, service := range file.Services() {
			name := service.Name()
			if _, ok := nameToService[name]; ok {
				return nil, fmt.Errorf("duplicate service in package %q: %q", pkg, name)
			}
			nameToService[name] = service
		}
	}
	return packageToNameToService, nil
}

// PackageToDirectlyImportedPackageToFileImports maps packages to directly imported packages
// to the FileImports that import this package.
//
// For example, if package a imports package b via c/d.proto and c/e.proto, this will have
// a -> b -> [c/d.proto, c/e.proto].
//
// A directly imported package will not be equal to the package, i.e. there will be no a -> a.
//
// Files with no packages are included with key "" to be consistent with other functions.
func PackageToDirectlyImportedPackageToFileImports(files ...File) (map[string]map[string][]FileImport, error) {
	filePathToFile, err := FilePathToFile(files...)
	if err != nil {
		return nil, err
	}
	packageToDirectlyImportedPackageToFileImports := make(map[string]map[string][]FileImport)
	for _, file := range files {
		pkg := file.Package()
		directlyImportedPackageToFileImports, ok := packageToDirectlyImportedPackageToFileImports[pkg]
		if !ok {
			directlyImportedPackageToFileImports = make(map[string][]FileImport)
			packageToDirectlyImportedPackageToFileImports[pkg] = directlyImportedPackageToFileImports
		}
		for _, fileImport := range file.FileImports() {
			if importedFile, ok := filePathToFile[fileImport.Import()]; ok {
				importedPkg := importedFile.Package()
				if importedPkg != pkg {
					directlyImportedPackageToFileImports[importedFile.Package()] = append(
						directlyImportedPackageToFileImports[importedPkg],
						fileImport,
					)
				}
			}
		}
	}
	return packageToDirectlyImportedPackageToFileImports, nil
}

// NameToMethod maps the Methods in the Service to a map from name to Method.
//
// Returns error if Methods do not have unique names within the Service, which should
// generally never happen for properly-formed Services.
func NameToMethod(service Service) (map[string]Method, error) {
	nameToMethod := make(map[string]Method)
	for _, method := range service.Methods() {
		name := method.Name()
		if _, ok := nameToMethod[name]; ok {
			return nil, fmt.Errorf("duplicate method: %q", name)
		}
		nameToMethod[name] = method
	}
	return nameToMethod, nil
}

// FullNameToMethod maps the Methods in the Files to a map from full name to Method.
//
// Returns error if Methods do not have unique full names within the Files, which should
// generally never happen for properly-formed Files.
func FullNameToMethod(files ...File) (map[string]Method, error) {
	fullNameToMethod := make(map[string]Method)
	for _, file := range files {
		for _, service := range file.Services() {
			for _, method := range service.Methods() {
				fullName := method.FullName()
				if _, ok := fullNameToMethod[fullName]; ok {
					return nil, fmt.Errorf("duplicate method: %q", fullName)
				}
				fullNameToMethod[fullName] = method
			}
		}
	}
	return fullNameToMethod, nil
}

// StringToReservedTagRange maps the ReservedTagRanges in the ReservedDescriptor to a map
// from string string to reserved TagRange.
//
// Ignores duplicates.
func StringToReservedTagRange(reservedDescriptor ReservedDescriptor) map[string]TagRange {
	stringToReservedTagRange := make(map[string]TagRange)
	for _, reservedTagRange := range reservedDescriptor.ReservedTagRanges() {
		stringToReservedTagRange[TagRangeString(reservedTagRange)] = reservedTagRange
	}
	return stringToReservedTagRange
}

// ValueToReservedName maps the ReservedNames in the ReservedDescriptor to a map
// from string value to ReservedName.
//
// Ignores duplicates.
func ValueToReservedName(reservedDescriptor ReservedDescriptor) map[string]ReservedName {
	valueToReservedName := make(map[string]ReservedName)
	for _, reservedName := range reservedDescriptor.ReservedNames() {
		valueToReservedName[reservedName.Value()] = reservedName
	}
	return valueToReservedName
}

// StringToExtensionMessageRange maps the ExtensionMessageRanges in the Message to a map
// from string string to ExtensionMessageRange.
//
// Ignores duplicates.
func StringToExtensionMessageRange(message Message) map[string]MessageRange {
	stringToExtensionMessageRange := make(map[string]MessageRange)
	for _, extensionMessageRange := range message.ExtensionMessageRanges() {
		stringToExtensionMessageRange[TagRangeString(extensionMessageRange)] = extensionMessageRange
	}
	return stringToExtensionMessageRange
}

// NumberInReservedRanges returns true if the number is in one of the Ranges.
func NumberInReservedRanges(number int, reservedRanges ...TagRange) bool {
	for _, reservedRange := range reservedRanges {
		start := reservedRange.Start()
		end := reservedRange.End()
		if number >= start && number <= end {
			return true
		}
	}
	return false
}

// NameInReservedNames returns true if the name is in one of the ReservedNames.
func NameInReservedNames(name string, reservedNames ...ReservedName) bool {
	for _, reservedName := range reservedNames {
		if name == reservedName.Value() {
			return true
		}
	}
	return false
}

// EnumIsSubset checks if subsetEnum is a subset of supersetEnum.
func EnumIsSubset(supersetEnum Enum, subsetEnum Enum) (bool, error) {
	supersetNameToEnumValue, err := NameToEnumValue(supersetEnum)
	if err != nil {
		return false, err
	}
	subsetNameToEnumValue, err := NameToEnumValue(subsetEnum)
	if err != nil {
		return false, err
	}
	for subsetName, subsetEnumValue := range subsetNameToEnumValue {
		supersetEnumValue, ok := supersetNameToEnumValue[subsetName]
		if !ok {
			// The enum value does not exist by name, this is not a superset.
			return false, nil
		}
		if subsetEnumValue.Number() != supersetEnumValue.Number() {
			// The enum values are not equal, this is not a superset.
			return false, nil
		}
	}
	// All enum values by name exist in the superset and have the same number,
	// subsetEnum is a subset of supersetEnum.
	return true, nil
}

// TagRangeString returns the string representation of the range.
func TagRangeString(tagRange TagRange) string {
	start := tagRange.Start()
	end := tagRange.End()
	if start == end {
		return fmt.Sprintf("[%d]", start)
	}
	if tagRange.Max() {
		return fmt.Sprintf("[%d,max]", start)
	}
	return fmt.Sprintf("[%d,%d]", start, end)
}

// CheckTagRangeIsSubset checks if supersetRanges is a superset of subsetRanges.
// If so, it returns true and nil. If not, it returns false with a slice of failing ranges from subsetRanges.
func CheckTagRangeIsSubset(supersetRanges []TagRange, subsetRanges []TagRange) (bool, []TagRange) {
	if len(subsetRanges) == 0 {
		return true, nil
	}

	if len(supersetRanges) == 0 {
		return false, subsetRanges
	}

	supersetTagRangeGroups := groupAdjacentTagRanges(supersetRanges)
	subsetTagRanges := sortTagRanges(subsetRanges)
	missingTagRanges := []TagRange{}

	for i, j := 0, 0; j < len(subsetTagRanges); j++ {
		for supersetTagRangeGroups[i].end < subsetTagRanges[j].Start() {
			if i++; i == len(supersetTagRangeGroups) {
				missingTagRanges = append(missingTagRanges, subsetTagRanges[j:]...)
				return false, missingTagRanges
			}
		}
		if supersetTagRangeGroups[i].start > subsetTagRanges[j].Start() ||
			supersetTagRangeGroups[i].end < subsetTagRanges[j].End() {
			missingTagRanges = append(missingTagRanges, subsetTagRanges[j])
		}
	}

	if len(missingTagRanges) != 0 {
		return false, missingTagRanges
	}

	return true, nil
}

// groupAdjacentTagRanges sorts and groups adjacent tag ranges.
func groupAdjacentTagRanges(ranges []TagRange) []tagRangeGroup {
	if len(ranges) == 0 {
		return []tagRangeGroup{}
	}

	sortedTagRanges := sortTagRanges(ranges)

	j := 0
	groupedTagRanges := make([]tagRangeGroup, 1, len(ranges))
	groupedTagRanges[j] = tagRangeGroup{
		ranges: sortedTagRanges[0:1],
		start:  sortedTagRanges[0].Start(),
		end:    sortedTagRanges[0].End(),
	}

	for i := 1; i < len(sortedTagRanges); i++ {
		if sortedTagRanges[i].Start() <= sortedTagRanges[i-1].End()+1 {
			if sortedTagRanges[i].End() > groupedTagRanges[j].end {
				groupedTagRanges[j].end = sortedTagRanges[i].End()
			}
			groupedTagRanges[j].ranges = groupedTagRanges[j].ranges[0 : len(groupedTagRanges[j].ranges)+1]
		} else {
			groupedTagRanges = append(groupedTagRanges, tagRangeGroup{
				ranges: sortedTagRanges[i : i+1],
				start:  sortedTagRanges[i].Start(),
				end:    sortedTagRanges[i].End(),
			})
			j++
		}
	}

	return groupedTagRanges
}

// sortTagRanges sorts tag ranges by their start, end components.
func sortTagRanges(ranges []TagRange) []TagRange {
	rangesCopy := make([]TagRange, len(ranges))
	copy(rangesCopy, ranges)

	sort.Slice(rangesCopy, func(i, j int) bool {
		return rangesCopy[i].Start() < rangesCopy[j].Start() ||
			(rangesCopy[i].Start() == rangesCopy[j].Start() &&
				rangesCopy[i].End() < rangesCopy[j].End())
	})

	return rangesCopy
}

func mapFiles(files []File, getKey func(File) string) (map[string][]File, error) {
	keyToFilePathToFile := make(map[string]map[string]File)
	for _, file := range files {
		if err := addUniqueFileToMap(keyToFilePathToFile, getKey(file), file); err != nil {
			return nil, err
		}
	}
	return mapToSortedFiles(keyToFilePathToFile), nil
}

func addUniqueFileToMap(keyToFilePathToFile map[string]map[string]File, key string, file File) error {
	filePathToFile, ok := keyToFilePathToFile[key]
	if !ok {
		filePathToFile = make(map[string]File)
		keyToFilePathToFile[key] = filePathToFile
	}
	if _, ok := filePathToFile[file.Path()]; ok {
		return fmt.Errorf("duplicate file: %s", file.Path())
	}
	filePathToFile[file.Path()] = file
	return nil
}

func mapToSortedFiles(keyToFileMap map[string]map[string]File) map[string][]File {
	keyToSortedFiles := make(map[string][]File, len(keyToFileMap))
	for key, fileMap := range keyToFileMap {
		files := make([]File, 0, len(fileMap))
		for _, file := range fileMap {
			files = append(files, file)
		}
		SortFiles(files)
		keyToSortedFiles[key] = files
	}
	return keyToSortedFiles
}

type tagRangeGroup struct {
	ranges []TagRange
	start  int
	end    int
}
