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

package bufformat

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.uber.org/multierr"
)

// FormatModule formats and writes the target module files into a read bucket.
func FormatModule(ctx context.Context, module bufmodule.Module) (_ storage.ReadBucket, retErr error) {
	fileInfos, err := module.TargetFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	jobs := make([]func(context.Context) error, len(fileInfos))
	for i, fileInfo := range fileInfos {
		fileInfo := fileInfo
		jobs[i] = func(ctx context.Context) (retErr error) {
			moduleFile, err := module.GetModuleFile(ctx, fileInfo.Path())
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, moduleFile.Close())
			}()
			fileNode, err := parser.Parse(moduleFile.ExternalPath(), moduleFile, reporter.NewHandler(nil))
			if err != nil {
				return err
			}
			writeObjectCloser, err := readWriteBucket.Put(ctx, moduleFile.Path())
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, writeObjectCloser.Close())
			}()
			if err := FormatFileNode(writeObjectCloser, fileNode); err != nil {
				return err
			}
			return writeObjectCloser.SetExternalPath(moduleFile.ExternalPath())
		}
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	return readWriteBucket, nil
}

// FormatFileNode formats the given file node and writ the result to dest.
func FormatFileNode(dest io.Writer, fileNode *ast.FileNode) error {
	formatter := newFormatter(dest, fileNode)
	return formatter.Run()
}
