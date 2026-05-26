// Copyright 2020-2026 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/printer"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/source/length"
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
	var matcher *fullNameMatcher
	if len(options.deprecatePrefixes) > 0 {
		matcher = newFullNameMatcher(options.deprecatePrefixes...)
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
			data, err := io.ReadAll(readObjectCloser)
			if err != nil {
				return err
			}
			file, err := parseFile(readObjectCloser, data)
			if err != nil {
				return err
			}
			if matcher != nil && applyDeprecations(file, matcher) {
				deprecationMatched.Store(true)
			}
			writeObjectCloser, err := readWriteBucket.Put(ctx, path)
			if err != nil {
				return err
			}
			defer func() {
				retErr = errors.Join(retErr, writeObjectCloser.Close())
			}()
			if err := FormatFile(writeObjectCloser, file); err != nil {
				return err
			}
			return writeObjectCloser.SetExternalPath(readObjectCloser.ExternalPath())
		}
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return nil, err
	}
	// If deprecation was requested but nothing matched, return an error.
	if matcher != nil && !deprecationMatched.Load() {
		return nil, fmt.Errorf("no types matched the specified deprecation prefixes")
	}
	return readWriteBucket, nil
}

// FormatFile formats the given file and writes the result to dest.
func FormatFile(dest io.Writer, file *ast.File) error {
	out, err := printer.PrintFile(printer.Options{
		Format:     true,
		Formatting: printer.Legacy(),
	}, file)
	if err != nil {
		return err
	}
	_, err = io.WriteString(dest, out)
	return err
}

// parseFile parses a .proto source file using the experimental parser.
//
// The parser may emit error-level diagnostics that are recoverable for
// formatting — e.g. edition 2024 import-ordering rule violations that
// canonicalization fixes anyway. We only fail when the parser produced
// no file at all, or when any top-level declaration is marked corrupt
// (signalling a syntactic failure that the formatter cannot recover
// from). This mirrors the legacy formatter's behavior of swallowing
// edition-2024-related errors while still failing on broken syntax.
func parseFile(fileInfo bufanalysis.FileInfo, data []byte) (*ast.File, error) {
	// Suppress non-error diagnostics at the source. We only ever surface
	// error-level diagnostics from this path.
	r := &report.Report{Options: report.Options{SuppressWarnings: true}}
	path := fileInfo.ExternalPath()
	file, _ := parser.Parse(path, source.NewFile(path, string(data)), r)
	if file == nil {
		return nil, fmt.Errorf("%s: parse failed", path)
	}
	for decl := range seq.Values(file.Decls()) {
		if def := decl.AsDef(); !def.IsZero() && def.IsCorrupt() {
			return nil, parseDiagnosticsAnnotationSet(fileInfo, r)
		}
	}
	return file, nil
}

// parseDiagnosticsAnnotationSet converts the error-level diagnostics into a
// file annotation set for rendering.
func parseDiagnosticsAnnotationSet(fileInfo bufanalysis.FileInfo, r *report.Report) error {
	var annotations []bufanalysis.FileAnnotation
	for _, diagnostic := range r.Diagnostics {
		primary := diagnostic.Primary()
		if primary.IsZero() {
			// Spanless diagnostics (e.g. companions to fatal file-open
			// errors) have no location to render and would be displayed
			// as "<input>:1:1:..."; skip them. Matches build_image.go.
			continue
		}
		start := primary.Location(primary.Start, length.Bytes)
		end := primary.Location(primary.End, length.Bytes)
		annotations = append(
			annotations,
			bufanalysis.NewFileAnnotation(
				fileInfo,
				start.Line,
				start.Column,
				end.Line,
				end.Column,
				"COMPILE",
				diagnostic.Message(),
				"", // pluginName
				"", // policyName
			),
		)
	}
	if len(annotations) == 0 {
		return fmt.Errorf("%s: parse failed", fileInfo.ExternalPath())
	}
	return bufanalysis.NewFileAnnotationSet(annotations...)
}
