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
	"fmt"
	"io"
	"sync/atomic"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
)

// FormatOption is an option for formatting.
type FormatOption func(*formatOptions)

// formatOptions contains options for formatting.
type formatOptions struct {
	deprecatePrefixes []string
}

// WithDeprecate adds a deprecation prefix. All types whose fully-qualified name
// starts with this prefix will have the deprecated option added to them.
// For fields and enum values, only exact matches are deprecated.
func WithDeprecate(fqnPrefix string) FormatOption {
	return func(opts *formatOptions) {
		opts.deprecatePrefixes = append(opts.deprecatePrefixes, fqnPrefix)
	}
}

// FormatModuleSet formats and writes the target files into a read bucket.
func FormatModuleSet(ctx context.Context, moduleSet bufmodule.ModuleSet, opts ...FormatOption) (_ storage.ReadBucket, retErr error) {
	return FormatBucket(
		ctx,
		bufmodule.ModuleReadBucketToStorageReadBucket(
			bufmodule.ModuleReadBucketWithOnlyTargetFiles(
				bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFilesForTargetModules(moduleSet),
			),
		),
		opts...,
	)
}

// FormatBucket formats the .proto files in the bucket and returns a new bucket with the formatted files.
// If WithDeprecate options are provided but no types match the prefixes, an error is returned.
func FormatBucket(ctx context.Context, bucket storage.ReadBucket, opts ...FormatOption) (_ storage.ReadBucket, retErr error) {
	options := &formatOptions{}
	for _, opt := range opts {
		opt(options)
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	paths, err := storage.AllPaths(ctx, storage.FilterReadBucket(bucket, storage.MatchPathExt(".proto")), "")
	if err != nil {
		return nil, err
	}
	// Track if any deprecation prefix matched across all files.
	var deprecationMatched atomic.Bool
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
			fileNode, err := parser.Parse(readObjectCloser.ExternalPath(), readObjectCloser, reporter.NewHandler(nil))
			if err != nil {
				return err
			}
			writeObjectCloser, err := readWriteBucket.Put(ctx, path)
			if err != nil {
				return err
			}
			defer func() {
				retErr = errors.Join(retErr, writeObjectCloser.Close())
			}()
			matched, err := formatFileNodeWithMatch(writeObjectCloser, fileNode, options)
			if err != nil {
				return err
			}
			if matched {
				deprecationMatched.Store(true)
			}
			return writeObjectCloser.SetExternalPath(readObjectCloser.ExternalPath())
		}
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	// If deprecation was requested but nothing matched, return an error.
	if len(options.deprecatePrefixes) > 0 && !deprecationMatched.Load() {
		return nil, fmt.Errorf("no types matched the specified deprecation prefixes")
	}
	return readWriteBucket, nil
}

// FormatFileNode formats the given file node and writes the result to dest.
func FormatFileNode(dest io.Writer, fileNode *ast.FileNode) error {
	_, err := formatFileNodeWithMatch(dest, fileNode, &formatOptions{})
	return err
}

// formatFileNode formats the given file node with options and writes the result to dest.
func formatFileNode(dest io.Writer, fileNode *ast.FileNode, options *formatOptions) error {
	_, err := formatFileNodeWithMatch(dest, fileNode, options)
	return err
}

// formatFileNodeWithMatch formats the given file node and returns whether any deprecation prefix matched.
func formatFileNodeWithMatch(dest io.Writer, fileNode *ast.FileNode, options *formatOptions) (bool, error) {
	// Construct the file descriptor to ensure the AST is valid. This will
	// capture unknown syntax like edition "2024" which at the current time is
	// not supported.
	if _, err := parser.ResultFromAST(fileNode, true, reporter.NewHandler(nil)); err != nil {
		return false, err
	}
	formatter := newFormatter(dest, fileNode, options)
	if err := formatter.Run(); err != nil {
		return false, err
	}
	return formatter.deprecationMatched, nil
}
