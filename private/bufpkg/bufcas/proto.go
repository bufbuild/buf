// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufcas

import (
	"bytes"
	"fmt"

	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/protobuf/proto"
)

// This file contains public functions to convert to and from Protobuf types.
//
// TODO: One could argue we should not be validating input Protobuf messages, as these should
// be validated at a higher level. Output messages we likely want to validate, although we
// could define that these should be done elsewhere, but generally we like to validate at
// construction time. Consider removing at least input Protobuf message validation.

var (
	// defaultValidate is the default Validator used in the absence of ProtoWithValidator
	// being passed for a given Protobuf conversion function.
	//
	// Disabling validation by default for now by not instantiating this.
	// TODO: discuss if we should do this by default, including on CLI side (probably not).
	defaultValidator *protovalidate.Validator
)

//func init() {
//var err error
//defaultValidator, err = protovalidate.New()
//if err != nil {
//panic(err.Error())
//}
//}

// BlobToProto converts the given Blob to a proto Blob.
func BlobToProto(blob Blob, options ...ProtoOption) (*storagev1beta1.Blob, error) {
	protoDigest, err := DigestToProto(blob.Digest(), doNotProtoValidate(options)...)
	if err != nil {
		return nil, err
	}
	protoBlob := &storagev1beta1.Blob{
		Digest:  protoDigest,
		Content: blob.Content(),
	}
	if err := protoValidate(options, protoBlob); err != nil {
		return nil, err
	}
	return protoBlob, nil
}

// BlobsToProto converts the given Blobs to proto Blobs.
func BlobsToProto(blobs []Blob, options ...ProtoOption) ([]*storagev1beta1.Blob, error) {
	return slicesextended.MapError(
		blobs,
		func(blob Blob) (*storagev1beta1.Blob, error) {
			return BlobToProto(blob, options...)
		},
	)
}

// ProtoToBlob converts the given proto Blob to a Blob.
//
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
func ProtoToBlob(protoBlob *storagev1beta1.Blob, options ...ProtoOption) (Blob, error) {
	if err := protoValidate(options, protoBlob); err != nil {
		return nil, err
	}
	digest, err := ProtoToDigest(protoBlob.Digest, doNotProtoValidate(options)...)
	if err != nil {
		return nil, err
	}
	return NewBlobForContent(bytes.NewReader(protoBlob.Content), BlobWithKnownDigest(digest))
}

// ProtoToBlobs converts the given proto Blobs to Blobs.
func ProtoToBlobs(protoBlobs []*storagev1beta1.Blob, options ...ProtoOption) ([]Blob, error) {
	return slicesextended.MapError(
		protoBlobs,
		func(protoBlob *storagev1beta1.Blob) (Blob, error) {
			return ProtoToBlob(protoBlob, options...)
		},
	)
}

// BlobSetToProtoBlobs converts the given BlobSet into proto Blobs.
func BlobSetToProtoBlobs(blobSet BlobSet, options ...ProtoOption) ([]*storagev1beta1.Blob, error) {
	blobs := blobSet.Blobs()
	protoBlobs := make([]*storagev1beta1.Blob, len(blobs))
	for i, blob := range blobs {
		// Rely on validation in BlobToProto.
		protoBlob, err := BlobToProto(blob, options...)
		if err != nil {
			return nil, err
		}
		protoBlobs[i] = protoBlob
	}
	return protoBlobs, nil
}

// ProtoBlobsToBlobSet converts the given proto Blobs into a BlobSet.
func ProtoBlobsToBlobSet(protoBlobs []*storagev1beta1.Blob, options ...ProtoOption) (BlobSet, error) {
	blobs := make([]Blob, len(protoBlobs))
	for i, protoBlob := range protoBlobs {
		// Rely on validation in ProtoToBlob.
		blob, err := ProtoToBlob(protoBlob, options...)
		if err != nil {
			return nil, err
		}
		blobs[i] = blob
	}
	return NewBlobSet(blobs)
}

