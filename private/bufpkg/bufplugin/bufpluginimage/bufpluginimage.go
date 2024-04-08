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

package bufpluginimage

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginexec"
	pluginv1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/plugin/v1beta1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"go.uber.org/zap"
)

type LintHandler interface {
	Check(
		ctx context.Context,
		container app.StderrContainer,
		config bufconfig.LintConfig,
		image bufimage.Image,
	) error
}

func NewLintHandler(
	logger *zap.Logger,
	runner command.Runner,
) LintHandler {
	return newLintHandler(logger, runner)
}

// *** PRIVATE ***

type lintHandler struct {
	logger *zap.Logger
	runner command.Runner
}

func newLintHandler(
	logger *zap.Logger,
	runner command.Runner,
) *lintHandler {
	return &lintHandler{
		logger: logger,
		runner: runner,
	}
}

func (l *lintHandler) Check(
	ctx context.Context,
	container app.StderrContainer,
	config bufconfig.LintConfig,
	image bufimage.Image,
) error {
	if config.Disabled() {
		return nil
	}
	lintPluginConfigs := config.Plugins()
	if len(lintPluginConfigs) == 0 {
		return nil
	}
	env := bufplugin.Env{
		Stderr: container.Stderr(),
	}
	request, err := bufplugin.NewLintRequest(
		&pluginv1beta1.LintRequest{
			Files: imageToProtoFiles(image),
		},
	)
	if err != nil {
		return err
	}
	pathToExternalPath := getPathToExternalPathForImage(image)
	responseWriter := bufplugin.NewLintResponseWriter()
	var fileAnnotations []bufanalysis.FileAnnotation
	for _, lintPluginConfig := range lintPluginConfigs {
		handler := bufpluginexec.NewLintHandler(
			l.runner,
			lintPluginConfig.Path(),
			lintPluginConfig.Args(),
		)
		if err := handler.Handle(ctx, env, responseWriter, request); err != nil {
			return err
		}
		protoLintResponse, err := responseWriter.ToProtoLintResponse()
		if err != nil {
			return err
		}
		for _, protoAnnotation := range protoLintResponse.GetAnnotations() {
			fileAnnotation, err := protoAnnotationToFileAnnotation(pathToExternalPath, protoAnnotation)
			if err != nil {
				return err
			}
			fileAnnotations = append(fileAnnotations, fileAnnotation)
		}
	}
	if len(fileAnnotations) > 0 {
		return bufanalysis.NewFileAnnotationSet(fileAnnotations...)
	}
	return nil
}

func imageToProtoFiles(image bufimage.Image) []*pluginv1beta1.File {
	var protoFiles []*pluginv1beta1.File
	for _, imageFile := range image.Files() {
		protoFiles = append(
			protoFiles,
			&pluginv1beta1.File{
				FileDescriptorProto: imageFile.FileDescriptorProto(),
				IsImport:            imageFile.IsImport(),
			},
		)
	}
	return protoFiles
}

func getPathToExternalPathForImage(image bufimage.Image) map[string]string {
	pathToExternalPath := make(map[string]string)
	for _, imageFile := range image.Files() {
		pathToExternalPath[imageFile.Path()] = imageFile.ExternalPath()
	}
	return pathToExternalPath
}

func protoAnnotationToFileAnnotation(
	pathToExternalPath map[string]string,
	protoAnnotation *pluginv1beta1.Annotation,
) (bufanalysis.FileAnnotation, error) {
	var fileInfo *fileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	if fileName := protoAnnotation.GetFileName(); fileName != "" {
		fileInfo = newFileInfo(fileName, pathToExternalPath[fileName])
		// TODO: Reconcile differences in semantics with bufanalysis.FileAnnotation
		// TODO: Why are we not using bufplugin.Annotation if we have it?
		startLine = int(protoAnnotation.GetStartLine())
		endLine = int(protoAnnotation.GetEndLine())
		startColumn = int(protoAnnotation.GetStartColumn())
		endColumn = int(protoAnnotation.GetEndColumn())
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		protoAnnotation.Id,
		protoAnnotation.Message,
	), nil
}

type fileInfo struct {
	path         string
	externalPath string
}

func newFileInfo(path string, externalPath string) *fileInfo {
	return &fileInfo{
		path:         path,
		externalPath: externalPath,
	}
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	if f.externalPath != "" {
		return f.externalPath
	}
	return f.path
}
