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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

var (
	manifestPath    = normalpath.Join("META-INF", "MANIFEST.MF")
	manifestContent = []byte(`Manifest-Version: 1.0
Created-By: 1.6.0 (protoc)

`)
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
	pluginOut string,
	requests ...*pluginpb.CodeGeneratorRequest,
) error {
	var files []*pluginpb.CodeGeneratorResponse_File
	var lock sync.Mutex
	switch len(requests) {
	case 0:
		return nil
	case 1:
		iFiles, err := e.runHandler(ctx, container, requests[0])
		if err != nil {
			return err
		}
		files = iFiles
	default:
		jobs := make([]func() error, len(requests))
		for i, request := range requests {
			request := request
			jobs[i] = func() error {
				iFiles, err := e.runHandler(ctx, container, request)
				if err != nil {
					return err
				}
				lock.Lock()
				files = append(files, iFiles...)
				lock.Unlock()
				return nil
			}
		}
		if err := thread.Parallelize(jobs...); err != nil {
			return err
		}
	}
	switch filepath.Ext(pluginOut) {
	case ".jar":
		return writeResponseFilesArchive(ctx, files, pluginOut, true)
	case ".zip":
		return writeResponseFilesArchive(ctx, files, pluginOut, false)
	case "":
		return writeResponseFilesDirectory(ctx, files, pluginOut)
	default:
		return fmt.Errorf("unsupported output: %s", pluginOut)
	}
}

func (e *executor) runHandler(
	ctx context.Context,
	container app.EnvStderrContainer,
	request *pluginpb.CodeGeneratorRequest,
) ([]*pluginpb.CodeGeneratorResponse_File, error) {
	response, err := runHandler(ctx, container, e.handler, request)
	if err != nil {
		return nil, err
	}
	if errString := response.GetError(); errString != "" {
		return nil, errors.New(errString)
	}
	return response.File, nil
}

func writeResponseFilesArchive(
	ctx context.Context,
	files []*pluginpb.CodeGeneratorResponse_File,
	outFilePath string,
	includeManifest bool,
) (retErr error) {
	fileInfo, err := os.Stat(filepath.Dir(outFilePath))
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("not a directory: %s", filepath.Dir(outFilePath))
	}
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	for _, file := range files {
		if file.GetInsertionPoint() != "" {
			return fmt.Errorf("insertion points not supported with archive output: %s", file.GetName())
		}
		if err := storage.PutPath(ctx, readBucketBuilder, file.GetName(), []byte(file.GetContent())); err != nil {
			return err
		}
	}
	if includeManifest {
		if err := storage.PutPath(ctx, readBucketBuilder, manifestPath, manifestContent); err != nil {
			return err
		}
	}
	readBucket, err := readBucketBuilder.ToReadBucket()
	if err != nil {
		return err
	}
	file, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, file.Close())
	}()
	// protoc does not compress
	return storagearchive.Zip(ctx, readBucket, file, false)
}

func writeResponseFilesDirectory(
	ctx context.Context,
	files []*pluginpb.CodeGeneratorResponse_File,
	outDirPath string,
) error {
	// this checks that the directory exists
	readWriteBucket, err := storageos.NewReadWriteBucket(outDirPath)
	if err != nil {
		return err
	}
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