// DigestToProto converts the given Digest to a proto Digest.
func DigestToProto(digest Digest, options ...ProtoOption) (*storagev1beta1.Digest, error) {
	protoDigestType, ok := digestTypeToProto[digest.Type()]
	// Technically we have already done this validation but just to be safe.
	if !ok {
		return nil, fmt.Errorf("unknown DigestType: %v", digest.Type())
	}
	protoDigest := &storagev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}
	if err := protoValidate(options, protoDigest); err != nil {
		return nil, err
	}
	return protoDigest, nil
}

// DigestsToProto converts the given Digests to proto Digests.
func DigestsToProto(digests []Digest, options ...ProtoOption) ([]*storagev1beta1.Digest, error) {
	return slicesextended.MapError(
		digests,
		func(digest Digest) (*storagev1beta1.Digest, error) {
			return DigestToProto(digest, options...)
		},
	)
}

// ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func ProtoToDigest(protoDigest *storagev1beta1.Digest, options ...ProtoOption) (Digest, error) {
	if err := protoValidate(options, protoDigest); err != nil {
		return nil, err
	}
	digestType, ok := protoToDigestType[protoDigest.Type]
	if !ok {
		return nil, fmt.Errorf("unknown proto Digest.Type: %v", protoDigest.Type)
	}
	if err := validateDigestParameters(digestType, protoDigest.Value); err != nil {
		return nil, err
	}
	return newDigest(digestType, protoDigest.Value), nil
}

// ProtoToDigests converts the given proto Digests to Digests.
func ProtoToDigests(protoDigests []*storagev1beta1.Digest, options ...ProtoOption) ([]Digest, error) {
	return slicesextended.MapError(
		protoDigests,
		func(protoDigest *storagev1beta1.Digest) (Digest, error) {
			return ProtoToDigest(protoDigest, options...)
		},
	)
}

// FileNodeToProto converts the given FileNode to a proto FileNode.
func FileNodeToProto(fileNode FileNode, options ...ProtoOption) (*storagev1beta1.FileNode, error) {
	protoDigest, err := DigestToProto(fileNode.Digest(), doNotProtoValidate(options)...)
	if err != nil {
		return nil, err
	}
	protoFileNode := &storagev1beta1.FileNode{
		Path:   fileNode.Path(),
		Digest: protoDigest,
	}
	if err := protoValidate(options, protoFileNode); err != nil {
		return nil, err
	}
	return protoFileNode, nil
}

// FileNodesToProto converts the given FileNodes to proto FileNodes.
func FileNodesToProto(fileNodes []FileNode, options ...ProtoOption) ([]*storagev1beta1.FileNode, error) {
	return slicesextended.MapError(
		fileNodes,
		func(fileNode FileNode) (*storagev1beta1.FileNode, error) {
			return FileNodeToProto(fileNode, options...)
		},
	)
}

// ProtoToFileNode converts the given proto FileNode to a FileNode.
//
// The path is validated to be normalized and non-empty.
func ProtoToFileNode(protoFileNode *storagev1beta1.FileNode, options ...ProtoOption) (FileNode, error) {
	if err := protoValidate(options, protoFileNode); err != nil {
		return nil, err
	}
	digest, err := ProtoToDigest(protoFileNode.Digest, doNotProtoValidate(options)...)
	if err != nil {
		return nil, err
	}
	return NewFileNode(protoFileNode.Path, digest)
}

// ProtoToFileNodes converts the given proto FileNodes to FileNodes.
func ProtoToFileNodes(protoFileNodes []*storagev1beta1.FileNode, options ...ProtoOption) ([]FileNode, error) {
	return slicesextended.MapError(
		protoFileNodes,
		func(protoFileNode *storagev1beta1.FileNode) (FileNode, error) {
			return ProtoToFileNode(protoFileNode, options...)
		},
	)
}

// FileSetToProtoManifestBlobAndBlobs converts the given FileSet into a proto Blob representing the
// Manifest, and a set of Blobs representing the Files.
func FileSetToProtoManifestBlobAndBlobs(
	fileSet FileSet,
	options ...ProtoOption,
) (*storagev1beta1.Blob, []*storagev1beta1.Blob, error) {
	// Rely on validation in ManifestToProtoBlob.
	protoManifestBlob, err := ManifestToProtoBlob(fileSet.Manifest(), options...)
	if err != nil {
		return nil, nil, err
	}
	// Rely on validation in BlobSetToProtoBlobs.
	protoBlobs, err := BlobSetToProtoBlobs(fileSet.BlobSet(), options...)
	if err != nil {
		return nil, nil, err
	}
	return protoManifestBlob, protoBlobs, nil
}

