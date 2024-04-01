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

package bufconfig

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// InputConfigType is an input config's type.
type InputConfigType int

const (
	// InputConfigTypeModule is the module input type.
	InputConfigTypeModule InputConfigType = iota + 1
	// InputConfigTypeDirectory is the directory input type.
	InputConfigTypeDirectory
	// InputConfigTypeGitRepo is the git repository input type.
	InputConfigTypeGitRepo
	// InputConfigTypeProtoFile is the proto file input type.
	InputConfigTypeProtoFile
	// InputConfigTypeTarball is the tarball input type.
	InputConfigTypeTarball
	// InputConfigTypeZipArchive is the zip archive input type.
	InputConfigTypeZipArchive
	// InputConfigTypeBinaryImage is the binary image input type.
	InputConfigTypeBinaryImage
	// InputConfigTypeJSONImage is the JSON image input type.
	InputConfigTypeJSONImage
	// InputConfigTypeTextImage is the text image input type.
	InputConfigTypeTextImage
	// InputConfigTypeYAMLImage is the yaml image input type.
	InputConfigTypeYAMLImage
)

// String implements fmt.Stringer.
func (i InputConfigType) String() string {
	s, ok := inputConfigTypeToString[i]
	if !ok {
		return strconv.Itoa(int(i))
	}
	return s
}

const (
	compressionKey         = "compression"
	branchKey              = "branch"
	commitKey              = "commit"
	tagKey                 = "tag"
	refKey                 = "ref"
	depthKey               = "depth"
	recurseSubmodulesKey   = "recurse_submodules"
	stripComponentsKey     = "strip_components"
	subDirKey              = "subdir"
	includePackageFilesKey = "include_package_files"
)

var (
	allowedOptionsForInputConfigType = map[InputConfigType](map[string]struct{}){
		InputConfigTypeGitRepo: {
			branchKey:            {},
			commitKey:            {},
			tagKey:               {},
			refKey:               {},
			depthKey:             {},
			recurseSubmodulesKey: {},
			subDirKey:            {},
		},
		InputConfigTypeModule:    {},
		InputConfigTypeDirectory: {},
		InputConfigTypeProtoFile: {
			includePackageFilesKey: {},
		},
		InputConfigTypeTarball: {
			compressionKey:     {},
			stripComponentsKey: {},
			subDirKey:          {},
		},
		InputConfigTypeZipArchive: {
			stripComponentsKey: {},
			subDirKey:          {},
		},
		InputConfigTypeBinaryImage: {
			compressionKey: {},
		},
		InputConfigTypeJSONImage: {
			compressionKey: {},
		},
		InputConfigTypeTextImage: {
			compressionKey: {},
		},
		InputConfigTypeYAMLImage: {
			compressionKey: {},
		},
	}
	inputConfigTypeToString = map[InputConfigType]string{
		InputConfigTypeGitRepo:     "git_repo",
		InputConfigTypeModule:      "module",
		InputConfigTypeDirectory:   "directory",
		InputConfigTypeProtoFile:   "proto_file",
		InputConfigTypeTarball:     "tarball",
		InputConfigTypeZipArchive:  "zip_archive",
		InputConfigTypeBinaryImage: "binary_image",
		InputConfigTypeJSONImage:   "json_image",
		InputConfigTypeTextImage:   "text_image",
		InputConfigTypeYAMLImage:   "yaml_image",
	}
	allInputConfigTypeString = stringutil.SliceToHumanString(
		slicesext.MapValuesToSortedSlice(inputConfigTypeToString),
	)
)

// InputConfig is an input configuration for code generation.
type InputConfig interface {
	// Type returns the input type. This is never the zero value.
	Type() InputConfigType
	// Location returns the location for the input. This is never empty.
	Location() string
	// Compression returns the compression scheme, not empty only if format is
	// one of tarball, binary image, json image or text image.
	Compression() string
	// StripComponents returns the number of directories to strip for tar or zip
	// inputs, not empty only if format is tarball or zip archive.
	StripComponents() uint32
	// SubDir returns the subdirectory to use, not empty only if format is one
	// git repo, tarball and zip archive.
	SubDir() string
	// Branch returns the git branch to checkout out, not empty only if format is git.
	Branch() string
	// CommitOrTag returns the full commit hash or tag to checkout, not empty only if format is git.
	CommitOrTag() string
	// Ref returns the git ref to checkout, not empty only if format is git.
	Ref() string
	// Depth returns the depth to clone the git repo with, not empty only if format is git.
	Depth() *uint32
	// RecurseSubmodules returns whether to clone submodules recursively. Not empty
	// only if input if git.
	RecurseSubmodules() bool
	// IncludePackageFiles returns other files in the same package as the proto file,
	// not empty only if format is proto file.
	IncludePackageFiles() bool
	// TargetPaths returns paths to generate for. An empty slice means to generate for all paths.
	TargetPaths() []string
	// ExcludePaths returns paths not to generate for.
	ExcludePaths() []string
	// IncludeTypes returns the types to generate. An empty slice means to generate for all types.
	IncludeTypes() []string

	isInputConfig()
}

