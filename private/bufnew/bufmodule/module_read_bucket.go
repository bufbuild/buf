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

package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleReadBucket is an object analogous to storage.ReadBucket that supplements ObjectInfos
// and Objects with the data on the Module that supplied them.
//
// ModuleReadBuckets talk in terms of Files and FileInfos. They are easily converted into
// storage.ReadBuckets.
//
// The contents of a ModuleReadBucket are specific to its context. In the context of a Module,
// a ModuleReadBucket will return .proto files, documentation file(s), and license file(s). However,
// in the context of converting a Workspace into its corresponding .proto files, a ModuleReadBucket
// will only contain .proto files.
type ModuleReadBucket interface {
	// GetFile gets the File within the Module as specified by the path.
	//
	// Returns an error with fs.ErrNotExist if the path is not part of the Module.
	GetFile(ctx context.Context, path string) (File, error)
	// StatFileInfo gets the FileInfo for the File within the Module as specified by the path.
	//
	// Returns an error with fs.ErrNotExist if the path is not part of the Module.
	StatFileInfo(ctx context.Context, path string) (FileInfo, error)
	// WalkFileInfos walks all Files in the Module, passing the FileInfo to a specified function.
	//
	// This will walk the .proto files, documentation file(s), and license files(s). This package
	// currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	WalkFileInfos(ctx context.Context, f func(FileInfo) error, options ...WalkFileInfosOption) error

	isModuleReadBucket()
}

// WalkFileInfosOption is an option for WalkFileInfos
type WalkFileInfosOption func(*walkFileInfosOptions)

// WalkFileInfosWithOnlyTargetFiles returns a new WalkFileInfosOption that will result in only
// FileInfos with IsTargetFile() set to true being walked.
//
// Note that no Files from a Module will have IsTargetFile() set to true if
// Module.IsTargetModule() is false.
//
// If specific Files were not targeted but the Module was targeted, all Files will have
// IsTargetFile() set to true, and this function will return all Files that WalkFileInfos does.
func WalkFileInfosWithOnlyTargetFiles() WalkFileInfosOption {
	return func(walkFileInfosOptions *walkFileInfosOptions) {
		walkFileInfosOptions.onlyTargetFiles = true
	}
}

// ModuleReadBucketToStorageReadBucket converts the given ModuleReadBucket to a storage.ReadBucket.
//
// All target and non-target Files are added.
//
// TODO: Add an option to allow only target Files to be added, if we require such a function.
func ModuleReadBucketToStorageReadBucket(bucket ModuleReadBucket) storage.ReadBucket {
	return newStorageReadBucket(bucket)
}

// ModuleReadBucketWithOnlyFileTypes returns a new ModuleReadBucket that only contains the given
// FileTypes.
//
// Common use case is to get only the .proto files.
func ModuleReadBucketWithOnlyFileTypes(
	moduleReadBucket ModuleReadBucket,
	fileTypes ...FileType,
) ModuleReadBucket {
	return newFilteredModuleReadBucket(moduleReadBucket, fileTypes)
}

// ModuleReadBucketWithOnlyProtoFiles is a convenience function that returns a new
// ModuleReadBucket that only contains the .proto files.
func ModuleReadBucketWithOnlyProtoFiles(moduleReadBucket ModuleReadBucket) ModuleReadBucket {
	return ModuleReadBucketWithOnlyFileTypes(moduleReadBucket, FileTypeProto)
}

// GetFileInfos is a convenience function that walks the ModuleReadBucket and gets
// all the FileInfos.
//
// Sorted by path.
func GetFileInfos(ctx context.Context, moduleReadBucket ModuleReadBucket) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := moduleReadBucket.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			fileInfos = append(fileInfos, fileInfo)
			return nil
		},
	); err != nil {
		return nil, err
	}
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
	return fileInfos, nil
}

