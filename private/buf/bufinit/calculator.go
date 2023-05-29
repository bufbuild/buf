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

	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.uber.org/zap"
)

type calculator struct {
	logger *zap.Logger
}

func newCalculator(logger *zap.Logger) *calculator {
	return &calculator{
		logger: logger,
	}
}

func (c *calculator) Calculate(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
) (_ *calculation, retErr error) {
	// TODO: if a file has a directory path that matches its package structure,
	// that is a good hint that a buf.yaml should be at the root of the package structure
	// need to make sure all files are covered by a buf.yaml
	// also need to make sure every file has exactly one buf.yaml
	// TODO: for common things like gogo, add a dep to buf.yaml if the file is not found

	calculation := newCalculation()
	defer func() {
		if checkedEntry := c.logger.Check(zap.DebugLevel, "calculation"); checkedEntry != nil {
			checkedEntry.Write(zap.Reflect("value", calculation))
		}
	}()

	if err := c.populateFileInfos(ctx, readWriteBucket, calculation); err != nil {
		return nil, err
	}
	if err := c.populateImportPathMaps(calculation); err != nil {
		return nil, err
	}

	if err := calculation.postValidate(); err != nil {
		return nil, err
	}
	return calculation, nil
}

func (c *calculator) populateFileInfos(
	ctx context.Context,
	readWriteBucket storage.ReadWriteBucket,
	calculation *calculation,
) error {
	return c.forEachFileNode(
		ctx,
		readWriteBucket,
		func(fileNode *ast.FileNode) error {
			fileInfo, err := newFileInfo(fileNode)
			if err != nil {
				return err
			}
			return calculation.addFileInfo(fileInfo)
		},
	)
}

// assumes populateFileInfos has been called
func (c *calculator) populateImportPathMaps(
	calculation *calculation,
) error {
	node := newReversePathTrieNode()
	for filePath := range calculation.FilePathToFileInfo {
		node.Insert(filePath)
	}
	for _, fileInfo := range calculation.FilePathToFileInfo {
		for _, importPath := range fileInfo.ImportPaths {
			importDirPaths, present := node.Get(importPath)
			if present {
				for _, importDirPath := range importDirPaths {
					if err := calculation.addImportDirPathAndImportPath(importDirPath, importPath); err != nil {
						return err
					}
				}
			} else if !datawkt.Exists(importPath) {
				if err := calculation.addMissingImportPathAndFilePath(importPath, fileInfo.Path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *calculator) forEachFileNode(
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
				c.logger.Debug("syntax_error", zap.String("file_path", readObject.Path()), zap.Error(err))
			}
			return f(fileNode)
		},
	)
}
