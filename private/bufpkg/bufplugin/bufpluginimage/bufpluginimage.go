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
	"errors"

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

type Handler interface {
	Check(
		ctx context.Context,
		container app.StderrContainer,
		config bufconfig.LintConfig,
		image bufimage.Image,
	) error
}

func NewHandler(
	logger *zap.Logger,
	runner command.Runner,
) Handler {
	return newHandler(logger, runner)
}

// *** PRIVATE ***

type handler struct {
	logger *zap.Logger
	runner command.Runner
}

func newHandler(
	logger *zap.Logger,
	runner command.Runner,
) *handler {
	return &handler{
		logger: logger,
		runner: runner,
	}
}

func (h *handler) Check(
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
	request, err := bufplugin.NewLintRequest(imageToProtoLintRequest(image))
	if err != nil {
		return err
	}
	responseWriter := bufplugin.NewLintResponseWriter()
	var fileAnnotations []bufanalysis.FileAnnotation
	for _, lintPluginConfig := range lintPluginConfigs {
		handler := bufpluginexec.NewLintHandler(
			h.runner,
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
			fileAnnotation, err := protoAnnotationToFileAnnotation(protoAnnotation)
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

func imageToProtoLintRequest(image bufimage.Image) *pluginv1beta1.LintRequest {
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
	return &pluginv1beta1.LintRequest{
		Files: protoFiles,
	}
}

func protoAnnotationToFileAnnotation(protoAnnotation *pluginv1beta1.Annotation) (bufanalysis.FileAnnotation, error) {
	// TODO: keep a map of path to external path for the input on the request, recreate
	return nil, errors.New("TODO")
}
