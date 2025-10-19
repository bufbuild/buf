// Copyright 2020-2025 Buf Technologies, Inc.
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
	"log/slog"
	"net/http"
	"slices"
	"sort"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"buf.build/go/protovalidate"
	"buf.build/go/protoyaml"
	"buf.build/go/standard/xio"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwkt/bufwktstore"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufreflect"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/httpauth"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/wasm"
	"google.golang.org/protobuf/proto"
)

// ImageWithConfig pairs an Image with its corresponding [bufmodule.Module] full name
// (which may be nil), [bufmodule.Module] opaque ID, and lint and breaking configurations.
type ImageWithConfig interface {
	bufimage.Image

	ModuleFullName() bufparse.FullName
	ModuleOpaqueID() string
	LintConfig() bufconfig.LintConfig
	BreakingConfig() bufconfig.BreakingConfig
	PluginConfigs() []bufconfig.PluginConfig
	PolicyConfigs() []bufconfig.PolicyConfig

	isImageWithConfig()
}

// Controller is the central entrypoint for the Buf CLI.
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
	// GetTargetImageWithConfigsAndCheckClient gets the target ImageWithConfigs
	// with a configured bufcheck Client.
	//
	// ImageWithConfig scopes the configuration per image for use with breaking
	// and lint checks. The check Client is bound to the input to ensure that the
	// correct remote plugin dependencies are used. A wasmRuntime is provided
	// to evaluate Wasm plugins.
	GetTargetImageWithConfigsAndCheckClient(
		ctx context.Context,
		input string,
		wasmRuntime wasm.Runtime,
		options ...FunctionOption,
	) ([]ImageWithConfig, bufcheck.Client, error)
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
	// GetCheckClientForWorkspace returns a new bufcheck Client for the given Workspace.
	//
	// Clients are bound to a specific Workspace to ensure that the correct
	// plugin dependencies are used. A wasmRuntime is provided to evaluate
	// Wasm plugins.
	GetCheckClientForWorkspace(
		ctx context.Context,
		workspace bufworkspace.Workspace,
		wasmRuntime wasm.Runtime,
	) (bufcheck.Client, error)
}

