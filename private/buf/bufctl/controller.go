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
	"io/fs"
	"net/http"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ImageWithConfig pairs an Image with lint and breaking configuration.
type ImageWithConfig interface {
	bufimage.Image
	LintConfig() bufconfig.LintConfig
	BreakingConfig() bufconfig.BreakingConfig

	isImageWithConfig()
}

// ProtoFileInfo is a minimal FileInfo that can be constructed from either
// a ModuleSet or an Image with no additional lazy calls.
//
// This is used by ls-files.
type ProtoFileInfo interface {
	storage.ObjectInfo

	ModuleFullName() bufmodule.ModuleFullName
	CommitID() string

	isProtoFileInfo()
}

type Controller interface {
	GetWorkspace(
		ctx context.Context,
		sourceOrModuleInput string,
		options ...FunctionOption,
	) (bufworkspace.Workspace, error)
	GetUpdateableWorkspace(
		ctx context.Context,
		dirPath string,
		options ...FunctionOption,
	) (bufworkspace.UpdateableWorkspace, error)
	GetImage(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) (bufimage.Image, error)
	GetImageForInputConfig(
		ctx context.Context,
		inputConfig bufconfig.InputConfig,
		options ...FunctionOption,
	) (bufimage.Image, error)
	GetImageForWorkspace(
		ctx context.Context,
		workspace bufworkspace.Workspace,
		options ...FunctionOption,
	) (bufimage.Image, error)
	GetTargetImageWithConfigs(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) ([]ImageWithConfig, error)
	// GetProtoFileInfos gets the .proto FileInfos for the given input.
	//
	// If WithFileInfosIncludeImports is set, imports are included, otherwise
	// just the targeted files are included.
	GetProtoFileInfos(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) ([]ProtoFileInfo, error)
	PutImage(
		ctx context.Context,
		imageOutput string,
		image bufimage.Image,
		options ...FunctionOption,
	) error
	GetMessage(
		ctx context.Context,
		schemaImage bufimage.Image,
		messageInput string,
		typeName string,
		defaultMessageEncoding buffetch.MessageEncoding,
		options ...FunctionOption,
	) (proto.Message, buffetch.MessageEncoding, error)
	PutMessage(
		ctx context.Context,
		schemaImage bufimage.Image,
		messageOutput string,
		message proto.Message,
		defaultMessageEncoding buffetch.MessageEncoding,
		options ...FunctionOption,
	) error
}

