// Copyright 2020 Buf Technologies Inc.
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
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"go.uber.org/zap"
)

var (
	fileSchemePrefixToFileScheme = map[string]FileScheme{
		"http://":  FileSchemeHTTP,
		"https://": FileSchemeHTTPS,
		"file://":  FileSchemeLocal,
	}
	gitSchemePrefixToGitScheme = map[string]GitScheme{
		"http://":  GitSchemeHTTP,
		"https://": GitSchemeHTTPS,
		"file://":  GitSchemeLocal,
		"ssh://":   GitSchemeSSH,
	}
)

type refParser struct {
	logger              *zap.Logger
	formatParser        func(string) (string, error)
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

func (a *refParser) GetRef(
	ctx context.Context,
	value string,
	options ...GetRefOption,
) (Ref, error) {
	getRefOptions := newGetRefOptions()
	for _, option := range options {
		option(getRefOptions)
	}
	return a.getRef(ctx, value, getRefOptions.allowedFormats)
}

func (a *refParser) getRef(
	ctx context.Context,
	value string,
	allowedFormats map[string]struct{},
) (Ref, error) {
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
		return a.getSingleRef(ctx, rawRef, singleFormatInfo.defaultCompressionType)
	}
	if archiveOK {
		return a.getArchiveRef(ctx, rawRef, archiveFormatInfo.archiveType, archiveFormatInfo.defaultCompressionType)
	}
	if dirOK {
		return a.getDirRef(ctx, rawRef)

	}
	if gitOK {
		return a.getGitRef(ctx, rawRef)
	}
	return nil, newFormatUnknownError(rawRef.Format)
}

func (a *refParser) getSingleRef(
	ctx context.Context,
	rawRef *rawRef,
	defaultCompressionType CompressionType,
) (SingleRef, error) {
	if rawRef.Path == "-" {
		return newSingleRef(
			rawRef.Format,
			"",
			FileSchemeStdio,
			defaultCompressionType,
		), nil
	}
	if rawRef.Path == app.DevNullFilePath {
		return newSingleRef(
			rawRef.Format,
			"",
			FileSchemeNull,
			defaultCompressionType,
		), nil
	}
	for prefix, fileScheme := range fileSchemePrefixToFileScheme {
		if strings.HasPrefix(rawRef.Path, prefix) {
			path := strings.TrimPrefix(rawRef.Path, prefix)
			if fileScheme == FileSchemeLocal {
				path = normalpath.Normalize(path)
			}
			if path == "" {
				return nil, newNoPathError(rawRef.Value)
			}
			return newSingleRef(
				rawRef.Format,
				path,
				fileScheme,
				defaultCompressionType,
			), nil
		}
	}
	path := normalpath.Normalize(rawRef.Path)
	if path == "" {
		return nil, newNoPathError(rawRef.Value)
	}
	return newSingleRef(
		rawRef.Format,
		path,
		FileSchemeLocal,
		defaultCompressionType,
	), nil
}

func (a *refParser) getArchiveRef(
	ctx context.Context,
	rawRef *rawRef,
	archiveType ArchiveType,
	defaultCompressionType CompressionType,
) (ArchiveRef, error) {
	singleRef, err := a.getSingleRef(ctx, rawRef, defaultCompressionType)
	if err != nil {
		return nil, err
	}
	return newArchiveRef(
		singleRef.Format(),
		singleRef.Path(),
		singleRef.FileScheme(),
		archiveType,
		singleRef.CompressionType(),
		rawRef.ArchiveStripComponents,
	), nil
}

func (a *refParser) getDirRef(
	ctx context.Context,
	rawRef *rawRef,
) (DirRef, error) {
	if rawRef.Path == "-" {
		return nil, newInvalidDirPathError(rawRef.Path)
	}
	if rawRef.Path == app.DevNullFilePath {
		return nil, newInvalidDirPathError(rawRef.Path)
	}
	path := normalpath.Normalize(rawRef.Path)
	if path == "" {
		return nil, newNoPathError(rawRef.Value)
	}
	return newDirRef(
		rawRef.Format,
		path,
	), nil
}

