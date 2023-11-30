// InputConfig is an input configuration.
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

package bufconfig

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/syserror"
)

// TODO: input type?
type InputConfigType int

const (
	InputConfigTypeModule InputConfigType = iota + 1
	InputConfigTypeDirectory
	InputConfigTypeGitRepo
	InputConfigTypeProtoFile
	InputConfigTypeTarball
	InputConfigTypeZipArchive
	InputConfigTypeBinaryImage
	InputConfigTypeJSONImage
	InputConfigTypeTextImage
)

// Implements fmt.Stringer
func (i InputConfigType) String() string {
	s, ok := inputConfigTypeToString[i]
	if !ok {
		return strconv.Itoa(int(i))
	}
	return s
}

const (
	// TODO: move string literal to maps
	formatGitRepo             = "git_repo"
	formatModule              = "module"
	formatDirectory           = "directory"
	formatProtoFile           = "proto_file"
	formatBinaryImage         = "binary_image"
	formatTarball             = "tarball"
	formatZipArchive          = "zip_archive"
	formatJSONImage           = "json_image"
	formatTextImage           = "text_image"
	optionCompression         = "compression"
	optionBranch              = "branch"
	optionTag                 = "tag"
	optionRef                 = "ref"
	optionDepth               = "depth"
	optionRecurseSubmodules   = "recurse_submodules"
	optionStripComponents     = "strip_components"
	optionSubdir              = "subdir"
	optionIncludePackageFiles = "include_package_files"
)

var allowedOptionsForFormat = map[InputConfigType](map[string]bool){
	InputConfigTypeGitRepo: {
		optionBranch:            true,
		optionTag:               true,
		optionRef:               true,
		optionDepth:             true,
		optionRecurseSubmodules: true,
		optionSubdir:            true,
	},
	InputConfigTypeModule:    {},
	InputConfigTypeDirectory: {},
	InputConfigTypeProtoFile: {
		optionIncludePackageFiles: true,
	},
	InputConfigTypeTarball: {
		optionCompression:     true,
		optionStripComponents: true,
		optionSubdir:          true,
	},
	InputConfigTypeZipArchive: {
		optionStripComponents: true,
		optionSubdir:          true,
	},
	InputConfigTypeBinaryImage: {
		optionCompression: true,
	},
	InputConfigTypeJSONImage: {
		optionCompression: true,
	},
	InputConfigTypeTextImage: {
		optionCompression: true,
	},
}

var inputConfigTypeToString = map[InputConfigType]string{
	InputConfigTypeGitRepo:     formatGitRepo,
	InputConfigTypeModule:      formatModule,
	InputConfigTypeDirectory:   formatDirectory,
	InputConfigTypeProtoFile:   formatProtoFile,
	InputConfigTypeTarball:     formatTarball,
	InputConfigTypeZipArchive:  formatZipArchive,
	InputConfigTypeBinaryImage: formatBinaryImage,
	InputConfigTypeJSONImage:   formatJSONImage,
	InputConfigTypeTextImage:   formatTextImage,
}

// InputConfig is an input configuration for code generation.
type InputConfig interface {
	// Type returns the input type.
	Type() InputConfigType
	// Location returns the location for the input.
	Location() string
	// Compression returns the compression scheme, not empty only if format is
	// one of tarball, binary image, json image or text image.
	Compression() string
	// StripComponents returns the number of directories to strip for tar or zip
	// inputs, not empty only if format is tarball or zip archive.
	StripComponents() *uint32
	// SubDir returns the subdirectory to use, not empty only if format is one
	// git repo, tarball and zip archive.
	SubDir() string
	// Branch returns the git branch to checkout out, not empty only if format is git.
	Branch() string
	// Tag returns the git tag to checkout, not empty only if format is git.
	Tag() string
	// Ref returns the git ref to checkout, not empty only if format is git.
	Ref() string
	// Ref returns the depth to clone the git repo with, not empty only if format is git.
	Depth() *uint32
	// RecurseSubmodules returns whether to clone submodules recursively. Not empty
	// only if input if git.
	RecurseSubmodules() bool
	// IncludePackageFiles returns other files in the same package as the proto file,
	// not empty only if format is proto file.
	IncludePackageFiles() bool
	// IncludePaths returns paths to generate for.
	IncludePaths() []string
	// ExcludePaths returns paths not to generate for.
	ExcludePaths() []string
	// IncludeTypes returns the types to generate. If GenerateConfig.GenerateTypeConfig()
	// returns a non-empty list of types.
	IncludeTypes() []string

	isInputConfig()
}