// ProtoManifestBlobAndBlobsToFileSet converts the given manifest Blob and set of Blobs representing
// the Files into a FileSet.
//
// Validation is done to ensure the Manifest exactly matches the BlobSet.
func ProtoManifestBlobAndBlobsToFileSet(
	protoManifestBlob *storagev1beta1.Blob,
	protoBlobs []*storagev1beta1.Blob,
	options ...ProtoOption,
) (FileSet, error) {
	// Rely on validation in ProtoBlobToManifest.
	manifest, err := ProtoBlobToManifest(protoManifestBlob, options...)
	if err != nil {
		return nil, err
	}
	// Rely on validation in ProtoBlobsToBlobSet.
	blobSet, err := ProtoBlobsToBlobSet(protoBlobs, options...)
	if err != nil {
		return nil, err
	}
	return NewFileSet(manifest, blobSet)
}

// ManifestToProtoBlob converts the string representation of the given Manifest into a proto Blob.
//
// The Manifest is assumed to be non-nil.
func ManifestToProtoBlob(manifest Manifest, options ...ProtoOption) (*storagev1beta1.Blob, error) {
	blob, err := ManifestToBlob(manifest)
	if err != nil {
		return nil, err
	}
	// Rely on validation in BlobToProto.
	return BlobToProto(blob, options...)
}

// ProtoBlobToManifest converts the given proto Blob representing the string representation of a
// Manifest into a Manifest.
//
// The proto Blob is assumed to be non-nil.
//
// This function returns ParseErrors as it is effectively parsing the Manifest.
func ProtoBlobToManifest(protoBlob *storagev1beta1.Blob, options ...ProtoOption) (Manifest, error) {
	// Rely on validation in ProtoToBlob.
	blob, err := ProtoToBlob(protoBlob, options...)
	if err != nil {
		return nil, err
	}
	return BlobToManifest(blob)
}

// ProtoOption is an option for a Protobuf conversion function.
type ProtoOption func(*protoOptions)

// ProtoWithValidator says to use the given ProtoValidator.
//
// The default is to use a global instance of *protovalidator.Validator constructed for this package.
func ProtoWithValidator(validator ProtoValidator) ProtoOption {
	return func(protoOptions *protoOptions) {
		protoOptions.validator = validator
	}
}

// ProtoValidator is a validator for Protobuf messages.
type ProtoValidator interface {
	Validate(proto.Message) error
}

// *** PRIVATE ***

type protoOptions struct {
	validator     ProtoValidator
	doNotValidate bool
}

func newProtoOptions() *protoOptions {
	return &protoOptions{}
}

// convenience function to do the validation we actually want based on the options values.
//
// If we have more ProtoOptions in the future, we can choose to instead pass *protoOptions
// instead of []ProtoOption, however the benefit will be minimal.
func protoValidate(options []ProtoOption, message proto.Message) error {
	protoOptions := newProtoOptions()
	for _, option := range options {
		option(protoOptions)
	}
	if protoOptions.doNotValidate {
		return nil
	}
	if protoOptions.validator != nil {
		return protoOptions.validator.Validate(message)
	}
	if defaultValidator != nil {
		// Need to do this as opposed to setting protoOptions.validator and
		// then checking if protoOptions.Validator != nil because of Golang
		// interface nil weirdness.
		return defaultValidator.Validate(message)
	}
	return nil
}

// convenience function to not do validation when we call Protobuf conversion functions within
// other Protobuf conversion functions.
//
// If we were to just blanket pass the options recursively, we would get double validation, as
// validation itself is recursive. This avoids that for calls within this package.
//
// Having this option allows us to add future ProtoOptions with less chance of introducing a bug,
// ie the call path is the same.
func doNotProtoValidate(options []ProtoOption) []ProtoOption {
	return append(options, doNotProtoValidateOption)
}

func doNotProtoValidateOption(protoOptions *protoOptions) {
	// We cannot simply set validator to nil, as options can be passed in any order.
	// Technically we control the order within this package but this deals with that.
	protoOptions.doNotValidate = true
}
