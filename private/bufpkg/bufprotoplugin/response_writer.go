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

package bufprotoplugin

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"unicode"
	"unicode/utf8"

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
				return &fs.PathError{Op: "stat", Path: file.GetName(), Err: fs.ErrNotExist}
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
	var found bool
	for i := 0; targetScanner.Scan(); i++ {
		if i > 0 {
			// These writes cannot fail, they will panic if they cannot allocate.
			_, _ = postInsertionContent.Write(newline)
		}
		targetLine := targetScanner.Bytes()
		if !bytes.Contains(targetLine, match) {
			// These writes cannot fail, they will panic if they cannot allocate.
			_, _ = postInsertionContent.Write(targetLine)
			continue
		}
		// For each line in then new content, apply the
		// same amount of whitespace. This is important
		// for specific languages, e.g. Python.
		whitespace := leadingWhitespace(targetLine)

		// Insert the content from the insertion point file. Handle newlines in
		// a platform-agnostic manner.
		insertedContentReader := strings.NewReader(insertionPointFile.GetContent())
		writeWithPrefixAndLineEnding(postInsertionContent, insertedContentReader, whitespace, newline)

		// Code inserted at this point is placed immediately
		// above the line containing the insertion point, so
		// we include it last.
		// These writes cannot fail, they will panic if they cannot
		// allocate
		_, _ = postInsertionContent.Write(targetLine)
		found = true
	}
	if err := targetScanner.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("could not find insertion point %q in %q", insertionPointFile.GetInsertionPoint(), insertionPointFile.GetName())
	}
	return postInsertionContent.Bytes(), nil
}

// leadingWhitespace iterates through the given string,
// and returns the leading whitespace substring, if any,
// respecting utf-8 encoding.
//
//	leadingWhitespace("\u205F   foo ") -> "\u205F   "
func leadingWhitespace(buf []byte) []byte {
	leadingSize := 0
	iterBuf := buf
	for len(iterBuf) > 0 {
		r, size := utf8.DecodeRune(iterBuf)
		// protobuf strings must always be valid UTF8
		// https://developers.google.com/protocol-buffers/docs/proto3#scalar
		// Additionally, utf8.RuneError is not a space so we'll terminate
		// and return the leading, valid, UTF8 whitespace sequence.
		if !unicode.IsSpace(r) {
			out := make([]byte, leadingSize)
			copy(out, buf)
			return out
		}
		leadingSize += size
		iterBuf = iterBuf[size:]
	}
	return buf
}

// writeWithPrefixAndLineEnding iterates over each of the given reader's lines
// prepends prefix, and appends the newline sequence.
func writeWithPrefixAndLineEnding(dst *bytes.Buffer, src io.Reader, prefix, newline []byte) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		// These writes cannot fail, they will panic if they cannot allocate.
		_, _ = dst.Write(prefix)
		_, _ = dst.Write(scanner.Bytes())
		_, _ = dst.Write(newline)
	}
}

type writeResponseOptions struct {
	insertionPointReadBucket storage.ReadBucket
}

func newWriteResponseOptions() *writeResponseOptions {
	return &writeResponseOptions{}
}
