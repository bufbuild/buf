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

package bufgenv2

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/plugingen"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodifyv2"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

const (
	defaultJavaPackagePrefix = "com"
	defaultInput             = "."
)

type Generator struct {
	logger            *zap.Logger
	generator         plugingen.Generator
	imageConfigReader bufwire.ImageConfigReader
	readWriteBucket   storage.ReadWriteBucket
}

func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor *bufwasm.WASMPluginExecutor,
	clientConfig *connectclient.Config,
	imageConfigReader bufwire.ImageConfigReader,
	readWriteBucket storage.ReadWriteBucket,
) *Generator {
	return &Generator{
		logger: logger,
		generator: plugingen.NewGenerator(
			logger,
			storageosProvider,
			runner,
			wasmPluginExecutor,
			clientConfig,
		),
		imageConfigReader: imageConfigReader,
		readWriteBucket:   readWriteBucket,
	}
}

func (g *Generator) Generate(
	ctx context.Context,
	container appflag.Container,
	genTemplatePath string,
	moduleConfigPathOverride string,
	inputSpecified string,
	baseOutDir string,
	typesIncludedOverride []string,
	pathsSpecifiedOverride []string,
	pathsExcludedOverride []string,
	includeImportsOverride bool,
	includeWellKnownTypesOverride bool,
	errorFormat string,
	fileAnnotationErr error,
	// wasm is turned off in v2
	_ bool,
) error {
	genConfig, err := ReadConfigV2(
		ctx,
		g.logger,
		g.readWriteBucket,
		internal.ReadConfigWithOverride(genTemplatePath),
	)
	if err != nil {
		return err
	}
	var inputImages []bufimage.Image
	if inputSpecified != "" || len(genConfig.Inputs) == 0 {
		input := defaultInput
		if inputSpecified != "" {
			input = inputSpecified
		}
		inputRef, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
		if err != nil {
			return err
		}
		inputImage, err := internal.GetInputImage(
			ctx,
			container,
			inputRef,
			g.imageConfigReader,
			moduleConfigPathOverride,
			pathsSpecifiedOverride,
			pathsExcludedOverride,
			errorFormat,
			typesIncludedOverride,
			fileAnnotationErr,
		)
		if err != nil {
			return err
		}
		inputImages = []bufimage.Image{inputImage}
	} else {
		for _, inputConfig := range genConfig.Inputs {
			pathsSpecified := inputConfig.IncludePaths
			if len(pathsSpecifiedOverride) > 0 {
				pathsSpecified = pathsSpecifiedOverride
			}
			pathsExcluded := inputConfig.ExcludePaths
			if len(pathsExcludedOverride) > 0 {
				pathsExcluded = pathsExcludedOverride
			}
			typesIncluded := inputConfig.Types
			if len(typesIncludedOverride) > 0 {
				typesIncluded = typesIncludedOverride
			}
			inputImage, err := internal.GetInputImage(
				ctx,
				container,
				inputConfig.InputRef,
				g.imageConfigReader,
				moduleConfigPathOverride,
				pathsSpecified,
				pathsExcluded,
				errorFormat,
				typesIncluded,
				fileAnnotationErr,
			)
			if err != nil {
				return err
			}
			inputImages = append(inputImages, inputImage)
		}
	}
	generateOptions := []plugingen.GenerateOption{
		plugingen.GenerateWithBaseOutDirPath(baseOutDir),
	}
	if includeImportsOverride {
		generateOptions = append(
			generateOptions,
			plugingen.GenerateWithAlwaysIncludeImports(),
		)
	}
	if includeWellKnownTypesOverride {
		generateOptions = append(
			generateOptions,
			plugingen.GenerateWithAlwaysIncludeWellKnownTypes(),
		)
	}
	for _, inputImage := range inputImages {
		// TODO: modify this image
		if err := g.generator.Generate(
			ctx,
			container,
			genConfig.Plugins,
			inputImage,
			generateOptions...,
		); err != nil {
			return err
		}
	}
	return nil
}

