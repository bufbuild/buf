// Package protodescpb is meant to provide an easy mechanism to switch between
// github.com/golang/protobuf and google.golang.org/protobuf in the future.
package protodescpb

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// TODO: evaluate the normalization of names
// Right now we make sure every input name is normalized and valid.
// This is always the case with buf input, but may not be the case with protoc input.

// Marshaler says that a Protobuf type can be marshalled.
type Marshaler interface {
	// MarshalWire marshals the backing type to wire format.
	MarshalWire() ([]byte, error)
	// MarshalJSON marshals the backing type to JSON format.
	MarshalJSON() ([]byte, error)
	// MarshalJSONIndent marshals the backing type to JSON format with indents.
	MarshalJSONIndent() ([]byte, error)
	// MarshalText marshals the backing type to text format.
	MarshalText() ([]byte, error)
}

// BaseFileDescriptorSet is an interface to wrap FileDescriptorSet implementations.
//
// Fields should not be modified.
type BaseFileDescriptorSet interface {
	Marshaler

	// GetFile gets all the FileDescriptors.
	GetFile() []FileDescriptor
}

// FileDescriptorSet is an interface to wrap FileDescriptorSet implementations.
//
// Fields should not be modified.
type FileDescriptorSet interface {
	BaseFileDescriptorSet

	// Equal calls proto.Equal on the backing FileDescriptorSets.
	Equal(FileDescriptorSet) bool
}

// FileDescriptor is an interface to wrap FileDescriptorProto implementations.
//
// Fields should not be modified.
type FileDescriptor interface {
	GetDependency() []string
	GetEnumType() []*descriptor.EnumDescriptorProto
	GetExtension() []*descriptor.FieldDescriptorProto
	GetMessageType() []*descriptor.DescriptorProto
	GetName() string
	GetOptions() *descriptor.FileOptions
	GetPackage() string
	GetPublicDependency() []int32
	GetService() []*descriptor.ServiceDescriptorProto
	GetSourceCodeInfo() *descriptor.SourceCodeInfo
	GetSyntax() string
	GetWeakDependency() []int32
}

// NewFileDescriptorSet returns a new FileDescriptorSet for the golang/protobuf v1 FileDescriptorSet.
//
// Validates that the input fileDescriptorSet is not empty, that the File field is not empty,
// and every FileDescriptorProto has a non-empty name.
func NewFileDescriptorSet(backing *descriptor.FileDescriptorSet) (FileDescriptorSet, error) {
	return newFileDescriptorSet(backing)
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	return &v
}

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int32 {
	p := new(int32)
	*p = int32(v)
	return p
}

// Int64 is a helper routine that allocates a new int64 value
// to store v and returns a pointer to it.
func Int64(v int64) *int64 {
	return &v
}

// Float32 is a helper routine that allocates a new float32 value
// to store v and returns a pointer to it.
func Float32(v float32) *float32 {
	return &v
}

// Float64 is a helper routine that allocates a new float64 value
// to store v and returns a pointer to it.
func Float64(v float64) *float64 {
	return &v
}

// Uint32 is a helper routine that allocates a new uint32 value
// to store v and returns a pointer to it.
func Uint32(v uint32) *uint32 {
	return &v
}

// Uint64 is a helper routine that allocates a new uint64 value
// to store v and returns a pointer to it.
func Uint64(v uint64) *uint64 {
	return &v
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	return &v
}
