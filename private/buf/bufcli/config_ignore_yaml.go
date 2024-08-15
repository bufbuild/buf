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

package bufcli

import (
	"bytes"
	"io"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
)

// AllLintFormatStrings are all format strings for lint.
var AllLintFormatStrings = append(
	bufanalysis.AllFormatStrings,
	"config-ignore-yaml",
)

// PrintFileAnnotationSetLintConfigIgnoreYAMLV1 prints the FileAnnotationSet to the Writer
// for the lint config-ignore-yaml format.
//
// TODO FUTURE: This is messed.
func PrintFileAnnotationSetLintConfigIgnoreYAMLV1(
	writer io.Writer,
	fileAnnotationSet bufanalysis.FileAnnotationSet,
) error {
	ignoreIDToPathMap := make(map[string]map[string]struct{})
	for _, fileAnnotation := range fileAnnotationSet.FileAnnotations() {
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
