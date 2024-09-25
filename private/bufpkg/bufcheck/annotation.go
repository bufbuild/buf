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

package bufcheck

import (
	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

type annotation struct {
	check.Annotation

	pluginName string
}

func newAnnotation(checkAnnotation check.Annotation, pluginName string) *annotation {
	return &annotation{
		Annotation: checkAnnotation,
		pluginName: pluginName,
	}
}

func (a *annotation) PluginName() string {
	return a.pluginName
}

func annotationsToFileAnnotations(
	pathToExternalPath map[string]string,
	annotations []*annotation,
) []bufanalysis.FileAnnotation {
	return slicesext.Map(
		annotations,
		func(annotation *annotation) bufanalysis.FileAnnotation {
			return annotationToFileAnnotation(pathToExternalPath, annotation)
		},
	)
}

func annotationToFileAnnotation(
	pathToExternalPath map[string]string,
	annotation *annotation,
) bufanalysis.FileAnnotation {
	fileLocation := annotation.FileLocation()
	if fileLocation == nil {
		// We have to do this or we get a weird fileInfo != nil but it is nil thing.
		return bufanalysis.NewFileAnnotation(
			nil,
			0,
			0,
			0,
			0,
			annotation.RuleID(),
			annotation.Message(),
			annotation.PluginName(),
		)
	}
	path := fileLocation.FileDescriptor().ProtoreflectFileDescriptor().Path()
	// While it never should, it is OK if pathToExternalPath returns "" for a given path.
	// We handle this in fileInfo.
	fileInfo := newFileInfo(path, pathToExternalPath[path])
	startLine := fileLocation.StartLine() + 1
	startColumn := fileLocation.StartColumn() + 1
	endLine := fileLocation.EndLine() + 1
	endColumn := fileLocation.EndColumn() + 1
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		annotation.RuleID(),
		annotation.Message(),
		annotation.PluginName(),
	)
}