// GetTargetFileInfos is a convenience function that walks the ModuleReadBucket and gets
// all the FileInfos where IsTargetFile() is set to true.
//
// Sorted by path.
func GetTargetFileInfos(ctx context.Context, moduleReadBucket ModuleReadBucket) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := moduleReadBucket.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			fileInfos = append(fileInfos, fileInfo)
			return nil
		},
		WalkFileInfosWithOnlyTargetFiles(),
	); err != nil {
		return nil, err
	}
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
	return fileInfos, nil
}

// GetDocFile gets the singular documentation File for the Module, if it exists.
//
// When creating a Module from a Bucket, we check the file paths buf.md, README.md, and README.markdown
// to exist, in that order. The first one to exist is chosen as the documentation File that is considered
// part of the Module, and any others are discarded. This function will return that File that was chosen.
//
// Returns an error with fs.ErrNotExist if no documentation file exists.
func GetDocFile(ctx context.Context, moduleReadBucket ModuleReadBucket) (File, error) {
	if docFilePath := getDocFilePathForModuleReadBucket(ctx, moduleReadBucket); docFilePath != "" {
		return moduleReadBucket.GetFile(ctx, docFilePath)
	}
	return nil, fs.ErrNotExist
}

// GetLicenseFile gets the license File for the Module, if it exists.
//
// Returns an error with fs.ErrNotExist if the license File does not exist.
func GetLicenseFile(ctx context.Context, moduleReadBucket ModuleReadBucket) (File, error) {
	return moduleReadBucket.GetFile(ctx, licenseFilePath)
}

// *** PRIVATE ***

// moduleReadBucket

type moduleReadBucket struct {
	getBucket func() (storage.ReadBucket, error)
	module    Module
	// We have to store a deterministic ordering of targetPaths so that Walk
	// has the same iteration order every time. We could have a different iteration order,
	// as storage.ReadBucket.Walk doesn't guarantee any iteration order, but that seems wonky.
	targetPaths          []string
	targetPathMap        map[string]struct{}
	targetExcludePathMap map[string]struct{}
}

// module cannot be assumed to be functional yet.
// Do not call any functions on module.
func newModuleReadBucket(
	ctx context.Context,
	getBucket func() (storage.ReadBucket, error),
	module Module,
	targetPaths []string,
	targetExcludePaths []string,
) *moduleReadBucket {

	return &moduleReadBucket{
		getBucket: sync.OnceValues(
			func() (storage.ReadBucket, error) {
				bucket, err := getBucket()
				if err != nil {
					return nil, err
				}
				docFilePath := getDocFilePathForStorageReadBucket(ctx, bucket)
				return storage.MapReadBucket(
					bucket,
					storage.MatchOr(
						storage.MatchPathExt(".proto"),
						storage.MatchPathEqual(licenseFilePath),
						storage.MatchPathEqual(docFilePath),
					),
				), nil
			},
		),
		module:               module,
		targetPaths:          targetPaths,
		targetPathMap:        slicesextended.ToMap(targetPaths),
		targetExcludePathMap: slicesextended.ToMap(targetExcludePaths),
	}
}

func (b *moduleReadBucket) GetFile(ctx context.Context, path string) (File, error) {
	fileInfo, err := b.StatFileInfo(ctx, path)
	if err != nil {
		return nil, err
	}
	bucket, err := b.getBucket()
	if err != nil {
		return nil, err
	}
	readObjectCloser, err := bucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	return newFile(fileInfo, readObjectCloser), nil
}

func (b *moduleReadBucket) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	bucket, err := b.getBucket()
	if err != nil {
		return nil, err
	}
	objectInfo, err := bucket.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	return b.newFileInfo(objectInfo)
}

