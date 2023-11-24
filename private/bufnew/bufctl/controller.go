// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/proto"
)

type controller struct {
	container          Container
	moduleDataProvider bufmodule.ModuleDataProvider

	disableSymlinks bool

	commandRunner        command.Runner
	storageosProvider    storageos.Provider
	buffetchRefParser    buffetch.RefParser
	buffetchReader       buffetch.Reader
	buffetchWriter       buffetch.Writer
	bufimagebuildBuilder bufimagebuild.Builder
}

func newController(
	container Container,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) *controller {
	controller := &controller{
		container:          container,
		moduleDataProvider: moduleDataProvider,
	}
	for _, option := range options {
		option(controller)
	}
	controller.commandRunner = command.NewRunner()
	controller.storageosProvider = newStorageosProvider(controller.disableSymlinks)
	controller.buffetchRefParser = buffetch.NewRefParser(container.Logger())
	controller.buffetchReader = buffetch.NewReader(
		container.Logger(),
		controller.storageosProvider,
		httpClient,
		httpauthAuthenticator,
		git.NewCloner(
			container.Logger(),
			controller.storageosProvider,
			controller.commandRunner,
			gitClonerOptions,
		),
		moduleKeyProvider,
	)
	controller.buffetchWriter = buffetch.NewWriter(container.Logger())
	controller.bufimagebuildBuilder = bufimagebuild.NewBuilder(container.Logger())
	return controller
}

func (c *controller) GetWorkspace(
	ctx context.Context,
	sourceOrModuleInput string,
	options ...FunctionOption,
) (bufworkspace.Workspace, error) {
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	sourceOrModuleRef, err := c.buffetchRefParser.GetSourceOrModuleRef(ctx, sourceOrModuleInput)
	if err != nil {
		return nil, err
	}
	switch t := sourceOrModuleRef.(type) {
	case buffetch.ProtoFileRef:
		return nil, errors.New("TODO")
	case buffetch.SourceRef:
		return c.getWorkspaceForSourceRef(ctx, t, functionOptions)
	case buffetch.ModuleRef:
		return c.getWorkspaceForModuleRef(ctx, t, functionOptions)
	default:
		// This is a system error.
		return nil, fmt.Errorf("invalid SourceOrModuleRef: %T", sourceOrModuleRef)
	}
}

func (c *controller) GetImage(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		return nil, nil, errors.New("TODO")
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
		return c.buildImage(ctx, workspace, functionOptions)
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
		return c.buildImage(ctx, workspace, functionOptions)
	case buffetch.MessageRef:
		image, err := c.getImageForMessageRef(ctx, t, functionOptions)
		return image, nil, err
	default:
		// This is a system error.
		return nil, nil, fmt.Errorf("invalid Ref: %T", ref)
	}
}

func (c *controller) PutImage(
	ctx context.Context,
	container app.EnvStdoutContainer,
	imageOutput string,
	image bufimage.Image,
	options ...FunctionOption,
) (retErr error) {
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	messageRef, err := c.buffetchRefParser.GetMessageRef(ctx, imageOutput)
	if err != nil {
		return err
	}
	// Stop short for performance.
	if messageRef.IsNull() {
		return nil
	}
	writeImage := image
	if functionOptions.excludeImports {
		writeImage = bufimage.ImageWithoutImports(image)
	}
	var message proto.Message
	if functionOptions.asFileDescriptorSet {
		message = bufimage.ImageToFileDescriptorSet(writeImage)
	} else {
		message = bufimage.ImageToProtoImage(writeImage)
	}
	data, err := c.marshalImage(ctx, message, image, messageRef)
	if err != nil {
		return err
	}
	writeCloser, err := c.buffetchWriter.PutMessageFile(ctx, c.container, messageRef)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, writeCloser.Close())
	}()
	_, err = writeCloser.Write(data)
	return err
}