func (a *refParser) getGitRef(
	ctx context.Context,
	rawRef *rawRef,
) (GitRef, error) {
	if rawRef.Path == "-" {
		return nil, newInvalidGitPathError(rawRef.Path)
	}
	if rawRef.Path == app.DevNullFilePath {
		return nil, newInvalidGitPathError(rawRef.Path)
	}
	if rawRef.GitBranch == "" && rawRef.GitTag == "" {
		// already did this in getRawRef but just in case
		return nil, newMustSpecifyGitRepositoryRefNameError(rawRef.Path)
	}
	if rawRef.GitBranch != "" && rawRef.GitTag != "" {
		// already did this in getRawRef but just in case
		return nil, newCannotSpecifyMultipleGitRepositoryRefNamesError()
	}
	var gitRefName git.RefName
	if rawRef.GitBranch != "" {
		gitRefName = git.NewBranchRefName(rawRef.GitBranch)
	} else {
		gitRefName = git.NewTagRefName(rawRef.GitTag)
	}
	gitScheme, path, err := getGitSchemeAndPath(rawRef)
	if err != nil {
		return nil, err
	}
	return newGitRef(
		rawRef.Format,
		path,
		gitScheme,
		gitRefName,
		rawRef.GitRecurseSubmodules,
	), nil
}

func getGitSchemeAndPath(rawRef *rawRef) (GitScheme, string, error) {
	if rawRef.Path == "-" {
		return 0, "", newInvalidGitPathError(rawRef.Path)
	}
	if rawRef.Path == app.DevNullFilePath {
		return 0, "", newInvalidGitPathError(rawRef.Path)
	}
	for prefix, gitScheme := range gitSchemePrefixToGitScheme {
		if strings.HasPrefix(rawRef.Path, prefix) {
			path := strings.TrimPrefix(rawRef.Path, prefix)
			if gitScheme == GitSchemeLocal {
				path = normalpath.Normalize(path)
			}
			if path == "" {
				return 0, "", newNoPathError(rawRef.Value)
			}
			return gitScheme, path, nil
		}
	}
	path := normalpath.Normalize(rawRef.Path)
	if path == "" {
		return 0, "", newNoPathError(rawRef.Value)
	}
	return GitSchemeLocal, path, nil
}

// create with getRawRef
// a validated rawRef per the below rules is returned
type rawRef struct {
	// The original value
	// Will always be set
	Value string
	// Will always be set
	// Not normalized yet
	Path string
	// Will always be set
	// Set via formatParser if not explicitly set
	Format string
	// Only set for git formats
	// Only one of GitBranch and GitTag will be set
	GitBranch string
	// Only set for git formats
	// Only one of GitBranch and GitTag will be set
	GitTag string
	// Only set for git formats
	GitRecurseSubmodules bool
	// Only set for archive formats
	ArchiveStripComponents uint32
}

// validated per rules on rawRef
func (a *refParser) getRawRef(value string) (*rawRef, error) {
	// path is never empty after returning from this function
	path, options, err := getRawPathAndOptions(value)
	if err != nil {
		return nil, err
	}
	rawRef := &rawRef{
		Value: value,
		Path:  path,
	}
	if a.formatParser != nil {
		impliedFormat, err := a.formatParser(path)
		if err != nil {
			return nil, err
		}
		rawRef.Format = impliedFormat
	}
	for key, value := range options {
		switch key {
		case "format":
			if path == app.DevNullFilePath {
				return nil, newFormatOverrideNotAllowedForDevNullError(app.DevNullFilePath)
			}
			rawRef.Format = value
		case "branch":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyMultipleGitRepositoryRefNamesError()
			}
			rawRef.GitBranch = value
		case "tag":
			if rawRef.GitBranch != "" || rawRef.GitTag != "" {
				return nil, newCannotSpecifyMultipleGitRepositoryRefNamesError()
			}
			rawRef.GitTag = value
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
	// not a git format
	if _, ok := a.gitFormatToInfo[rawRef.Format]; !ok {
		if rawRef.GitBranch != "" || rawRef.GitTag != "" || rawRef.GitRecurseSubmodules {
			return nil, newOptionsInvalidForFormatError(rawRef.Format, value)
		}
	} else {
		if rawRef.GitBranch == "" && rawRef.GitTag == "" {
			return nil, newMustSpecifyGitRepositoryRefNameError(rawRef.Path)
		}
	}
	// not an archive format
	if _, ok := a.archiveFormatToInfo[rawRef.Format]; !ok {
		if rawRef.ArchiveStripComponents > 0 {
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

type getRefOptions struct {
	allowedFormats map[string]struct{}
}

func newGetRefOptions() *getRefOptions {
	return &getRefOptions{
		allowedFormats: make(map[string]struct{}),
	}
}
