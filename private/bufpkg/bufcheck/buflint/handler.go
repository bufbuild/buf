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

package buflint

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/buflintplugin"
	"github.com/bufbuild/buf/private/bufpkg/buflintplugin/buflintpluginexec"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	lintv1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/plugin/lint/v1beta1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

type handler struct {
	logger        *zap.Logger
	commandRunner command.Runner
	tracer        tracing.Tracer
	runner        *internal.Runner
}

func newHandler(logger *zap.Logger, commandRunner command.Runner, tracer tracing.Tracer) *handler {
	return &handler{
		logger:        logger,
		commandRunner: commandRunner,
		tracer:        tracer,
		// linting allows for comment ignores
		// note that comment ignores still need to be enabled within the config
		// for a given check, this just says that comment ignores are allowed
		// in the first place
		runner: internal.NewRunner(
			logger,
			tracer,
			internal.RunnerWithIgnorePrefix(buflintcheck.CommentIgnorePrefix),
		),
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
	files, err := bufprotosource.NewFiles(ctx, image)
	if err != nil {
		return err
	}
	internalConfig, err := internalConfigForConfig(config)
	if err != nil {
		return err
	}
	var fileAnnotationSet bufanalysis.FileAnnotationSet
	if err := h.runner.Check(ctx, internalConfig, nil, files); err != nil {
		// If an error other than a FileAnnotationSet, return now.
		if !errors.As(err, &fileAnnotationSet) {
			return err
		}
	}
	lintPluginConfigs := config.Plugins()
	if len(lintPluginConfigs) == 0 {
		return fileAnnotationSet
	}
	env := buflintplugin.Env{
		Stderr: container.Stderr(),
	}
	request, err := buflintplugin.NewRequest(imageToProtoLintRequest(image))
	if err != nil {
		return err
	}
	responseWriter := buflintplugin.NewResponseWriter()
	var pluginFileAnnotations []bufanalysis.FileAnnotation
	// Otherwise, also check plugins.
	for _, lintPluginConfig := range lintPluginConfigs {
		handler := buflintpluginexec.NewHandler(
			h.commandRunner,
			lintPluginConfig.Path(),
			lintPluginConfig.Args(),
		)
		if err := handler.Handle(ctx, env, responseWriter, request); err != nil {
			// Always an error not related to annotations based on how we designed API for now.
			return err
		}
		// TODO: This breaks down the whole Handler model, since this has an error on it
		// and we've said that this error is handled and returned by Handler.
		protoLintResponse, err := responseWriter.ToProtoResponse()
		if err != nil {
			return err
		}
		for _, protoAnnotation := range protoLintResponse.GetAnnotations() {
			fileAnnotation, err := protoLintAnnotationToFileAnnotation(protoAnnotation)
			if err != nil {
				return err
			}
			pluginFileAnnotations = append(pluginFileAnnotations, fileAnnotation)
		}
	}
	if len(pluginFileAnnotations) == 0 {
		return fileAnnotationSet
	}
	if fileAnnotationSet == nil {
		return bufanalysis.NewFileAnnotationSet(pluginFileAnnotations...)
	}
	return bufanalysis.NewFileAnnotationSet(
		append(
			fileAnnotationSet.FileAnnotations(),
			pluginFileAnnotations...,
		)...,
	)
}

func imageToProtoLintRequest(image bufimage.Image) *lintv1beta1.Request {
	var protoLintFiles []*lintv1beta1.File
	for _, imageFile := range image.Files() {
		protoLintFiles = append(
			protoLintFiles,
			&lintv1beta1.File{
				FileDescriptorProto: imageFile.FileDescriptorProto(),
				IsImport:            imageFile.IsImport(),
			},
		)
	}
	return &lintv1beta1.Request{
		Files: protoLintFiles,
	}
}

func protoLintAnnotationToFileAnnotation(protoLintAnnotation *lintv1beta1.Annotation) (bufanalysis.FileAnnotation, error) {
	// TODO: keep a map of path to external path for the input on the request, recreate
	return nil, errors.New("TODO")
}
