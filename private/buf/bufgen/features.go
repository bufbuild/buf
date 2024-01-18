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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// requiredFeatures maps a feature to the set of files in an image that
// make use of that feature.
type requiredFeatures map[pluginpb.CodeGeneratorResponse_Feature][]string

type featureChecker func(options *descriptorpb.FileDescriptorProto) bool

// Map of all known features to functions that can check whether a given file
// uses said feature.
var allFeatures = map[pluginpb.CodeGeneratorResponse_Feature]featureChecker{
	pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL: fileHasProto3Optional,
}

// computeRequiredFeatures returns a map of required features to the files in
// the image that require that feature. After plugins are invoked, the plugins'
// responses are checked to make sure any required features were supported.
func computeRequiredFeatures(image bufimage.Image) requiredFeatures {
	features := requiredFeatures{}
	for feature, checker := range allFeatures {
		for _, imageFile := range image.Files() {
			if imageFile.IsImport() {
				// we only want to check the sources in the module, not their dependencies
				continue
			}
			if checker(imageFile.FileDescriptorProto()) {
				features[feature] = append(features[feature], imageFile.Path())
			}
		}
	}
	return features
}

func checkRequiredFeatures(
	container app.StderrContainer,
	required requiredFeatures,
	responses []*pluginpb.CodeGeneratorResponse,
	configs []*PluginConfig,
) {
	for responseIndex, response := range responses {
		if response == nil || response.GetError() != "" {
			// plugin failed, nothing to check
			continue
		}
		failed := requiredFeatures{}
		var failedFeatures []pluginpb.CodeGeneratorResponse_Feature
		supported := response.GetSupportedFeatures() // bit mask of features the plugin supports
		for feature, files := range required {
			featureMask := (uint64)(feature)
			if supported&featureMask != featureMask {
				// doh! Supported features don't include this one
				failed[feature] = files
				failedFeatures = append(failedFeatures, feature)
			}
		}
		if len(failed) > 0 {
			// TODO: plugin config to turn this into an error
			_, _ = fmt.Fprintf(container.Stderr(), "Warning: plugin %q does not support required features.\n",
				configs[responseIndex].PluginName())
			sort.Slice(failedFeatures, func(i, j int) bool {
				return failedFeatures[i].Number() < failedFeatures[j].Number()
			})
			for _, feature := range failedFeatures {
				files := failed[feature]
				_, _ = fmt.Fprintf(container.Stderr(), "  Feature %q is required by %d file(s):\n",
					featureName(feature), len(files))
				_, _ = fmt.Fprintf(container.Stderr(), "    %s\n", strings.Join(files, ","))
			}
		}
	}
}

func featureName(feature pluginpb.CodeGeneratorResponse_Feature) string {
	// FEATURE_PROTO3_OPTIONAL -> "proto3 optional"
	return strings.TrimSpace(
		strings.ToLower(
			strings.ReplaceAll(
				strings.TrimPrefix(feature.String(), "FEATURE"),
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
