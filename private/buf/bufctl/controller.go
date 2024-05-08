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

package bufctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sort"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwkt/bufwktstore"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protovalidate-go"
	"github.com/bufbuild/protoyaml-go"
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

type Controller interface {
	GetWorkspace(
		ctx context.Context,
		sourceOrModuleInput string,
		options ...FunctionOption,
	) (bufworkspace.Workspace, error)
	GetWorkspaceDepManager(
		ctx context.Context,
		dirPath string,
		options ...FunctionOption,
	) (bufworkspace.WorkspaceDepManager, error)
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
	// GetImportableImageFileInfos gets the importable .proto FileInfos for the given input.
	//
	// This includes all files that can be possible imported. For example, if a Module
	// is given, this will return FileInfos for the Module, its dependencies, and all of
	// the Well-Known Types.
	//
	// Returned ImageFileInfos are sorted by Path.
	GetImportableImageFileInfos(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) ([]bufimage.ImageFileInfo, error)
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
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	wktStore bufwktstore.Store,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (Controller, error) {
	return newController(
		logger,
		tracer,
		container,
		graphProvider,
		moduleKeyProvider,
		moduleDataProvider,
		commitProvider,
		wktStore,
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
	moduleDataProvider bufmodule.ModuleDataProvider
	graphProvider      bufmodule.GraphProvider
	commitProvider     bufmodule.CommitProvider
	wktStore           bufwktstore.Store

	disableSymlinks           bool
	fileAnnotationErrorFormat string
	fileAnnotationsToStdout   bool
	copyToInMemory            bool

	commandRunner               command.Runner
	storageosProvider           storageos.Provider
	buffetchRefParser           buffetch.RefParser
	buffetchReader              buffetch.Reader
	buffetchWriter              buffetch.Writer
	workspaceProvider           bufworkspace.WorkspaceProvider
	workspaceDepManagerProvider bufworkspace.WorkspaceDepManagerProvider
}

func newController(
	logger *zap.Logger,
	tracer tracing.Tracer,
	container app.EnvStdioContainer,
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	wktStore bufwktstore.Store,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (*controller, error) {
	controller := &controller{
		logger:             logger,
		tracer:             tracer,
		container:          container,
		graphProvider:      graphProvider,
		moduleDataProvider: moduleDataProvider,
		commitProvider:     commitProvider,
		wktStore:           wktStore,
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
	controller.workspaceProvider = bufworkspace.NewWorkspaceProvider(
		logger,
		tracer,
		graphProvider,
		moduleDataProvider,
		commitProvider,
	)
	controller.workspaceDepManagerProvider = bufworkspace.NewWorkspaceDepManagerProvider(
		logger,
		tracer,
	)
	return controller, nil
}

func (c *controller) GetWorkspace(
	ctx context.Context,
	sourceOrModuleInput string,
	options ...FunctionOption,
) (_ bufworkspace.Workspace, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
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

func (c *controller) GetWorkspaceDepManager(
	ctx context.Context,
	dirPath string,
	options ...FunctionOption,
) (_ bufworkspace.WorkspaceDepManager, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	dirRef, err := c.buffetchRefParser.GetDirRef(ctx, dirPath)
	if err != nil {
		return nil, err
	}
	return c.getWorkspaceDepManagerForDirRef(ctx, dirRef, functionOptions)
}

func (c *controller) GetImage(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (_ bufimage.Image, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
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
	functionOptions := newFunctionOptions(c)
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
	functionOptions := newFunctionOptions(c)
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
	functionOptions := newFunctionOptions(c)
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
		lintConfig := bufconfig.DefaultLintConfigV1
		breakingConfig := bufconfig.DefaultBreakingConfigV1
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
				// breaking config to apply.
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

func (c *controller) GetImportableImageFileInfos(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) (_ []bufimage.ImageFileInfo, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	// We never care about SourceCodeInfo here.
	functionOptions.imageExcludeSourceInfo = true
	// We always want to include imports for images.
	functionOptions.imageExcludeImports = false

	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, err
	}
	var imageFileInfos []bufimage.ImageFileInfo
	// For images, we want to get a bucket with no local paths. For everything else,
	// we want to get a bucket with local paths. datawkt.ReadBucket will result in
	// no local paths, otherwise we get the bucket from the bufwktstore.Store to
	// get local paths.
	wktBucket := datawkt.ReadBucket
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		workspace, err := c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		imageFileInfos, err = getImageFileInfosForModuleSet(ctx, workspace)
		if err != nil {
			return nil, err
		}
		wktBucket, err = c.wktStore.GetBucket(ctx)
		if err != nil {
			return nil, err
		}
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		imageFileInfos, err = getImageFileInfosForModuleSet(ctx, workspace)
		if err != nil {
			return nil, err
		}
		wktBucket, err = c.wktStore.GetBucket(ctx)
		if err != nil {
			return nil, err
		}
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		imageFileInfos, err = getImageFileInfosForModuleSet(ctx, workspace)
		if err != nil {
			return nil, err
		}
		wktBucket, err = c.wktStore.GetBucket(ctx)
		if err != nil {
			return nil, err
		}
	case buffetch.MessageRef:
		image, err := c.getImageForMessageRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		imageFiles := image.Files()
		imageFileInfos = make([]bufimage.ImageFileInfo, len(imageFiles))
		for i, imageFile := range imageFiles {
			imageFileInfos[i] = imageFile
		}
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid Ref: %T", ref)
	}
	imageFileInfos, err = bufimage.AppendWellKnownTypeImageFileInfos(
		ctx,
		wktBucket,
		imageFileInfos,
	)
	if err != nil {
		return nil, err
	}
	sort.Slice(
		imageFileInfos,
		func(i int, j int) bool {
			return imageFileInfos[i].Path() < imageFileInfos[j].Path()
		},
	)
	return imageFileInfos, nil
}

func (c *controller) PutImage(
	ctx context.Context,
	imageOutput string,
	image bufimage.Image,
	options ...FunctionOption,
) (retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	// Must be messageRefParser NOT c.buffetchRefParser as a NewMessageRefParser
	// defaults to a defaultMessageEncoding and not dir.
	messageRefParser := buffetch.NewMessageRefParser(c.logger)
	messageRef, err := messageRefParser.GetMessageRef(ctx, imageOutput)
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
		putMessage, err = bufimage.ImageToProtoImage(putImage)
		if err != nil {
			return err
		}
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
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	// Must be messageRefParser NOT c.buffetchRefParser as a NewMessageRefParser
	// defaults to a defaultMessageEncoding and not dir.
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
	var validator protoyaml.Validator
	if functionOptions.messageValidation {
		var err error
		validator, err = protovalidate.New()
		if err != nil {
			return nil, 0, err
		}
	}
	var unmarshaler protoencoding.Unmarshaler
	switch messageEncoding {
	case buffetch.MessageEncodingBinpb:
		unmarshaler = protoencoding.NewWireUnmarshaler(schemaImage.Resolver())
	case buffetch.MessageEncodingJSON:
		unmarshaler = protoencoding.NewJSONUnmarshaler(schemaImage.Resolver())
	case buffetch.MessageEncodingTxtpb:
		unmarshaler = protoencoding.NewTxtpbUnmarshaler(schemaImage.Resolver())
	case buffetch.MessageEncodingYAML:
		unmarshaler = protoencoding.NewYAMLUnmarshaler(
			schemaImage.Resolver(),
			protoencoding.YAMLUnmarshalerWithPath(messageRef.Path()),
			// This will pretty print validation errors.
			protoencoding.YAMLUnmarshalerWithValidator(validator),
		)
		validator = nil // Validation errors are handled by the unmarshaler.
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
	if validator != nil {
		if err := validator.Validate(message); err != nil {
			return nil, 0, err
		}
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
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	// Must be messageRefParser NOT c.buffetchRefParser as a NewMessageRefParser
	// defaults to a defaultMessageEncoding and not dir.
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
	image, err := c.buildImage(
		ctx,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
		functionOptions,
	)
	if err != nil {
		return nil, err
	}
	if err := c.warnUnconfiguredTransitiveImports(ctx, workspace, image); err != nil {
		return nil, err
	}
	return image, nil
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
		// TODO FUTURE: Feed flag names through to here.
		return nil, fmt.Errorf("--path is not valid for use with .proto file references")
	}
	if len(functionOptions.targetExcludePaths) > 0 {
		// Even though we didn't have an explicit error case, this never actually worked
		// properly in the pre-refactor buf CLI. We're going to call it unusable and this
		// not a breaking change - if anything, this is a bug fix.
		// TODO FUTURE: Feed flag names through to here.
		return nil, fmt.Errorf("--exclude-path is not valid for use with .proto file references")
	}
	readBucketCloser, bucketTargeting, err := c.buffetchReader.GetSourceReadBucketCloser(
		ctx,
		c.container,
		protoFileRef,
		functionOptions.getGetReadBucketCloserOptions()...,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	options := []bufworkspace.WorkspaceBucketOption{
		bufworkspace.WithProtoFileTargetPath(
			protoFileRef.ProtoFilePath(),
			protoFileRef.IncludePackageFiles(),
		),
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
		),
	}
	if functionOptions.ignoreAndDisallowV1BufWorkYAMLs {
		options = append(
			options,
			bufworkspace.WithIgnoreAndDisallowV1BufWorkYAMLs(),
		)
	}
	return c.workspaceProvider.GetWorkspaceForBucket(
		ctx,
		readBucketCloser,
		bucketTargeting,
		options...,
	)
}

func (c *controller) getWorkspaceForSourceRef(
	ctx context.Context,
	sourceRef buffetch.SourceRef,
	functionOptions *functionOptions,
) (_ bufworkspace.Workspace, retErr error) {
	readBucketCloser, bucketTargeting, err := c.buffetchReader.GetSourceReadBucketCloser(
		ctx,
		c.container,
		sourceRef,
		functionOptions.getGetReadBucketCloserOptions()...,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	options := []bufworkspace.WorkspaceBucketOption{
		bufworkspace.WithConfigOverride(
			functionOptions.configOverride,
		),
	}
	if functionOptions.ignoreAndDisallowV1BufWorkYAMLs {
		options = append(
			options,
			bufworkspace.WithIgnoreAndDisallowV1BufWorkYAMLs(),
		)
	}
	return c.workspaceProvider.GetWorkspaceForBucket(
		ctx,
		readBucketCloser,
		bucketTargeting,
		options...,
	)
}

func (c *controller) getWorkspaceDepManagerForDirRef(
	ctx context.Context,
	dirRef buffetch.DirRef,
	functionOptions *functionOptions,
) (_ bufworkspace.WorkspaceDepManager, retErr error) {
	readWriteBucket, bucketTargeting, err := c.buffetchReader.GetDirReadWriteBucket(
		ctx,
		c.container,
		dirRef,
		functionOptions.getGetReadWriteBucketOptions()...,
	)
	if err != nil {
		return nil, err
	}
	// WE DO NOT USE PATHS/EXCLUDE PATHS.
	// When we refactor functionOptions, we need to make sure we only include what we can pass to WorkspaceDepManager.
	return c.workspaceDepManagerProvider.GetWorkspaceDepManager(
		ctx,
		readWriteBucket,
		bucketTargeting,
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
	return c.workspaceProvider.GetWorkspaceForModuleKey(
		ctx,
		moduleKey,
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
		// No need to apply validation - Images do not use protovalidate.
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
		// TODO FUTURE: without allowNotExist, this results in silent behavior when --path is incorrect.
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
		if err := c.warnUnconfiguredTransitiveImports(ctx, workspace, image); err != nil {
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

// warnUnconfiguredTransitiveImports will print a warning whenever a file imports another file that
// is not in a local Module, or is not in the declared list of dependencies in your buf.yaml.
//
// If all the Modules in the Workspace are remote Modules, no warnings are printed.
func (c *controller) warnUnconfiguredTransitiveImports(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	image bufimage.Image,
) error {
	// First, figure out if we have a local module. If we don't, just return - the Workspace
	// was purely built from remote Modules, and therefore buf.yaml configured dependencies
	// do not apply.
	if slicesext.Count(workspace.Modules(), bufmodule.Module.IsLocal) == 0 {
		return nil
	}
	// Construct a struct map of all the ModuleFullName strings of the configured buf.yaml
	// Module dependencies, and the local Modules. These are considered OK to depend on
	// for non-imports in the Image.
	configuredModuleFullNameStrings, err := slicesext.MapError(
		workspace.ConfiguredDepModuleRefs(),
		func(moduleRef bufmodule.ModuleRef) (string, error) {
			moduleFullName := moduleRef.ModuleFullName()
			if moduleFullName == nil {
				return "", syserror.New("ModuleFullName nil on ModuleRef")
			}
			return moduleFullName.String(), nil
		},
	)
	if err != nil {
		return err
	}
	configuredModuleFullNameStringMap := slicesext.ToStructMap(configuredModuleFullNameStrings)
	for _, localModule := range bufmodule.ModuleSetLocalModules(workspace) {
		if moduleFullName := localModule.ModuleFullName(); moduleFullName != nil {
			configuredModuleFullNameStringMap[moduleFullName.String()] = struct{}{}
		}
	}

	// Construct a map from Image file path -> ModuleFullName string.
	//
	// If a given file in the Image did not have a ModuleFullName, it came from a local unnamed Module
	// in the Workspace, and we're safe to ignore it with respect to calculating the undeclared
	// transitive imports.
	pathToModuleFullNameString := make(map[string]string)
	for _, imageFile := range image.Files() {
		// If nil, this came from a local unnamed Module in the Workspace, and we're safe to ignore.
		if moduleFullName := imageFile.ModuleFullName(); moduleFullName != nil {
			pathToModuleFullNameString[imageFile.Path()] = moduleFullName.String()
		}
	}

	for _, imageFile := range image.Files() {
		// Ignore imports. We only care about non-imports.
		if imageFile.IsImport() {
			continue
		}
		for _, importPath := range imageFile.FileDescriptorProto().GetDependency() {
			moduleFullNameString, ok := pathToModuleFullNameString[importPath]
			if !ok {
				// The import was from a local unnamed Module in the Workspace.
				continue
			}
			if _, ok := configuredModuleFullNameStringMap[moduleFullNameString]; !ok {
				c.logger.Sugar().Warnf(
					`File %q imports %q, which is not in your workspace or in the dependencies declared in your buf.yaml, but is found in transitive dependency %q.
Declare %q in the deps key in your buf.yaml.`,
					imageFile.Path(),
					importPath,
					moduleFullNameString,
					moduleFullNameString,
				)
			}
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

func getImageFileInfosForModuleSet(ctx context.Context, moduleSet bufmodule.ModuleSet) ([]bufimage.ImageFileInfo, error) {
	// Sorted.
	fileInfos, err := bufmodule.GetFileInfos(
		ctx,
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
	)
	if err != nil {
		return nil, err
	}
	return slicesext.Map(
		fileInfos,
		func(fileInfo bufmodule.FileInfo) bufimage.ImageFileInfo {
			return bufimage.ImageFileInfoForModuleFileInfo(fileInfo)
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
// TODO FUTURE: redo functionOptions, this is a mess
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
			// bufimage expects normalized paths, so we need to normalize the paths
			// from functionOptions before passing them through.
			normalizedTargetPaths := make([]string, 0, len(functionOptions.targetPaths))
			normalizedExcludePaths := make([]string, 0, len(functionOptions.targetExcludePaths))
			for _, targetPath := range functionOptions.targetPaths {
				normalizedTargetPaths = append(normalizedTargetPaths, normalpath.Normalize(targetPath))
			}
			for _, excludePath := range functionOptions.targetExcludePaths {
				normalizedExcludePaths = append(normalizedExcludePaths, normalpath.Normalize(excludePath))
			}
			// TODO FUTURE: allowNotExist? Also, does this affect lint or breaking?
			newImage, err = bufimage.ImageWithOnlyPathsAllowNotExist(
				newImage,
				normalizedTargetPaths,
				normalizedExcludePaths,
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
		return newJSONMarshaler(image.Resolver(), messageRef), nil
	case buffetch.MessageEncodingTxtpb:
		return protoencoding.NewTxtpbMarshaler(image.Resolver()), nil
	case buffetch.MessageEncodingYAML:
		return newYAMLMarshaler(image.Resolver(), messageRef), nil
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
	// TODO FUTURE: get standard flag names and bindings into this package.
	fileAnnotationErrorFormatFlagName := "error-format"
	return appcmd.NewInvalidArgumentErrorf("--%s: invalid format: %q", fileAnnotationErrorFormatFlagName, fileAnnotationErrorFormat)
}
