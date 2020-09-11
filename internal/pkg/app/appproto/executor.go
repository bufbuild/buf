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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type executor struct {
	logger  *zap.Logger
	handler Handler
}

func newExecutor(
	logger *zap.Logger,
	handler Handler,
) *executor {
	return &executor{
		logger:  logger,
		handler: handler,
	}
}

func (e *executor) Execute(
	ctx context.Context,
	container app.EnvStderrContainer,
	readWriteBucket storage.ReadWriteBucket,
	request *pluginpb.CodeGeneratorRequest,
) error {
	response, err := runHandler(ctx, container, e.handler, request)
	if err != nil {
		return err
	}
	if errString := response.GetError(); errString != "" {
		return errors.New(errString)
	}
	return writeResponseFiles(ctx, response.File, readWriteBucket)
}

func writeResponseFiles(
	ctx context.Context,
	files []*pluginpb.CodeGeneratorResponse_File,
	readWriteBucket storage.ReadWriteBucket,
) error {
	for _, file := range files {
		if file.GetInsertionPoint() != "" {
			if err := applyInsertionPoint(ctx, file, readWriteBucket); err != nil {
				return err
			}
		} else if err := storage.PutPath(ctx, readWriteBucket, file.GetName(), []byte(file.GetContent())); err != nil {
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
	readWriteBucket storage.ReadWriteBucket,
) error {
	resultData, err := calculateInsertionPoint(ctx, file, readWriteBucket)
	if err != nil {
		return err
	}
	// This relies on storageos buckets maintaining existing file permissions
	return storage.PutPath(ctx, readWriteBucket, file.GetName(), resultData)
}

func calculateInsertionPoint(
	ctx context.Context,
	file *pluginpb.CodeGeneratorResponse_File,
	readWriteBucket storage.ReadWriteBucket,
) (_ []byte, retErr error) {
	targetReadObjectCloser, err := readWriteBucket.Get(ctx, file.GetName())
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, targetReadObjectCloser.Close())
	}()
	targetScanner := bufio.NewScanner(targetReadObjectCloser)
	match := fmt.Sprintf("@@protoc_insertion_point(%s)", file.GetInsertionPoint())
	var resultLines []string
	for targetScanner.Scan() {
		targetLine := targetScanner.Text()
		if !strings.Contains(targetLine, match) {
			resultLines = append(resultLines, targetLine)
			continue
		}
		// For each line in then new content, apply the
		// same amount of whitespace. This is important
		// for specific languages, e.g. Python.
		whitespace := leadingWhitespace(targetLine)

		// Create another scanner so that we can seamlessly handle
		// newlines in a platform-agnostic manner.
		insertedContentScanner := bufio.NewScanner(bytes.NewBufferString(file.GetContent()))
		insertedLines := scanWithPrefix(insertedContentScanner, whitespace)
		resultLines = append(resultLines, insertedLines...)

		// Code inserted at this point is placed immediately
		// above the line containing the insertion point, so
		// we include it last.
		resultLines = append(resultLines, targetLine)
	}
	// Now that we've applied the insertion point content into
	// the target's original content, we overwrite the target.
	return []byte(strings.Join(resultLines, "\n")), nil
}

// leadingWhitespace iterates through the given string,
// and returns the leading whitespace substring, if any.
//
//  leadingWhitespace("   foo ") -> "   "
func leadingWhitespace(s string) string {
	for i, r := range s {
		if !unicode.IsSpace(r) {
			return s[:i]
		}
	}
	return s
}

// scanWithPrefix iterates over each of the given scanner's lines
// and prepends each one with the given prefix.
func scanWithPrefix(scanner *bufio.Scanner, prefix string) []string {
	var result []string
	for scanner.Scan() {
		result = append(result, prefix+scanner.Text())
	}
	return result
}