func NewController(
	logger *zap.Logger,
	tracer tracing.Tracer,
	container app.EnvStdioContainer,
	clientProvider bufapi.ClientProvider,
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (Controller, error) {
	return newController(
		logger,
		tracer,
		container,
		clientProvider,
		graphProvider,
		moduleKeyProvider,
		moduleDataProvider,
		commitProvider,
		httpClient,
		httpauthAuthenticator,
		gitClonerOptions,
		options...,
	)
}

/// *** PRIVATE ***

// In theory, we want to keep this separate from our global variables in bufcli.
//
// Originally, this was in a different package, and we want to keep the option to split
// it out again. The separation of concerns here is that the controller doesnt itself
// deal in the global variables.
type controller struct {
	logger             *zap.Logger
	tracer             tracing.Tracer
	container          app.EnvStdioContainer
	clientProvider     bufapi.ClientProvider
	moduleDataProvider bufmodule.ModuleDataProvider
	graphProvider      bufmodule.GraphProvider
	commitProvider     bufmodule.CommitProvider

	disableSymlinks           bool
	fileAnnotationErrorFormat string
	fileAnnotationsToStdout   bool

	commandRunner     command.Runner
	storageosProvider storageos.Provider
	buffetchRefParser buffetch.RefParser
	buffetchReader    buffetch.Reader
	buffetchWriter    buffetch.Writer
}

func newController(
	logger *zap.Logger,
	tracer tracing.Tracer,
	container app.EnvStdioContainer,
	clientProvider bufapi.ClientProvider,
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (*controller, error) {
	controller := &controller{
		logger:             logger,
		tracer:             tracer,
		container:          container,
		clientProvider:     clientProvider,
		graphProvider:      graphProvider,
		moduleDataProvider: moduleDataProvider,
		commitProvider:     commitProvider,
	}
	for _, option := range options {
		option(controller)
	}
	if err := validateFileAnnotationErrorFormat(controller.fileAnnotationErrorFormat); err != nil {
		return nil, err
	}
	controller.commandRunner = command.NewRunner()
	controller.storageosProvider = newStorageosProvider(controller.disableSymlinks)
	controller.buffetchRefParser = buffetch.NewRefParser(logger)
	controller.buffetchReader = buffetch.NewReader(
		logger,
		controller.storageosProvider,
		httpClient,
		httpauthAuthenticator,
		git.NewCloner(
			logger,
			tracer,
			controller.storageosProvider,
			controller.commandRunner,
			gitClonerOptions,
		),
		moduleKeyProvider,
	)
	controller.buffetchWriter = buffetch.NewWriter(logger)
	return controller, nil
}

func (c *controller) GetWorkspace(
	ctx context.Context,
	sourceOrModuleInput string,
	options ...FunctionOption,
) (_ bufworkspace.Workspace, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
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
		return c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
	case buffetch.SourceRef:
		return c.getWorkspaceForSourceRef(ctx, t, functionOptions)
	case buffetch.ModuleRef:
		return c.getWorkspaceForModuleRef(ctx, t, functionOptions)
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid SourceOrModuleRef: %T", sourceOrModuleRef)
	}
}

func (c *controller) GetUpdateableWorkspace(
	ctx context.Context,
	dirPath string,
	options ...FunctionOption,
) (_ bufworkspace.UpdateableWorkspace, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	dirRef, err := c.buffetchRefParser.GetDirRef(ctx, dirPath)
	if err != nil {
		return nil, err
	}
	return c.getUpdateableWorkspaceForDirRef(ctx, dirRef, functionOptions)
}

func (c *controller) GetImage(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (_ bufimage.Image, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	return c.getImage(ctx, input, functionOptions)
}

func (c *controller) GetImageForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
	options ...FunctionOption,
) (_ bufimage.Image, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	return c.getImageForInputConfig(ctx, inputConfig, functionOptions)
}

func (c *controller) GetImageForWorkspace(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	options ...FunctionOption,
) (_ bufimage.Image, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	return c.getImageForWorkspace(ctx, workspace, functionOptions)
}

func (c *controller) GetTargetImageWithConfigs(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (_ []ImageWithConfig, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, err
	}
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		workspace, err := c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.buildTargetImageWithConfigs(ctx, workspace, functionOptions)
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.buildTargetImageWithConfigs(ctx, workspace, functionOptions)
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.buildTargetImageWithConfigs(ctx, workspace, functionOptions)
	case buffetch.MessageRef:
		image, err := c.getImageForMessageRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		bucket, err := c.storageosProvider.NewReadWriteBucket(
			".",
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return nil, err
		}
		lintConfig := bufconfig.DefaultLintConfig
		breakingConfig := bufconfig.DefaultBreakingConfig
		bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefixOrOverride(
			ctx,
			bucket,
			".",
			functionOptions.configOverride,
		)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			// We did not find a buf.yaml in our current directory, and there was no config override.
			// Use the defaults.
		} else {
			switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
			case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
				moduleConfigs := bufYAMLFile.ModuleConfigs()
				if len(moduleConfigs) != 1 {
					return nil, fmt.Errorf("expected 1 ModuleConfig for FileVersion %v, got %d", len(moduleConfigs), fileVersion)
				}
				lintConfig = moduleConfigs[0].LintConfig()
				breakingConfig = moduleConfigs[0].BreakingConfig()
			case bufconfig.FileVersionV2:
				// Do nothing. Use the default LintConfig and BreakingConfig. With
				// the new buf.yamls with multiple modules, we don't know what lint or
				// breaking config to apply. TODO is this right?
			default:
				return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
			}
		}
		return []ImageWithConfig{
			newImageWithConfig(
				image,
				lintConfig,
				breakingConfig,
			),
		}, nil
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid Ref: %T", ref)
	}
}

