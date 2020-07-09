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

// Package appproto contains helper functionality for protoc plugins.
package appproto

import (
	"context"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"google.golang.org/protobuf/types/pluginpb"
)

// ResponseWriter handles CodeGeneratorResponses.
type ResponseWriter interface {
	// Add adds the file to the response.
	//
	// Returns error if nil, the name is empty, or the name is already added.
	Add(*pluginpb.CodeGeneratorResponse_File) error
	// SetFeatureProto3Optional sets the proto3 optional feature.
	SetFeatureProto3Optional()
}

// Handler is a protoc plugin handler
type Handler interface {
	// Handle handles the plugin.
	//
	// This function can assume the request is valid.
	Handle(
		ctx context.Context,
		container app.EnvStderrContainer,
		responseWriter ResponseWriter,
		request *pluginpb.CodeGeneratorRequest,
	) error
}

// HandlerFunc is a handler function.
type HandlerFunc func(
	context.Context,
	app.EnvStderrContainer,
	ResponseWriter,
	*pluginpb.CodeGeneratorRequest,
) error

// Handle implements Handler.
func (h HandlerFunc) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter ResponseWriter,
	request *pluginpb.CodeGeneratorRequest,
) error {
	return h(ctx, container, responseWriter, request)
}

// NewRunFunc returns a new RunFunc for app.Main and app.Run.
func NewRunFunc(handler Handler) func(context.Context, app.Container) error {
	return func(ctx context.Context, container app.Container) error {
		input, err := ioutil.ReadAll(container.Stdin())
		if err != nil {
			return err
		}
		request := &pluginpb.CodeGeneratorRequest{}
		// We do not know the FileDescriptorSet before unmarshaling this
		if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(input, request); err != nil {
			return err
		}
		response, err := Execute(ctx, container, handler, request)
		if err != nil {
			return err
		}
		data, err := protoencoding.NewWireMarshaler().Marshal(response)
		if err != nil {
			return err
		}
		_, err = container.Stdout().Write(data)
		return err
	}
}

// Execute executes the given handler.
func Execute(
	ctx context.Context,
	container app.EnvStderrContainer,
	handler Handler,
	request *pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	if err := protodescriptor.ValidateCodeGeneratorRequest(request); err != nil {
		return nil, err
	}
	responseWriter := newResponseWriter()
	response := responseWriter.toResponse(handler.Handle(ctx, container, responseWriter, request))
	if err := protodescriptor.ValidateCodeGeneratorResponse(response); err != nil {
		return nil, err
	}
	return response, nil
}
