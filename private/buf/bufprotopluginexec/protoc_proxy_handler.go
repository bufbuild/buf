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

package bufprotopluginexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protoplugin"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type protocProxyHandler struct {
	storageosProvider storageos.Provider
	runner            command.Runner
	tracer            tracing.Tracer
	protocPath        string
	protocExtraArgs   []string
	pluginName        string
}

func newProtocProxyHandler(
	storageosProvider storageos.Provider,
	runner command.Runner,
	tracer tracing.Tracer,
	protocPath string,
	protocExtraArgs []string,
	pluginName string,
) *protocProxyHandler {
	return &protocProxyHandler{
		storageosProvider: storageosProvider,
		runner:            runner,
		tracer:            tracer,
		protocPath:        protocPath,
		protocExtraArgs:   protocExtraArgs,
		pluginName:        pluginName,
	}
}

func (h *protocProxyHandler) Handle(
	ctx context.Context,
	pluginEnv protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) (retErr error) {
	ctx, span := h.tracer.Start(
		ctx,
		tracing.WithErr(&retErr),
		tracing.WithAttributes(
			attribute.Key("plugin").String(filepath.Base(h.pluginName)),
		),
	)
	defer span.End()

	// We should send the complete FileDescriptorSet with source-retention options to --descriptor_set_in.
	//
	// This is used via the FileDescriptorSet below.
	request, err := request.WithSourceRetentionOptions()
	if err != nil {
		return err
	}

	protocVersion, err := h.getProtocVersion(ctx, pluginEnv)
	if err != nil {
		return err
	}
	if h.pluginName == "kotlin" && !getKotlinSupportedAsBuiltin(protocVersion) {
		return fmt.Errorf("kotlin is not supported for protoc version %s", versionString(protocVersion))
	}
	if h.pluginName == "rust" && !getRustSupportedAsBuiltin(protocVersion) {
		return fmt.Errorf("rust is not supported for protoc version %s", versionString(protocVersion))
	}
	// When we create protocProxyHandlers in NewHandler, we always prefer protoc-gen-.* plugins
	// over builtin plugins, so we only get here if we did not find protoc-gen-js, so this
	// is an error
	if h.pluginName == "js" && !getJSSupportedAsBuiltin(protocVersion) {
		return errors.New("js moved to a separate plugin hosted at https://github.com/protocolbuffers/protobuf-javascript in v21, you must install this plugin")
	}
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: request.AllFileDescriptorProtos(),
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
	args := slicesext.Concat(h.protocExtraArgs, []string{
		fmt.Sprintf("--descriptor_set_in=%s", descriptorFilePath),
		fmt.Sprintf("--%s_out=%s", h.pluginName, tmpDir.AbsPath()),
	})
	if getSetExperimentalAllowProto3OptionalFlag(protocVersion) {
		args = append(
			args,
			"--experimental_allow_proto3_optional",
		)
	}
	if parameter := request.Parameter(); parameter != "" {
		args = append(
			args,
			fmt.Sprintf("--%s_opt=%s", h.pluginName, parameter),
		)
	}
	args = append(
		args,
		request.CodeGeneratorRequest().GetFileToGenerate()...,
	)
	stdin := ioext.DiscardReader
	if descriptorFilePath != "" && descriptorFilePath == app.DevStdinFilePath {
		stdin = bytes.NewReader(fileDescriptorSetData)
	}
	if err := h.runner.Run(
		ctx,
		h.protocPath,
		command.RunWithArgs(args...),
		command.RunWithEnviron(pluginEnv.Environ),
		command.RunWithStdin(stdin),
		command.RunWithStderr(pluginEnv.Stderr),
	); err != nil {
		// TODO: strip binary path as well?
		// We don't know if this is a system error or plugin error, so we assume system error
		return handlePotentialTooManyFilesError(err)
	}
	if getFeatureProto3OptionalSupported(protocVersion) {
		responseWriter.SetFeatureProto3Optional()
	}
	// We always claim support for all Editions in the response because the invocation to
	// "protoc" will fail if it can't handle the input editions. That way, we don't have to
	// track which protoc versions support which editions and synthesize this information.
	// And that also lets us support users passing "--experimental_editions" to protoc.
	responseWriter.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_PROTO2, descriptorpb.Edition_EDITION_MAX)

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
			responseWriter.AddFile(readObject.Path(), string(data))
			return nil
		},
	)
}

func (h *protocProxyHandler) getProtocVersion(
	ctx context.Context,
	pluginEnv protoplugin.PluginEnv,
) (*pluginpb.Version, error) {
	stdoutBuffer := bytes.NewBuffer(nil)
	if err := h.runner.Run(
		ctx,
		h.protocPath,
		command.RunWithArgs(slicesext.Concat(h.protocExtraArgs, []string{"--version"})...),
		command.RunWithEnviron(pluginEnv.Environ),
		command.RunWithStdout(stdoutBuffer),
	); err != nil {
		// TODO: strip binary path as well?
		return nil, handlePotentialTooManyFilesError(err)
	}
	return parseVersionForCLIVersion(strings.TrimSpace(stdoutBuffer.String()))
}
