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

package bufcheck

import (
	checkv1beta1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/check/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

func imageToProtoFiles(image bufimage.Image) []*checkv1beta1.File {
	if image == nil {
		return nil
	}
	return slicesext.Map(image.Files(), imageFileToProtoFile)
}

func imageFileToProtoFile(imageFile bufimage.ImageFile) *checkv1beta1.File {
	return &checkv1beta1.File{
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
