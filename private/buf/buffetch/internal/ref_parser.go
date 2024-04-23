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

package internal

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

type refParser struct {
	logger                *zap.Logger
	rawRefProcessor       func(*RawRef) error
	singleFormatToInfo    map[string]*singleFormatInfo
	archiveFormatToInfo   map[string]*archiveFormatInfo
	dirFormatToInfo       map[string]*dirFormatInfo
	gitFormatToInfo       map[string]*gitFormatInfo
	moduleFormatToInfo    map[string]*moduleFormatInfo
	protoFileFormatToInfo map[string]*protoFileFormatInfo
}

func newRefParser(logger *zap.Logger, options ...RefParserOption) *refParser {
	refParser := &refParser{
		logger:                logger,
		singleFormatToInfo:    make(map[string]*singleFormatInfo),
		archiveFormatToInfo:   make(map[string]*archiveFormatInfo),
		dirFormatToInfo:       make(map[string]*dirFormatInfo),
		gitFormatToInfo:       make(map[string]*gitFormatInfo),
		moduleFormatToInfo:    make(map[string]*moduleFormatInfo),
		protoFileFormatToInfo: make(map[string]*protoFileFormatInfo),
	}
	for _, option := range options {
		option(refParser)
	}
	return refParser
}

func (a *refParser) GetParsedRef(
	ctx context.Context,
	value string,
	options ...GetParsedRefOption,
) (ParsedRef, error) {
	getParsedRefOptions := newGetParsedRefOptions()
	for _, option := range options {
		option(getParsedRefOptions)
	}
	return a.getParsedRef(ctx, value, getParsedRefOptions.allowedFormats)
}

func (a *refParser) GetParsedRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
	options ...GetParsedRefOption,
) (ParsedRef, error) {
	getParsedRefOptions := newGetParsedRefOptions()
	for _, option := range options {
		option(getParsedRefOptions)
	}
	return a.getParsedRefForInputConfig(ctx, inputConfig, getParsedRefOptions.allowedFormats)
}

func (a *refParser) getParsedRef(
	ctx context.Context,
	value string,
	allowedFormats map[string]struct{},
) (ParsedRef, error) {
	// path is never empty after returning from this function
	path, options, err := getRawPathAndOptions(value)
	if err != nil {
		return nil, err
	}
	rawRef, err := a.getRawRef(path, value, options)
	if err != nil {
		return nil, err
	}
	return a.parseRawRef(rawRef, allowedFormats)
}

func (a *refParser) getParsedRefForInputConfig(
	ctx context.Context,
	inputConfig bufconfig.InputConfig,
	allowedFormats map[string]struct{},
) (ParsedRef, error) {
	rawRef, err := a.getRawRefForInputConfig(inputConfig)
	if err != nil {
		return nil, err
	}
	return a.parseRawRef(rawRef, allowedFormats)
}