// TODO this would be part of a runner or likewise
// this is just for demonstration of bringing the management stuff into one function
// ApplyManagement modifies an image based on managed mode configuration.
func ApplyManagement(image bufimage.Image, managedConfig *ManagedConfig) error {
	markSweeper := bufimagemodifyv2.NewMarkSweeper(image)
	for _, imageFile := range image.Files() {
		if err := applyManagementForFile(markSweeper, imageFile, managedConfig); err != nil {
			return err
		}
	}
	return markSweeper.Sweep()
}

// DisableFunc decides whether a file option should be disabled for a file.
type DisabledFunc func(FileOption, bufimage.ImageFile) bool

// TODO: likely want something like *string or otherwise, see https://github.com/bufbuild/buf/issues/1949
// OverrideFunc is specific to a file option, and returns what thie file option
// should be overridden to for this file.
type OverrideFunc func(bufimage.ImageFile) (string, error)

// ReadConfigV2 reads V2 configuration.
func ReadConfigV2(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
	options ...internal.ReadConfigOption,
) (*Config, error) {
	provider := internal.NewConfigDataProvider(logger)
	return readConfigV2(
		ctx,
		logger,
		provider,
		readBucket,
		options...,
	)
}

// Config is a configuration.
type Config struct {
	Managed *ManagedConfig
	Plugins []plugingen.PluginConfig
	Inputs  []*InputConfig
}

// TODO: We use nil or not to denote enabled or not, but that deems dangerous
// ManagedConfig is a managed mode configuration.
type ManagedConfig struct {
	DisabledFunc             DisabledFunc
	FileOptionToOverrideFunc map[FileOption]OverrideFunc
}

// InputConfig is an input configuration.
type InputConfig struct {
	InputRef     buffetch.Ref
	Types        []string
	ExcludePaths []string
	IncludePaths []string
}

