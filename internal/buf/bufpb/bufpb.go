// Package bufpb provides wrappers to the Buf Protobuf API.
package bufpb

import (
	"bytes"

	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// TODO: evaluate the normalization of names
// Right now we make sure every input name is normalized and valid.
// This is always the case with buf input, but may not be the case with protoc input.

var jsonUnmarshaler = &jsonpb.Unmarshaler{
	AllowUnknownFields: true,
}

// Image is an interface to wrap imagev1beta1.Images.
//
// Fields should not be modified.
type Image interface {
	protodescpb.BaseFileDescriptorSet

	GetBufbuildImageExtension() *imagev1beta1.ImageExtension

	// ImportNames returns the sorted import names.
	ImportNames() ([]string, error)

	// WithoutImports returns a copy of the Image without imports.
	//
	// If GetBufbuildImageExtension() is nil, returns the original Image.
	// If there are no imports, returns the original Image.
	//
	// Backing FileDescriptorProtos are not copied, only the references are copied.
	// This will result in unknown fields being dropped from the backing Image, but not
	// the backing FileDescriptorProtos.
	//
	// Validates the output.
	WithoutImports() (Image, error)

	// WithSpecificNames returns a copy of the Image with only the Files with the given names.
	//
	// Names are normalized and validated.
	// If allowNotExist is false, the specific names must exist on the input image.
	// Backing FileDescriptorProtos are not copied, only the references are copied.
	// Validates the output.
	//
	WithSpecificNames(allowNotExist bool, specificNames ...string) (Image, error)

	// ToFileDescriptorSet converts the Image to a native FileDescriptorSet.
	//
	// This strips the backing ImageExtension and then re-validates using protodescpb validation.
	//
	// Backing FileDescriptorProtos are not copied, only the references are copied.
	// This will result in unknown fields being dropped from the backing FileDescriptorSet, but not
	// the backing FileDescriptorProtos.
	ToFileDescriptorSet() (protodescpb.FileDescriptorSet, error)

	// ToCodeGeneratorRequest converts the image to a CodeGeneratorRequest.
	//
	// The files to generate must be within the Image.
	// Files to generate are normalized and validated.
	//
	// TODO: this should only be for testing, move somewhere else if possible
	ToCodeGeneratorRequest(parameter string, fileToGenerate ...string) (*plugin_go.CodeGeneratorRequest, error)
}

// NewImage returns a new validated Image for the imagev1beta1.Image.
func NewImage(backing *imagev1beta1.Image) (Image, error) {
	return newImage(backing)
}

// UnmarshalWireDataImage returns a new validated Image for the imagev1beta1.Image.
func UnmarshalWireDataImage(data []byte) (Image, error) {
	backing := &imagev1beta1.Image{}
	if err := proto.Unmarshal(data, backing); err != nil {
		return nil, err
	}
	return NewImage(backing)
}

// UnmarshalJSONDataImage returns a new validated Image for the imagev1beta1.Image.
func UnmarshalJSONDataImage(data []byte) (Image, error) {
	backing := &imagev1beta1.Image{}
	if err := jsonUnmarshaler.Unmarshal(bytes.NewReader(data), backing); err != nil {
		return nil, err
	}
	return NewImage(backing)
}

// CodeGeneratorRequestToImage converts the CodeGeneratorRequest to an Image.
func CodeGeneratorRequestToImage(request *plugin_go.CodeGeneratorRequest) (Image, error) {
	backing := &imagev1beta1.Image{
		File: request.GetProtoFile(),
	}
	image, err := NewImage(backing)
	if err != nil {
		return nil, err
	}
	return image.WithSpecificNames(false, request.FileToGenerate...)
}