func (a *refParser) getRawRef(
	path string,
	// Used to reference the input config in error messages.
	displayName string,
	options map[string]string,
) (*RawRef, error) {
	rawRef := &RawRef{
		Path:                path,
		UnrecognizedOptions: make(map[string]string),
	}
	if a.rawRefProcessor != nil {
		if err := a.rawRefProcessor(rawRef); err != nil {
			return nil, err
		}
	}
	for key, value := range options {
		switch key {
		case "format":
			if app.IsDevNull(path) {
				return nil, NewFormatOverrideNotAllowedForDevNullError(app.DevNullFilePath)
			}
			rawRef.Format = value
		case "compression":
			compressionType, err := parseCompressionType(value)
			if err != nil {
				return nil, err
			}
			rawRef.CompressionType = compressionType
		case "branch":
			rawRef.GitBranch = value
		case "tag", "commit":
			rawRef.GitCommitOrTag = value
		case "ref":
			rawRef.GitRef = value
		case "depth":
			depth, err := parseGitDepth(value)
			if err != nil {
				return nil, err
			}
			rawRef.GitDepth = depth
		case "recurse_submodules":
			// TODO FUTURE: need to refactor to make sure this is not set for any non-git input
			// ie right now recurse_submodules=false will not error
			switch value {
			case "true":
				rawRef.GitRecurseSubmodules = true
			case "false":
			default:
				return nil, NewOptionsCouldNotParseRecurseSubmodulesError(value)
			}
		case "strip_components":
			// TODO FUTURE: need to refactor to make sure this is not set for any non-tarball
			// ie right now strip_components=0 will not error
			stripComponents, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, NewOptionsCouldNotParseStripComponentsError(value)
			}
			rawRef.ArchiveStripComponents = uint32(stripComponents)
		case "subdir":
			subDirPath, err := parseSubDirPath(value)
			if err != nil {
				return nil, err
			}
			rawRef.SubDirPath = subDirPath
		case "include_package_files":
			switch value {
			case "true":
				rawRef.IncludePackageFiles = true
			case "false", "":
				rawRef.IncludePackageFiles = false
			default:
				return nil, NewOptionsInvalidValueForKeyError(key, value)
			}
		default:
			rawRef.UnrecognizedOptions[key] = value
		}
	}
	// This cannot be set ahead of time, it can only happen after all options are read.
	if rawRef.Format == "git" && rawRef.GitDepth == 0 {
		// Default to 1
		rawRef.GitDepth = 1
		if rawRef.GitRef != "" {
			// Default to 50 when using ref
			rawRef.GitDepth = 50
		}
	}
	if err := a.validateRawRef(displayName, rawRef); err != nil {
		return nil, err
	}
	return rawRef, nil
}

func (a *refParser) getRawRefForInputConfig(
	inputConfig bufconfig.InputConfig,
) (*RawRef, error) {
	rawRef := &RawRef{
		Path:                inputConfig.Location(),
		UnrecognizedOptions: make(map[string]string),
	}
	if a.rawRefProcessor != nil {
		if err := a.rawRefProcessor(rawRef); err != nil {
			return nil, err
		}
	}
	switch inputConfig.Type() {
	case bufconfig.InputConfigTypeModule:
		rawRef.Format = "mod"
	case bufconfig.InputConfigTypeDirectory:
		rawRef.Format = "dir"
	case bufconfig.InputConfigTypeGitRepo:
		rawRef.Format = "git"
	case bufconfig.InputConfigTypeProtoFile:
		rawRef.Format = "protofile"
	case bufconfig.InputConfigTypeTarball:
		rawRef.Format = "tar"
	case bufconfig.InputConfigTypeZipArchive:
		rawRef.Format = "zip"
	case bufconfig.InputConfigTypeBinaryImage:
		rawRef.Format = "binpb"
	case bufconfig.InputConfigTypeJSONImage:
		rawRef.Format = "json"
	case bufconfig.InputConfigTypeTextImage:
		rawRef.Format = "txtpb"
	case bufconfig.InputConfigTypeYAMLImage:
		rawRef.Format = "yaml"
	default:
		return nil, syserror.Newf("unknown InputConfigType: %v", inputConfig.Type())
	}
	// This cannot be set ahead of time, it can only happen after all options are read.
	if inputConfig.Type() == bufconfig.InputConfigTypeGitRepo {
		// TODO FUTURE: might change rawRef.Depth into a pointer or use some other way to handle the case where 0 is specified
		if inputConfig.Depth() != nil {
			if *inputConfig.Depth() == 0 {
				return nil, NewDepthZeroError()
			}
			rawRef.GitDepth = *inputConfig.Depth()
		}
		rawRef.GitBranch = inputConfig.Branch()
		rawRef.GitCommitOrTag = inputConfig.CommitOrTag()
		rawRef.GitRef = inputConfig.Ref()
		rawRef.GitRecurseSubmodules = inputConfig.RecurseSubmodules()
		if rawRef.GitDepth == 0 {
			// Default to 1
			rawRef.GitDepth = 1
			if rawRef.GitRef != "" {
				// Default to 50 when using ref
				rawRef.GitDepth = 50
			}
		}
	}
	var err error
	if compression := inputConfig.Compression(); compression != "" {
		rawRef.CompressionType, err = parseCompressionType(compression)
		if err != nil {
			return nil, err
		}
	}
	rawRef.IncludePackageFiles = inputConfig.IncludePackageFiles()
	rawRef.ArchiveStripComponents = inputConfig.StripComponents()
	rawRef.SubDirPath, err = parseSubDirPath(inputConfig.SubDir())
	if err != nil {
		return nil, err
	}
	if err := a.validateRawRef(inputConfig.Location(), rawRef); err != nil {
		return nil, err
	}
	return rawRef, nil
}

