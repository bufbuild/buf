// Copyright 2020-2022 Buf Technologies, Inc.
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

package appprotoexec

import (
	"context"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type templateHandler struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
	pluginPath        string
}

func newTemplateHandler(
	logger *zap.Logger,
	pluginPath string,
) *templateHandler {
	// TODO: The storageos.Provider should be thread in from the layer above.
	// This is instantiated here for convenience, but will be refactored later.
	storageosProvider := storageos.NewProvider()
	return &templateHandler{
		logger:            logger.Named("appprotoexec"),
		storageosProvider: storageosProvider,
		pluginPath:        pluginPath,
	}
}

func (h *templateHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
	_, span := trace.StartSpan(ctx, "template_plugin")
	span.AddAttributes(trace.StringAttribute("plugin", filepath.Base(h.pluginPath)))
	defer span.End()
	readWriteBucket, err := h.storageosProvider.NewReadWriteBucket(h.pluginPath, storageos.ReadWriteBucketWithSymlinksIfSupported())
	if err != nil {
		return err
	}
	templateEngine, err := newTemplateEngine(storage.MapReadBucket(readWriteBucket, storage.MatchPathExt(".tmpl")))
	if err != nil {
		return err
	}
	response, err := templateEngine.Generate(ctx, request)
	if err != nil {
		return err
	}
	response, err = normalizeCodeGeneratorResponse(response)
	if err != nil {
		return err
	}
	if response.GetSupportedFeatures()&uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) != 0 {
		responseWriter.SetFeatureProto3Optional()
	}
	for _, file := range response.File {
		if err := responseWriter.AddFile(file); err != nil {
			return err
		}
	}
	// plugin.proto specifies that only non-empty errors are considered errors.
	// This is also consistent with protoc's behavior.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/069f989b483e63005f87ab309de130677718bbec/src/google/protobuf/compiler/plugin.proto#L100-L108.
	if response.GetError() != "" {
		responseWriter.AddError(response.GetError())
	}
	return nil
}
