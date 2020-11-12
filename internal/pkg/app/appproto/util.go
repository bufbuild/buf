// Copyright 2020 Buf Technologies, Inc.
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

package appproto

import (
	"context"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"google.golang.org/protobuf/types/pluginpb"
)

func runHandler(
	ctx context.Context,
	container app.EnvStderrContainer,
	handler Handler,
	request *pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	if err := protodescriptor.ValidateCodeGeneratorRequest(request); err != nil {
		return nil, err
	}
	responseWriter := newResponseWriter(container)
	response := responseWriter.toResponse(handler.Handle(ctx, container, responseWriter, request))
	if err := protodescriptor.ValidateCodeGeneratorResponse(response); err != nil {
		return nil, err
	}
	return response, nil
}
