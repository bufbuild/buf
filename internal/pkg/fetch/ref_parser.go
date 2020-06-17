// Copyright 2020 Buf Technologies, Inc.
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

package fetch

import (
	"context"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/git"
	"go.uber.org/zap"
)

var (
	knownCompressionTypeStrings = []string{
		"none",
		"gzip",
		"zstd",
	}
)

type refParser struct {
	logger              *zap.Logger
	rawRefProcessor     func(*RawRef) error
	singleFormatToInfo  map[string]*singleFormatInfo
	archiveFormatToInfo map[string]*archiveFormatInfo
	dirFormatToInfo     map[string]*dirFormatInfo
	gitFormatToInfo     map[string]*gitFormatInfo
}

func newRefParser(logger *zap.Logger, options ...RefParserOption) *refParser {
	refParser := &refParser{
		logger:              logger,
		singleFormatToInfo:  make(map[string]*singleFormatInfo),
		archiveFormatToInfo: make(map[string]*archiveFormatInfo),
		dirFormatToInfo:     make(map[string]*dirFormatInfo),
		gitFormatToInfo:     make(map[string]*gitFormatInfo),
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

func (a *refParser) getParsedRef(
	ctx context.Context,
	value string,
	allowedFormats map[string]struct{},
) (ParsedRef, error) {
	rawRef, err := a.getRawRef(value)
	if err != nil {
		return nil, err
	}
	singleFormatInfo, singleOK := a.singleFormatToInfo[rawRef.Format]
	archiveFormatInfo, archiveOK := a.archiveFormatToInfo[rawRef.Format]
	_, dirOK := a.dirFormatToInfo[rawRef.Format]
	_, gitOK := a.gitFormatToInfo[rawRef.Format]
	if !(singleOK || archiveOK || dirOK || gitOK) {
		return nil, newFormatUnknownError(rawRef.Format)
	}
	if len(allowedFormats) > 0 {
		if _, ok := allowedFormats[rawRef.Format]; !ok {
			return nil, newFormatNotAllowedError(rawRef.Format, allowedFormats)
		}
	}
	if singleOK {
		return getSingleRef(rawRef, singleFormatInfo.defaultCompressionType)
	}
	if archiveOK {
		return getArchiveRef(rawRef, archiveFormatInfo.archiveType, archiveFormatInfo.defaultCompressionType)
	}
	if dirOK {
		return getDirRef(rawRef)

	}
	if gitOK {
		return getGitRef(rawRef)
	}
	return nil, newFormatUnknownError(rawRef.Format)
}

// validated per rules on rawRef
func (a *refParser) getRawRef(value string) (*RawRef, error) {
	// path is never empty after returning from this function
	path, options, err := getRawPathAndOptions(value)
	if err != nil {
		return nil, err
	}
	rawRef := &RawRef{
		Path: path,
	}
	if a.rawRefProcessor != nil {
		if err := a.rawRefProcessor(rawRef); err != nil {
			return nil, err
		}
	}
	for key, value := range options {
		switch key {
		case "format":
			if path == app.DevNullFilePath {
				return nil, newFormatOverrideNotAllowedForDevNullError(app.DevNullFilePath)
			}
			rawRef.Format = value
		case "compression":
			switch value {
			case "none":
				rawRef.CompressionType = CompressionTypeNone
			case "gzip":
				rawRef.CompressionType = CompressionTypeGzip
			case "zstd":
				rawRef.CompressionType = CompressionTypeZstd
			default:
				return nil, newCompressionUnknownError(value, knownCompressionTypeStrings...)
			}
		case "branch":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyGitBranchAndTagError()
			}
			rawRef.GitBranch = value
		case "tag":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyGitBranchAndTagError()
			}
			rawRef.GitTag = value
		case "ref":
			rawRef.GitRef = value
		case "depth":
			depth, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, newDepthParseError(value)
			}
			if depth == 0 {
				return nil, newDepthZeroError()
			}
			rawRef.GitDepth = uint32(depth)
		case "recurse_submodules":
			// TODO: need to refactor to make sure this is not set for any non-git input
			// ie right now recurse_submodules=false will not error
			switch value {
			case "true":
				rawRef.GitRecurseSubmodules = true
			case "false":
			default:
				return nil, newOptionsCouldNotParseRecurseSubmodulesError(value)
			}
		case "strip_components":
			// TODO: need to refactor to make sure this is not set for any non-tarball
			// ie right now strip_components=0 will not error
			stripComponents, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, newOptionsCouldNotParseStripComponentsError(value)
			}
			rawRef.ArchiveStripComponents = uint32(stripComponents)
		default:
			return nil, newOptionsInvalidKeyError(key)
		}
	}

	if rawRef.Format == "" {
		return nil, newFormatCannotBeDeterminedError(value)
	}

	_, gitOK := a.gitFormatToInfo[rawRef.Format]
	archiveFormatInfo, archiveOK := a.archiveFormatToInfo[rawRef.Format]
	_, singleOK := a.singleFormatToInfo[rawRef.Format]
	if gitOK {
		if rawRef.GitRef != "" && rawRef.GitTag != "" {
			return nil, newCannotSpecifyTagWithRefError()
		}
		if rawRef.GitDepth == 0 {
			// Default to 1
			rawRef.GitDepth = 1
			if rawRef.GitRef != "" {
				// Default to 50 when using ref
				rawRef.GitDepth = 50
			}
		}
	} else {
		if rawRef.GitBranch != "" || rawRef.GitTag != "" || rawRef.GitRef != "" || rawRef.GitRecurseSubmodules || rawRef.GitDepth > 0 {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	}
	// not an archive format
	if !archiveOK {
		if rawRef.ArchiveStripComponents > 0 {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	} else {
		if archiveFormatInfo.archiveType == ArchiveTypeZip && rawRef.CompressionType != 0 {
			return nil, newCannotSpecifyCompressionForZipError()
		}
	}
	if !singleOK && !archiveOK {
		if rawRef.CompressionType != 0 {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	}
	return rawRef, nil
}

// rawPath will be non-empty
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
) (ParsedSingleRef, error) {
	compressionType := rawRef.CompressionType
	if compressionType == 0 {
		compressionType = defaultCompressionType
	}
	return newSingleRef(
		rawRef.Format,
		rawRef.Path,
		compressionType,
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
	gitRefName, err := getGitRefName(rawRef.Path, rawRef.GitBranch, rawRef.GitTag, rawRef.GitRef)
	if err != nil {
		return nil, err
	}
	return newGitRef(
		rawRef.Format,
		rawRef.Path,
		gitRefName,
		rawRef.GitDepth,
		rawRef.GitRecurseSubmodules,
	)
}

func getGitRefName(path string, branch string, tag string, ref string) (git.Name, error) {
	if branch == "" && tag == "" && ref == "" {
		return nil, nil
	}
	if branch != "" && tag != "" {
		// already did this in getRawRef but just in case
		return nil, newCannotSpecifyGitBranchAndTagError()
	}
	if ref != "" && tag != "" {
		// already did this in getRawRef but just in case
		return nil, newCannotSpecifyTagWithRefError()
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
	return git.NewTagName(tag), nil
}

// options

type singleFormatInfo struct {
	defaultCompressionType CompressionType
}

func newSingleFormatInfo() *singleFormatInfo {
	return &singleFormatInfo{
		defaultCompressionType: CompressionTypeNone,
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

type getParsedRefOptions struct {
	allowedFormats map[string]struct{}
}

func newGetParsedRefOptions() *getParsedRefOptions {
	return &getParsedRefOptions{
		allowedFormats: make(map[string]struct{}),
	}
}
