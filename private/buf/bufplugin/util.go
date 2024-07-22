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

package bufplugin

import (
	checkv1beta1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/check/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/bufplugin-go/bufplugincheck"
)

func imageToProtoFiles(image bufimage.Image) []*checkv1beta1.File {
	return slicesext.Map(image.Files(), imageFileToProtoFile)
}

func imageFileToProtoFile(imageFile bufimage.ImageFile) *checkv1beta1.File {
	return &checkv1beta1.File{
		FileDescriptorProto: imageFile.FileDescriptorProto(),
		IsImport:            imageFile.IsImport(),
	}
}

func annotationsToFileAnnotations(annotations []bufplugincheck.Annotation) []bufanalysis.FileAnnotation {
	return slicesext.Map(annotations, annotationToFileAnnotation)
}

func annotationToFileAnnotation(annotation bufplugincheck.Annotation) bufanalysis.FileAnnotation {
	if annotation == nil {
		return nil
	}
	var fileInfo *fileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	if location := annotation.Location(); location != nil {
		fileInfo = newFileInfo(location.FileName())
		startLine = location.StartLine() + 1
		startColumn = location.StartColumn() + 1
		endLine = location.EndLine() + 1
		endColumn = location.EndColumn() + 1
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		annotation.ID(),
		annotation.Message(),
	)
}
