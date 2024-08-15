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

package bufcheckclient

import (
	checkv1beta1 "buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go/buf/plugin/check/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/bufplugin-go/check"
)

func imageToProtoFiles(image bufimage.Image) []*checkv1beta1.File {
	if image == nil {
		return nil
	}
	return slicesext.Map(image.Files(), imageFileToProtoFile)
}

func imageFileToProtoFile(imageFile bufimage.ImageFile) *checkv1beta1.File {
	return &checkv1beta1.File{
		FileDescriptorProto: imageFile.FileDescriptorProto(),
		IsImport:            imageFile.IsImport(),
	}
}

// imageToPathToExternalPath returns a map from path to external path for all ImageFiles in the Image.
//
// We do not transmit external path information over the wire to plugins, so we need to keep track
// of this on the client side to properly construct bufanalysis.FileAnnotations when we get back
// check.Annotations. This is used in annotationToFileAnnotation.
func imageToPathToExternalPath(image bufimage.Image) map[string]string {
	imageFiles := image.Files()
	pathToExternalPath := make(map[string]string, len(imageFiles))
	for _, imageFile := range imageFiles {
		// We know that Images do not have overlapping paths.
		pathToExternalPath[imageFile.Path()] = imageFile.ExternalPath()
	}
	return pathToExternalPath
}

func annotationsToFileAnnotations(
	pathToExternalPath map[string]string,
	annotations []check.Annotation,
) []bufanalysis.FileAnnotation {
	return slicesext.Map(
		annotations,
		func(annotation check.Annotation) bufanalysis.FileAnnotation {
			return annotationToFileAnnotation(pathToExternalPath, annotation)
		},
	)
}

func annotationToFileAnnotation(
	pathToExternalPath map[string]string,
	annotation check.Annotation,
) bufanalysis.FileAnnotation {
	if annotation == nil {
		return nil
	}
	var fileInfo *fileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	if location := annotation.Location(); location != nil {
		path := location.File().FileDescriptor().Path()
		// While it never should, it is OK if pathToExternalPath returns "" for a given path.
		// We handle this in fileInfo.
		fileInfo = newFileInfo(path, pathToExternalPath[path])
		startLine = location.StartLine() + 1
		startColumn = location.StartColumn() + 1
		endLine = location.EndLine() + 1
		endColumn = location.EndColumn() + 1
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		annotation.RuleID(),
		annotation.Message(),
	)
}

// Returns Rules in same order as in allRules.
func rulesForType(allRules []check.Rule, ruleType check.RuleType) []check.Rule {
	return slicesext.Filter(allRules, func(rule check.Rule) bool { return rule.Type() == ruleType })
}

// Returns Rules in same order as in allRules.
func rulesForRuleIDs(allRules []check.Rule, ruleIDs []string) []check.Rule {
	rules := make([]check.Rule, 0, len(allRules))
	ruleIDMap := slicesext.ToStructMap(ruleIDs)
	for _, rule := range allRules {
		if _, ok := ruleIDMap[rule.ID()]; ok {
			rules = append(rules, rule)
		}
	}
	return rules
}
