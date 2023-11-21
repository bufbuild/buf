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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

const (
	// defaultBufYAMLFileName is the default file name you should use for buf.yaml Files.
	defaultBufYAMLFileName = "buf.yaml"
)

var (
	// otherBufYAMLFileNames are all file names we have ever used for configuration files.
	//
	// Originally we thought we were going to move to buf.mod, and had this around for
	// a while, but then reverted back to buf.yaml. We still need to support buf.mod as
	// we released with it, however.
	otherBufYAMLFileNames = []string{
		"buf.mod",
	}
)

// BufYAMLFile represents a buf.yaml file.
type BufYAMLFile interface {
	File

	// ModuleConfigs returns the ModuleConfigs for the File.
	//
	// For v1 buf.yaml, this will only have a single ModuleConfig.
	ModuleConfigs() []ModuleConfig
	// GenerateConfigs returns the GenerateConfigs for the File.
	//
	// For v1 buf.yaml, this will be empty.
	GenerateConfigs() []GenerateConfig

	isBufYAMLFile()
}

// GetBufYAMLFileForPrefix gets the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, defaultBufYAMLFileName, otherBufYAMLFileNames, readBufYAMLFile)
}

// GetBufYAMLFileForPrefix gets the buf.yaml file version at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, defaultBufYAMLFileName, otherBufYAMLFileNames)
}

// PutBufYAMLFileForPrefix puts the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be written to prefix/buf.yaml.
func PutBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufYAMLFileName, writeBufYAMLFile)
}

// ReadBufYAMLFile reads the BufYAMLFile from the io.Reader.
func ReadBufYAMLFile(reader io.Reader) (BufYAMLFile, error) {
	return readFile(reader, "config file", readBufYAMLFile)
}

// WriteBufYAMLFile writes the BufYAMLFile to the io.Writer.
func WriteBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	return writeFile(writer, "config file", bufYAMLFile, writeBufYAMLFile)
}

// *** PRIVATE ***

type bufYAMLFile struct {
	fileVersion     FileVersion
	moduleConfigs   []ModuleConfig
	generateConfigs []GenerateConfig
}

func newBufYAMLFile(
	fileVersion FileVersion,
	moduleConfigs []ModuleConfig,
	generateConfigs []GenerateConfig,
) (*bufYAMLFile, error) {
	// TODO: validation
	return &bufYAMLFile{
		fileVersion:     fileVersion,
		moduleConfigs:   moduleConfigs,
		generateConfigs: generateConfigs,
	}, nil
}

func (c *bufYAMLFile) FileVersion() FileVersion {
	return c.fileVersion
}

func (c *bufYAMLFile) ModuleConfigs() []ModuleConfig {
	return slicesextended.Copy(c.moduleConfigs)
}

func (c *bufYAMLFile) GenerateConfigs() []GenerateConfig {
	return slicesextended.Copy(c.generateConfigs)
}

func (*bufYAMLFile) isBufYAMLFile() {}
func (*bufYAMLFile) isFile()        {}

