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
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ImageWithConfig pairs an Image with lint and breaking configuration.
type ImageWithConfig interface {
	bufimage.Image
	LintConfig() bufconfig.LintConfig
	BreakingConfig() bufconfig.BreakingConfig
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
	GetTargetImageWithConfigs(
		ctx context.Context,
		input string,
		options ...FunctionOption,
	) ([]ImageWithConfig, error)
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
	container app.EnvStdioContainer,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (Controller, error) {
	return newController(
		logger,
		container,
		moduleKeyProvider,
		moduleDataProvider,
		httpClient,
		httpauthAuthenticator,
		gitClonerOptions,
		options...,
	)
}

type ControllerOption func(*controller)

func WithDisableSymlinks(disableSymlinks bool) ControllerOption {
	return func(controller *controller) {
		controller.disableSymlinks = disableSymlinks
	}
}

func WithFileAnnotationErrorFormat(fileAnnotationErrorFormat string) ControllerOption {
	return func(controller *controller) {
		controller.fileAnnotationErrorFormat = fileAnnotationErrorFormat
	}
}

func WithFileAnnotationsToStdout() ControllerOption {
	return func(controller *controller) {
		controller.fileAnnotationsToStdout = true
	}
}

// TODO: split up to per-function.
type FunctionOption func(*functionOptions)

func WithTargetPaths(targetPaths []string, targetExcludePaths []string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.targetPaths = targetPaths
		functionOptions.targetExcludePaths = targetExcludePaths
	}
}

func WithImageExcludeSourceInfo(imageExcludeSourceInfo bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageExcludeSourceInfo = imageExcludeSourceInfo
	}
}

func WithImageExcludeImports(imageExcludeImports bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageExcludeImports = imageExcludeImports
	}
}

func WithImageTypes(imageTypes []string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageTypes = imageTypes
	}
}

func WithImageAsFileDescriptorSet(imageAsFileDescriptorSet bool) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.imageAsFileDescriptorSet = imageAsFileDescriptorSet
	}
}

// WithConfigOverride applies the config override.
//
// This flag will only work if no buf.work.yaml is detected, and the buf.yaml is a
// v1beta1 buf.yaml, v1 buf.yaml, or no buf.yaml. This flag will not work if a buf.work.yaml
// is detected, or a v2 buf.yaml is detected.
//
// If used with an image or module ref, this has no effect on the build, i.e. excludes are
// not respected, and the module name is ignored. This matches old behavior.
//
// This implements the soon-to-be-deprected --config flag.
//
// See bufconfig.GetBufYAMLFileForPrefixOrOverride for more details.
//
// *** DO NOT USE THIS OUTSIDE OF THE CLI AND/OR IF YOU DON'T UNDERSTAND IT. ***
// *** DO NOT ADD THIS TO ANY NEW COMMANDS. ***
//
// Current commands that use this: build, breaking, lint, generate, format,
// export, ls-breaking-rules, ls-lint-rules.
func WithConfigOverride(configOverride string) FunctionOption {
	return func(functionOptions *functionOptions) {
		functionOptions.configOverride = configOverride
	}
}

/// *** PRIVATE ***

// In theory, we want to keep this separate from our global variables in bufcli.
//
// Originally, this was in a different package, and we want to keep the option to split
// it out again. The separation of concerns here is that the controller doesnt itself
// deal in the global variables.
type controller struct {
	logger             *zap.Logger
	container          app.EnvStdioContainer
	moduleDataProvider bufmodule.ModuleDataProvider

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
	container app.EnvStdioContainer,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (*controller, error) {
	controller := &controller{
		logger:             logger,
		container:          container,
		moduleDataProvider: moduleDataProvider,
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
) (bufworkspace.UpdateableWorkspace, error) {
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
) (bufimage.Image, error) {
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
		return c.buildImage(
			ctx,
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
			functionOptions,
		)
	case buffetch.SourceRef:
		workspace, err := c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.buildImage(
			ctx,
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
			functionOptions,
		)
	case buffetch.ModuleRef:
		workspace, err := c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, err
		}
		return c.buildImage(
			ctx,
			bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace),
			functionOptions,
		)
	case buffetch.MessageRef:
		return c.getImageForMessageRef(ctx, t, functionOptions)
	default:
		// This is a system error.
		return nil, syserror.Newf("invalid Ref: %T", ref)
	}
}

