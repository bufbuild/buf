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

package bufpluginexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type protocProxyHandler struct {
	storageosProvider storageos.Provider
	runner            command.Runner
	protocPath        string
	pluginName        string
	tracer            trace.Tracer
}

func newProtocProxyHandler(
	storageosProvider storageos.Provider,
	runner command.Runner,
	protocPath string,
	pluginName string,
) *protocProxyHandler {
	return &protocProxyHandler{
		storageosProvider: storageosProvider,
		runner:            runner,
		protocPath:        protocPath,
		pluginName:        pluginName,
		tracer:            otel.GetTracerProvider().Tracer("bufbuild/buf"),
	}
}

func (h *protocProxyHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) (retErr error) {
	ctx, span := h.tracer.Start(ctx, "protoc_proxy", trace.WithAttributes(
		attribute.Key("plugin").String(filepath.Base(h.pluginName)),
	))
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	protocVersion, err := h.getProtocVersion(ctx, container)
	if err != nil {
		return err
	}
	if h.pluginName == "kotlin" && !getKotlinSupportedAsBuiltin(protocVersion) {
		return fmt.Errorf("kotlin is not supported for protoc version %s", versionString(protocVersion))
	}
	// When we create protocProxyHandlers in NewHandler, we always prefer protoc-gen-.* plugins
	// over builtin plugins, so we only get here if we did not find protoc-gen-js, so this
	// is an error
	if h.pluginName == "js" && !getJSSupportedAsBuiltin(protocVersion) {
		return errors.New("js moved to a separate plugin hosted at https://github.com/protocolbuffers/protobuf-javascript in v21, you must install this plugin")
	}
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: request.ProtoFile,
	}
	fileDescriptorSetData, err := protoencoding.NewWireMarshaler().Marshal(fileDescriptorSet)
	if err != nil {
		return err
	}
	descriptorFilePath := app.DevStdinFilePath
	var tmpFile tmp.File
	if descriptorFilePath == "" {
		// since we have no stdin file (i.e. Windows), we're going to have to use a temporary file
		tmpFile, err = tmp.NewFileWithData(fileDescriptorSetData)
		if err != nil {
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, tmpFile.Close())
		}()
		descriptorFilePath = tmpFile.AbsPath()
	}
	tmpDir, err := tmp.NewDir()
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, tmpDir.Close())
	}()
	args := []string{
		fmt.Sprintf("--descriptor_set_in=%s", descriptorFilePath),
		fmt.Sprintf("--%s_out=%s", h.pluginName, tmpDir.AbsPath()),
	}
	if getSetExperimentalAllowProto3OptionalFlag(protocVersion) {
		args = append(
			args,
			"--experimental_allow_proto3_optional",
		)
	}
	if parameter := request.GetParameter(); parameter != "" {
		args = append(
			args,
			fmt.Sprintf("--%s_opt=%s", h.pluginName, parameter),
		)
	}
	args = append(
		args,
		request.FileToGenerate...,
	)
	stdin := ioext.DiscardReader
	if descriptorFilePath != "" && descriptorFilePath == app.DevStdinFilePath {
		stdin = bytes.NewReader(fileDescriptorSetData)
	}
	if err := h.runner.Run(
		ctx,
		h.protocPath,
		command.RunWithArgs(args...),
		command.RunWithEnv(app.EnvironMap(container)),
		command.RunWithStdin(stdin),
		command.RunWithStderr(container.Stderr()),
	); err != nil {
		// TODO: strip binary path as well?
		// We don't know if this is a system error or plugin error, so we assume system error
		return handlePotentialTooManyFilesError(err)
	}
	if getFeatureProto3OptionalSupported(protocVersion) {
		responseWriter.SetFeatureProto3Optional()
	}
	// no need for symlinks here, and don't want to support
	readWriteBucket, err := h.storageosProvider.NewReadWriteBucket(tmpDir.AbsPath())
	if err != nil {
		return err
	}
	return storage.WalkReadObjects(
		ctx,
		readWriteBucket,
		"",
		func(readObject storage.ReadObject) error {
			data, err := io.ReadAll(readObject)
			if err != nil {
				return err
			}
			return responseWriter.AddFile(
				&pluginpb.CodeGeneratorResponse_File{
					Name:    proto.String(readObject.Path()),
					Content: proto.String(string(data)),
				},
			)
		},
	)
}

func (h *protocProxyHandler) getProtocVersion(
	ctx context.Context,
	container app.EnvContainer,
) (*pluginpb.Version, error) {
	stdoutBuffer := bytes.NewBuffer(nil)
	if err := h.runner.Run(
		ctx,
		h.protocPath,
		command.RunWithArgs("--version"),
		command.RunWithEnv(app.EnvironMap(container)),
		command.RunWithStdout(stdoutBuffer),
	); err != nil {
		// TODO: strip binary path as well?
		return nil, handlePotentialTooManyFilesError(err)
	}
	return parseVersionForCLIVersion(strings.TrimSpace(stdoutBuffer.String()))
}