func (b *moduleReadBucket) WalkFileInfos(
	ctx context.Context,
	fn func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	walkFileInfosOptions := newWalkFileInfosOptions()
	for _, option := range options {
		option(walkFileInfosOptions)
	}
	bucket, err := b.getBucket()
	if err != nil {
		return err
	}
	walkFunc := func(objectInfo storage.ObjectInfo) error {
		fileInfo, err := b.newFileInfo(objectInfo)
		if err != nil {
			return err
		}
		if walkFileInfosOptions.onlyTargetFiles && !fileInfo.IsTargetFile() {
			return nil
		}
		return fn(fileInfo)
	}
	// If we have target paths, we do not want to walk to whole bucket.
	// For example, we do --path path/to/file.proto for googleapis, we don't want to
	// walk all of googleapis to find the single file.
	//
	// Instead, we walk the specific targets.
	// Note that storage.ReadBucket.Walk allows calling a file path as a prefix.
	//
	// Use targetPaths instead of targetPathMap to have a deterministic iteration order at this level.
	if len(b.targetPaths) > 0 {
		for _, targetPath := range b.targetPaths {
			if err := bucket.Walk(ctx, targetPath, walkFunc); err != nil {
				return err
			}
		}
		return nil
	}
	return bucket.Walk(ctx, "", walkFunc)
}

func (*moduleReadBucket) isModuleReadBucket() {}

func (b *moduleReadBucket) newFileInfo(objectInfo storage.ObjectInfo) (FileInfo, error) {
	fileType, err := classifyPathFileType(objectInfo.Path())
	if err != nil {
		// Given our matching in the constructor, all file paths should be classified.
		// A lack of classification is a system error.
		return nil, err
	}
	return newFileInfo(objectInfo, b.module, fileType, b.getIsTargetedFileForPath(objectInfo.Path())), nil
}

func (b *moduleReadBucket) getIsTargetedFileForPath(path string) bool {
	if !b.module.IsTargetModule() {
		// If the Module is not targeted, the file is automatically not targeted.
		//
		// Note we can change IsTargetModule via setIsTargetModule during ModuleSetBuilder building,
		// so we do not want to cache this value.
		return false
	}
	switch {
	case len(b.targetPathMap) == 0 && len(b.targetExcludePathMap) == 0:
		// If we did not target specific Files, all Files in a targeted Module are targeted.
		return true
	case len(b.targetPathMap) == 0 && len(b.targetExcludePathMap) != 0:
		// We only have exclude paths, no paths.
		return !normalpath.MapHasEqualOrContainingPath(b.targetExcludePathMap, path, normalpath.Relative)
	case len(b.targetPathMap) != 0 && len(b.targetExcludePathMap) == 0:
		// We only have paths, no exclude paths.
		return normalpath.MapHasEqualOrContainingPath(b.targetPathMap, path, normalpath.Relative)
	default:
		// We have both paths and exclude paths.
		return normalpath.MapHasEqualOrContainingPath(b.targetPathMap, path, normalpath.Relative) &&
			!normalpath.MapHasEqualOrContainingPath(b.targetExcludePathMap, path, normalpath.Relative)
	}
}

// filteredModuleReadBucket

type filteredModuleReadBucket struct {
	delegate    ModuleReadBucket
	fileTypeMap map[FileType]struct{}
}

func newFilteredModuleReadBucket(
	delegate ModuleReadBucket,
	fileTypes []FileType,
) *filteredModuleReadBucket {
	return &filteredModuleReadBucket{
		delegate:    delegate,
		fileTypeMap: fileTypeSliceToMap(fileTypes),
	}
}

func (f *filteredModuleReadBucket) GetFile(ctx context.Context, path string) (File, error) {
	// Stat'ing the filtered bucket, not the delegate.
	if _, err := f.StatFileInfo(ctx, path); err != nil {
		return nil, err
	}
	return f.delegate.GetFile(ctx, path)
}

func (f *filteredModuleReadBucket) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	fileInfo, err := f.delegate.StatFileInfo(ctx, path)
	if err != nil {
		return nil, err
	}
	if _, ok := f.fileTypeMap[fileInfo.FileType()]; !ok {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	return fileInfo, nil
}

