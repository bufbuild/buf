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
		return bufanalysis.NewFileAnnotationSet(
			protoAnnotationsToFileAnnotations(
				protoAnnotations,
			)...,
		)
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

func protoAnnotationsToFileAnnotations(protoAnnotations []*checkv1beta1.Annotation) []bufanalysis.FileAnnotation {
	return slicesext.Map(protoAnnotations, protoAnnotationToFileAnnotation)
}

func protoAnnotationToFileAnnotation(protoAnnotation *checkv1beta1.Annotation) bufanalysis.FileAnnotation {
	if protoAnnotation == nil {
		return nil
	}
	var fileInfo *fileInfo
	if fileName := protoAnnotation.GetFileName(); fileName != "" {
		fileInfo = newFileInfo(fileName)
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		int(protoAnnotation.GetStartLine()),
		int(protoAnnotation.GetStartColumn()),
		int(protoAnnotation.GetEndLine()),
		int(protoAnnotation.GetEndColumn()),
		protoAnnotation.GetId(),
		protoAnnotation.GetMessage(),
	)
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
