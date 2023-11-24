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
	"bytes"
	"io"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
)

var (
	DefaultLintConfig LintConfig = defaultLintConfigV1

	defaultLintConfigV1Beta1 = NewLintConfig(
		defaultCheckConfigV1Beta1,
		"",
		false,
		false,
		false,
		"",
		false,
	)
	defaultLintConfigV1 = NewLintConfig(
		defaultCheckConfigV1,
		"",
		false,
		false,
		false,
		"",
		false,
	)
	defaultLintConfigV2 = NewLintConfig(
		defaultCheckConfigV2,
		"",
		false,
		false,
		false,
		"",
		false,
	)
)

// LintConfig is lint configuration for a specific Module.
type LintConfig interface {
	CheckConfig

	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool

	isLintConfig()
}

// NewLintConfig returns a new LintConfig.
func NewLintConfig(
	checkConfig CheckConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
) LintConfig {
	return newLintConfig(
		checkConfig,
		enumZeroValueSuffix,
		rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix,
		allowCommentIgnores,
	)
}

// PrintFileAnnotationsConfigIgnoreYAMLV1 prints the FileAnnotations to the Writer
// for the config-ignore-yaml format.
//
// TODO: This is messed.
func PrintFileAnnotationsConfigIgnoreYAMLV1(
	writer io.Writer,
	fileAnnotations []bufanalysis.FileAnnotation,
) error {
	if len(fileAnnotations) == 0 {
		return nil
	}
	ignoreIDToPathMap := make(map[string]map[string]struct{})
	for _, fileAnnotation := range fileAnnotations {
		fileInfo := fileAnnotation.FileInfo()
		if fileInfo == nil || fileAnnotation.Type() == "" {
			continue
		}
		pathMap, ok := ignoreIDToPathMap[fileAnnotation.Type()]
		if !ok {
			pathMap = make(map[string]struct{})
			ignoreIDToPathMap[fileAnnotation.Type()] = pathMap
		}
		pathMap[fileInfo.Path()] = struct{}{}
	}
	if len(ignoreIDToPathMap) == 0 {
		return nil
	}

	sortedIgnoreIDs := make([]string, 0, len(ignoreIDToPathMap))
	ignoreIDToSortedPaths := make(map[string][]string, len(ignoreIDToPathMap))
	for id, pathMap := range ignoreIDToPathMap {
		sortedIgnoreIDs = append(sortedIgnoreIDs, id)
		paths := make([]string, 0, len(pathMap))
		for path := range pathMap {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		ignoreIDToSortedPaths[id] = paths
	}
	sort.Strings(sortedIgnoreIDs)

	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`version: v1
lint:
  ignore_only:
`)
	for _, id := range sortedIgnoreIDs {
		_, _ = buffer.WriteString("    ")
		_, _ = buffer.WriteString(id)
		_, _ = buffer.WriteString(":\n")
		for _, rootPath := range ignoreIDToSortedPaths[id] {
			_, _ = buffer.WriteString("      - ")
			_, _ = buffer.WriteString(rootPath)
			_, _ = buffer.WriteString("\n")
		}
	}
	_, err := writer.Write(buffer.Bytes())
	return err
}

// *** PRIVATE ***

type lintConfig struct {
	CheckConfig

	enumZeroValueSuffix                  string
	rpcAllowSameRequestResponse          bool
	rpcAllowGoogleProtobuEmptyRequests   bool
	rpcAllowGoogleProtobufEmptyResponses bool
	serviceSuffix                        string
	allowCommentIgnores                  bool
}

func newLintConfig(
	checkConfig CheckConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobuEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
) *lintConfig {
	return &lintConfig{
		CheckConfig:                          checkConfig,
		enumZeroValueSuffix:                  enumZeroValueSuffix,
		rpcAllowSameRequestResponse:          rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobuEmptyRequests:   rpcAllowGoogleProtobuEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses: rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix:                        serviceSuffix,
		allowCommentIgnores:                  allowCommentIgnores,
	}
}

func (l *lintConfig) EnumZeroValueSuffix() string {
	return l.enumZeroValueSuffix
}

func (l *lintConfig) RPCAllowSameRequestResponse() bool {
	return l.rpcAllowSameRequestResponse
}

func (l *lintConfig) RPCAllowGoogleProtobufEmptyRequests() bool {
	return l.rpcAllowGoogleProtobuEmptyRequests
}

func (l *lintConfig) RPCAllowGoogleProtobufEmptyResponses() bool {
	return l.rpcAllowGoogleProtobufEmptyResponses
}

func (l *lintConfig) ServiceSuffix() string {
	return l.serviceSuffix
}

func (l *lintConfig) AllowCommentIgnores() bool {
	return l.allowCommentIgnores
}

func (*lintConfig) isLintConfig() {}
