// Copyright 2020-2021 Buf Technologies, Inc.
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
	"bytes"
	"context"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

var defaultVersion = newVersion(
	DefaultMajorVersion,
	DefaultMinorVersion,
	DefaultPatchVersion,
	DefaultSuffixVersion,
)

type binaryHandler struct {
	logger     *zap.Logger
	pluginPath string
}

func newBinaryHandler(
	logger *zap.Logger,
	pluginPath string,
) *binaryHandler {
	return &binaryHandler{
		logger:     logger.Named("appprotoexec"),
		pluginPath: pluginPath,
	}
}

func (h *binaryHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseWriter,
	request *pluginpb.CodeGeneratorRequest,
) error {
	ctx, span := trace.StartSpan(ctx, "plugin_proxy")
	span.AddAttributes(trace.StringAttribute("plugin", filepath.Base(h.pluginPath)))
	defer span.End()
	unsetRequestVersion := false
	if request.CompilerVersion == nil {
		unsetRequestVersion = true
		request.CompilerVersion = defaultVersion
	}
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request)
	if unsetRequestVersion {
		request.CompilerVersion = nil
	}
	if err != nil {
		return err
	}
	responseBuffer := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, h.pluginPath)
	cmd.Env = app.Environ(container)
	cmd.Stdin = bytes.NewReader(requestData)
	cmd.Stdout = responseBuffer
	cmd.Stderr = container.Stderr()
	if err := cmd.Run(); err != nil {
		// TODO: strip binary path as well?
		return err
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(responseBuffer.Bytes(), response); err != nil {
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
	// call AddError even if this is empty
	if response.Error != nil {
		responseWriter.AddError(response.GetError())
	}
	return nil
}

func newVersion(major int32, minor int32, patch int32, suffix string) *pluginpb.Version {
	version := &pluginpb.Version{
		Major: proto.Int32(major),
		Minor: proto.Int32(minor),
		Patch: proto.Int32(patch),
	}
	if suffix != "" {
		version.Suffix = proto.String(suffix)
	}
	return version
}
