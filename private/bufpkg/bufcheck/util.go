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

package bufcheck

import (
	"slices"

	descriptorv1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/descriptor/v1"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func imageToProtoFileDescriptors(image bufimage.Image) ([]*descriptorv1.FileDescriptor, error) {
	if image == nil {
		return nil, nil
	}
	descriptors := slicesext.Map(image.Files(), imageToProtoFileDescriptor)
	// We need to ensure that if a FileDescriptorProto includes a Go
	// feature set extension, that it matches the library version in
	// gofeaturespb. The library will use the gofeaturespb.E_Go extension
	// type to determine how to parse the file. This must match the expected
	// go type to avoid panics on getting the extension when using
	// proto.GetExtension or protodesc.NewFiles. We therefore reparse all
	// extensions if it may contain any Go feature set extensions, with
	// a resolver that includes the Go feature set resolver.
	// See the issue: https://github.com/golang/protobuf/issues/1669
	const goFeaturesImportPath = "google/protobuf/go_features.proto"
	var reparseDescriptors []*descriptorv1.FileDescriptor
	for _, descriptor := range descriptors {
		fileDescriptorProto := descriptor.FileDescriptorProto
		// Trigger reparsing on any file that includes the gofeatures import.
		if slices.Contains(fileDescriptorProto.Dependency, goFeaturesImportPath) {
			reparseDescriptors = append(reparseDescriptors, descriptor)
		}
	}
	if len(reparseDescriptors) == 0 {
		return descriptors, nil
	}
	goFeaturesResolver, err := protoencoding.NewGoFeaturesResolver()
	if err != nil {
		return nil, err
	}
	resolver := protoencoding.CombineResolvers(
		goFeaturesResolver,
		protoencoding.NewLazyResolver(slicesext.Map(descriptors, func(fileDescriptor *descriptorv1.FileDescriptor) *descriptorpb.FileDescriptorProto {
			return fileDescriptor.FileDescriptorProto
		})...),
	)
	for _, descriptor := range reparseDescriptors {
		// We clone the FileDescriptorProto to avoid modifying the original.
		fileDescriptorProto := &descriptorpb.FileDescriptorProto{}
		proto.Merge(fileDescriptorProto, descriptor.FileDescriptorProto)
		if err := protoencoding.ReparseExtensions(resolver, fileDescriptorProto.ProtoReflect()); err != nil {
			return nil, err
		}
		descriptor.FileDescriptorProto = fileDescriptorProto
	}
	return descriptors, nil
}

func imageToProtoFileDescriptor(imageFile bufimage.ImageFile) *descriptorv1.FileDescriptor {
	return &descriptorv1.FileDescriptor{
		FileDescriptorProto: imageFile.FileDescriptorProto(),
		IsImport:            imageFile.IsImport(),
		IsSyntaxUnspecified: imageFile.IsSyntaxUnspecified(),
		UnusedDependency:    imageFile.UnusedDependencyIndexes(),
	}
}

// imageToPathToExternalPath returns a map from path to external path for all ImageFiles in the Image.
//
// We do not transmit external path information over the wire to plugins, so we need to keep track
// of this on the client side to properly construct bufanalysis.FileAnnotations when we get back
// check.Annotations. This is used in annotationToFileAnnotation.
func imageToPathToExternalPath(image bufimage.Image) map[string]string {
	imageFiles := image.Files()
	pathToExternalPath := make(map[string]string, len(imageFiles))
	for _, imageFile := range imageFiles {
		// We know that Images do not have overlapping paths.
		pathToExternalPath[imageFile.Path()] = imageFile.ExternalPath()
	}
	return pathToExternalPath
}