func (c *controller) GetProtoFileInfos(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (_ []ProtoFileInfo, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	// We never care about SourceCodeInfo here.
	functionOptions.imageExcludeSourceInfo = true

	if functionOptions.protoFileInfosIncludeImports {
		// There are cleaner ways we could do this on per-ref basis, but this matches
		// what we did in the pre-buf-refactor, and it's simple and fine. We could
		// optimize this later if we really wanted.
		image, err := c.getImage(ctx, input, functionOptions)
		if err != nil {
			return nil, err
		}
		return getProtoFileInfosForImage(image)
	}
	// We now know that we don't want imports. Just get the targets. We set up
	// functionOptions to do this for images here too.
	functionOptions.imageExcludeImports = true

	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, err
	}
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		workspace, err := c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return getProtoFileInfosForModuleSet(ctx, workspace)
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return getProtoFileInfosForModuleSet(ctx, workspace)
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return getProtoFileInfosForModuleSet(ctx, workspace)
	case buffetch.MessageRef:
		image, err := c.getImageForMessageRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return getProtoFileInfosForImage(image)
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid Ref: %T", ref)
	}
}

func (c *controller) PutImage(
	ctx context.Context,
	imageOutput string,
	image bufimage.Image,
	options ...FunctionOption,
) (retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
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
	marshaler, err := newProtoencodingMarshaler(image, messageRef)
	if err != nil {
		return err
	}
	putImage, err := filterImage(image, functionOptions, false)
	if err != nil {
		return err
	}
	var putMessage proto.Message
	if functionOptions.imageAsFileDescriptorSet {
		putMessage = bufimage.ImageToFileDescriptorSet(putImage)
	} else {
		putMessage = bufimage.ImageToProtoImage(putImage)
	}
	data, err := marshaler.Marshal(putMessage)
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

func (c *controller) GetMessage(
	ctx context.Context,
	schemaImage bufimage.Image,
	messageInput string,
	typeName string,
	defaultMessageEncoding buffetch.MessageEncoding,
	options ...FunctionOption,
) (_ proto.Message, _ buffetch.MessageEncoding, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	messageRefParser := buffetch.NewMessageRefParser(
		c.logger,
		buffetch.MessageRefParserWithDefaultMessageEncoding(
			defaultMessageEncoding,
		),
	)
	messageRef, err := messageRefParser.GetMessageRef(ctx, messageInput)
	if err != nil {
		return nil, 0, err
	}
	messageEncoding := messageRef.MessageEncoding()
	if messageRef.IsNull() {
		return nil, messageEncoding, nil
	}
	resolver, err := protoencoding.NewResolver(
		bufimage.ImageToFileDescriptorProtos(schemaImage)...,
	)
	if err != nil {
		return nil, 0, err
	}
	var unmarshaler protoencoding.Unmarshaler
	switch messageEncoding {
	case buffetch.MessageEncodingBinpb:
		unmarshaler = protoencoding.NewWireUnmarshaler(resolver)
	case buffetch.MessageEncodingJSON:
		unmarshaler = protoencoding.NewJSONUnmarshaler(resolver)
	case buffetch.MessageEncodingTxtpb:
		unmarshaler = protoencoding.NewTxtpbUnmarshaler(resolver)
	case buffetch.MessageEncodingYAML:
		unmarshaler = protoencoding.NewYAMLUnmarshaler(
			resolver,
			protoencoding.YAMLUnmarshalerWithPath(messageRef.Path()),
		)
	default:
		// This is a system error.
		return nil, 0, syserror.Newf("unknown MessageEncoding: %v", messageEncoding)
	}
	readCloser, err := c.buffetchReader.GetMessageFile(ctx, c.container, messageRef)
	if err != nil {
		return nil, 0, err
	}
	data, err := ioext.ReadAllAndClose(readCloser)
	if err != nil {
		return nil, 0, err
	}
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("length of data read from %q was zero", messageInput)
	}
	message, err := bufreflect.NewMessage(ctx, schemaImage, typeName)
	if err != nil {
		return nil, 0, err
	}
	if err := unmarshaler.Unmarshal(data, message); err != nil {
		return nil, 0, err
	}
	return message, messageEncoding, nil
}