// NewGitRepoInputConfig returns an input config for a git repo.
func NewGitRepoInputConfig(
	location string,
	subDir string,
	branch string,
	commitOrTag string,
	ref string,
	depth *uint32,
	recurseSubModules bool,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for git repository")
	}
	return &inputConfig{
		inputConfigType:   InputConfigTypeGitRepo,
		location:          location,
		subDir:            subDir,
		branch:            branch,
		commitOrTag:       commitOrTag,
		ref:               ref,
		depth:             depth,
		recurseSubmodules: recurseSubModules,
	}, nil
}

// NewModuleInputConfig returns an input config for a module.
func NewModuleInputConfig(
	location string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for module")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeModule,
		location:        location,
	}, nil
}

// NewDirectoryInputConfig returns an input config for a directory.
func NewDirectoryInputConfig(
	location string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for directory")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeDirectory,
		location:        location,
	}, nil
}

// NewProtoFileInputConfig returns an input config for a proto file.
func NewProtoFileInputConfig(
	location string,
	includePackageFiles bool,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for proto file")
	}
	return &inputConfig{
		inputConfigType:     InputConfigTypeProtoFile,
		location:            location,
		includePackageFiles: includePackageFiles,
	}, nil
}

// NewTarballInputConfig returns an input config for a tarball.
func NewTarballInputConfig(
	location string,
	subDir string,
	compression string,
	stripComponents uint32,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for tarball")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeTarball,
		location:        location,
		subDir:          subDir,
		compression:     compression,
		stripComponents: stripComponents,
	}, nil
}

// NewZipArchiveInputConfig returns an input config for a zip archive.
func NewZipArchiveInputConfig(
	location string,
	subDir string,
	stripComponents uint32,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for zip archive")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeZipArchive,
		location:        location,
		subDir:          subDir,
		stripComponents: stripComponents,
	}, nil
}

// NewBinaryImageInputConfig returns an input config for a binary image.
func NewBinaryImageInputConfig(
	location string,
	compression string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for binary image")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeBinaryImage,
		location:        location,
		compression:     compression,
	}, nil
}

// NewJSONImageInputConfig returns an input config for a JSON image.
func NewJSONImageInputConfig(
	location string,
	compression string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for JSON image")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeJSONImage,
		location:        location,
		compression:     compression,
	}, nil
}

// NewTextImageInputConfig returns an input config for a text image.
func NewTextImageInputConfig(
	location string,
	compression string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for text image")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeTextImage,
		location:        location,
		compression:     compression,
	}, nil
}

// NewYAMLImageInputConfig returns an input config for a yaml image.
func NewYAMLImageInputConfig(
	location string,
	compression string,
) (InputConfig, error) {
	if location == "" {
		return nil, errors.New("empty location for yaml image")
	}
	return &inputConfig{
		inputConfigType: InputConfigTypeYAMLImage,
		location:        location,
		compression:     compression,
	}, nil
}

// *** PRIVATE ***

type inputConfig struct {
	inputConfigType     InputConfigType
	location            string
	compression         string
	stripComponents     uint32
	subDir              string
	branch              string
	commitOrTag         string
	ref                 string
	depth               *uint32
	recurseSubmodules   bool
	includePackageFiles bool
	includeTypes        []string
	targetPaths         []string
	excludePaths        []string
}

