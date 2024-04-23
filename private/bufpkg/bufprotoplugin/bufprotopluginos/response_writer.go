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

package bufprotopluginos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

// Constants used to create .jar files.
var (
	manifestPath    = normalpath.Join("META-INF", "MANIFEST.MF")
	manifestContent = []byte(`Manifest-Version: 1.0
Created-By: 1.6.0 (protoc)

`)
)

type responseWriter struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
	responseWriter    bufprotoplugin.ResponseWriter
	// If set, create directories if they don't already exist.
	createOutDirIfNotExists bool
	// Cache the readWriteBuckets by their respective output paths.
	// These builders are transformed to storage.ReadBuckets and written
	// to disk once the responseWriter is flushed.
	//
	// Note that output paths are used as-is with respect to the
	// caller's configuration. It's possible that a single invocation
	// will specify the same filepath in multiple ways, e.g. "." and
	// "$(pwd)". However, we intentionally treat these as distinct paths
	// to mirror protoc's insertion point behavior.
	//
	// For example, the following command will fail because protoc treats
	// "." and "$(pwd)" as distinct paths:
	//
	// $ protoc example.proto --insertion-point-receiver_out=. --insertion-point-writer_out=$(pwd)
	//
	readWriteBuckets map[string]storage.ReadWriteBucket
	// Cache the functions used to flush all of the responses to disk.
	// This holds all of the buckets in-memory so that we only write
	// the results to disk if all of the responses are successful.
	closers []func() error
	lock    sync.RWMutex
}

func newResponseWriter(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	options ...ResponseWriterOption,
) *responseWriter {
	responseWriterOptions := newResponseWriterOptions()
	for _, option := range options {
		option(responseWriterOptions)
	}
	return &responseWriter{
		logger:                  logger,
		storageosProvider:       storageosProvider,
		responseWriter:          bufprotoplugin.NewResponseWriter(logger),
		createOutDirIfNotExists: responseWriterOptions.createOutDirIfNotExists,
		readWriteBuckets:        make(map[string]storage.ReadWriteBucket),
	}
}

func (w *responseWriter) AddResponse(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	pluginOut string,
) error {
	// It's important that we get a consistent output path
	// so that we use the same in-memory bucket for paths
	// set to the same directory.
	//
	// filepath.Abs calls filepath.Clean.
	//
	// For example:
	//
	// --insertion-point-receiver_out=insertion --insertion-point-writer_out=./insertion/ --insertion-point_writer_out=/foo/insertion
	absPluginOut, err := filepath.Abs(normalpath.Unnormalize(pluginOut))
	if err != nil {
		return err
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.addResponse(
		ctx,
		response,
		absPluginOut,
		w.createOutDirIfNotExists,
	)
}

func (w *responseWriter) Close() error {
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, closeFunc := range w.closers {
		if err := closeFunc(); err != nil {
			// Although unlikely, if an error happens here,
			// some generated files could be written to disk,
			// whereas others aren't.
			//
			// Regardless, we stop at the first error so that
			// we don't unncessarily write more results.
			return err
		}
	}
	// Re-initialize the cached values to be safe.
	w.readWriteBuckets = make(map[string]storage.ReadWriteBucket)
	w.closers = nil
	return nil
}

func (w *responseWriter) addResponse(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	pluginOut string,
	createOutDirIfNotExists bool,
) error {
	switch filepath.Ext(pluginOut) {
	case ".jar":
		return w.writeZip(
			ctx,
			response,
			pluginOut,
			true,
			createOutDirIfNotExists,
		)
	case ".zip":
		return w.writeZip(
			ctx,
			response,
			pluginOut,
			false,
			createOutDirIfNotExists,
		)
	default:
		return w.writeDirectory(
			ctx,
			response,
			pluginOut,
			createOutDirIfNotExists,
		)
	}
}

func (w *responseWriter) writeZip(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	outFilePath string,
	includeManifest bool,
	createOutDirIfNotExists bool,
) (retErr error) {
	outDirPath := filepath.Dir(outFilePath)
	if readWriteBucket, ok := w.readWriteBuckets[outFilePath]; ok {
		// We already have a readWriteBucket for this outFilePath, so
		// we can write to the same bucket.
		if err := w.responseWriter.WriteResponse(
			ctx,
			readWriteBucket,
			response,
			bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
		); err != nil {
			return err
		}
		return nil
	}
	// OK to use os.Stat instead of os.Lstat here.
	fileInfo, err := os.Stat(outDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if createOutDirIfNotExists {
				if err := os.MkdirAll(outDirPath, 0755); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return err
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("not a directory: %s", outDirPath)
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	if includeManifest {
		if err := storage.PutPath(ctx, readWriteBucket, manifestPath, manifestContent); err != nil {
			return err
		}
	}
	if err := w.responseWriter.WriteResponse(
		ctx,
		readWriteBucket,
		response,
		bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
	); err != nil {
		return err
	}
	// Add this readWriteBucket to the set so that other plugins
	// can write to the same files (re: insertion points).
	w.readWriteBuckets[outFilePath] = readWriteBucket
	w.closers = append(w.closers, func() (retErr error) {
		// We're done writing all of the content into this
		// readWriteBucket, so we zip it when we flush.
		file, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		defer func() {
			retErr = multierr.Append(retErr, file.Close())
		}()
		// protoc does not compress.
		return storagearchive.Zip(ctx, readWriteBucket, file, false)
	})
	return nil
}

func (w *responseWriter) writeDirectory(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	outDirPath string,
	createOutDirIfNotExists bool,
) error {
	if readWriteBucket, ok := w.readWriteBuckets[outDirPath]; ok {
		// We already have a readWriteBucket for this outDirPath, so
		// we can write to the same bucket.
		if err := w.responseWriter.WriteResponse(
			ctx,
			readWriteBucket,
			response,
			bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
		); err != nil {
			return err
		}
		return nil
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	if err := w.responseWriter.WriteResponse(
		ctx,
		readWriteBucket,
		response,
		bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
	); err != nil {
		return err
	}
	// Add this readWriteBucket to the set so that other plugins
	// can write to the same files (re: insertion points).
	w.readWriteBuckets[outDirPath] = readWriteBucket
	w.closers = append(w.closers, func() error {
		if createOutDirIfNotExists {
			if err := os.MkdirAll(outDirPath, 0755); err != nil {
				return err
			}
		}
		// This checks that the directory exists.
		osReadWriteBucket, err := w.storageosProvider.NewReadWriteBucket(
			outDirPath,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		if _, err := storage.Copy(ctx, readWriteBucket, osReadWriteBucket); err != nil {
			return err
		}
		return nil
	})
	return nil
}

type responseWriterOptions struct {
	createOutDirIfNotExists bool
}

func newResponseWriterOptions() *responseWriterOptions {
	return &responseWriterOptions{}
}