func (c *controller) PutMessage(
	ctx context.Context,
	schemaImage bufimage.Image,
	messageOutput string,
	message proto.Message,
	defaultMessageEncoding buffetch.MessageEncoding,
	options ...FunctionOption,
) (retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions()
	for _, option := range options {
		option(functionOptions)
	}
	messageRefParser := buffetch.NewMessageRefParser(
		c.logger,
		buffetch.MessageRefParserWithDefaultMessageEncoding(
			defaultMessageEncoding,
		),
	)
	messageRef, err := messageRefParser.GetMessageRef(ctx, messageOutput)
	if err != nil {
		return err
	}
	if messageRef.IsNull() {
		return nil
	}
	marshaler, err := newProtoencodingMarshaler(schemaImage, messageRef)
	if err != nil {
		return err
	}
	data, err := marshaler.Marshal(message)
	if err != nil {
		return err
	}
	writeCloser, err := c.buffetchWriter.PutMessageFile(ctx, c.container, messageRef)
	if err != nil {
		return err
	}
	_, err = writeCloser.Write(data)
	return multierr.Append(err, writeCloser.Close())
}

func (c *controller) getImage(
	ctx context.Context,
	input string,
	functionOptions *functionOptions,
) (bufimage.Image, error) {
	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, err
	}
	return c.getImageForRef(ctx, ref, functionOptions)
}

func (c *controller) getImageForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
	functionOptions *functionOptions,
) (bufimage.Image, error) {
	ref, err := c.buffetchRefParser.GetRefForInputConfig(ctx, inputConfig)
	if err != nil {
		return nil, err
	}
	return c.getImageForRef(ctx, ref, functionOptions)
}

func (c *controller) getImageForRef(
	ctx context.Context,
	ref buffetch.Ref,
	functionOptions *functionOptions,
) (bufimage.Image, error) {
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		workspace, err := c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.getImageForWorkspace(ctx, workspace, functionOptions)
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.getImageForWorkspace(ctx, workspace, functionOptions)
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.getImageForWorkspace(ctx, workspace, functionOptions)
	case buffetch.MessageRef:
		return c.getImageForMessageRef(ctx, t, functionOptions)
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid Ref: %T", ref)
	}
}

func (c *controller) getImageForWorkspace(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	functionOptions *functionOptions,
) (bufimage.Image, error) {
	if err := c.warnDeps(workspace); err != nil {
		return nil, err
	}
	return c.buildImage(
		ctx,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
		functionOptions,
	)
}

func (c *controller) getWorkspaceForProtoFileRef(
	ctx context.Context,
	protoFileRef buffetch.ProtoFileRef,
	functionOptions *functionOptions,
) (_ bufworkspace.Workspace, retErr error) {
	if len(functionOptions.targetPaths) > 0 {
		// Even though we didn't have an explicit error case, this never actually worked
		// properly in the pre-refactor buf CLI. We're going to call it unusable and this
		// not a breaking change - if anything, this is a bug fix.
		// TODO: Feed flag names through to here.
		return nil, fmt.Errorf("--path is not valid for use with .proto file references")
	}
	if len(functionOptions.targetExcludePaths) > 0 {
		// Even though we didn't have an explicit error case, this never actually worked
		// properly in the pre-refactor buf CLI. We're going to call it unusable and this
		// not a breaking change - if anything, this is a bug fix.
		// TODO: Feed flag names through to here.
		return nil, fmt.Errorf("--exclude-path is not valid for use with .proto file references")
	}
	readBucketCloser, err := c.buffetchReader.GetSourceReadBucketCloser(
		ctx,
		c.container,
		protoFileRef,
		functionOptions.getGetBucketOptions()...,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	// The ProtoFilePath is still relative to the input bucket, not the bucket
	// retrieved from buffetch. Treat the path just as we do with targetPaths
	// and externalPaths in withPathsForBucketExtender.
	protoFilePath, err := readBucketCloser.PathForExternalPath(protoFileRef.ProtoFilePath())
	if err != nil {
		return nil, err
	}
	return bufworkspace.NewWorkspaceForBucket(
		ctx,
		c.logger,
		c.tracer,
		readBucketCloser,
		c.clientProvider,
		c.moduleDataProvider,
		bufworkspace.WithTargetSubDirPath(
			readBucketCloser.SubDirPath(),
		),
		bufworkspace.WithProtoFileTargetPath(
			protoFilePath,
			protoFileRef.IncludePackageFiles(),
		),
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
		),
	)
}

