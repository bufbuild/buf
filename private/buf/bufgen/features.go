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

package bufgen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Map of all known features to functions that can check whether a given file
// uses said feature.
var featureToFeatureChecker = map[pluginpb.CodeGeneratorResponse_Feature]featureChecker{
	pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL:   fileHasProto3Optional,
	pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS: fileHasEditions,
}

// requiredFeatures maps a feature to the set of files in an image that
// make use of that feature.
type requiredFeatures struct {
	featureToFilenames map[pluginpb.CodeGeneratorResponse_Feature][]string
	editionToFilenames map[descriptorpb.Edition][]string
	minEdition         descriptorpb.Edition
	maxEdition         descriptorpb.Edition
}

func newRequiredFeatures() *requiredFeatures {
	return &requiredFeatures{
		featureToFilenames: map[pluginpb.CodeGeneratorResponse_Feature][]string{},
		editionToFilenames: map[descriptorpb.Edition][]string{},
	}
}

type featureChecker func(options *descriptorpb.FileDescriptorProto) bool

// computeRequiredFeatures returns a map of required features to the files in
// the image that require that feature. After plugins are invoked, the plugins'
// responses are checked to make sure any required features were supported.
func computeRequiredFeatures(image bufimage.Image) *requiredFeatures {
	features := newRequiredFeatures()
	for _, imageFile := range image.Files() {
		if imageFile.IsImport() {
			// we only want to check the sources in the module, not their dependencies
			continue
		}
		// Collect all required feature enum values.
		for feature, checker := range featureToFeatureChecker {
			if checker(imageFile.FileDescriptorProto()) {
				features.featureToFilenames[feature] = append(features.featureToFilenames[feature], imageFile.Path())
			}
		}
		// We also collect the range of required editions.
		if !fileHasEditions(imageFile.FileDescriptorProto()) {
			continue
		}
		edition := imageFile.FileDescriptorProto().GetEdition()
		features.editionToFilenames[edition] = append(features.editionToFilenames[edition], imageFile.Path())
		if features.minEdition == 0 || edition < features.minEdition {
			features.minEdition = edition
		}
		if edition > features.maxEdition {
			features.maxEdition = edition
		}
	}
	return features
}

