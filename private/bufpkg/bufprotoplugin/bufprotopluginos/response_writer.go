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

package bufprotopluginos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"buf.build/go/standard/xpath/xfilepath"
	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/thread"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	jarExt = ".jar"
	zipExt = ".zip"
)

// Constants used to create .jar files.
var (
	manifestPath    = normalpath.Join("META-INF", "MANIFEST.MF")
	manifestContent = []byte(`Manifest-Version: 1.0
Created-By: 1.6.0 (protoc)

`)
)

type responseWriter struct {
	logger            *slog.Logger
	storageosProvider storageos.Provider
	responseWriter    bufprotoplugin.ResponseWriter
	// If set, create directories if they don't already exist.
	createOutDirIfNotExists bool
	// If set, delete files from output directories that were not written
	// during generation.
	deleteOuts bool
	// Cache the readWriteBuckets by their respective output paths.
	// These builders are transformed to storage.ReadBuckets and written
	// to disk once the responseWriter is flushed.
	//
	// Note that output paths are used as-is with respect to the
	// caller's configuration. It's possible that a single invocation
	// will specify the same filepath in multiple ways, e.g. "." and
	// "$(pwd)". However, we intentionally treat these as distinct paths
	// to mirror protoc's insertion point behavior.
	//
	// For example, the following command will fail because protoc treats
	// "." and "$(pwd)" as distinct paths:
	//
	// $ protoc example.proto --insertion-point-receiver_out=. --insertion-point-writer_out=$(pwd)
	//
	readWriteBuckets map[string]storage.ReadWriteBucket
	// Cache the functions used to flush all of the responses to disk.
	// This holds all of the buckets in-memory so that we only write
	// the results to disk if all of the responses are successful.
	closers []func(ctx context.Context) error
	lock    sync.RWMutex
}

func newResponseWriter(
	logger *slog.Logger,
	storageosProvider storageos.Provider,
	options ...ResponseWriterOption,
) *responseWriter {
	responseWriterOptions := newResponseWriterOptions()
	for _, option := range options {
		option(responseWriterOptions)
	}
	return &responseWriter{
		logger:                  logger,
		storageosProvider:       storageosProvider,
		responseWriter:          bufprotoplugin.NewResponseWriter(logger),
		createOutDirIfNotExists: responseWriterOptions.createOutDirIfNotExists,
		deleteOuts:              responseWriterOptions.deleteOuts,
		readWriteBuckets:        make(map[string]storage.ReadWriteBucket),
	}
}

func (w *responseWriter) AddResponse(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	pluginOut string,
) error {
	// It's important that we get a consistent output path
	// so that we use the same in-memory bucket for paths
	// set to the same directory.
	//
	// filepath.Abs calls filepath.Clean.
	//
	// For example:
	//
	// --insertion-point-receiver_out=insertion --insertion-point-writer_out=./insertion/ --insertion-point_writer_out=/foo/insertion
	absPluginOut, err := filepath.Abs(normalpath.Unnormalize(pluginOut))
	if err != nil {
		return err
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.addResponse(
		ctx,
		response,
		absPluginOut,
		w.createOutDirIfNotExists,
	)
}

func (w *responseWriter) Close(ctx context.Context) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, closeFunc := range w.closers {
		if err := closeFunc(ctx); err != nil {
			// Although unlikely, if an error happens here,
			// some generated files could be written to disk,
			// whereas others aren't.
			//
			// Regardless, we stop at the first error so that
			// we don't unnecessarily write more results.
			return err
		}
	}
	// Collect the set of generated paths per directory output before
	// clearing state, so the delete phase knows which files to keep.
	var dirOutputPaths map[string]map[string]struct{}
	if w.deleteOuts {
		dirOutputPaths = make(map[string]map[string]struct{}, len(w.readWriteBuckets))
		for outPath, readWriteBucket := range w.readWriteBuckets {
			if isArchivePath(outPath) {
				continue
			}
			paths, err := storage.AllPaths(ctx, readWriteBucket, "")
			if err != nil {
				return err
			}
			pathSet := make(map[string]struct{}, len(paths))
			for _, path := range paths {
				pathSet[path] = struct{}{}
			}
			dirOutputPaths[outPath] = pathSet
		}
	}
	// Re-initialize the cached values to be safe.
	w.readWriteBuckets = make(map[string]storage.ReadWriteBucket)
	w.closers = nil
	if !w.deleteOuts {
		return nil
	}
	// Delete stale files and remove empty directories.
	for outDirPath, retainPaths := range dirOutputPaths {
		if err := w.deleteStaleFilesAndEmptyDirs(ctx, outDirPath, retainPaths); err != nil {
			return err
		}
	}
	return nil
}