// NewController returns a new Controller.
func NewController(
	logger *slog.Logger,
	container app.EnvStdioContainer,
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	pluginKeyProvider bufplugin.PluginKeyProvider,
	pluginDataProvider bufplugin.PluginDataProvider,
	policyKeyProvider bufpolicy.PolicyKeyProvider,
	policyDataProvider bufpolicy.PolicyDataProvider,
	wktStore bufwktstore.Store,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (Controller, error) {
	return newController(
		logger,
		container,
		graphProvider,
		moduleKeyProvider,
		moduleDataProvider,
		commitProvider,
		pluginKeyProvider,
		pluginDataProvider,
		policyKeyProvider,
		policyDataProvider,
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
	logger             *slog.Logger
	container          app.EnvStdioContainer
	moduleDataProvider bufmodule.ModuleDataProvider
	graphProvider      bufmodule.GraphProvider
	commitProvider     bufmodule.CommitProvider
	pluginKeyProvider  bufplugin.PluginKeyProvider
	pluginDataProvider bufplugin.PluginDataProvider
	policyKeyProvider  bufpolicy.PolicyKeyProvider
	policyDataProvider bufpolicy.PolicyDataProvider
	wktStore           bufwktstore.Store

	disableSymlinks           bool
	fileAnnotationErrorFormat string
	fileAnnotationsToStdout   bool
	copyToInMemory            bool

	storageosProvider           storageos.Provider
	buffetchRefParser           buffetch.RefParser
	buffetchReader              buffetch.Reader
	buffetchWriter              buffetch.Writer
	workspaceProvider           bufworkspace.WorkspaceProvider
	workspaceDepManagerProvider bufworkspace.WorkspaceDepManagerProvider
}

func newController(
	logger *slog.Logger,
	container app.EnvStdioContainer,
	graphProvider bufmodule.GraphProvider,
	moduleKeyProvider bufmodule.ModuleKeyProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	pluginKeyProvider bufplugin.PluginKeyProvider,
	pluginDataProvider bufplugin.PluginDataProvider,
	policyKeyProvider bufpolicy.PolicyKeyProvider,
	policyDataProvider bufpolicy.PolicyDataProvider,
	wktStore bufwktstore.Store,
	httpClient *http.Client,
	httpauthAuthenticator httpauth.Authenticator,
	gitClonerOptions git.ClonerOptions,
	options ...ControllerOption,
) (*controller, error) {
	controller := &controller{
		logger:             logger,
		container:          container,
		graphProvider:      graphProvider,
		moduleDataProvider: moduleDataProvider,
		commitProvider:     commitProvider,
		pluginKeyProvider:  pluginKeyProvider,
		pluginDataProvider: pluginDataProvider,
		policyKeyProvider:  policyKeyProvider,
		policyDataProvider: policyDataProvider,
		wktStore:           wktStore,
	}
	for _, option := range options {
		option(controller)
	}
	if err := validateFileAnnotationErrorFormat(controller.fileAnnotationErrorFormat); err != nil {
		return nil, err
	}
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
			gitClonerOptions,
		),
		moduleKeyProvider,
	)
	controller.buffetchWriter = buffetch.NewWriter(logger)
	controller.workspaceProvider = bufworkspace.NewWorkspaceProvider(
		logger,
		graphProvider,
		moduleDataProvider,
		commitProvider,
		pluginKeyProvider,
	)
	controller.workspaceDepManagerProvider = bufworkspace.NewWorkspaceDepManagerProvider(
		logger,
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

func (c *controller) GetTargetImageWithConfigsAndCheckClient(
	ctx context.Context,
	input string,
	wasmRuntime wasm.Runtime,
	options ...FunctionOption,
) (_ []ImageWithConfig, _ bufcheck.Client, retErr error) {
	defer c.handleFileAnnotationSetRetError(&retErr)
	functionOptions := newFunctionOptions(c)
	for _, option := range options {
		option(functionOptions)
	}
	ref, err := c.buffetchRefParser.GetRef(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var workspace bufworkspace.Workspace
	switch t := ref.(type) {
	case buffetch.ProtoFileRef:
		workspace, err = c.getWorkspaceForProtoFileRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
	case buffetch.SourceRef:
		workspace, err = c.getWorkspaceForSourceRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
	case buffetch.ModuleRef:
		workspace, err = c.getWorkspaceForModuleRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
	case buffetch.MessageRef:
		image, err := c.getImageForMessageRef(ctx, t, functionOptions)
		if err != nil {
			return nil, nil, err
		}
		bucket, err := c.storageosProvider.NewReadWriteBucket(
			".",
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return nil, nil, err
		}
		lintConfig := bufconfig.DefaultLintConfigV1
		breakingConfig := bufconfig.DefaultBreakingConfigV1
		var (
			pluginConfigs            []bufconfig.PluginConfig
			policyConfigs            []bufconfig.PolicyConfig
			pluginKeyProvider        = bufplugin.NopPluginKeyProvider
			policyKeyProvider        = bufpolicy.NopPolicyKeyProvider
			policyPluginKeyProvider  = bufpolicy.NopPolicyPluginKeyProvider
			policyPluginDataProvider = bufpolicy.NopPolicyPluginDataProvider
		)
		bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefixOrOverride(
			ctx,
			bucket,
			".",
			functionOptions.configOverride,
		)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, err
			}
			// We did not find a buf.yaml in our current directory, and there was no config override.
			// Use the defaults.
		} else {
			if topLevelLintConfig := bufYAMLFile.TopLevelLintConfig(); topLevelLintConfig == nil {
				// Ensure that this is a v2 config
				if fileVersion := bufYAMLFile.FileVersion(); fileVersion != bufconfig.FileVersionV2 {
					return nil, nil, syserror.Newf("non-v2 version with no top-level lint config: %s", fileVersion)
				}
				// v2 config without a top-level lint config, use v2 default
				lintConfig = bufconfig.DefaultLintConfigV2
			} else {
				lintConfig = topLevelLintConfig
			}
			if topLevelBreakingConfig := bufYAMLFile.TopLevelBreakingConfig(); topLevelBreakingConfig == nil {
				if fileVersion := bufYAMLFile.FileVersion(); fileVersion != bufconfig.FileVersionV2 {
					return nil, nil, syserror.Newf("non-v2 version with no top-level breaking config: %s", fileVersion)
				}
				// v2 config without a top-level breaking config, use v2 default
				breakingConfig = bufconfig.DefaultBreakingConfigV2
			} else {
				breakingConfig = topLevelBreakingConfig
			}
			// The directory path is resolved to a buf.yaml file and a buf.lock file. If the
			// buf.yaml file is found, the PluginConfigs from the buf.yaml file and the PluginKeys
			// from the buf.lock file are resolved to create the PluginKeyProvider.
			pluginConfigs = bufYAMLFile.PluginConfigs()
			policyConfigs = bufYAMLFile.PolicyConfigs()
			// If a config override is provided, the PluginConfig remote Refs use the BSR
			// to resolve the PluginKeys. No buf.lock is required.
			// If the buf.yaml file is not found, the bufplugin.NopPluginKeyProvider is returned.
			// If the buf.lock file is not found, the bufplugin.NopPluginKeyProvider is returned.
			if functionOptions.configOverride != "" {
				// To support remote plugins in the override, we need to resolve the remote
				// Refs to PluginKeys. A buf.lock file is not required for this operation.
				// We use the BSR to resolve any remote plugin Refs.
				pluginKeyProvider = c.pluginKeyProvider
			} else if bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
				var (
					remotePluginKeys             []bufplugin.PluginKey
					remotePolicyKeys             []bufpolicy.PolicyKey
					policyNameToRemotePluginKeys map[string][]bufplugin.PluginKey
				)
				if bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
					ctx,
					bucket,
					// buf.lock files live next to the buf.yaml
					".",
				); err != nil {
					if !errors.Is(err, fs.ErrNotExist) {
						return nil, nil, err
					}
					// We did not find a buf.lock in our current directory.
					// Remote plugins and policies are not available.
				} else {
					remotePluginKeys = bufLockFile.RemotePluginKeys()
					remotePolicyKeys = bufLockFile.RemotePolicyKeys()
					policyNameToRemotePluginKeys = bufLockFile.PolicyNameToRemotePluginKeys()
				}
				pluginKeyProvider, err = newStaticPluginKeyProviderForPluginConfigs(
					pluginConfigs,
					remotePluginKeys,
				)
				if err != nil {
					return nil, nil, err
				}
				policyKeyProvider, err = newStaticPolicyKeyProviderForPolicyConfigs(
					policyConfigs,
					remotePolicyKeys,
				)
				if err != nil {
					return nil, nil, err
				}
				policyPluginKeyProvider, err = newStaticPolicyPluginKeyProviderForPolicyConfigs(
					policyConfigs,
					policyNameToRemotePluginKeys,
				)
				if err != nil {
					return nil, nil, err
				}
				policyPluginDataProvider, err = newStaticPolicyPluginDataProviderForPolicyConfigs(
					c.pluginDataProvider,
					policyConfigs,
				)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		imageWithConfigs := []ImageWithConfig{
			newImageWithConfig(
				image,
				nil, // No module name for a single message ref
				"",  // No module opaque ID for a single message ref
				lintConfig,
				breakingConfig,
				pluginConfigs,
				policyConfigs,
			),
		}
		checkClient, err := bufcheck.NewClient(
			c.logger,
			bufcheck.ClientWithStderr(c.container.Stderr()),
			bufcheck.ClientWithRunnerProvider(
				bufcheck.NewLocalRunnerProvider(wasmRuntime),
			),
			bufcheck.ClientWithLocalWasmPluginsFromOS(),
			bufcheck.ClientWithRemoteWasmPlugins(pluginKeyProvider, c.pluginDataProvider),
			bufcheck.ClientWithLocalPoliciesFromOS(),
			bufcheck.ClientWithRemotePolicies(
				policyKeyProvider,
				c.policyDataProvider,
				policyPluginKeyProvider,
				policyPluginDataProvider,
			),
		)
		if err != nil {
			return nil, nil, err
		}
		return imageWithConfigs, checkClient, nil
	default:
		// This is a system error.
		return nil, nil, syserror.Newf("invalid Ref: %T", ref)
	}
	targetImageWithConfigs, err := c.buildTargetImageWithConfigs(ctx, workspace, functionOptions)
	if err != nil {
		return nil, nil, err
	}
	checkClient, err := c.GetCheckClientForWorkspace(ctx, workspace, wasmRuntime)
	if err != nil {
		return nil, nil, err
	}
	return targetImageWithConfigs, checkClient, err
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
		retErr = errors.Join(retErr, writeCloser.Close())
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
		protovalidateValidator, err := protovalidate.New()
		if err != nil {
			return nil, 0, err
		}
		validator = yamlValidator{protovalidateValidator}
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
	data, err := xio.ReadAllAndClose(readCloser)
	if err != nil {
		return nil, 0, err
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
	return errors.Join(err, writeCloser.Close())
}

func (c *controller) GetCheckClientForWorkspace(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	wasmRuntime wasm.Runtime,
) (_ bufcheck.Client, retErr error) {
	pluginKeyProvider, err := newStaticPluginKeyProviderForPluginConfigs(
		workspace.PluginConfigs(),
		workspace.RemotePluginKeys(),
	)
	if err != nil {
		return nil, err
	}
	policyKeyProvider, err := newStaticPolicyKeyProviderForPolicyConfigs(
		workspace.PolicyConfigs(),
		workspace.RemotePolicyKeys(),
	)
	if err != nil {
		return nil, err
	}
	policyPluginKeyProvider, err := newStaticPolicyPluginKeyProviderForPolicyConfigs(
		workspace.PolicyConfigs(),
		workspace.PolicyNameToRemotePluginKeys(),
	)
	if err != nil {
		return nil, err
	}
	policyPluginDataProvider, err := newStaticPolicyPluginDataProviderForPolicyConfigs(
		c.pluginDataProvider,
		workspace.PolicyConfigs(),
	)
	if err != nil {
		return nil, err
	}
	return bufcheck.NewClient(
		c.logger,
		bufcheck.ClientWithStderr(c.container.Stderr()),
		bufcheck.ClientWithRunnerProvider(
			bufcheck.NewLocalRunnerProvider(wasmRuntime),
		),
		bufcheck.ClientWithLocalWasmPluginsFromOS(),
		bufcheck.ClientWithRemoteWasmPlugins(
			pluginKeyProvider,
			c.pluginDataProvider,
		),
		bufcheck.ClientWithLocalPoliciesFromOS(),
		bufcheck.ClientWithRemotePolicies(
			policyKeyProvider,
			c.policyDataProvider,
			policyPluginKeyProvider,
			policyPluginDataProvider,
		),
	)
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
		retErr = errors.Join(retErr, readBucketCloser.Close())
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
		retErr = errors.Join(retErr, readBucketCloser.Close())
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
		retErr = errors.Join(retErr, readCloser.Close())
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
		for _, fileDescriptorProto := range protoImage.GetFile() {
			fileDescriptorProto.ClearSourceCodeInfo()
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
		c.logger,
		moduleReadBucket,
		options...,
	)
	if err != nil {
		return nil, err
	}
	return filterImage(image, functionOptions, true)
}

// buildTargetImageWithConfigs builds an image for each module in the workspace.
// This is used to associate LintConfig and BreakingConfig on a per-module basis.
func (c *controller) buildTargetImageWithConfigs(
	ctx context.Context,
	workspace bufworkspace.Workspace,
	functionOptions *functionOptions,
) ([]ImageWithConfig, error) {
	modules := bufmodule.ModuleSetTargetModules(workspace)
	imageWithConfigs := make([]ImageWithConfig, 0, len(modules))
	for _, module := range modules {
		c.logger.DebugContext(
			ctx,
			"building image for target module",
			slog.String("moduleOpaqueID", module.OpaqueID()),
			slog.String("moduleDescription", module.Description()),
		)
		opaqueID := module.OpaqueID()
		// We need to make sure that all dependencies are non-targets, so that they
		// end up as imports in the resulting image.
		moduleSet, err := workspace.WithTargetOpaqueIDs(opaqueID)
		if err != nil {
			return nil, err
		}
		// The moduleReadBucket may include more modules than the target module
		// and its dependencies. This is because the moduleSet is constructed from
		// the workspace. Targeting the module does not remove non-related modules.
		// Build image will use the target info to build the image for the specific
		// module. Non-targeted modules will not be included in the image.
		moduleReadBucket := bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet)
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
				module.FullName(),
				module.OpaqueID(),
				workspace.GetLintConfigForOpaqueID(module.OpaqueID()),
				workspace.GetBreakingConfigForOpaqueID(module.OpaqueID()),
				workspace.PluginConfigs(),
				workspace.PolicyConfigs(),
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
	if xslices.Count(workspace.Modules(), bufmodule.Module.IsLocal) == 0 {
		return nil
	}
	// Construct a struct map of all the FullName strings of the configured buf.yaml
	// Module dependencies, and the local Modules. These are considered OK to depend on
	// for non-imports in the Image.
	configuredFullNameStrings, err := xslices.MapError(
		workspace.ConfiguredDepModuleRefs(),
		func(moduleRef bufparse.Ref) (string, error) {
			moduleFullName := moduleRef.FullName()
			if moduleFullName == nil {
				return "", syserror.New("FullName nil on ModuleRef")
			}
			return moduleFullName.String(), nil
		},
	)
	if err != nil {
		return err
	}
	configuredFullNameStringMap := xslices.ToStructMap(configuredFullNameStrings)
	for _, localModule := range bufmodule.ModuleSetLocalModules(workspace) {
		if moduleFullName := localModule.FullName(); moduleFullName != nil {
			configuredFullNameStringMap[moduleFullName.String()] = struct{}{}
		}
	}

	// Construct a map from Image file path -> FullName string.
	//
	// If a given file in the Image did not have a FullName, it came from a local unnamed Module
	// in the Workspace, and we're safe to ignore it with respect to calculating the undeclared
	// transitive imports.
	pathToFullNameString := make(map[string]string)
	for _, imageFile := range image.Files() {
		// If nil, this came from a local unnamed Module in the Workspace, and we're safe to ignore.
		if moduleFullName := imageFile.FullName(); moduleFullName != nil {
			pathToFullNameString[imageFile.Path()] = moduleFullName.String()
		}
	}

	for _, imageFile := range image.Files() {
		// Ignore imports. We only care about non-imports.
		if imageFile.IsImport() {
			continue
		}
		for _, importPath := range imageFile.FileDescriptorProto().GetDependency() {
			moduleFullNameString, ok := pathToFullNameString[importPath]
			if !ok {
				// The import was from a local unnamed Module in the Workspace.
				continue
			}
			if _, ok := configuredFullNameStringMap[moduleFullNameString]; !ok {
				c.logger.Warn(fmt.Sprintf(
					`File %q imports %q, which is not in your workspace or in the dependencies declared in your buf.yaml, but is found in transitive dependency %q.
Declare %q in the deps key in your buf.yaml.`,
					imageFile.Path(),
					importPath,
					moduleFullNameString,
					moduleFullNameString,
				))
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

// Implements [protoyaml.Validator] using [protovalidate.Validator]. This allows us to pass
// a [protovalidate.Validator] to the [protoencoding.NewYAMLUnmarshaler] and validate types
// when unmarshalling.
type yamlValidator struct{ protovalidateValidator protovalidate.Validator }

func (v yamlValidator) Validate(msg proto.Message) error {
	return v.protovalidateValidator.Validate(msg)
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
	return xslices.Map(
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
	return protoencoding.NewResolver(firstProtoImage.GetFile()...)
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
	includeTypes := functionOptions.imageIncludeTypes
	excludeTypes := functionOptions.imageExcludeTypes
	if len(includeTypes) > 0 || len(excludeTypes) > 0 {
		newImage, err = bufimageutil.FilterImage(
			newImage,
			bufimageutil.WithIncludeTypes(includeTypes...),
			bufimageutil.WithExcludeTypes(excludeTypes...),
			bufimageutil.WithMutateInPlace(),
		)
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
	if slices.Contains(bufanalysis.AllFormatStrings, fileAnnotationErrorFormat) {
		return nil
	}
	// TODO FUTURE: get standard flag names and bindings into this package.
	fileAnnotationErrorFormatFlagName := "error-format"
	return appcmd.NewInvalidArgumentErrorf("--%s: invalid format: %q", fileAnnotationErrorFormatFlagName, fileAnnotationErrorFormat)
}

// newStaticPluginKeyProvider creates a new PluginKeyProvider for the set of PluginKeys.
//
// The PluginKeys come from the buf.lock file. The PluginKeyProvider is static
// and does not change. PluginConfigs are validated to ensure that all remote
// PluginConfigs are pinned in the buf.lock file.
func newStaticPluginKeyProviderForPluginConfigs(
	pluginConfigs []bufconfig.PluginConfig,
	pluginKeys []bufplugin.PluginKey,
) (_ bufplugin.PluginKeyProvider, retErr error) {
	// Validate that all remote PluginConfigs are present in the buf.lock file.
	pluginKeysByFullName, err := xslices.ToUniqueValuesMap(pluginKeys, func(pluginKey bufplugin.PluginKey) string {
		return pluginKey.FullName().String()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate remote PluginKeys: %w", err)
	}
	// Remote PluginConfig Refs are any PluginConfigs that have a Ref.
	remotePluginRefs := xslices.Filter(
		xslices.Map(pluginConfigs, func(pluginConfig bufconfig.PluginConfig) bufparse.Ref {
			return pluginConfig.Ref()
		}),
		func(pluginRef bufparse.Ref) bool {
			return pluginRef != nil
		},
	)
	for _, remotePluginRef := range remotePluginRefs {
		if _, ok := pluginKeysByFullName[remotePluginRef.FullName().String()]; !ok {
			return nil, fmt.Errorf(`remote plugin %q is not in the buf.lock file, use "buf plugin update" to pin remote refs`, remotePluginRef)
		}
	}
	return bufplugin.NewStaticPluginKeyProvider(pluginKeys)
}

// newStaticPolicyKeyProvider creates a new PolicyKeyProvider for the set of PolicyKeys.
//
// The PolicyKeys come from the buf.lock file. The PolicyKeyProvider is static
// and does not change. PolicyConfigs are validated to ensure that all remote
// PolicyConfigs are pinned in the buf.lock file.
func newStaticPolicyKeyProviderForPolicyConfigs(
	policyConfigs []bufconfig.PolicyConfig,
	policyKeys []bufpolicy.PolicyKey,
) (_ bufpolicy.PolicyKeyProvider, retErr error) {
	// Validate that all remote PolicyConfigs are present in the buf.lock file.
	policyKeysByFullName, err := xslices.ToUniqueValuesMap(policyKeys, func(policyKey bufpolicy.PolicyKey) string {
		return policyKey.FullName().String()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate remote PolicyKeys: %w", err)
	}
	// Remote PolicyConfig Refs are any PolicyConfigs that have a Ref.
	remotePolicyRefs := xslices.Filter(
		xslices.Map(policyConfigs, func(policyConfig bufconfig.PolicyConfig) bufparse.Ref {
			return policyConfig.Ref()
		}),
		func(policyRef bufparse.Ref) bool {
			return policyRef != nil
		},
	)
	for _, remotePolicyRef := range remotePolicyRefs {
		if _, ok := policyKeysByFullName[remotePolicyRef.FullName().String()]; !ok {
			return nil, fmt.Errorf(`remote policy %q is not in the buf.lock file, use "buf policy update" to pin remote refs`, remotePolicyRef)
		}
	}
	return bufpolicy.NewStaticPolicyKeyProvider(policyKeys)
}

// newStaticPolicyPluginKeyProviderForPolicyConfigs creates a new PolicyPluginKeyProvider for the set of PolicyConfigs.
func newStaticPolicyPluginKeyProviderForPolicyConfigs(
	policyConfigs []bufconfig.PolicyConfig,
	policyNameToRemotePluginKeys map[string][]bufplugin.PluginKey,
) (bufpolicy.PolicyPluginKeyProvider, error) {
	policyNames, err := xslices.ToUniqueValuesMap(policyConfigs, func(policyConfig bufconfig.PolicyConfig) string {
		policyName := policyConfig.Name()
		if policyConfig.Ref() != nil {
			// Remote policies are required to be referenced by their full name.
			// The buf.lock stores the name.
			policyName = policyConfig.Ref().FullName().String()
		}
		return policyName
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate policy names in policy configs: %w", err)
	}
	// We do not validate that all remote PolicyConfig plugins are present in the buf.lock file.
	// This would require loading the PolicyConfig data. Check is defered to the runtime.
	for policyName, remotePluginKeys := range policyNameToRemotePluginKeys {
		_, err := xslices.ToUniqueValuesMap(remotePluginKeys, func(pluginKey bufplugin.PluginKey) string {
			return pluginKey.FullName().String()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to validate remote PluginKeys for Policy %q: %w", policyName, err)
		}
		if _, ok := policyNames[policyName]; !ok {
			return nil, fmt.Errorf("remote plugins configured for unknown policy %q", policyName)
		}
	}
	return bufpolicy.NewStaticPolicyPluginKeyProvider(policyNameToRemotePluginKeys)
}

// newStaticPolicyPluginDataProviderForPolicyConfigs creates a new PolicyPluginKeyProvider for the set of PolicyConfigs.
//
// The pluginDataProvider is shared across all policies.
func newStaticPolicyPluginDataProviderForPolicyConfigs(
	pluginDataProvider bufplugin.PluginDataProvider,
	policyConfigs []bufconfig.PolicyConfig,
) (bufpolicy.PolicyPluginDataProvider, error) {
	policyNameToPluginDataProvider := make(map[string]bufplugin.PluginDataProvider)
	for _, policyConfig := range policyConfigs {
		policyName := policyConfig.Name()
		if _, ok := policyNameToPluginDataProvider[policyName]; ok {
			return nil, fmt.Errorf("duplicate policy name %q found in policy configs", policyName)
		}
		// We use the same pluginDataProvider for all policies.
		policyNameToPluginDataProvider[policyName] = pluginDataProvider
	}
	return bufpolicy.NewStaticPolicyPluginDataProvider(policyNameToPluginDataProvider)
}