func (a *refParser) parseRawRef(
	rawRef *RawRef,
	allowedFormats map[string]struct{},
) (ParsedRef, error) {
	singleFormatInfo, singleOK := a.singleFormatToInfo[rawRef.Format]
	archiveFormatInfo, archiveOK := a.archiveFormatToInfo[rawRef.Format]
	_, dirOK := a.dirFormatToInfo[rawRef.Format]
	_, gitOK := a.gitFormatToInfo[rawRef.Format]
	_, moduleOK := a.moduleFormatToInfo[rawRef.Format]
	_, protoFileOK := a.protoFileFormatToInfo[rawRef.Format]
	if !(singleOK || archiveOK || dirOK || gitOK || moduleOK || protoFileOK) {
		return nil, NewFormatUnknownError(rawRef.Format)
	}
	if len(allowedFormats) > 0 {
		if _, ok := allowedFormats[rawRef.Format]; !ok {
			return nil, NewFormatNotAllowedError(rawRef.Format, allowedFormats)
		}
	}
	if !singleOK && len(rawRef.UnrecognizedOptions) > 0 {
		// Only single refs allow custom options. In every other case, this is an error.
		//
		// We verify unrecognized options match what is expected in getSingleRef.
		keys := make([]string, 0, len(rawRef.UnrecognizedOptions))
		for key := range rawRef.UnrecognizedOptions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil, NewOptionsInvalidKeysError(keys...)
	}
	if singleOK {
		return getSingleRef(rawRef, singleFormatInfo.defaultCompressionType, singleFormatInfo.customOptionKeys)
	}
	if archiveOK {
		return getArchiveRef(rawRef, archiveFormatInfo.archiveType, archiveFormatInfo.defaultCompressionType)
	}
	if protoFileOK {
		return getProtoFileRef(rawRef)
	}
	if dirOK {
		return getDirRef(rawRef)
	}
	if gitOK {
		return getGitRef(rawRef)
	}
	if moduleOK {
		return getModuleRef(rawRef)
	}
	return nil, NewFormatUnknownError(rawRef.Format)
}