func (f *filteredModuleReadBucket) WalkFileInfos(
	ctx context.Context,
	fn func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	return f.delegate.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if _, ok := f.fileTypeMap[fileInfo.FileType()]; !ok {
				return nil
			}
			return fn(fileInfo)
		},
		options...,
	)
}

func (*filteredModuleReadBucket) isModuleReadBucket() {}

// multiModuleReadBucket

type multiModuleReadBucket struct {
	delegates []ModuleReadBucket
}

func newMultiModuleReadBucket(
	delegates []ModuleReadBucket,
) *multiModuleReadBucket {
	return &multiModuleReadBucket{
		delegates: delegates,
	}
}

func (m *multiModuleReadBucket) GetFile(ctx context.Context, path string) (File, error) {
	_, delegateIndex, err := m.getFileInfoAndDelegateIndex(ctx, "read", path)
	if err != nil {
		return nil, err
	}
	return m.delegates[delegateIndex].GetFile(ctx, path)
}

func (m *multiModuleReadBucket) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	fileInfo, _, err := m.getFileInfoAndDelegateIndex(ctx, "stat", path)
	return fileInfo, err
}

func (m *multiModuleReadBucket) WalkFileInfos(
	ctx context.Context,
	fn func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	seenPathToFileInfo := make(map[string]FileInfo)
	for _, delegate := range m.delegates {
		if err := delegate.WalkFileInfos(
			ctx,
			func(fileInfo FileInfo) error {
				path := fileInfo.Path()
				if existingFileInfo, ok := seenPathToFileInfo[path]; ok {
					// This does not return all paths that are matching, unlike GetFile and StatFileInfo.
					// We do not want to continue iterating, as calling WalkFileInfos on the same path
					// could cause errors downstream as callers expect a single call per path.
					return newExistsMultipleModulesError(path, existingFileInfo, fileInfo)
				}
				seenPathToFileInfo[path] = fileInfo
				return fn(fileInfo)
			},
			options...,
		); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiModuleReadBucket) getFileInfoAndDelegateIndex(
	ctx context.Context,
	op string,
	path string,
) (FileInfo, int, error) {
	var fileInfos []FileInfo
	var delegateIndexes []int
	for i, delegate := range m.delegates {
		fileInfo, err := delegate.StatFileInfo(ctx, path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, 0, err
		}
		fileInfos = append(fileInfos, fileInfo)
		delegateIndexes = append(delegateIndexes, i)
	}
	switch len(fileInfos) {
	case 0:
		return nil, 0, &fs.PathError{Op: op, Path: path, Err: fs.ErrNotExist}
	case 1:
		return fileInfos[0], delegateIndexes[0], nil
	default:
		return nil, 0, newExistsMultipleModulesError(path, fileInfos...)
	}
}

func (*multiModuleReadBucket) isModuleReadBucket() {}

// storageReadBucket

type storageReadBucket struct {
	delegate ModuleReadBucket
}

func newStorageReadBucket(delegate ModuleReadBucket) *storageReadBucket {
	return &storageReadBucket{
		delegate: delegate,
	}
}

func (s *storageReadBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	return s.delegate.GetFile(ctx, path)
}

func (s *storageReadBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return s.delegate.StatFileInfo(ctx, path)
}

func (s *storageReadBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	return s.delegate.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			if !normalpath.EqualsOrContainsPath(prefix, fileInfo.Path(), normalpath.Relative) {
				return nil
			}
			return f(fileInfo)
		},
	)
}

func newExistsMultipleModulesError(path string, fileInfos ...FileInfo) error {
	return fmt.Errorf(
		"%s exists in multiple Modules: %v",
		path,
		strings.Join(
			slicesextended.Map(
				fileInfos,
				func(fileInfo FileInfo) string {
					return fileInfo.Module().OpaqueID()
				},
			),
			",",
		),
	)
}

type walkFileInfosOptions struct {
	onlyTargetFiles bool
}

func newWalkFileInfosOptions() *walkFileInfosOptions {
	return &walkFileInfosOptions{}
}
