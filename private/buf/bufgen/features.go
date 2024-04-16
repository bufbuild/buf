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
	"github.com/bufbuild/buf/private/pkg/app"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// requiredFeatures maps a feature to the set of files in an image that
// make use of that feature.
type requiredFeatures struct {
	flags                  map[pluginpb.CodeGeneratorResponse_Feature][]string
	editions               map[descriptorpb.Edition][]string
	minEdition, maxEdition descriptorpb.Edition
}

func newRequiredFeatures() *requiredFeatures {
	return &requiredFeatures{
		flags:    map[pluginpb.CodeGeneratorResponse_Feature][]string{},
		editions: map[descriptorpb.Edition][]string{},
	}
}

type featureChecker func(options *descriptorpb.FileDescriptorProto) bool

// Map of all known features to functions that can check whether a given file
// uses said feature.
var allFeatures = map[pluginpb.CodeGeneratorResponse_Feature]featureChecker{
	pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL:   fileHasProto3Optional,
	pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS: fileHasEditions,
}

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
		for feature, checker := range allFeatures {
			if checker(imageFile.FileDescriptorProto()) {
				features.flags[feature] = append(features.flags[feature], imageFile.Path())
			}
		}
		// We also collect the range of required editions.
		if !fileHasEditions(imageFile.FileDescriptorProto()) {
			continue
		}
		edition := imageFile.FileDescriptorProto().GetEdition()
		features.editions[edition] = append(features.editions[edition], imageFile.Path())
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
	container app.StderrContainer,
	required *requiredFeatures,
	responses []*pluginpb.CodeGeneratorResponse,
	configs []bufconfig.GeneratePluginConfig,
) error {
	var failedPlugins []string
	for responseIndex, response := range responses {
		if response == nil || response.GetError() != "" {
			// plugin failed, nothing to check
			continue
		}

		failed := newRequiredFeatures()
		var failedFeatures []pluginpb.CodeGeneratorResponse_Feature
		var failedEditions []descriptorpb.Edition
		supported := response.GetSupportedFeatures() // bit mask of features the plugin supports
		for feature, files := range required.flags {
			featureMask := uint64(feature)
			if supported&featureMask != featureMask {
				// doh! Supported features don't include this one
				failed.flags[feature] = files
				failedFeatures = append(failedFeatures, feature)
			}
		}
		if supported&uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS) != 0 && len(required.editions) > 0 {
			// Plugin supports editions, and files include editions. So make sure
			// the plugin supports precisely the right editions.
			requiredEditions := make([]descriptorpb.Edition, 0, len(required.editions))
			for edition := range required.editions {
				requiredEditions = append(requiredEditions, edition)
			}
			sort.Slice(requiredEditions, func(i, j int) bool {
				return requiredEditions[i] < requiredEditions[j]
			})
			for _, requiredEdition := range requiredEditions {
				if int32(requiredEdition) < response.GetMinimumEdition() ||
					int32(requiredEdition) > response.GetMaximumEdition() {
					failed.editions[requiredEdition] = required.editions[requiredEdition]
					failedEditions = append(failedEditions, requiredEdition)
				}
			}
		}

		pluginName := configs[responseIndex].Name()
		if len(failedFeatures) > 0 {
			_, _ = fmt.Fprintf(
				container.Stderr(),
				"Plugin %q does not support required feature(s).\n",
				pluginName)
			sort.Slice(failedFeatures, func(i, j int) bool {
				return failedFeatures[i] < failedFeatures[j]
			})
			for _, feature := range failedFeatures {
				files := failed.flags[feature]
				_, _ = fmt.Fprintf(
					container.Stderr(),
					"  Feature %q is required by %d file(s):\n",
					featureName(feature), len(files))
				_, _ = fmt.Fprintf(
					container.Stderr(),
					"    %s\n",
					strings.Join(files, ","))
			}
		}

		if len(failedEditions) > 0 {
			_, _ = fmt.Fprintf(
				container.Stderr(),
				"Plugin %q does not support required edition(s).\n",
				pluginName)
			sort.Slice(failedEditions, func(i, j int) bool {
				return failedEditions[i] < failedEditions[j]
			})
			for _, edition := range failedEditions {
				files := failed.editions[edition]
				_, _ = fmt.Fprintf(
					container.Stderr(),
					"  Edition %q is required by %d file(s):\n",
					editionName(edition), len(files))
				_, _ = fmt.Fprintf(
					container.Stderr(),
					"    %s\n",
					strings.Join(files, ","))
			}
		}

		if len(failedFeatures) > 0 || len(failedEditions) > 0 {
			failedPlugins = append(failedPlugins, pluginName)
		}
	}
	switch len(failedPlugins) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("plugin %s is unable to generate code for all input files", failedPlugins[0])
	default:
		return fmt.Errorf("plugins [%v] are unable to generate code for all input files", strings.Join(failedPlugins, ","))
	}
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
