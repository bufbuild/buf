// Copyright 2020-2022 Buf Technologies, Inc.
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
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// requiredFeatures maps a feature to the set of files in an image that
// depend on that make use of that feature.
type requiredFeatures map[pluginpb.CodeGeneratorResponse_Feature][]string

type featureChecker func(options *descriptorpb.FileDescriptorProto) bool

var allFeatures = map[pluginpb.CodeGeneratorResponse_Feature]featureChecker{
	pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL: fileHasProto3Optional,
}

// computeRequiredFeatures returns a map of required features to files in the
// image that require that feature. After plugins are invoked, the plugins'
// response is checked to make sure that any required features were supported.
func computeRequiredFeatures(img bufimage.Image) requiredFeatures {
	features := requiredFeatures{}
	for feature, checker := range allFeatures {
		for _, file := range img.Files() {
			if file.IsImport() {
				// we only want to check the sources in the module, not their dependencies
				continue
			}
			if checker(file.Proto()) {
				features[feature] = append(features[feature], file.Path())
			}
		}
	}
	return features
}

func checkRequiredFeatures(req requiredFeatures, resps []*pluginpb.CodeGeneratorResponse) {
	for _, resp := range resps {
		if resp == nil || resp.GetError() != "" {
			// plugin failed, nothing to check
			continue
		}
		failed := requiredFeatures{}
		var failedFeatures []pluginpb.CodeGeneratorResponse_Feature
		supported := resp.GetSupportedFeatures() // bit mask of features the plugin supports
		for feature, files := range req {
			featureMask := (uint64)(feature)
			if supported&featureMask != featureMask {
				// doh! Supported features don't include this one
				failed[feature] = files
				failedFeatures = append(failedFeatures, feature)
			}
		}
		if len(failed) > 0 {
			var buf bytes.Buffer
			buf.WriteString("Plugin does not support required features.\n")
			sort.Slice(failedFeatures, func(i, j int) bool {
				return failedFeatures[i].Number() < failedFeatures[j].Number()
			})
			for _, feature := range failedFeatures {
				files := failed[feature]
				// bytes.Buffer does not generate I/O errors, so no need to check err result
				_, _ = fmt.Fprintf(&buf, "%v, required by %d file(s):\n  ", feature, len(files))
				_, _ = fmt.Fprintln(&buf, strings.Join(files, ","))
			}
			// clear out code gen results and replace with an error
			resp.File = nil
			resp.Error = proto.String(buf.String())
		}
	}
}

func fileHasProto3Optional(fd *descriptorpb.FileDescriptorProto) bool {
	if fd.GetSyntax() != "proto3" {
		// can't have proto3 optional unless syntax is proto3
		return false
	}
	for _, msg := range fd.MessageType {
		if msgHasProto3Optional(msg) {
			return true
		}
	}
	return false
}

func msgHasProto3Optional(md *descriptorpb.DescriptorProto) bool {
	for _, fld := range md.Field {
		if fld.GetProto3Optional() {
			return true
		}
	}
	for _, nested := range md.NestedType {
		if msgHasProto3Optional(nested) {
			return true
		}
	}
	return false
}
