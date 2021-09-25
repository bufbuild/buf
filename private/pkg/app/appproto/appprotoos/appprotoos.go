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

// Package appprotoos does OS-specific generation.
package appprotoos

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

// Generator is used to generate code to the OS filesystem.
type Generator interface {
	// Generate generates to the os filesystem, switching on the file extension.
	// If there is a .jar extension, this generates a jar. If there is a .zip
	// extension, this generates a zip. If there is no extension, this outputs
	// to the directory. The corresponding CodeGeneratorResponse written is returned.
	Generate(
		ctx context.Context,
		container app.EnvStderrContainer,
		pluginName string,
		requests []*pluginpb.CodeGeneratorRequest,
		options ...GenerateOption,
	) (*pluginpb.CodeGeneratorResponse, error)
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) Generator {
	return newGenerator(logger, storageosProvider)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithPluginPath returns a new GenerateOption that uses the given
// path to the plugin.
func GenerateWithPluginPath(pluginPath string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.pluginPath = pluginPath
	}
}

// BucketResponseWriter writes CodeGeneratorResponses to the OS filesystem.
type BucketResponseWriter interface {
	// Close writes all of the responses to disk. No further calls can be
	// made to the BucketResponseWriter after this call.
	io.Closer

	// AddResponse adds the response to the writer, switching on the file extension.
	// If there is a .jar extension, this generates a jar. If there is a .zip
	// extension, this generates a zip. If there is no extension, this outputs
	// to the directory.
	AddResponse(
		ctx context.Context,
		response *pluginpb.CodeGeneratorResponse,
		pluginOut string,
	) error
}

// NewBucketResponseWriter returns a new BucketResponseWriter.
func NewBucketResponseWriter(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	options ...BucketResponseWriterOption,
) BucketResponseWriter {
	return newBucketResponseWriter(
		logger,
		storageosProvider,
		options...,
	)
}

// BucketResponseWriterOption is an option for the BucketResponseWriter.
type BucketResponseWriterOption func(*bucketResponseWriterOptions)

// BucketResponseWriterWithCreateOutDirIfNotExists returns a new BucketResponseWriterOption that creates
// the directory if it does not exist.
func BucketResponseWriterWithCreateOutDirIfNotExists() BucketResponseWriterOption {
	return func(bucketResponseWriterOptions *bucketResponseWriterOptions) {
		bucketResponseWriterOptions.createOutDirIfNotExists = true
	}
}
