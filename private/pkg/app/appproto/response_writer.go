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

package appproto

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type responseWriter struct {
	logger *zap.Logger
}

func newResponseWriter(
	logger *zap.Logger,
) *responseWriter {
	return &responseWriter{
		logger: logger,
	}
}

func (h *responseWriter) WriteResponse(
	ctx context.Context,
	writeBucket storage.WriteBucket,
	response *pluginpb.CodeGeneratorResponse,
	options ...WriteResponseOption,
) error {
	writeResponseOptions := newWriteResponseOptions()
	for _, option := range options {
		option(writeResponseOptions)
	}
	for _, file := range response.File {
		if file.GetInsertionPoint() != "" {
			if writeResponseOptions.insertionPointReadBucket == nil {
				return storage.NewErrNotExist(file.GetName())
			}
			if err := applyInsertionPoint(ctx, file, writeResponseOptions.insertionPointReadBucket, writeBucket); err != nil {
				return err
			}
		} else if err := storage.PutPath(ctx, writeBucket, file.GetName(), []byte(file.GetContent())); err != nil {
			return err
		}
	}
	return nil
}

// applyInsertionPoint inserts the content of the given file at the insertion point that it specfiies.
// For more details on insertion points, see the following:
//
// https://github.com/protocolbuffers/protobuf/blob/f5bdd7cd56aa86612e166706ed8ef139db06edf2/src/google/protobuf/compiler/plugin.proto#L135-L171
func applyInsertionPoint(
	ctx context.Context,
	file *pluginpb.CodeGeneratorResponse_File,
	readBucket storage.ReadBucket,
	writeBucket storage.WriteBucket,
) (retErr error) {
	targetReadObjectCloser, err := readBucket.Get(ctx, file.GetName())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, targetReadObjectCloser.Close())
	}()
	resultData, err := writeInsertionPoint(ctx, file, targetReadObjectCloser)
	if err != nil {
		return err
	}
	// This relies on storageos buckets maintaining existing file permissions
	return storage.PutPath(ctx, writeBucket, file.GetName(), resultData)
}

// writeInsertionPoint writes the insertion point defined in insertionPointFile
// to the targetFile and returns the result as []byte. The caller must ensure the
// provided targetFile matches the file requested in insertionPointFile.Name.
func writeInsertionPoint(
	ctx context.Context,
	insertionPointFile *pluginpb.CodeGeneratorResponse_File,
	targetFile io.Reader,
) (_ []byte, retErr error) {
	targetScanner := bufio.NewScanner(targetFile)
	match := []byte("@@protoc_insertion_point(" + insertionPointFile.GetInsertionPoint() + ")")
	postInsertionContent := bytes.NewBuffer(nil)
	postInsertionContent.Grow(averageGeneratedFileSize)
	// TODO: We should respect the line endings in the generated file. This would
	// require either targetFile being an io.ReadSeeker and in the worst case
	// doing 2 full scans of the file (if it is a single line), or implementing
	// bufio.Scanner.Scan() inline
	newline := []byte{'\n'}
	for targetScanner.Scan() {
		targetLine := targetScanner.Bytes()
		if !bytes.Contains(targetLine, match) {
			// these writes cannot fail, they will panic if they cannot
			// allocate
			_, _ = postInsertionContent.Write(targetLine)
			_, _ = postInsertionContent.Write(newline)
			continue
		}
		// For each line in then new content, apply the
		// same amount of whitespace. This is important
		// for specific languages, e.g. Python.
		whitespace := leadingWhitespace(targetLine)

		// Create another scanner so that we can seamlessly handle
		// newlines in a platform-agnostic manner.
		insertedContentScanner := bufio.NewScanner(bytes.NewBufferString(insertionPointFile.GetContent()))
		insertedContent := scanWithPrefixAndLineEnding(insertedContentScanner, whitespace, newline)
		// This write cannot fail, it will panic if it cannot
		// allocate
		_, _ = postInsertionContent.Write(insertedContent)

		// Code inserted at this point is placed immediately
		// above the line containing the insertion point, so
		// we include it last.
		// These writes cannot fail, they will panic if they cannot
		// allocate
		_, _ = postInsertionContent.Write(targetLine)
		_, _ = postInsertionContent.Write(newline)
	}

	if err := targetScanner.Err(); err != nil {
		return nil, err
	}

	// trim the trailing newline
	postInsertionBytes := postInsertionContent.Bytes()
	return postInsertionBytes[:len(postInsertionBytes)-1], nil
}

type writeResponseOptions struct {
	insertionPointReadBucket storage.ReadBucket
}

func newWriteResponseOptions() *writeResponseOptions {
	return &writeResponseOptions{}
}