// NewInputConfig returns a new input config.
func NewInputConfig(
	inputType InputConfigType,
	location string,
	compression string,
	stripComponents *uint32,
	subDir string,
	branch string,
	tag string,
	ref string,
	depth *uint32,
	recurseSubmodules bool,
	includePackageFiles bool,
	includePaths []string,
	excludePaths []string,
	includeTypes []string,
) InputConfig {
	return &inputConfig{
		inputType:           inputType,
		location:            location,
		compression:         compression,
		stripComponents:     stripComponents,
		subDir:              subDir,
		branch:              branch,
		tag:                 tag,
		ref:                 ref,
		depth:               depth,
		recurseSubmodules:   recurseSubmodules,
		includePackageFiles: includePackageFiles,
		includePaths:        includePaths,
		excludePaths:        excludePaths,
		includeTypes:        includeTypes,
	}
}

// NewGitRepoInputConfig returns an input config for a git repo.
func NewGitRepoInputConfig(
	location string,
	subDir string,
	branch string,
	tag string,
	ref string,
	depth *uint32,
	recurseSubModules bool,
) InputConfig {
	return &inputConfig{
		inputType:         InputConfigTypeGitRepo,
		location:          location,
		subDir:            subDir,
		branch:            branch,
		tag:               tag,
		ref:               ref,
		depth:             depth,
		recurseSubmodules: recurseSubModules,
	}
}

// NewModuleInputConfig returns an input config for a module.
func NewModuleInputConfig(
	location string,
) InputConfig {
	return &inputConfig{
		inputType: InputConfigTypeModule,
		location:  location,
	}
}

// NewDirectoryInputConfig returns an input config for a directory.
func NewDirectoryInputConfig(
	location string,
) InputConfig {
	return &inputConfig{
		inputType: InputConfigTypeDirectory,
		location:  location,
	}
}

// NewProtoFileInputConfig returns an input config for a proto file.
func NewProtoFileInputConfig(
	location string,
) InputConfig {
	return &inputConfig{
		inputType: InputConfigTypeProtoFile,
		location:  location,
	}
}

// NewTarballInputConfig returns an input config for a tarball.
func NewTarballInputConfig(
	location string,
	subDir string,
	compression string,
	stripComponents *uint32,
) InputConfig {
	return &inputConfig{
		inputType:       InputConfigTypeTarball,
		location:        location,
		subDir:          subDir,
		compression:     compression,
		stripComponents: stripComponents,
	}
}

// NewZipArchiveInputConfig returns an input config for a zip archive.
func NewZipArchiveInputConfig(
	location string,
	subDir string,
	stripComponents *uint32,
) InputConfig {
	return &inputConfig{
		inputType:       InputConfigTypeZipArchive,
		location:        location,
		subDir:          subDir,
		stripComponents: stripComponents,
	}
}

// NewBinaryImageInputConfig returns an input config for a binary image.
func NewBinaryImageInputConfig(
	location string,
	compression string,
) InputConfig {
	return &inputConfig{
		inputType:   InputConfigTypeBinaryImage,
		location:    location,
		compression: compression,
	}
}

// NewJSONImageInputConfig returns an input config for a JSON image.
func NewJSONImageInputConfig(
	location string,
	compression string,
) InputConfig {
	return &inputConfig{
		inputType:   InputConfigTypeJSONImage,
		location:    location,
		compression: compression,
	}
}

// NewTextImageInputConfig returns an input config for a text image.
func NewTextImageInputConfig(
	location string,
	compression string,
) InputConfig {
	return &inputConfig{
		inputType:   InputConfigTypeTextImage,
		location:    location,
		compression: compression,
	}
}

// *** PRIVATE ***