func (c *controller) getWorkspaceForSourceRef(
	ctx context.Context,
	sourceRef buffetch.SourceRef,
	functionOptions *functionOptions,
) (_ bufworkspace.Workspace, retErr error) {
	readBucketCloser, err := c.buffetchReader.GetSourceBucket(ctx, c.container, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	functionOptions, err = functionOptions.withPathsForReadBucketCloser(readBucketCloser)
	if err != nil {
		return nil, err
	}
	return bufworkspace.NewWorkspaceForBucket(
		ctx,
		readBucketCloser,
		c.moduleDataProvider,
		bufworkspace.WorkspaceWithTargetSubDirPath(
			readBucketCloser.SubDirPath(),
		),
		bufworkspace.WorkspaceWithTargetPaths(
			functionOptions.targetPaths,
			functionOptions.targetExcludePaths,
		),
	)
}

func (c *controller) getWorkspaceForModuleRef(
	ctx context.Context,
	moduleRef buffetch.ModuleRef,
	functionOptions *functionOptions,
) (bufworkspace.Workspace, error) {
	moduleKey, err := c.buffetchReader.GetModuleKey(ctx, c.container, moduleRef)
	if err != nil {
		return nil, err
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, c.moduleDataProvider)
	moduleSetBuilder.AddRemoteModule(
		moduleKey,
		true,
		bufmodule.RemoteModuleWithTargetPaths(
			functionOptions.targetPaths,
			functionOptions.targetExcludePaths,
		),
	)
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return bufworkspace.NewWorkspaceForModuleSet(moduleSet)
}

func (c *controller) getImageForMessageRef(
	ctx context.Context,
	messageRef buffetch.MessageRef,
	functionOptions *functionOptions,
) (_ bufimage.Image, retErr error) {
	readCloser, err := c.buffetchReader.GetMessageFile(ctx, c.container, messageRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := io.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}

	protoImage := &imagev1.Image{}
	var imageFromProtoOptions []bufimage.NewImageForProtoOption
	switch messageEncoding := messageRef.MessageEncoding(); messageEncoding {
	// we have to double parse due to custom options
	// See https://github.com/golang/protobuf/issues/1123
	case buffetch.MessageEncodingBinpb:
		if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
	case buffetch.MessageEncodingJSON:
		resolver, err := c.bootstrapResolver(ctx, protoencoding.NewJSONUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewJSONUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingTxtpb:
		resolver, err := c.bootstrapResolver(ctx, protoencoding.NewTxtpbUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewTxtpbUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingYAML:
		resolver, err := c.bootstrapResolver(ctx, protoencoding.NewYAMLUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewYAMLUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	default:
		return nil, err
	}
	if functionOptions.excludeSourceInfo {
		for _, fileDescriptorProto := range protoImage.File {
			fileDescriptorProto.SourceCodeInfo = nil
		}
	}

	image, err := bufimage.NewImageForProto(protoImage, imageFromProtoOptions...)
	if err != nil {
		return nil, err
	}
	// TODO: allowNotExist?
	return bufimage.ImageWithOnlyPathsAllowNotExist(
		image,
		functionOptions.targetPaths,
		functionOptions.targetExcludePaths,
	)
}

func (c *controller) buildImage(
	ctx context.Context,
	moduleSet bufmodule.ModuleSet,
	functionOptions *functionOptions,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	var options []bufimagebuild.BuildOption
	if functionOptions.excludeSourceInfo {
		options = append(options, bufimagebuild.WithExcludeSourceCodeInfo())
	}
	return c.bufimagebuildBuilder.Build(
		ctx,
		moduleSet,
		options...,
	)
}

func (c *controller) bootstrapResolver(
	ctx context.Context,
	unresolving protoencoding.Unmarshaler,
	data []byte,
) (protoencoding.Resolver, error) {
	firstProtoImage := &imagev1.Image{}
	if err := unresolving.Unmarshal(data, firstProtoImage); err != nil {
		return nil, err
	}
	return protoencoding.NewResolver(firstProtoImage.File...)
}

func (c *controller) marshalImage(
	ctx context.Context,
	message proto.Message,
	image bufimage.Image,
	messageRef buffetch.MessageRef,
) ([]byte, error) {
	switch messageEncoding := messageRef.MessageEncoding(); messageEncoding {
	case buffetch.MessageEncodingBinpb:
		return protoencoding.NewWireMarshaler().Marshal(message)
	case buffetch.MessageEncodingJSON:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return newJSONMarshaler(resolver, messageRef).Marshal(message)
	case buffetch.MessageEncodingTxtpb:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return protoencoding.NewTxtpbMarshaler(resolver).Marshal(message)
	case buffetch.MessageEncodingYAML:
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return newYAMLMarshaler(resolver, messageRef).Marshal(message)
	default:
		// This is a system error.
		return nil, fmt.Errorf("unknown MessageEncoding: %v", messageEncoding)
	}
}

func newStorageosProvider(disableSymlinks bool) storageos.Provider {
	var options []storageos.ProviderOption
	if !disableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	return storageos.NewProvider(options...)
}

func newJSONMarshaler(
	resolver protoencoding.Resolver,
	messageRef buffetch.MessageRef,
) protoencoding.Marshaler {
	jsonMarshalerOptions := []protoencoding.JSONMarshalerOption{
		//protoencoding.JSONMarshalerWithIndent(),
	}
	if messageRef.UseProtoNames() {
		jsonMarshalerOptions = append(
			jsonMarshalerOptions,
			protoencoding.JSONMarshalerWithUseProtoNames(),
		)
	}
	if messageRef.UseEnumNumbers() {
		jsonMarshalerOptions = append(
			jsonMarshalerOptions,
			protoencoding.JSONMarshalerWithUseEnumNumbers(),
		)
	}
	return protoencoding.NewJSONMarshaler(resolver, jsonMarshalerOptions...)
}

func newYAMLMarshaler(
	resolver protoencoding.Resolver,
	messageRef buffetch.MessageRef,
) protoencoding.Marshaler {
	yamlMarshalerOptions := []protoencoding.YAMLMarshalerOption{
		protoencoding.YAMLMarshalerWithIndent(),
	}
	if messageRef.UseProtoNames() {
		yamlMarshalerOptions = append(
			yamlMarshalerOptions,
			protoencoding.YAMLMarshalerWithUseProtoNames(),
		)
	}
	if messageRef.UseEnumNumbers() {
		yamlMarshalerOptions = append(
			yamlMarshalerOptions,
			protoencoding.YAMLMarshalerWithUseEnumNumbers(),
		)
	}
	return protoencoding.NewYAMLMarshaler(resolver, yamlMarshalerOptions...)
}

type functionOptions struct {
	targetPaths         []string
	targetExcludePaths  []string
	excludeImports      bool
	excludeSourceInfo   bool
	asFileDescriptorSet bool
}

func newFunctionOptions() *functionOptions {
	return &functionOptions{}
}

func (f *functionOptions) withPathsForReadBucketCloser(
	readBucketCloser buffetch.ReadBucketCloser,
) (*functionOptions, error) {
	c := &functionOptions{
		targetPaths:         f.targetPaths,
		targetExcludePaths:  f.targetExcludePaths,
		excludeImports:      f.excludeImports,
		excludeSourceInfo:   f.excludeSourceInfo,
		asFileDescriptorSet: f.asFileDescriptorSet,
	}
	for _, targetPath := range c.targetPaths {
		targetPath, err := readBucketCloser.PathForExternalPath(targetPath)
		if err != nil {
			return nil, err
		}
		c.targetPaths = append(c.targetPaths, targetPath)
	}
	for _, targetExcludePath := range c.targetExcludePaths {
		targetExcludePath, err := readBucketCloser.PathForExternalPath(targetExcludePath)
		if err != nil {
			return nil, err
		}
		c.targetExcludePaths = append(c.targetExcludePaths, targetExcludePath)
	}
	return c, nil
}
