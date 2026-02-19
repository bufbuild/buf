// Copyright 2020-2025 Buf Technologies, Inc.
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
	"errors"
	"io"
	"log/slog"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/printer"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
)

// FormatModuleSet formats and writes the target files into a read bucket.
func FormatModuleSet(ctx context.Context, moduleSet bufmodule.ModuleSet) (_ storage.ReadBucket, retErr error) {
	return FormatBucket(
		ctx,
		bufmodule.ModuleReadBucketToStorageReadBucket(
			bufmodule.ModuleReadBucketWithOnlyTargetFiles(
				bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(moduleSet),
			),
		),
	)
}

// FormatBucket formats the .proto files in the bucket and returns a new bucket with the formatted files.
func FormatBucket(ctx context.Context, bucket storage.ReadBucket) (_ storage.ReadBucket, retErr error) {
	readWriteBucket := storagemem.NewReadWriteBucket()
	paths, err := storage.AllPaths(ctx, storage.FilterReadBucket(bucket, storage.MatchPathExt(".proto")), "")
	if err != nil {
		return nil, err
	}
	jobs := make([]func(context.Context) error, len(paths))
	for i, path := range paths {
		jobs[i] = func(ctx context.Context) (retErr error) {
			readObjectCloser, err := bucket.Get(ctx, path)
			if err != nil {
				return err
			}
			defer func() {
				retErr = errors.Join(retErr, readObjectCloser.Close())
			}()
			data, err := io.ReadAll(readObjectCloser)
			if err != nil {
				return err
			}
			sourceFile := source.NewFile(readObjectCloser.ExternalPath(), string(data))
			r := &report.Report{}
			parsed, _ := parser.Parse(path, sourceFile, r)
			for _, d := range r.Diagnostics {
				if d.Level() <= report.Error {
					slog.WarnContext(ctx, "parse error during format", "path", path, "error", d.Message())
				}
			}
			formatted := FormatFile(parsed)
			writeObjectCloser, err := readWriteBucket.Put(ctx, path)
			if err != nil {
				return err
			}
			defer func() {
				retErr = errors.Join(retErr, writeObjectCloser.Close())
			}()
			if _, err := io.WriteString(writeObjectCloser, formatted); err != nil {
				return err
			}
			return writeObjectCloser.SetExternalPath(readObjectCloser.ExternalPath())
		}
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	return readWriteBucket, nil
}

// FormatFile formats the file and returns the result as a string.
func FormatFile(file *ast.File) string {
	return printer.PrintFile(printer.Options{Format: true}, file)
}
