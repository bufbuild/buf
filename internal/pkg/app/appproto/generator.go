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
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger  *zap.Logger
	handler Handler
}

func newGenerator(
	logger *zap.Logger,
	handler Handler,
) *generator {
	return &generator{
		logger:  logger,
		handler: handler,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	writeBucket storage.WriteBucket,
	requests []*pluginpb.CodeGeneratorRequest,
	options ...GenerateOption,
) error {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	files, err := g.getResponseFiles(ctx, container, requests)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.GetInsertionPoint() != "" {
			if generateOptions.insertionPointReadBucket == nil {
				return storage.NewErrNotExist(file.GetName())
			}
			if err := applyInsertionPoint(ctx, file, generateOptions.insertionPointReadBucket, writeBucket); err != nil {
				return err
			}
		} else if err := storage.PutPath(ctx, writeBucket, file.GetName(), []byte(file.GetContent())); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) getResponseFiles(
	ctx context.Context,
	container app.EnvStderrContainer,
	requests []*pluginpb.CodeGeneratorRequest,
) ([]*pluginpb.CodeGeneratorResponse_File, error) {
	responseWriter := newResponseWriter(container)
	jobs := make([]func() error, len(requests))
	for i, request := range requests {
		request := request
		jobs[i] = func() error {
			if err := protodescriptor.ValidateCodeGeneratorRequest(request); err != nil {
				return err
			}
			return g.handler.Handle(ctx, container, responseWriter, request)
		}
	}
	if err := thread.Parallelize(jobs...); err != nil {
		return nil, err
	}
	response := responseWriter.toResponse()
	if err := protodescriptor.ValidateCodeGeneratorResponse(response); err != nil {
		return nil, err
	}
	if errString := response.GetError(); errString != "" {
		return nil, errors.New(errString)
	}
	return response.File, nil
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
) error {
	resultData, err := calculateInsertionPoint(ctx, file, readBucket)
	if err != nil {
		return err
	}
	// This relies on storageos buckets maintaining existing file permissions
	return storage.PutPath(ctx, writeBucket, file.GetName(), resultData)
}

func calculateInsertionPoint(
	ctx context.Context,
	file *pluginpb.CodeGeneratorResponse_File,
	readBucket storage.ReadBucket,
) (_ []byte, retErr error) {
	targetReadObjectCloser, err := readBucket.Get(ctx, file.GetName())
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

type generateOptions struct {
	insertionPointReadBucket storage.ReadBucket
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