func (c *controller) GetTargetImageWithConfigs(
	ctx context.Context,
	input string,
	options ...FunctionOption,
) ([]ImageWithConfig, error) {
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

func (c *controller) PutImage(
	ctx context.Context,
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
) (proto.Message, buffetch.MessageEncoding, error) {
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
) error {
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
	return bufworkspace.NewWorkspaceForBucket(
		ctx,
		readBucketCloser,
		c.moduleDataProvider,
		bufworkspace.WithTargetSubDirPath(
			readBucketCloser.SubDirPath(),
		),
		bufworkspace.WithProtoFileTargetPath(
			protoFileRef.ProtoFilePath(),
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
		readBucketCloser,
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
		readWriteBucket,
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
		moduleKey,
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
	image, fileAnnotations, err := bufimage.BuildImage(
		ctx,
		moduleReadBucket,
		options...,
	)
	if err != nil {
		return nil, err
	}
	if len(fileAnnotations) > 0 {
		writer := c.container.Stderr()
		if c.fileAnnotationsToStdout {
			writer = c.container.Stdout()
		}
		if err := bufanalysis.PrintFileAnnotations(
			writer,
			fileAnnotations,
			c.fileAnnotationErrorFormat,
		); err != nil {
			return nil, err
		}
		return nil, ErrFileAnnotation
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
	return imageWithConfigs, nil
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

type imageWithConfig struct {
	bufimage.Image

	lintConfig     bufconfig.LintConfig
	breakingConfig bufconfig.BreakingConfig
}

func newImageWithConfig(
	image bufimage.Image,
	lintConfig bufconfig.LintConfig,
	breakingConfig bufconfig.BreakingConfig,
) *imageWithConfig {
	return &imageWithConfig{
		Image:          image,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
	}
}

func (i *imageWithConfig) LintConfig() bufconfig.LintConfig {
	return i.lintConfig
}

func (i *imageWithConfig) BreakingConfig() bufconfig.BreakingConfig {
	return i.breakingConfig
}

type functionOptions struct {
	targetPaths              []string
	targetExcludePaths       []string
	imageExcludeSourceInfo   bool
	imageExcludeImports      bool
	imageTypes               []string
	imageAsFileDescriptorSet bool
	configOverride           string
}

func newFunctionOptions() *functionOptions {
	return &functionOptions{}
}

func (f *functionOptions) withPathsForBucketExtender(
	bucketExtender buffetch.BucketExtender,
) (*functionOptions, error) {
	deref := *f
	c := &deref
	for i, inputTargetPath := range c.targetPaths {
		targetPath, err := bucketExtender.PathForExternalPath(inputTargetPath)
		if err != nil {
			return nil, err
		}
		c.targetPaths[i] = targetPath
	}
	for i, inputTargetExcludePath := range c.targetExcludePaths {
		targetExcludePath, err := bucketExtender.PathForExternalPath(inputTargetExcludePath)
		if err != nil {
			return nil, err
		}
		c.targetExcludePaths[i] = targetExcludePath
	}
	return c, nil
}

func (f *functionOptions) getGetBucketOptions() []buffetch.GetBucketOption {
	if f.configOverride != "" {
		// If we have a config override, we do not search for buf.yamls or buf.work.yamls,
		// instead acting as if the config override was the only configuration file available.
		//
		// Note that this is slightly different behavior than the pre-refactor CLI had, but this
		// was always the intended behavior. The pre-refactor CLI would error if you had a buf.work.yaml,
		// and did the same search behavior for buf.yamls, which didn't really make sense. In the new
		// world where buf.yamls also represent the behavior of buf.work.yamls, you should be able
		// to specify whatever want here.
		return []buffetch.GetBucketOption{
			buffetch.GetBucketWithNoSearch(),
		}
	}
	return nil
}