func checkRequiredFeatures(
	logger *zap.Logger,
	required *requiredFeatures,
	responses []*pluginpb.CodeGeneratorResponse,
	configs []bufconfig.GeneratePluginConfig,
) error {
	var errs []error
	for responseIndex, response := range responses {
		if response == nil || response.GetError() != "" {
			// plugin failed, nothing to check
			continue
		}

		failed := newRequiredFeatures()
		var failedFeatures []pluginpb.CodeGeneratorResponse_Feature
		var failedEditions []descriptorpb.Edition
		supported := response.GetSupportedFeatures() // bit mask of features the plugin supports
		for feature, files := range required.featureToFilenames {
			featureMask := uint64(feature)
			if supported&featureMask != featureMask {
				// doh! Supported features don't include this one
				failed.featureToFilenames[feature] = files
				failedFeatures = append(failedFeatures, feature)
			}
		}
		pluginName := configs[responseIndex].Name()
		if supported&uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS) != 0 && len(required.editionToFilenames) > 0 {
			// Plugin supports editions, and files include editions.
			// First, let's make sure that the plugin set the min/max edition fields correctly.
			if response.MinimumEdition == nil {
				return fmt.Errorf(
					"plugin %q advertises that it supports editions but did not indicate a minimum supported edition",
					pluginName,
				)
			}
			if response.MaximumEdition == nil {
				return fmt.Errorf(
					"plugin %q advertises that it supports editions but did not indicate a maximum supported edition",
					pluginName,
				)
			}
			if response.GetMaximumEdition() < response.GetMinimumEdition() {
				return fmt.Errorf(
					"plugin %q indicates a maximum supported edition (%v) that is less than its minimum supported edition (%v)",
					pluginName,
					descriptorpb.Edition(response.GetMaximumEdition()),
					descriptorpb.Edition(response.GetMinimumEdition()),
				)
			}

			// And also make sure the plugin supports precisely the right editions.
			requiredEditions := make([]descriptorpb.Edition, 0, len(required.editionToFilenames))
			for edition := range required.editionToFilenames {
				requiredEditions = append(requiredEditions, edition)
			}
			sort.Slice(requiredEditions, func(i, j int) bool {
				return requiredEditions[i] < requiredEditions[j]
			})
			for _, requiredEdition := range requiredEditions {
				if int32(requiredEdition) < response.GetMinimumEdition() ||
					int32(requiredEdition) > response.GetMaximumEdition() {
					failed.editionToFilenames[requiredEdition] = required.editionToFilenames[requiredEdition]
					failedEditions = append(failedEditions, requiredEdition)
				}
			}
		}

		if len(failedFeatures) > 0 {
			sort.Slice(failedFeatures, func(i, j int) bool {
				return failedFeatures[i] < failedFeatures[j]
			})
			for _, feature := range failedFeatures {
				// For CLI versions pre-1.32.0, we logged unsupported features. However, this is an
				// unsafe behavior for editions. So, in keeping with pre-1.32.0 CLI versions, we
				// warn for proto3 optional, but error if editions are required (BSR-3931).
				if feature == pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL {
					warningMessage := fmt.Sprintf("plugin %q does not support required features.\n", pluginName)

					files := failed.featureToFilenames[feature]
					warningMessage = fmt.Sprintln(
						warningMessage,
						fmt.Sprintf(" Feature %q is required by %d file(s):", featureName(feature), len(files)),
					)
					warningMessage = fmt.Sprintln(warningMessage, fmt.Sprintf("   %s", strings.Join(files, ",")))
					logger.Warn(strings.TrimSpace(warningMessage))
					continue
				}
				featureErrs := slicesext.Map(
					failed.featureToFilenames[feature],
					func(fileName string) error {
						return fmt.Errorf("plugin %q does not support feature %q which is required by %q", pluginName, featureName(feature), fileName)
					},
				)
				errs = append(errs, featureErrs...)
			}
		}
		if len(failedEditions) > 0 {
			sort.Slice(failedEditions, func(i, j int) bool {
				return failedEditions[i] < failedEditions[j]
			})
			for _, edition := range failedEditions {
				for _, file := range failed.editionToFilenames[edition] {
					errs = append(errs, fmt.Errorf("plugin %q does not support edition %q which is required by %q",
						pluginName, editionName(edition), file))
				}
			}
		}
	}
	return multierr.Combine(errs...)
}

func featureName(feature pluginpb.CodeGeneratorResponse_Feature) string {
	// FEATURE_PROTO3_OPTIONAL -> "proto3 optional"
	return enumReadableName(feature, "FEATURE")
}

func editionName(edition descriptorpb.Edition) string {
	// EDITION_2023 -> "2023"
	return enumReadableName(edition, "EDITION")
}

func enumReadableName(
	enum interface {
		protoreflect.Enum
		String() string
	},
	prefix string,
) string {
	return strings.TrimSpace(
		strings.ToLower(
			strings.ReplaceAll(
				strings.TrimPrefix(enum.String(), prefix),
				"_", " ")))
}

func fileHasProto3Optional(fileDescriptorProto *descriptorpb.FileDescriptorProto) bool {
	if fileDescriptorProto.GetSyntax() != "proto3" {
		// can't have proto3 optional unless syntax is proto3
		return false
	}
	for _, msg := range fileDescriptorProto.MessageType {
		if messageHasProto3Optional(msg) {
			return true
		}
	}
	return false
}

func messageHasProto3Optional(descriptorProto *descriptorpb.DescriptorProto) bool {
	for _, fld := range descriptorProto.Field {
		if fld.GetProto3Optional() {
			return true
		}
	}
	for _, nested := range descriptorProto.NestedType {
		if messageHasProto3Optional(nested) {
			return true
		}
	}
	return false
}

func fileHasEditions(fileDescriptorProto *descriptorpb.FileDescriptorProto) bool {
	return fileDescriptorProto.GetSyntax() == "editions"
}