// TODO: port tests from bufmoduleconfig, buflintconfig, bufbreakingconfig
func readBufYAMLFile(reader io.Reader, allowJSON bool) (BufYAMLFile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	fileVersion, err := getFileVersionForData(data, allowJSON)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1, FileVersionV1:
		var externalBufYAMLFile externalBufYAMLFileV1OrV1Beta1
		if err := getUnmarshalStrict(allowJSON)(data, &externalBufYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		if fileVersion == FileVersionV1 && len(externalBufYAMLFile.Build.Roots) > 0 {
			return nil, fmt.Errorf("build.roots cannot be set on version %v: %v", fileVersion, externalBufYAMLFile.Build.Roots)
		}
		rootToExcludes, err := getRootToExcludes(externalBufYAMLFile.Build.Roots, externalBufYAMLFile.Build.Excludes)
		if err != nil {
			return nil, err
		}
		moduleFullName, err := bufmodule.ParseModuleFullName(externalBufYAMLFile.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid module name: %w", err)
		}
		return newBufYAMLFile(
			fileVersion,
			[]ModuleConfig{
				newModuleConfig(
					".",
					moduleFullName,
					rootToExcludes,
					newLintConfig(
						newCheckConfig(
							fileVersion,
							externalBufYAMLFile.Lint.Use,
							externalBufYAMLFile.Lint.Except,
							externalBufYAMLFile.Lint.Ignore,
							externalBufYAMLFile.Lint.IgnoreOnly,
						),
						externalBufYAMLFile.Lint.EnumZeroValueSuffix,
						externalBufYAMLFile.Lint.RPCAllowSameRequestResponse,
						externalBufYAMLFile.Lint.RPCAllowGoogleProtobufEmptyRequests,
						externalBufYAMLFile.Lint.RPCAllowGoogleProtobufEmptyResponses,
						externalBufYAMLFile.Lint.ServiceSuffix,
						externalBufYAMLFile.Lint.AllowCommentIgnores,
					),
					newBreakingConfig(
						newCheckConfig(
							fileVersion,
							externalBufYAMLFile.Breaking.Use,
							externalBufYAMLFile.Breaking.Except,
							externalBufYAMLFile.Breaking.Ignore,
							externalBufYAMLFile.Breaking.IgnoreOnly,
						),
						externalBufYAMLFile.Breaking.IgnoreUnstablePackages,
					),
				),
			},
			nil,
		)
	case FileVersionV2:
		return nil, newUnsupportedFileVersionError(fileVersion)
	default:
		// This is a system error since we've already parsed.
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1:
		return errors.New("TODO")
	case FileVersionV1:
		return errors.New("TODO")
	case FileVersionV2:
		return newUnsupportedFileVersionError(fileVersion)
	default:
		// This is a system error since we've already parsed.
		return fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func getRootToExcludes(roots []string, fullExcludes []string) (map[string][]string, error) {
	if len(roots) == 0 {
		roots = []string{"."}
	}

	rootToExcludes := make(map[string][]string)
	roots, err := normalizeAndCheckPaths(roots, "root")
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		// we already checked duplicates, but just in case
		if _, ok := rootToExcludes[root]; ok {
			return nil, fmt.Errorf("unexpected duplicate root: %q", root)
		}
		rootToExcludes[root] = []string{}
	}
	if len(fullExcludes) == 0 {
		return rootToExcludes, nil
	}

	// This also verifies that fullExcludes is unique.
	fullExcludes, err = normalizeAndCheckPaths(fullExcludes, "exclude")
	if err != nil {
		return nil, err
	}
	// Verify that no exclude equals a root directly and only directories are specified.
	for _, fullExclude := range fullExcludes {
		if normalpath.Ext(fullExclude) == ".proto" {
			return nil, fmt.Errorf("excludes can only be directories but file %s discovered", fullExclude)
		}
		if _, ok := rootToExcludes[fullExclude]; ok {
			return nil, fmt.Errorf("%s is both a root and exclude, which means the entire root is excluded, which is not valid", fullExclude)
		}
	}

	// Verify that all excludes are within a root.
	rootMap := slicesextended.ToMap(roots)
	for _, fullExclude := range fullExcludes {
		switch matchingRoots := normalpath.MapAllEqualOrContainingPaths(rootMap, fullExclude, normalpath.Relative); len(matchingRoots) {
		case 0:
			return nil, fmt.Errorf("exclude %s is not contained in any root, which is not valid", fullExclude)
		case 1:
			root := matchingRoots[0]
			exclude, err := normalpath.Rel(root, fullExclude)
			if err != nil {
				return nil, err
			}
			// Just in case.
			exclude, err = normalpath.NormalizeAndValidate(exclude)
			if err != nil {
				return nil, err
			}
			rootToExcludes[root] = append(rootToExcludes[root], exclude)
		default:
			// This should never happen, but just in case.
			return nil, fmt.Errorf("exclude %q was in multiple roots %v", fullExclude, matchingRoots)
		}
	}

	for root, excludes := range rootToExcludes {
		uniqueSortedExcludes := stringutil.SliceToUniqueSortedSliceFilterEmptyStrings(excludes)
		if len(excludes) != len(uniqueSortedExcludes) {
			// This should never happen, but just in case.
			return nil, fmt.Errorf("excludes %v are not unique", excludes)
		}
		rootToExcludes[root] = uniqueSortedExcludes
	}
	return rootToExcludes, nil
}

// externalBufYAMLFileV1Beta1 represents the v1 or v1beta1 buf.yaml file, which have
// the same shape EXCEPT build.roots.
//
// Note that the lint and breaking ids/categories DID change between v1beta1 and v1, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileV1OrV1Beta1 struct {
	Version  string                                 `json:"version,omitempty" yaml:"version,omitempty"`
	Name     string                                 `json:"name,omitempty" yaml:"name,omitempty"`
	Deps     []string                               `json:"deps,omitempty" yaml:"deps,omitempty"`
	Build    externalBufYAMLFileBuildV1OrV1Beta1    `json:"build,omitempty" yaml:"build,omitempty"`
	Breaking externalBufYAMLFileBreakingV1OrV1Beta1 `json:"breaking,omitempty" yaml:"breaking,omitempty"`
	Lint     externalBufYAMLFileLintV1OrV1Beta1     `json:"lint,omitempty" yaml:"lint,omitempty"`
}

// externalBufYAMLFileBuildV1OrV1Beta1 represents build configuation within a v1 or
// v1beta1 buf.yaml file, which have the same shape except for roots.
type externalBufYAMLFileBuildV1OrV1Beta1 struct {
	// Roots are only valid in v1beta! Validate that this is not set for v1.
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}

// externalBufYAMLFileBreakingV1OrV1Beta1 represents breaking configuation within a v1 or
// v1beta1 buf.yaml file, which have the same shape.
//
// Note that the lint and breaking ids/categories DID change between v1beta1 and v1, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileBreakingV1OrV1Beta1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	/// IgnoreOnly are the ID/category to paths to ignore.
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

// externalBufYAMLFileLintV1OrV1Beta1 represents lint configuation within a v1 or
// v1beta1 buf.yaml file, which have the same shape.
//
// Note that the lint and breaking ids/categories DID change between v1beta1 and v1, make
// sure to deal with this when parsing what to set as defaults, or how to interpret categories.
type externalBufYAMLFileLintV1OrV1Beta1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// Ignore are the paths to ignore.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	/// IgnoreOnly are the ID/category to paths to ignore.
	IgnoreOnly                           map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	EnumZeroValueSuffix                  string              `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool                `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool                `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool                `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string              `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
	AllowCommentIgnores                  bool                `json:"allow_comment_ignores,omitempty" yaml:"allow_comment_ignores,omitempty"`
}