func (w *responseWriter) addResponse(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	pluginOut string,
	createOutDirIfNotExists bool,
) error {
	// Validate on the first time we see each output path when deleteOuts is
	// enabled, before committing to any destructive operations.
	if w.deleteOuts {
		if _, seen := w.readWriteBuckets[pluginOut]; !seen {
			if err := w.validateDeleteOutPath(pluginOut); err != nil {
				return err
			}
		}
	}
	switch filepath.Ext(pluginOut) {
	case jarExt:
		return w.writeZip(
			ctx,
			response,
			pluginOut,
			true,
			createOutDirIfNotExists,
		)
	case zipExt:
		return w.writeZip(
			ctx,
			response,
			pluginOut,
			false,
			createOutDirIfNotExists,
		)
	default:
		return w.writeDirectory(
			ctx,
			response,
			pluginOut,
			createOutDirIfNotExists,
		)
	}
}

func (w *responseWriter) writeZip(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	outFilePath string,
	includeManifest bool,
	createOutDirIfNotExists bool,
) error {
	outDirPath := filepath.Dir(outFilePath)
	if readWriteBucket, ok := w.readWriteBuckets[outFilePath]; ok {
		// We already have a readWriteBucket for this outFilePath, so
		// we can write to the same bucket.
		if err := w.responseWriter.WriteResponse(
			ctx,
			readWriteBucket,
			response,
			bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
		); err != nil {
			return err
		}
		return nil
	}
	// OK to use os.Stat instead of os.Lstat here.
	fileInfo, err := os.Stat(outDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			if createOutDirIfNotExists {
				if err := os.MkdirAll(outDirPath, 0755); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return err
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("not a directory: %s", outDirPath)
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	if includeManifest {
		if err := storage.PutPath(ctx, readWriteBucket, manifestPath, manifestContent); err != nil {
			return err
		}
	}
	if err := w.responseWriter.WriteResponse(
		ctx,
		readWriteBucket,
		response,
		bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
	); err != nil {
		return err
	}
	// Add this readWriteBucket to the set so that other plugins
	// can write to the same files (re: insertion points).
	w.readWriteBuckets[outFilePath] = readWriteBucket
	w.closers = append(w.closers, func(ctx context.Context) error {
		// Zip the generated content into a buffer so we can compare it with
		// the existing file before deciding whether to write. This preserves
		// the modification time when the output is unchanged.
		var buf bytes.Buffer
		// protoc does not compress.
		if err := storagearchive.Zip(ctx, readWriteBucket, &buf, false); err != nil {
			return err
		}
		newContent := buf.Bytes()
		existingContent, err := os.ReadFile(outFilePath)
		if err == nil && bytes.Equal(existingContent, newContent) {
			return nil
		}
		file, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(newContent)
		return errors.Join(writeErr, file.Close())
	})
	return nil
}

func (w *responseWriter) writeDirectory(
	ctx context.Context,
	response *pluginpb.CodeGeneratorResponse,
	outDirPath string,
	createOutDirIfNotExists bool,
) error {
	if readWriteBucket, ok := w.readWriteBuckets[outDirPath]; ok {
		// We already have a readWriteBucket for this outDirPath, so
		// we can write to the same bucket.
		if err := w.responseWriter.WriteResponse(
			ctx,
			readWriteBucket,
			response,
			bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
		); err != nil {
			return err
		}
		return nil
	}
	readWriteBucket := storagemem.NewReadWriteBucket()
	if err := w.responseWriter.WriteResponse(
		ctx,
		readWriteBucket,
		response,
		bufprotoplugin.WriteResponseWithInsertionPointReadBucket(readWriteBucket),
	); err != nil {
		return err
	}
	// Add this readWriteBucket to the set so that other plugins
	// can write to the same files (re: insertion points).
	w.readWriteBuckets[outDirPath] = readWriteBucket
	w.closers = append(w.closers, func(ctx context.Context) error {
		if createOutDirIfNotExists {
			if err := os.MkdirAll(outDirPath, 0755); err != nil {
				return err
			}
		}
		// This checks that the directory exists.
		osReadWriteBucket, err := w.storageosProvider.NewReadWriteBucket(
			outDirPath,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		return w.copySkipUnchanged(ctx, readWriteBucket, osReadWriteBucket)
	})
	return nil
}

// copySkipUnchanged copies all paths from the source bucket to the destination,
// skipping any path whose content already matches what is on disk. This preserves
// mtimes for unchanged generated files so that mtime-based build systems do not
// rebuild unnecessarily.
func (w *responseWriter) copySkipUnchanged(
	ctx context.Context,
	from storage.ReadBucket,
	to storage.ReadWriteBucket,
) error {
	paths, err := storage.AllPaths(ctx, from, "")
	if err != nil {
		return err
	}
	jobs := make([]func(context.Context) error, len(paths))
	for i, path := range paths {
		jobs[i] = func(ctx context.Context) error {
			newData, err := storage.ReadPath(ctx, from, path)
			if err != nil {
				return err
			}
			existingData, err := storage.ReadPath(ctx, to, path)
			if err == nil && bytes.Equal(existingData, newData) {
				w.logger.DebugContext(ctx, "skipping unchanged generated file", slog.String("path", path))
				return nil
			}
			// Not-exist, read error, or content differs: fall through to write.
			// We intentionally swallow read errors here; this comparison is an
			// optimization and must not cause generate to fail.
			return storage.PutPath(ctx, to, path, newData)
		}
	}
	return thread.Parallelize(ctx, jobs)
}

// deleteStaleFilesAndEmptyDirs deletes files present in outDirPath that are
// not in retainPaths, then removes any directories that are now empty.
func (w *responseWriter) deleteStaleFilesAndEmptyDirs(
	ctx context.Context,
	outDirPath string,
	retainPaths map[string]struct{},
) error {
	osReadWriteBucket, err := w.storageosProvider.NewReadWriteBucket(
		outDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Output directory doesn't exist; nothing to delete.
			return nil
		}
		return err
	}
	existingPaths, err := storage.AllPaths(ctx, osReadWriteBucket, "")
	if err != nil {
		return err
	}
	var deleteJobs []func(context.Context) error
	for _, existingPath := range existingPaths {
		if _, ok := retainPaths[existingPath]; !ok {
			deleteJobs = append(deleteJobs, func(ctx context.Context) error {
				w.logger.DebugContext(ctx, "deleting stale generated file", slog.String("path", existingPath))
				if err := osReadWriteBucket.Delete(ctx, existingPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				return nil
			})
		}
	}
	if err := thread.Parallelize(ctx, deleteJobs); err != nil {
		return err
	}
	return removeEmptyDirs(outDirPath)
}

// removeEmptyDirs recursively removes all empty directories under rootDir.
// It processes children before parents so that a chain of directories that
// are empty after their children are removed will be fully cleaned up.
// The rootDir itself is never removed.
//
// This operates directly on the filesystem because the storage abstraction
// only models files, not directories.
func removeEmptyDirs(rootDir string) error {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			childDir := filepath.Join(rootDir, entry.Name())
			if err := removeEmptyDirs(childDir); err != nil {
				return err
			}
			// Re-check after recursing into children: the child directory
			// may now be empty if all its contents were removed.
			childEntries, err := os.ReadDir(childDir)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return err
			}
			if len(childEntries) == 0 {
				if err := os.Remove(childDir); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
	}
	return nil
}

// validateDeleteOutPath checks that the output path is safe to delete from.
// It prevents accidentally deleting files from the current working directory,
// which could happen if a user configures out as ".".
// The path is already absolute (via filepath.Abs in AddResponse).
func (w *responseWriter) validateDeleteOutPath(absOutPath string) error {
	pwd, err := osext.Getwd()
	if err != nil {
		return err
	}
	resolvedPwd, err := resolveCleanPath(pwd)
	if err != nil {
		return err
	}
	resolvedOut, err := resolveCleanPath(absOutPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if resolvedOut == resolvedPwd {
		return errors.New("cannot use --clean if your plugin will output to the current directory")
	}
	return nil
}

type responseWriterOptions struct {
	createOutDirIfNotExists bool
	deleteOuts              bool
}

func newResponseWriterOptions() *responseWriterOptions {
	return &responseWriterOptions{}
}

// resolveCleanPath returns the real, cleaned absolute path, following symlinks.
func resolveCleanPath(path string) (string, error) {
	path, err := xfilepath.RealClean(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}

// isArchivePath returns true if the given path has a .zip or .jar extension.
func isArchivePath(path string) bool {
	ext := filepath.Ext(path)
	return ext == zipExt || ext == jarExt
}