type inputConfig struct {
	inputType   InputConfigType
	location    string
	compression string
	// TODO: does it make sense to be a pointer?
	stripComponents     *uint32
	subDir              string
	branch              string
	tag                 string
	ref                 string
	depth               *uint32
	recurseSubmodules   bool
	includePackageFiles bool
	includeTypes        []string
	excludePaths        []string
	includePaths        []string
}

func newInputConfigFromExternalInputConfigV2(externalConfig externalInputConfigV2) (InputConfig, error) {
	inputConfig := &inputConfig{}
	var inputTypes []InputConfigType
	var options []string
	if externalConfig.Module != nil {
		inputTypes = append(inputTypes, InputConfigTypeModule)
		inputConfig.location = *externalConfig.Module
	}
	if externalConfig.Directory != nil {
		inputTypes = append(inputTypes, InputConfigTypeDirectory)
		inputConfig.location = *externalConfig.Directory
	}
	if externalConfig.ProtoFile != nil {
		inputTypes = append(inputTypes, InputConfigTypeProtoFile)
		inputConfig.location = *externalConfig.ProtoFile
	}
	if externalConfig.BinaryImage != nil {
		inputTypes = append(inputTypes, InputConfigTypeBinaryImage)
		inputConfig.location = *externalConfig.BinaryImage
	}
	if externalConfig.Tarball != nil {
		inputTypes = append(inputTypes, InputConfigTypeTarball)
		inputConfig.location = *externalConfig.Tarball
	}
	if externalConfig.ZipArchive != nil {
		inputTypes = append(inputTypes, InputConfigTypeZipArchive)
		inputConfig.location = *externalConfig.ZipArchive
	}
	if externalConfig.JSONImage != nil {
		inputTypes = append(inputTypes, InputConfigTypeJSONImage)
		inputConfig.location = *externalConfig.JSONImage
	}
	if externalConfig.TextImage != nil {
		inputTypes = append(inputTypes, InputConfigTypeTextImage)
		inputConfig.location = *externalConfig.TextImage
	}
	if externalConfig.GitRepo != nil {
		inputTypes = append(inputTypes, InputConfigTypeGitRepo)
		inputConfig.location = *externalConfig.GitRepo
	}
	if externalConfig.Compression != nil {
		options = append(options, optionCompression)
		inputConfig.compression = *externalConfig.Compression
	}
	if externalConfig.StripComponents != nil {
		options = append(options, optionStripComponents)
		inputConfig.stripComponents = externalConfig.StripComponents
	}
	if externalConfig.Subdir != nil {
		options = append(options, optionSubdir)
		inputConfig.subDir = *externalConfig.Subdir
	}
	if externalConfig.Branch != nil {
		options = append(options, optionBranch)
		inputConfig.branch = *externalConfig.Branch
	}
	if externalConfig.Tag != nil {
		options = append(options, optionTag)
		inputConfig.tag = *externalConfig.Tag
	}
	if externalConfig.Ref != nil {
		options = append(options, optionRef)
		inputConfig.ref = *externalConfig.Ref
	}
	if externalConfig.Depth != nil {
		options = append(options, optionDepth)
		inputConfig.depth = externalConfig.Depth
	}
	if externalConfig.RecurseSubmodules != nil {
		options = append(options, optionRecurseSubmodules)
		inputConfig.recurseSubmodules = *externalConfig.RecurseSubmodules
	}
	if externalConfig.IncludePackageFiles != nil {
		options = append(options, optionIncludePackageFiles)
		inputConfig.includePackageFiles = *externalConfig.IncludePackageFiles
	}
	if len(inputTypes) == 0 {
		return nil, errors.New("must specify input type")
	}
	if len(inputTypes) > 1 {
		// TODO: print out all types allowed
		return nil, fmt.Errorf("exactly one input type can be specified")
	}
	format := inputTypes[0]
	allowedOptions, ok := allowedOptionsForFormat[format]
	if !ok {
		// this should not happen
		return nil, fmt.Errorf("unable to find allowed options for format %v", format)
	}
	for _, option := range options {
		if !allowedOptions[option] {
			return nil, fmt.Errorf("option %s is not allowed for format %v", option, format)
		}
	}
	return inputConfig, nil
}