func (a *refParser) validateRawRef(
	displayName string,
	rawRef *RawRef,
) error {
	// probably move everything below this point to a new function, perhaps called validateRawRef
	if rawRef.Format == "" {
		return NewFormatCannotBeDeterminedError(displayName)
	}
	_, gitOK := a.gitFormatToInfo[rawRef.Format]
	archiveFormatInfo, archiveOK := a.archiveFormatToInfo[rawRef.Format]
	_, singleOK := a.singleFormatToInfo[rawRef.Format]
	if gitOK {
		if rawRef.GitBranch != "" && rawRef.GitCommitOrTag != "" {
			return NewCannotSpecifyGitBranchAndCommitOrTagError()
		}
		if rawRef.GitRef != "" && rawRef.GitCommitOrTag != "" {
			return NewCannotSpecifyCommitOrTagWithRefError()
		}
	} else {
		if rawRef.GitBranch != "" || rawRef.GitCommitOrTag != "" || rawRef.GitRef != "" || rawRef.GitRecurseSubmodules || rawRef.GitDepth > 0 {
			return NewOptionsInvalidForFormatError(rawRef.Format, displayName, "git options set")
		}
	}
	// not an archive format
	if !archiveOK {
		if rawRef.ArchiveStripComponents > 0 {
			return NewOptionsInvalidForFormatError(rawRef.Format, displayName, "archive options set")
		}
	} else {
		if archiveFormatInfo.archiveType == ArchiveTypeZip && rawRef.CompressionType != 0 {
			return NewCannotSpecifyCompressionForZipError()
		}
	}
	if !singleOK && !archiveOK {
		if rawRef.CompressionType != 0 {
			return NewOptionsInvalidForFormatError(rawRef.Format, displayName, "compression set")
		}
	}
	if !archiveOK && !gitOK {
		if rawRef.SubDirPath != "" {
			return NewOptionsInvalidForFormatError(rawRef.Format, displayName, "subdir set")
		}
	}
	return nil
}

// empty value is an error
func parseCompressionType(value string) (CompressionType, error) {
	switch value {
	case "none":
		return CompressionTypeNone, nil
	case "gzip":
		return CompressionTypeGzip, nil
	case "zstd":
		return CompressionTypeZstd, nil
	default:
		return 0, NewCompressionUnknownError(value)
	}
}

func parseGitDepth(value string) (uint32, error) {
	depth, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, NewDepthParseError(value)
	}
	if depth == 0 {
		return 0, NewDepthZeroError()
	}
	return uint32(depth), nil
}

func parseSubDirPath(value string) (string, error) {
	subDirPath, err := normalpath.NormalizeAndValidate(value)
	if err != nil {
		return "", err
	}
	if subDirPath == "." {
		return "", nil
	}
	return subDirPath, nil
}

// getRawPathAndOptions returns the raw path and options from the value provided,
// the rawPath will be non-empty when returning without error here.
func getRawPathAndOptions(value string) (string, map[string]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, newValueEmptyError()
	}

	switch splitValue := strings.Split(value, "#"); len(splitValue) {
	case 1:
		return value, nil, nil
	case 2:
		path := strings.TrimSpace(splitValue[0])
		optionsString := strings.TrimSpace(splitValue[1])
		if path == "" {
			return "", nil, newValueStartsWithHashtagError(value)
		}
		if optionsString == "" {
			return "", nil, newValueEndsWithHashtagError(value)
		}
		options := make(map[string]string)
		for _, pair := range strings.Split(optionsString, ",") {
			split := strings.Split(pair, "=")
			if len(split) != 2 {
				return "", nil, newOptionsInvalidError(optionsString)
			}
			key := strings.TrimSpace(split[0])
			value := strings.TrimSpace(split[1])
			if key == "" || value == "" {
				return "", nil, newOptionsInvalidError(optionsString)
			}
			if _, ok := options[key]; ok {
				return "", nil, newOptionsDuplicateKeyError(key)
			}
			options[key] = value
		}
		return path, options, nil
	default:
		return "", nil, newValueMultipleHashtagsError(value)
	}
}

func getSingleRef(
	rawRef *RawRef,
	defaultCompressionType CompressionType,
	customOptionKeys map[string]struct{},
) (ParsedSingleRef, error) {
	compressionType := rawRef.CompressionType
	if compressionType == 0 {
		compressionType = defaultCompressionType
	}
	var invalidKeys []string
	for key := range rawRef.UnrecognizedOptions {
		if _, ok := customOptionKeys[key]; !ok {
			invalidKeys = append(invalidKeys, key)
		}
	}
	if len(invalidKeys) > 0 {
		return nil, NewOptionsInvalidKeysError(invalidKeys...)
	}
	return newSingleRef(
		rawRef.Format,
		rawRef.Path,
		compressionType,
		rawRef.UnrecognizedOptions,
	)
}