func (c *controller) getWorkspaceForSourceRef(
	ctx context.Context,
	sourceRef buffetch.SourceRef,
	functionOptions *functionOptions,
) (_ bufworkspace.Workspace, retErr error) {
	readBucketCloser, err := c.buffetchReader.GetSourceReadBucketCloser(
		ctx,
		c.container,
		sourceRef,
		functionOptions.getGetBucketOptions()...,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	functionOptions, err = functionOptions.withPathsForBucketExtender(readBucketCloser)
	if err != nil {
		return nil, err
	}
	return bufworkspace.NewWorkspaceForBucket(
		ctx,
		c.logger,
		c.tracer,
		readBucketCloser,
		c.clientProvider,
		c.moduleDataProvider,
		bufworkspace.WithTargetSubDirPath(
			readBucketCloser.SubDirPath(),
		),
		bufworkspace.WithTargetPaths(
			functionOptions.targetPaths,
			functionOptions.targetExcludePaths,
		),
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
		),
	)
}

func (c *controller) getUpdateableWorkspaceForDirRef(
	ctx context.Context,
	dirRef buffetch.DirRef,
	functionOptions *functionOptions,
) (_ bufworkspace.UpdateableWorkspace, retErr error) {
	readWriteBucket, err := c.buffetchReader.GetDirReadWriteBucket(
		ctx,
		c.container,
		dirRef,
		functionOptions.getGetBucketOptions()...,
	)
	if err != nil {
		return nil, err
	}
	functionOptions, err = functionOptions.withPathsForBucketExtender(readWriteBucket)
	if err != nil {
		return nil, err
	}
	return bufworkspace.NewUpdateableWorkspaceForBucket(
		ctx,
		c.logger,
		c.tracer,
		readWriteBucket,
		c.clientProvider,
		c.moduleDataProvider,
		bufworkspace.WithTargetSubDirPath(
			readWriteBucket.SubDirPath(),
		),
		bufworkspace.WithTargetPaths(
			functionOptions.targetPaths,
			functionOptions.targetExcludePaths,
		),
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
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
	return bufworkspace.NewWorkspaceForModuleKey(
		ctx,
		c.logger,
		c.tracer,
		moduleKey,
		c.graphProvider,
		c.moduleDataProvider,
		bufworkspace.WithTargetPaths(
			functionOptions.targetPaths,
			functionOptions.targetExcludePaths,
		),
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
		),
	)
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
		resolver, err := bootstrapResolver(protoencoding.NewJSONUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewJSONUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingTxtpb:
		resolver, err := bootstrapResolver(protoencoding.NewTxtpbUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewTxtpbUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	case buffetch.MessageEncodingYAML:
		resolver, err := bootstrapResolver(protoencoding.NewYAMLUnmarshaler(nil), data)
		if err != nil {
			return nil, err
		}
		if err := protoencoding.NewYAMLUnmarshaler(resolver).Unmarshal(data, protoImage); err != nil {
			return nil, err
		}
		// we've already re-parsed, by unmarshalling 2x above
		imageFromProtoOptions = append(imageFromProtoOptions, bufimage.WithNoReparse())
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown MessageEncoding: %v", messageEncoding)
	}

	if functionOptions.imageExcludeSourceInfo {
		for _, fileDescriptorProto := range protoImage.File {
			fileDescriptorProto.SourceCodeInfo = nil
		}
	}

	image, err := bufimage.NewImageForProto(protoImage, imageFromProtoOptions...)
	if err != nil {
		return nil, err
	}
	return filterImage(image, functionOptions, false)
}

func (c *controller) buildImage(
	ctx context.Context,
	moduleReadBucket bufmodule.ModuleReadBucket,
	functionOptions *functionOptions,
) (bufimage.Image, error) {
	var options []bufimage.BuildImageOption
	if functionOptions.imageExcludeSourceInfo {
		options = append(options, bufimage.WithExcludeSourceCodeInfo())
	}
	image, err := bufimage.BuildImage(
		ctx,
		c.tracer,
		moduleReadBucket,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return filterImage(image, functionOptions, true)
}

func (c *controller) buildTargetImageWithConfigs(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	functionOptions *functionOptions,
) ([]ImageWithConfig, error) {
	if err := c.warnDeps(workspace); err != nil {
		return nil, err
	}
	modules := bufmodule.ModuleSetTargetModules(workspace)
	imageWithConfigs := make([]ImageWithConfig, 0, len(modules))
	for _, module := range modules {
		c.logger.Debug(
			"building image for target module",
			zap.String("moduleOpaqueID", module.OpaqueID()),
		)
		opaqueID := module.OpaqueID()
		// We need to make sure that all dependencies are non-targets, so that they
		// end up as imports in the resulting image.
		moduleSet, err := workspace.WithTargetOpaqueIDs(opaqueID)
		if err != nil {
			return nil, err
		}
		module := moduleSet.GetModuleForOpaqueID(opaqueID)
		if module == nil {
			return nil, syserror.Newf("new ModuleSet from WithTargetOpaqueIDs did not have opaqueID %q", opaqueID)
		}
		moduleReadBucket, err := bufmodule.ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles(module)
		if err != nil {
			return nil, err
		}
		targetFileInfos, err := bufmodule.GetTargetFileInfos(ctx, moduleReadBucket)
		if err != nil {
			return nil, err
		}
		// This may happen after path targeting. We may have a Module that itself was targeted,
		// but no target files remain. In this case, this isn't a target image.
		//
		// TODO: without allowNotExist, this results in silent behavior when --path is incorrect.
		if len(targetFileInfos) == 0 {
			continue
		}
		image, err := c.buildImage(
			ctx,
			moduleReadBucket,
			functionOptions,
		)
		if err != nil {
			return nil, err
		}
		imageWithConfigs = append(
			imageWithConfigs,
			newImageWithConfig(
				image,
				workspace.GetLintConfigForOpaqueID(module.OpaqueID()),
				workspace.GetBreakingConfigForOpaqueID(module.OpaqueID()),
			),
		)
	}
	if len(imageWithConfigs) == 0 {
		// If we had no target modules, or no target files within the modules after path filtering, this is an error.
		// We could have a better user error than this. This gets back to the lack of allowNotExist.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	return imageWithConfigs, nil
}

// warnDeps warns on either unused deps in your buf.yaml, or transitive deps that were
// not in your buf.yaml.
//
// Only call this if you are building an image. This results in ModuleDeps calls that
// you don't want to invoke unless you are building - they'll result in import reading,
// which can cause issues. If this happens for all workspaces, you'll see integration
// test errors, and correctly so. In the pre-refactor world, we only did this with
// image building, so we keep it that way for now.
func (c *controller) warnDeps(workspace bufworkspace.Workspace) error {
	malformedDeps, err := bufworkspace.MalformedDepsForWorkspace(workspace)
	if err != nil {
		return err
	}
	for _, malformedDep := range malformedDeps {
		switch t := malformedDep.Type(); t {
		case bufworkspace.MalformedDepTypeUndeclared:
			c.logger.Sugar().Warnf(
				"Module %s is a transitive remote dependency not declared in your buf.yaml deps. Add %s to your deps.",
				malformedDep.ModuleFullName(),
				malformedDep.ModuleFullName(),
			)
		case bufworkspace.MalformedDepTypeUnused:
			if workspace.GetModuleForModuleFullName(malformedDep.ModuleFullName()) != nil {
				c.logger.Sugar().Warnf(
					`Module %s is declared in your buf.yaml deps but is a module in your workspace. Declaring a dep within your workspace has no effect.`,
					malformedDep.ModuleFullName(),
				)
			} else {
				c.logger.Sugar().Warnf(
					`Module %s is declared in your buf.yaml deps but is unused.`,
					malformedDep.ModuleFullName(),
				)
			}
		default:
			return fmt.Errorf("unknown MalformedDepType: %v", t)
		}
	}
	return nil
}

// handleFileAnnotationSetError will attempt to handle the error as a FileAnnotationSet, and if so, print
// the FileAnnotationSet to the writer with the given error format while returning ErrFileAnnotation.
//
// Otherwise, the original error is returned.
func (c *controller) handleFileAnnotationSetRetError(retErrAddr *error) {
	if *retErrAddr == nil {
		return
	}
	var fileAnnotationSet bufanalysis.FileAnnotationSet
	if errors.As(*retErrAddr, &fileAnnotationSet) {
		writer := c.container.Stderr()
		if c.fileAnnotationsToStdout {
			writer = c.container.Stdout()
		}
		if err := bufanalysis.PrintFileAnnotationSet(
			writer,
			fileAnnotationSet,
			c.fileAnnotationErrorFormat,
		); err != nil {
			*retErrAddr = err
			return
		}
		*retErrAddr = ErrFileAnnotation
	}
}

// We expect that we only want target files when we call this.
func getProtoFileInfosForModuleSet(ctx context.Context, moduleSet bufmodule.ModuleSet) ([]ProtoFileInfo, error) {
	targetFileInfos, err := bufmodule.GetTargetFileInfos(
		ctx,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
	)
	if err != nil {
		return nil, err
	}
	return slicesext.Map(
		targetFileInfos,
		func(fileInfo bufmodule.FileInfo) ProtoFileInfo {
			return newModuleProtoFileInfo(fileInfo)
		},
	), nil
}

// Any import filtering is expected to be done before this.
func getProtoFileInfosForImage(image bufimage.Image) ([]ProtoFileInfo, error) {
	return slicesext.Map(
		image.Files(),
		func(imageFile bufimage.ImageFile) ProtoFileInfo {
			return newImageProtoFileInfo(imageFile)
		},
	), nil
}

func bootstrapResolver(
	unmarshaler protoencoding.Unmarshaler,
	data []byte,
) (protoencoding.Resolver, error) {
	firstProtoImage := &imagev1.Image{}
	if err := unmarshaler.Unmarshal(data, firstProtoImage); err != nil {
		return nil, err
	}
	return protoencoding.NewResolver(firstProtoImage.File...)
}

// WE DO NOT FILTER IF WE ALREADY FILTERED ON BUILDING OF A WORKSPACE
// Also, paths are still external paths at this point if this came from a workspace
// TODO: redo functionOptions, this is a mess
func filterImage(
	image bufimage.Image,
	functionOptions *functionOptions,
	imageCameFromAWorkspace bool,
) (bufimage.Image, error) {
	newImage := image
	var err error
	if functionOptions.imageExcludeImports {
		newImage = bufimage.ImageWithoutImports(newImage)
	}
	if len(functionOptions.imageTypes) > 0 {
		newImage, err = bufimageutil.ImageFilteredByTypes(newImage, functionOptions.imageTypes...)
		if err != nil {
			return nil, err
		}
	}
	if !imageCameFromAWorkspace {
		if len(functionOptions.targetPaths) > 0 || len(functionOptions.targetExcludePaths) > 0 {
			// TODO: allowNotExist?
			// TODO: Also, does this affect lint or breaking?
			newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
				newImage,
				functionOptions.targetPaths,
				functionOptions.targetExcludePaths,
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return newImage, nil
}

func newStorageosProvider(disableSymlinks bool) storageos.Provider {
	var options []storageos.ProviderOption
	if !disableSymlinks {
		options = append(options, storageos.ProviderWithSymlinks())
	}
	return storageos.NewProvider(options...)
}

func newProtoencodingMarshaler(
	image bufimage.Image,
	messageRef buffetch.MessageRef,
) (protoencoding.Marshaler, error) {
	switch messageEncoding := messageRef.MessageEncoding(); messageEncoding {
	case buffetch.MessageEncodingBinpb:
		return protoencoding.NewWireMarshaler(), nil
	case buffetch.MessageEncodingJSON:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return newJSONMarshaler(resolver, messageRef), nil
	case buffetch.MessageEncodingTxtpb:
		// TODO: verify that image is complete
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return protoencoding.NewTxtpbMarshaler(resolver), nil
	case buffetch.MessageEncodingYAML:
		resolver, err := protoencoding.NewResolver(bufimage.ImageToFileDescriptorProtos(image)...)
		if err != nil {
			return nil, err
		}
		return newYAMLMarshaler(resolver, messageRef), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown MessageEncoding: %v", messageEncoding)
	}
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

func validateFileAnnotationErrorFormat(fileAnnotationErrorFormat string) error {
	if fileAnnotationErrorFormat == "" {
		return nil
	}
	for _, formatString := range bufanalysis.AllFormatStrings {
		if fileAnnotationErrorFormat == formatString {
			return nil
		}
	}
	// TODO: get standard flag names and bindings into this package.
	fileAnnotationErrorFormatFlagName := "error-format"
	return appcmd.NewInvalidArgumentErrorf("--%s: invalid format: %q", fileAnnotationErrorFormatFlagName, fileAnnotationErrorFormat)
}
