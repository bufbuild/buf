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

// Package bufprotoplugin contains helper functionality for protoc plugins.
//
// Note this is currently implicitly tested through buf's protoc command.
// If this were split out into a separate package, testing would need to be
// moved to this package.
package bufprotoplugin

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/protoplugin"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	// Our generated files in `private/gen/proto` are on average 15KB which isn't
	// an unreasonable amount of memory to reserve each time we process an insertion
	// point and will save a significant number of allocations.
	averageGeneratedFileSize = 15 * 1024
)

// Generator executes the Handler using protoc's plugin execution logic.
//
// If multiple requests are specified, these are executed in parallel and the
// result is combined into one response that is written.
type Generator interface {
	// Generate generates a CodeGeneratorResponse for the given CodeGeneratorRequests.
	//
	// A new ResponseBuilder is constructed for every invocation of Generate and is
	// used to consolidate all of the CodeGeneratorResponse_Files returned from a single
	// plugin into a single CodeGeneratorResponse.
	Generate(
		ctx context.Context,
		container app.EnvStderrContainer,
		requests []*pluginpb.CodeGeneratorRequest,
	) (*pluginpb.CodeGeneratorResponse, error)
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	handler protoplugin.Handler,
) Generator {
	return newGenerator(logger, handler)
}

// ResponseWriter handles the response and writes it to the given storage.WriteBucket
// without executing any plugins and handles insertion points as needed.
type ResponseWriter interface {
	// WriteResponse writes to the bucket with the given response. In practice, the
	// WriteBucket is most often an in-memory bucket.
	//
	// CodeGeneratorResponses are consolidated into the bucket, and insertion points
	// are applied in-place so that they can only access the files created in a single
	// generation invocation (just like protoc).
	WriteResponse(
		ctx context.Context,
		writeBucket storage.WriteBucket,
		response *pluginpb.CodeGeneratorResponse,
		options ...WriteResponseOption,
	) error
}

// NewResponseWriter returns a new ResponseWriter.
func NewResponseWriter(logger *zap.Logger) ResponseWriter {
	return newResponseWriter(logger)
}

// WriteResponseOption is an option for WriteResponse.
type WriteResponseOption func(*writeResponseOptions)

// WriteResponseWithInsertionPointReadBucket returns a new WriteResponseOption that uses the given
// ReadBucket to read from for insertion points.
//
// If this is not specified, insertion points are not supported.
func WriteResponseWithInsertionPointReadBucket(
	insertionPointReadBucket storage.ReadBucket,
) WriteResponseOption {
	return func(writeResponseOptions *writeResponseOptions) {
		writeResponseOptions.insertionPointReadBucket = insertionPointReadBucket
	}
}

// PluginResponse encapsulates a CodeGeneratorResponse,
// along with the name of the plugin that created it.
type PluginResponse struct {
	Response   *pluginpb.CodeGeneratorResponse
	PluginName string
	PluginOut  string
}

// NewPluginResponse retruns a new *PluginResponse.
func NewPluginResponse(
	response *pluginpb.CodeGeneratorResponse,
	pluginName string,
	pluginOut string,
) *PluginResponse {
	return &PluginResponse{
		Response:   response,
		PluginName: pluginName,
		PluginOut:  pluginOut,
	}
}

// ValidatePluginResponses validates that each file is only defined by a single *PluginResponse.
func ValidatePluginResponses(pluginResponses []*PluginResponse) error {
	seen := make(map[string]string)
	for _, pluginResponse := range pluginResponses {
		for _, file := range pluginResponse.Response.File {
			if file.GetInsertionPoint() != "" {
				// We expect insertion points to write
				// to files that already exist.
				continue
			}
			fileName := filepath.Join(pluginResponse.PluginOut, file.GetName())
			if pluginName, ok := seen[fileName]; ok {
				return fmt.Errorf(
					"file %q was generated multiple times: once by plugin %q and again by plugin %q",
					fileName,
					pluginName,
					pluginResponse.PluginName,
				)
			}
			seen[fileName] = pluginResponse.PluginName
		}
		// Note: we used to verify that the plugin set min/max edition correctly if it set the
		// SUPPORTS_EDITIONS feature. But some plugins in the protoc codebase, from when editions
		// were still experimental, advertise SUPPORTS_EDITIONS but did not set those fields. So
		// we defer the check and only complain if/when the input actually contains Editions files.
	}
	return nil
}