func getArchiveRef(
	rawRef *RawRef,
	archiveType ArchiveType,
	defaultCompressionType CompressionType,
) (ParsedArchiveRef, error) {
	compressionType := rawRef.CompressionType
	if compressionType == 0 {
		compressionType = defaultCompressionType
	}
	return newArchiveRef(
		rawRef.Format,
		rawRef.Path,
		archiveType,
		compressionType,
		rawRef.ArchiveStripComponents,
		rawRef.SubDirPath,
	)
}

func getDirRef(
	rawRef *RawRef,
) (ParsedDirRef, error) {
	return newDirRef(
		rawRef.Format,
		rawRef.Path,
	)
}

func getGitRef(
	rawRef *RawRef,
) (ParsedGitRef, error) {
	gitRefName, err := getGitRefName(rawRef.Path, rawRef.GitBranch, rawRef.GitCommitOrTag, rawRef.GitRef)
	if err != nil {
		return nil, err
	}
	return newGitRef(
		rawRef.Format,
		rawRef.Path,
		gitRefName,
		rawRef.GitDepth,
		rawRef.GitRecurseSubmodules,
		rawRef.SubDirPath,
	)
}

func getModuleRef(
	rawRef *RawRef,
) (ParsedModuleRef, error) {
	return newModuleRef(
		rawRef.Format,
		rawRef.Path,
	)
}

func getGitRefName(path string, branch string, commitOrTag string, ref string) (git.Name, error) {
	if branch == "" && commitOrTag == "" && ref == "" {
		return nil, nil
	}
	if branch != "" && commitOrTag != "" {
		// already did this in getRawRef but just in case
		return nil, NewCannotSpecifyGitBranchAndCommitOrTagError()
	}
	if ref != "" && commitOrTag != "" {
		// already did this in getRawRef but just in case
		return nil, NewCannotSpecifyCommitOrTagWithRefError()
	}
	if ref != "" && branch != "" {
		return git.NewRefNameWithBranch(ref, branch), nil
	}
	if ref != "" {
		return git.NewRefName(ref), nil
	}
	if branch != "" {
		return git.NewBranchName(branch), nil
	}
	return git.NewTagName(commitOrTag), nil
}

func getProtoFileRef(rawRef *RawRef) (ParsedProtoFileRef, error) {
	return newProtoFileRef(
		rawRef.Format,
		rawRef.Path,
		rawRef.IncludePackageFiles,
	)
}

// options

type singleFormatInfo struct {
	defaultCompressionType CompressionType
	customOptionKeys       map[string]struct{}
}

func newSingleFormatInfo() *singleFormatInfo {
	return &singleFormatInfo{
		defaultCompressionType: CompressionTypeNone,
		customOptionKeys:       make(map[string]struct{}),
	}
}

type archiveFormatInfo struct {
	archiveType            ArchiveType
	defaultCompressionType CompressionType
}

func newArchiveFormatInfo(archiveType ArchiveType) *archiveFormatInfo {
	return &archiveFormatInfo{
		archiveType:            archiveType,
		defaultCompressionType: CompressionTypeNone,
	}
}

type dirFormatInfo struct{}

func newDirFormatInfo() *dirFormatInfo {
	return &dirFormatInfo{}
}

type gitFormatInfo struct{}

func newGitFormatInfo() *gitFormatInfo {
	return &gitFormatInfo{}
}

type moduleFormatInfo struct{}

func newModuleFormatInfo() *moduleFormatInfo {
	return &moduleFormatInfo{}
}

type getParsedRefOptions struct {
	allowedFormats map[string]struct{}
}

type protoFileFormatInfo struct{}

func newProtoFileFormatInfo() *protoFileFormatInfo {
	return &protoFileFormatInfo{}
}

func newGetParsedRefOptions() *getParsedRefOptions {
	return &getParsedRefOptions{
		allowedFormats: make(map[string]struct{}),
	}
}
