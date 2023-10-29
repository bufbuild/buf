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
	defaultValidator *protovalidate.Validator
)

func init() {
	var err error
	defaultValidator, err = protovalidate.New()
	if err != nil {
		panic(err.Error())
	}
}

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
	return newDigest(digestType, protoDigest.Value)
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

// ManifestBlobAndBlobsToFileSet converts the given manifest Blob and set of Blobs representing
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

// BlobToManifest converts the given proto Blob representing the string representation of a
// Manifest into a Manifest.
//
// # The proto Blob is assumed to be non-nil
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
	if protoOptions.validator == nil {
		protoOptions.validator = defaultValidator
	}
	return protoOptions.validator.Validate(message)
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