func newInputConfigFromExternalV2(externalConfig externalInputConfigV2) (InputConfig, error) {
	// We only allow one of these to be set.
	inputConfig := &inputConfig{}
	var inputConfigTypes []InputConfigType
	if externalConfig.Module != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeModule)
		inputConfig.location = *externalConfig.Module
	}
	if externalConfig.Directory != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeDirectory)
		inputConfig.location = *externalConfig.Directory
	}
	if externalConfig.ProtoFile != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeProtoFile)
		inputConfig.location = *externalConfig.ProtoFile
	}
	if externalConfig.BinaryImage != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeBinaryImage)
		inputConfig.location = *externalConfig.BinaryImage
	}
	if externalConfig.Tarball != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeTarball)
		inputConfig.location = *externalConfig.Tarball
	}
	if externalConfig.ZipArchive != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeZipArchive)
		inputConfig.location = *externalConfig.ZipArchive
	}
	if externalConfig.JSONImage != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeJSONImage)
		inputConfig.location = *externalConfig.JSONImage
	}
	if externalConfig.TextImage != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeTextImage)
		inputConfig.location = *externalConfig.TextImage
	}
	if externalConfig.YAMLImage != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeYAMLImage)
		inputConfig.location = *externalConfig.YAMLImage
	}
	if externalConfig.GitRepo != nil {
		inputConfigTypes = append(inputConfigTypes, InputConfigTypeGitRepo)
		inputConfig.location = *externalConfig.GitRepo
	}
	if len(inputConfigTypes) == 0 {
		return nil, fmt.Errorf("must specify one of %s", allInputConfigTypeString)
	}
	if len(inputConfigTypes) > 1 {
		return nil, fmt.Errorf("exactly one of %s must be specified", allInputConfigTypeString)
	}
	inputConfigType := inputConfigTypes[0]
	inputConfig.inputConfigType = inputConfigType
	// Types, TargetPaths, and ExcludePaths.
	inputConfig.includeTypes = externalConfig.Types
	inputConfig.targetPaths = externalConfig.TargetPaths
	inputConfig.excludePaths = externalConfig.ExcludePaths
	// Options depending on input format.
	var options []string
	if externalConfig.Compression != nil {
		options = append(options, compressionKey)
		inputConfig.compression = *externalConfig.Compression
	}
	if externalConfig.StripComponents != nil {
		options = append(options, stripComponentsKey)
		inputConfig.stripComponents = *externalConfig.StripComponents
	}
	if externalConfig.Subdir != nil {
		options = append(options, subDirKey)
		inputConfig.subDir = *externalConfig.Subdir
	}
	if externalConfig.Branch != nil {
		options = append(options, branchKey)
		inputConfig.branch = *externalConfig.Branch
	}
	if externalConfig.Commit != nil && externalConfig.Tag != nil {
		return nil, fmt.Errorf("commit and tag options cannot be used at the same time; use one or the other")
	}
	if externalConfig.Commit != nil {
		options = append(options, commitKey)
		inputConfig.commitOrTag = *externalConfig.Commit
	}
	if externalConfig.Tag != nil {
		options = append(options, tagKey)
		inputConfig.commitOrTag = *externalConfig.Tag
	}
	if externalConfig.Ref != nil {
		options = append(options, refKey)
		inputConfig.ref = *externalConfig.Ref
	}
	if externalConfig.Depth != nil {
		options = append(options, depthKey)
		inputConfig.depth = externalConfig.Depth
	}
	if externalConfig.RecurseSubmodules != nil {
		options = append(options, recurseSubmodulesKey)
		inputConfig.recurseSubmodules = *externalConfig.RecurseSubmodules
	}
	if externalConfig.IncludePackageFiles != nil {
		options = append(options, includePackageFilesKey)
		inputConfig.includePackageFiles = *externalConfig.IncludePackageFiles
	}
	allowedOptions, ok := allowedOptionsForInputConfigType[inputConfigType]
	if !ok {
		return nil, syserror.Newf("unable to find allowed options for InputConfigType %v", inputConfigType)
	}
	for _, option := range options {
		if _, ok := allowedOptions[option]; !ok {
			return nil, fmt.Errorf("option %s is not allowed for InputConfigType %v", option, inputConfigType)
		}
	}
	return inputConfig, nil
}

func (i *inputConfig) Type() InputConfigType {
	return i.inputConfigType
}

func (i *inputConfig) Location() string {
	return i.location
}

func (i *inputConfig) Compression() string {
	return i.compression
}

func (i *inputConfig) StripComponents() uint32 {
	return i.stripComponents
}

func (i *inputConfig) SubDir() string {
	return i.subDir
}

func (i *inputConfig) Branch() string {
	return i.branch
}

func (i *inputConfig) CommitOrTag() string {
	return i.commitOrTag
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

func (i *inputConfig) TargetPaths() []string {
	return i.targetPaths
}

func (i *inputConfig) IncludeTypes() []string {
	return i.includeTypes
}

func (i *inputConfig) isInputConfig() {}

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
	case InputConfigTypeYAMLImage:
		externalInputConfigV2.YAMLImage = toPointer(inputConfig.Location())
	default:
		return externalInputConfigV2, syserror.Newf("unknown input config type: %v", inputConfig.Type())
	}
	if inputConfig.Branch() != "" {
		externalInputConfigV2.Branch = toPointer(inputConfig.Branch())
	}
	if inputConfig.Ref() != "" {
		externalInputConfigV2.Ref = toPointer(inputConfig.Ref())
	}
	if inputConfig.CommitOrTag() != "" {
		externalInputConfigV2.Commit = toPointer(inputConfig.CommitOrTag())
	}
	externalInputConfigV2.Depth = inputConfig.Depth()
	if inputConfig.RecurseSubmodules() {
		externalInputConfigV2.RecurseSubmodules = toPointer(inputConfig.RecurseSubmodules())
	}
	if inputConfig.Compression() != "" {
		externalInputConfigV2.Compression = toPointer(inputConfig.Compression())
	}
	if inputConfig.StripComponents() != 0 {
		externalInputConfigV2.StripComponents = toPointer(inputConfig.StripComponents())
	}
	if inputConfig.SubDir() != "" {
		externalInputConfigV2.Subdir = toPointer(inputConfig.SubDir())
	}
	if inputConfig.IncludePackageFiles() {
		externalInputConfigV2.IncludePackageFiles = toPointer(inputConfig.IncludePackageFiles())
	}
	externalInputConfigV2.TargetPaths = inputConfig.TargetPaths()
	externalInputConfigV2.ExcludePaths = inputConfig.ExcludePaths()
	externalInputConfigV2.Types = inputConfig.IncludeTypes()
	return externalInputConfigV2, nil
}
