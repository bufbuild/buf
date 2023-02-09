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

package main

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	appproto.Main(context.Background(), appproto.HandlerFunc(handle))
}

func handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
	if err := responseWriter.AddFile(
		&pluginpb.CodeGeneratorResponse_File{
			Name:           proto.String("test.txt"),
			InsertionPoint: proto.String("example"),
			Content: proto.String(`
			// Include this comment on the 'example' insertion point.
			  // This is another example where whitespaces are preserved.
			  // And this demonstrates a newline literal (\n).
			// And don't forget the windows newline literal (\r\n).
		`),
		},
	); err != nil {
		return err
	}
	if err := responseWriter.AddFile(
		&pluginpb.CodeGeneratorResponse_File{
			Name:           proto.String("test.txt"),
			InsertionPoint: proto.String("other"),
			Content: proto.String(`
			// Include this comment on the 'other' insertion point.
		`),
		},
	); err != nil {
		return err
	}
	return nil
}
