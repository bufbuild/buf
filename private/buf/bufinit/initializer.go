// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufinit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.uber.org/zap"
)

type initializer struct {
	logger *zap.Logger
}

func newInitializer(logger *zap.Logger) *initializer {
	return &initializer{
		logger: logger,
	}
}

func (i *initializer) Initialize(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
	options ...InitializeOption,
) error {
	initializeOptions := &initializeOptions{}
	for _, option := range options {
		option(initializeOptions)
	}
	return i.initialize(ctx, readWriteBucket)
}

func (i *initializer) initialize(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
) error {
	// TODO: if a file has a directory path that matches its package structure,
	// that is a good hint that a buf.yaml should be at the root of the package structure
	// need to make sure all files are covered by a buf.yaml
	// also need to make sure every file has exactly one buf.yaml
	// TODO: for common things like gogo, add a dep to buf.yaml if the file is not found

	fileInfos, err := i.getSortedFileInfos(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	if checkedEntry := i.logger.Check(zap.DebugLevel, "file_infos"); checkedEntry != nil {
		data, err := json.Marshal(fileInfos)
		if err != nil {
			return err
		}
		checkedEntry.Write(zap.String("value", string(data)))
	}
	node := newReversePathTrieNode()
	for _, fileInfo := range fileInfos {
		node.Insert(fileInfo.Path)
	}
	directoryMap := make(map[string]struct{})
	missingImportPathToFilePaths := make(map[string][]string)
	for _, fileInfo := range fileInfos {
		for _, importPath := range fileInfo.ImportPaths {
			directories, present := node.Get(importPath)
			if present {
				for _, directory := range directories {
					directoryMap[directory] = struct{}{}
				}
			} else if !datawkt.Exists(importPath) {
				missingImportPathToFilePaths[importPath] = append(missingImportPathToFilePaths[importPath], fileInfo.Path)
			}
		}
	}
	if len(missingImportPathToFilePaths) > 0 {
		for importPath, paths := range missingImportPathToFilePaths {
			fmt.Printf("%s is imported by %v but is not found in the current directory.\n", importPath, stringutil.SliceToHumanString(paths))
		}
		fmt.Println()
		fmt.Println("Given that you have missing imports, buf will not be able to build directly.")
		fmt.Println()
	}
	directories := stringutil.MapToSortedSlice(directoryMap)
	if len(directories) > 0 {
		fmt.Println("Directories that need a buf.yaml:")
		fmt.Println()
		for _, directory := range directories {
			fmt.Println(directory)
		}
	} else {
		fmt.Println("No directories need a buf.yaml.")
	}
	return nil
}

func (i *initializer) getSortedFileInfos(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
) ([]*fileInfo, error) {
	var fileInfos []*fileInfo
	i.forEachFileNode(
		ctx,
		readWriteBucket,
		func(fileNode *ast.FileNode) error {
			fileInfo, err := newFileInfo(fileNode)
			if err != nil {
				return err
			}
			fileInfos = append(fileInfos, fileInfo)
			return nil
		},
	)
	sortFileInfos(fileInfos)
	return fileInfos, nil
}

func (i *initializer) forEachFileNode(
	ctx context.Context,
	readBucket storage.ReadBucket,
	f func(*ast.FileNode) error,
) error {
	handler := reporter.NewHandler(
		reporter.NewReporter(
			func(reporter.ErrorWithPos) error {
				// never aborts
				return nil
			},
			nil,
		),
	)
	return storage.WalkReadObjects(
		ctx,
		storage.MapReadBucket(
			readBucket,
			storage.MatchPathExt(".proto"),
		),
		"",
		func(readObject storage.ReadObject) error {
			// This can return an error and non-nil AST.
			// readObject.Path() will always be normalized.
			fileNode, err := parser.Parse(readObject.Path(), readObject, handler)
			if fileNode == nil {
				// No AST implies an I/O error trying to read the file contents. Consider this a real error.
				return err
			}
			if err != nil {
				// There was a syntax error, but we still have a partial AST we can examine.
				i.logger.Debug("syntax_error", zap.String("file_path", readObject.Path()), zap.Error(err))
			}
			return f(fileNode)
		},
	)
}

type initializeOptions struct{}
