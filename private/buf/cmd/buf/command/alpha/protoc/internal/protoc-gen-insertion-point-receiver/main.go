// Copyright 2020-2025 Buf Technologies, Inc.
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

package main

import (
	"context"

	"github.com/bufbuild/protoplugin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	protoplugin.Main(protoplugin.HandlerFunc(handle))
}

func handle(
	_ context.Context,
	_ protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	_ protoplugin.Request,
) error {
	responseWriter.AddCodeGeneratorResponseFiles(
		&pluginpb.CodeGeneratorResponse_File{
			Name: proto.String("test.txt"),
			Content: proto.String(`
		// The following line represents an insertion point named 'example'.
		// We include a few indentation to verify the whitespace is preserved
		// in the inserted content.
		//
		//     @@protoc_insertion_point(example)
		//
		// The 'other' insertion point is also included so that we verify
		// multiple insertion points can be written in a single invocation.
		//
		//   @@protoc_insertion_point(other)
		//
		// Note that all text should be added above the insertion points.
		`),
		},
	)
	return nil
}
