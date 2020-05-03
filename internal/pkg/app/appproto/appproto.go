// Copyright 2020 Buf Technologies Inc.
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
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto"
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// ResponseWriter is a response writer.
//
// Not thread-safe.
type ResponseWriter interface {
	// WriteCodeGeneratorResponseFile adds the file to the response.
	//
	// Can be called multiple times.
	WriteCodeGeneratorResponseFile(*plugin_go.CodeGeneratorResponse_File)
	// WriteError writes the error to the response.
	//
	// Can be called multiple times. Errors will be concatenated by newlines.
	// Resulting error string will have spaces trimmed before creating the response.
	WriteError(string)
}

// Main runs the application using the OS Container and calling os.Exit on the return value of Run.
func Main(
	ctx context.Context,
	f func(
		ctx context.Context,
		container app.EnvStderrContainer,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	),
) {
	app.Main(ctx, newRunFunc(f))
}

// Run runs the application using the container.
func Run(
	ctx context.Context,
	container app.Container,
	f func(
		ctx context.Context,
		container app.EnvStderrContainer,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	),
) error {
	return app.Run(ctx, container, newRunFunc(f))
}

func newRunFunc(
	f func(
		ctx context.Context,
		container app.EnvStderrContainer,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	),
) func(context.Context, app.Container) error {
	return func(ctx context.Context, container app.Container) error {
		return run(ctx, container, f)
	}
}

func run(
	ctx context.Context,
	container app.Container,
	f func(
		ctx context.Context,
		container app.EnvStderrContainer,
		responseWriter ResponseWriter,
		request *plugin_go.CodeGeneratorRequest,
	),
) error {
	input, err := ioutil.ReadAll(container.Stdin())
	if err != nil {
		return err
	}
	request := &plugin_go.CodeGeneratorRequest{}
	if err := utilproto.UnmarshalWire(input, request); err != nil {
		return err
	}
	responseWriter := newResponseWriter()
	f(ctx, container, responseWriter, request)
	response := responseWriter.ToCodeGeneratorResponse()
	data, err := utilproto.MarshalWire(response)
	if err != nil {
		return err
	}
	_, err = container.Stdout().Write(data)
	return err
}

type responseWriter struct {
	files         []*plugin_go.CodeGeneratorResponse_File
	errorMessages []string
}

func newResponseWriter() *responseWriter {
	return &responseWriter{}
}

func (r *responseWriter) WriteCodeGeneratorResponseFile(file *plugin_go.CodeGeneratorResponse_File) {
	r.files = append(r.files, file)
}

func (r *responseWriter) WriteError(errorMessage string) {
	r.errorMessages = append(r.errorMessages, errorMessage)
}

func (r *responseWriter) ToCodeGeneratorResponse() *plugin_go.CodeGeneratorResponse {
	var err *string
	if errorMessage := strings.TrimSpace(strings.Join(r.errorMessages, "\n")); errorMessage != "" {
		err = proto.String(errorMessage)
	}
	return &plugin_go.CodeGeneratorResponse{
		File:  r.files,
		Error: err,
	}
}
