// Copyright 2020-2021 Buf Technologies, Inc.
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
//
// Note this is currently implicitly tested through buf's protoc command.
// If this were split out into a separate package, testing would need to be moved to this package.
package appproto

import (
	"context"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

// ResponseWriter handles CodeGeneratorResponses.
type ResponseWriter interface {
	// Add adds the file to the response.
	//
	// Returns error if nil or the name is empty.
	// Warns to stderr if the name is already added or the name is not normalized.
	AddFile(*pluginpb.CodeGeneratorResponse_File) error
	// AddError adds the error message to the response.
	//
	// If there is an existing error message, this will be concatenated with a newline.
	// If message is empty, a message "error" will be added.
	AddError(message string)
	// SetFeatureProto3Optional sets the proto3 optional feature.
	SetFeatureProto3Optional()
}

// Handler is a protoc plugin handler
type Handler interface {
	// Handle handles the plugin.
	//
	// This function can assume the request is valid.
	// This should only return error on system error.
	// Plugin generation errors should be added with AddError.
	// See https://github.com/protocolbuffers/protobuf/blob/95e6c5b4746dd7474d540ce4fb375e3f79a086f8/src/google/protobuf/compiler/plugin.proto#L100
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

// Main runs the plugin using app.Main and the Handler.
func Main(ctx context.Context, handler Handler) {
	app.Main(ctx, newRunFunc(handler))
}

// Run runs the plugin using app.Main and the Handler.
//
// The exit code can be determined using app.GetExitCode.
func Run(ctx context.Context, container app.Container, handler Handler) error {
	return app.Run(ctx, container, newRunFunc(handler))
}

// Generator executes the Handler using protoc's plugin execution logic.
//
// This invokes a Handler and writes out the response to the output location,
// additionally accounting for insertion point logic.
//
// If multiple requests are specified, these are executed in parallel and the
// result is combined into one response that is written.
type Generator interface {
	// Generate generates to the bucket.
	Generate(
		ctx context.Context,
		container app.EnvStderrContainer,
		writeBucket storage.WriteBucket,
		requests []*pluginpb.CodeGeneratorRequest,
		options ...GenerateOption,
	) error
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithInsertionPointReadBucket returns a new GenerateOption that uses the given
// ReadBucket to read from for insertion points.
//
// If this is not specified, insertion points are not supported.
func GenerateWithInsertionPointReadBucket(
	insertionPointReadBucket storage.ReadBucket,
) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.insertionPointReadBucket = insertionPointReadBucket
	}
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	handler Handler,
) Generator {
	return newGenerator(logger, handler)
}

// newRunFunc returns a new RunFunc for app.Main and app.Run.
func newRunFunc(handler Handler) func(context.Context, app.Container) error {
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
		if err := protodescriptor.ValidateCodeGeneratorRequest(request); err != nil {
			return err
		}
		responseWriter := newResponseWriter(container)
		if err := handler.Handle(ctx, container, responseWriter, request); err != nil {
			return err
		}
		response := responseWriter.toResponse()
		if err := protodescriptor.ValidateCodeGeneratorResponse(response); err != nil {
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