// ExternalConfigV2 is an external configuration.
type ExternalConfigV2 struct {
	// Must be V2 in this current code setup, but we'd want this to be alongside V1
	Version string                   `json:"version,omitempty" yaml:"version,omitempty"`
	Managed ExternalManagedConfigV2  `json:"managed,omitempty" yaml:"managed,omitempty"`
	Plugins []ExternalPluginConfigV2 `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Inputs  []ExternalInputConfigV2  `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

// ExternalManagedConfigV2 is an external managed mode configuration.
type ExternalManagedConfigV2 struct {
	Enable   bool                              `json:"enable,omitempty" yaml:"enable,omitempty"`
	Disable  []ExternalManagedDisableConfigV2  `json:"disable,omitempty" yaml:"disable,omitempty"`
	Override []ExternalManagedOverrideConfigV2 `json:"override,omitempty" yaml:"override,omitempty"`
}

// IsEmpty returns true if the config is empty.
func (m ExternalManagedConfigV2) IsEmpty() bool {
	return !m.Enable && len(m.Disable) == 0 && len(m.Override) == 0
}

// ExternalManagedDisableConfigV2 is an external configuration that disables file options in
// managed mode.
type ExternalManagedDisableConfigV2 struct {
	// Must be validated to be a valid FileOption
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// ExternalManagedOverrideConfigV2 is an external configuration that overrides file options in
// managed mode.
type ExternalManagedOverrideConfigV2 struct {
	// Must be validated to be a valid FileOption
	// Required
	FileOption string `json:"file_option,omitempty" yaml:"file_option,omitempty"`
	// Must be validated to be a valid module path
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Must be normalized and validated
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	// Only one of Value and Prefix can be set
	// TODO: may be interface{}, what to do about boo, optimize_mode, etc
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
}

// ExternalPluginConfigV2 is an external plugin configuration.
type ExternalPluginConfigV2 struct {
	// Only one of Remote, Binary, Wasm, ProtocBuiltin can be set
	Remote *string `json:"remote,omitempty" yaml:"remote,omitempty"`
	// Can be multiple arguments
	// All arguments must be strings
	Binary        interface{} `json:"binary,omitempty" yaml:"binary,omitempty"`
	ProtocBuiltin *string     `json:"protoc_builtin,omitempty" yaml:"protoc_builtin,omitempty"`
	// Only valid with Remote
	Revision *int `json:"revision,omitempty" yaml:"revision,omitempty"`
	// Only valid with ProtocBuiltin
	ProtocPath *string `json:"protoc_path,omitempty" yaml:"protoc_path,omitempty"`
	// Required
	Out string `json:"out,omitempty" yaml:"out,omitempty"`
	// Can be one string or multiple strings
	Opt            interface{} `json:"opt,omitempty" yaml:"opt,omitempty"`
	IncludeImports bool        `json:"include_imports,omitempty" yaml:"include_imports,omitempty"`
	IncludeWKT     bool        `json:"include_wkt,omitempty" yaml:"include_wkt,omitempty"`
	// Must be a valid Strategy, only valid with ProtoBuiltin and Binary
	Strategy *string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// ExternalInputConfigV2 is an external input configuration.
type ExternalInputConfigV2 struct {
	// One and only one of Module, Directory, ProtoFile, Tarball, ZipArchive, BinaryImage,
	// JSONImage and GitRepo must be specified as the format.
	Module      *string `json:"module,omitempty" yaml:"module,omitempty"`
	Directory   *string `json:"directory,omitempty" yaml:"directory,omitempty"`
	ProtoFile   *string `json:"proto_file,omitempty" yaml:"proto_file,omitempty"`
	Tarball     *string `json:"tarball,omitempty" yaml:"tarball,omitempty"`
	ZipArchive  *string `json:"zip_archive,omitempty" yaml:"zip_archive,omitempty"`
	BinaryImage *string `json:"binary_image,omitempty" yaml:"binary_image,omitempty"`
	JSONImage   *string `json:"json_image,omitempty" yaml:"json_image,omitempty"`
	GitRepo     *string `json:"git_repo,omitempty" yaml:"git_repo,omitempty"`
	// Compression, StripComponents, Subdir, Branch, Tag, Ref, Depth, RecurseSubmodules
	// and IncludePackageFils are available for only some formats.
	Compression         *string `json:"compression,omitempty" yaml:"compression,omitempty"`
	StripComponents     *uint32 `json:"strip_components,omitempty" yaml:"strip_components,omitempty"`
	Subdir              *string `json:"subdir,omitempty" yaml:"subdir,omitempty"`
	Branch              *string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Tag                 *string `json:"tag,omitempty" yaml:"tag,omitempty"`
	Ref                 *string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Depth               *uint32 `json:"depth,omitempty" yaml:"depth,omitempty"`
	RecurseSubmodules   *bool   `json:"recurse_submodules,omitempty" yaml:"recurse_submodules,omitempty"`
	IncludePackageFiles *bool   `json:"include_package_files,omitempty" yaml:"include_package_files,omitempty"`
	// Types, IncludePaths and ExcludePaths are available for all formats.
	Types        []string `json:"types,omitempty" yaml:"types,omitempty"`
	IncludePaths []string `json:"include_paths,omitempty" yaml:"include_paths,omitempty"`
	ExcludePaths []string `json:"exclude_paths,omitempty" yaml:"exclude_paths,omitempty"`
}

func applyManagementForFile(
	marker bufimagemodifyv2.Marker,
	imageFile bufimage.ImageFile,
	managedConfig *ManagedConfig,
) error {
	for _, fileOption := range AllFileOptions {
		if managedConfig.DisabledFunc(fileOption, imageFile) {
			continue
		}
		var valueOrPrefix string
		var err error
		overrideFunc, ok := managedConfig.FileOptionToOverrideFunc[fileOption]
		if ok {
			valueOrPrefix, err = overrideFunc(imageFile)
			if err != nil {
				return err
			}
		}
		// TODO do the rest
		switch fileOption {
		case FileOptionJavaPackage:
			// Will need to do *string or similar for unset
			if valueOrPrefix == "" {
				valueOrPrefix = defaultJavaPackagePrefix
			}
			if err := bufimagemodifyv2.ModifyJavaPackage(marker, imageFile, valueOrPrefix); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown FileOption: %q", fileOption)
		}
	}
	return nil
}
