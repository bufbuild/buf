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
	"context"

	checkv1beta1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/check/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/gen/proto/pluginrpc/buf/plugin/check/v1beta1/v1beta1pluginrpc"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/pluginrpc-go"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type lintClient struct {
	client pluginrpc.Client
}

func newLintClient(
	runner pluginrpc.Runner,
	programName string,
) *lintClient {
	return &lintClient{
		client: pluginrpc.NewClient(runner, programName),
	}
}

func (l *lintClient) Lint(ctx context.Context, image bufimage.Image) error {
	lintServiceClient, err := v1beta1pluginrpc.NewLintServiceClient(l.client)
	if err != nil {
		return err
	}
	response, err := lintServiceClient.Lint(
		ctx,
		&checkv1beta1.LintRequest{
			Files: imageToProtoFiles(image),
		},
	)
	if err != nil {
		return err
	}
	if protoAnnotations := response.GetAnnotations(); len(protoAnnotations) > 0 {
		protoregistryFiles, err := protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(image))
		if err != nil {
			return err
		}
		fileAnnotatations, err := protoAnnotationsToFileAnnotations(
			protoregistryFiles,
			protoAnnotations,
		)
		if err != nil {
			return err
		}
		return bufanalysis.NewFileAnnotationSet(fileAnnotatations...)
	}
	return nil
}

func imageToProtoFiles(image bufimage.Image) []*checkv1beta1.File {
	return slicesext.Map(image.Files(), imageFileToProtoFile)
}

func imageFileToProtoFile(imageFile bufimage.ImageFile) *checkv1beta1.File {
	return &checkv1beta1.File{
		FileDescriptorProto: imageFile.FileDescriptorProto(),
		IsImport:            imageFile.IsImport(),
	}
}

func protoAnnotationsToFileAnnotations(
	protoregistryFiles *protoregistry.Files,
	protoAnnotations []*checkv1beta1.Annotation,
) ([]bufanalysis.FileAnnotation, error) {
	return slicesext.MapError(
		protoAnnotations,
		func(protoAnnotation *checkv1beta1.Annotation) (bufanalysis.FileAnnotation, error) {
			return protoAnnotationToFileAnnotation(protoregistryFiles, protoAnnotation)
		},
	)
}

func protoAnnotationToFileAnnotation(
	protoregistryFiles *protoregistry.Files,
	protoAnnotation *checkv1beta1.Annotation,
) (bufanalysis.FileAnnotation, error) {
	if protoAnnotation == nil {
		return nil, nil
	}
	var fileInfo *fileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	if location := protoAnnotation.GetLocation(); location != nil {
		name := location.GetName()
		fileInfo = newFileInfo(name)
		if path := location.GetPath(); len(path) > 0 {
			fileDescriptor, err := protoregistryFiles.FindFileByPath(name)
			if err != nil {
				return nil, err
			}
			if sourceLocation := fileDescriptor.SourceLocations().ByPath(path); !isSourceLocationEqualToZeroValue(sourceLocation) {
				startLine = sourceLocation.StartLine + 1
				startColumn = sourceLocation.StartColumn + 1
				endLine = sourceLocation.EndLine + 1
				endColumn = sourceLocation.EndColumn + 1
			}
		}
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		protoAnnotation.GetId(),
		protoAnnotation.GetMessage(),
	), nil
}

type fileInfo struct {
	path string
}

func newFileInfo(path string) *fileInfo {
	return &fileInfo{
		path: path,
	}
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	return f.path
}

// The protoreflect API is a disaster. It says that "If there is no SourceLocation,
// the zero value is returned", but equality is not easy because SourceLocation contains
// slices. This is just a mess.
func isSourceLocationEqualToZeroValue(sourceLocation protoreflect.SourceLocation) bool {
	return len(sourceLocation.Path) == 0 &&
		sourceLocation.StartLine == 0 &&
		sourceLocation.StartColumn == 0 &&
		sourceLocation.EndLine == 0 &&
		sourceLocation.EndColumn == 0 &&
		len(sourceLocation.LeadingDetachedComments) == 0 &&
		sourceLocation.LeadingComments == "" &&
		sourceLocation.TrailingComments == "" &&
		sourceLocation.Next == 0
}