func newExternalInputConfigV2FromInputConfig(
	inputConfig InputConfig,
) (externalInputConfigV2, error) {
	externalInputConfigV2 := externalInputConfigV2{}
	switch inputConfig.Type() {
	case InputConfigTypeGitRepo:
		externalInputConfigV2.GitRepo = toPointer(inputConfig.Location())
	case InputConfigTypeDirectory:
		externalInputConfigV2.Directory = toPointer(inputConfig.Location())
	case InputConfigTypeModule:
		externalInputConfigV2.Module = toPointer(inputConfig.Location())
	case InputConfigTypeProtoFile:
		externalInputConfigV2.ProtoFile = toPointer(inputConfig.Location())
	case InputConfigTypeZipArchive:
		externalInputConfigV2.ZipArchive = toPointer(inputConfig.Location())
	case InputConfigTypeTarball:
		externalInputConfigV2.Tarball = toPointer(inputConfig.Location())
	case InputConfigTypeBinaryImage:
		externalInputConfigV2.BinaryImage = toPointer(inputConfig.Location())
	case InputConfigTypeJSONImage:
		externalInputConfigV2.JSONImage = toPointer(inputConfig.Location())
	case InputConfigTypeTextImage:
		externalInputConfigV2.TextImage = toPointer(inputConfig.Location())
	default:
		return externalInputConfigV2, syserror.Newf("unknown input config type: %v", inputConfig.Type())
	}
	if inputConfig.Branch() != "" {
		externalInputConfigV2.Branch = toPointer(inputConfig.Branch())
	}
	if inputConfig.Ref() != "" {
		externalInputConfigV2.Ref = toPointer(inputConfig.Ref())
	}
	if inputConfig.Tag() != "" {
		externalInputConfigV2.Tag = toPointer(inputConfig.Tag())
	}
	externalInputConfigV2.Depth = inputConfig.Depth()
	// TODO: make RecurseSubmodules return a pointer for more accurate representation
	if inputConfig.RecurseSubmodules() {
		externalInputConfigV2.RecurseSubmodules = toPointer(inputConfig.RecurseSubmodules())
	}
	if inputConfig.Compression() != "" {
		externalInputConfigV2.Compression = toPointer(inputConfig.Compression())
	}
	externalInputConfigV2.StripComponents = inputConfig.StripComponents()
	if inputConfig.SubDir() != "" {
		externalInputConfigV2.Subdir = toPointer(inputConfig.SubDir())
	}
	// TODO: make IncludePackageFiles return a pointer for more accurate representation
	if inputConfig.IncludePackageFiles() {
		externalInputConfigV2.IncludePackageFiles = toPointer(inputConfig.IncludePackageFiles())
	}
	externalInputConfigV2.IncludePaths = inputConfig.IncludePaths()
	externalInputConfigV2.ExcludePaths = inputConfig.ExcludePaths()
	externalInputConfigV2.Types = inputConfig.IncludeTypes()
	return externalInputConfigV2, nil
}

func (i *inputConfig) Type() InputConfigType {
	return i.inputType
}

func (i *inputConfig) Location() string {
	return i.location
}

func (i *inputConfig) Compression() string {
	return i.compression
}

func (i *inputConfig) StripComponents() *uint32 {
	return i.stripComponents
}

func (i *inputConfig) SubDir() string {
	return i.subDir
}

func (i *inputConfig) Branch() string {
	return i.branch
}

func (i *inputConfig) Tag() string {
	return i.tag
}

func (i *inputConfig) Ref() string {
	return i.ref
}

func (i *inputConfig) Depth() *uint32 {
	return i.depth
}

func (i *inputConfig) RecurseSubmodules() bool {
	return i.recurseSubmodules
}

func (i *inputConfig) IncludePackageFiles() bool {
	return i.includePackageFiles
}

func (i *inputConfig) ExcludePaths() []string {
	return i.excludePaths
}

func (i *inputConfig) IncludePaths() []string {
	return i.includePaths
}

func (i *inputConfig) IncludeTypes() []string {
	return i.includeTypes
}

func (i *inputConfig) isInputConfig() {}
