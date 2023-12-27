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

	"github.com/bufbuild/buf/private/bufpkg/bufprotocompile"
	"github.com/bufbuild/buf/private/pkg/cache"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/protocompile/parser/fastscan"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
	//
	// A ModuleReadBucket directly derived from a Module will always have at least one .proto file.
	// If this is not the case, WalkFileInfos will return an error when called.
	WalkFileInfos(ctx context.Context, f func(FileInfo) error, options ...WalkFileInfosOption) error

	// ShouldBeSelfContained returns true if the ModuleReadBucket was constructed with the intention
	// that it would be self-contained with respect to its .proto files. That is, every .proto
	// file in the ModuleReadBucket only imports other files from the ModuleReadBucket.
	//
	// It is possible for a bucket to be marked as ShouldBeSelfContained without it actually
	// being self-contained.
	//
	// A ModuleReadBucket is self-contained if it was constructed from
	// ModuleSetToModuleReadBucketWithOnlyProtoFiles or
	// ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles.
	//
	// A ModuleReadBucket as inherited from a Module is not self-contained.
	//
	// A ModuleReadBucket filtered to anything but FileTypeProto is not self-contained.
	ShouldBeSelfContained() bool

	// getFastscanResultForPath gets the fastscan.Result for the File path of a File within the ModuleReadBucket.
	//
	// This should only be used by Modules and FileInfos.
	//
	// returns errIsWKT if the filePath is a WKT.
	// returns an error with fs.ErrNotExist if the file is not found.
	getFastscanResultForPath(ctx context.Context, path string) (fastscan.Result, error)

	isModuleReadBucket()
}

// WalkFileInfosOption is an option for WalkFileInfos
type WalkFileInfosOption func(*walkFileInfosOptions)

// WalkFileInfosWithOnlyTargetFiles returns a new WalkFileInfosOption that only walks the target files.
func WalkFileInfosWithOnlyTargetFiles() WalkFileInfosOption {
	return func(walkFileInfosOptions *walkFileInfosOptions) {
		walkFileInfosOptions.onlyTargetFiles = true
	}
}

// ModuleReadBucketToStorageReadBucket converts the given ModuleReadBucket to a storage.ReadBucket.
//
// All Files (whether targets or non-targets) are added.
func ModuleReadBucketToStorageReadBucket(bucket ModuleReadBucket) storage.ReadBucket {
	return newStorageReadBucket(bucket)
}

// ModuleReadBucketWithOnlyTargetFiles returns a new ModuleReadBucket that only contains
// target Files.
func ModuleReadBucketWithOnlyTargetFiles(moduleReadBucket ModuleReadBucket) ModuleReadBucket {
	return newTargetedModuleReadBucket(moduleReadBucket)
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

// GetFilePaths is a convenience function that gets all the target and non-target
// file paths for the ModuleReadBucket.
//
// Sorted.
func GetFilePaths(ctx context.Context, moduleReadBucket ModuleReadBucket) ([]string, error) {
	fileInfos, err := GetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return nil, err
	}
	return slicesext.Map(fileInfos, func(fileInfo FileInfo) string { return fileInfo.Path() }), nil
}

// GetTargetFilePaths is a convenience function that gets all the target
// file paths for the ModuleReadBucket.
//
// Sorted.
func GetTargetFilePaths(ctx context.Context, moduleReadBucket ModuleReadBucket) ([]string, error) {
	fileInfos, err := GetTargetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return nil, err
	}
	return slicesext.Map(fileInfos, func(fileInfo FileInfo) string { return fileInfo.Path() }), nil
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

// GetDocStorageReadBucket gets a storage.ReadBucket that just contains the documentation file(s).
//
// This is needed for i.e. using RootToExcludes in NewWorkspaceForBucket.
func GetDocStorageReadBucket(ctx context.Context, bucket storage.ReadBucket) storage.ReadBucket {
	return storage.MapReadBucket(
		bucket,
		storage.MatchPathEqual(getDocFilePathForStorageReadBucket(ctx, bucket)),
	)
}

// GetLicenseStorageReadBucket gets a storage.ReadBucket that just contains the license file(s).
//
// This is needed for i.e. using RootToExcludes in NewWorkspaceForBucket.
func GetLicenseStorageReadBucket(bucket storage.ReadBucket) storage.ReadBucket {
	return storage.MapReadBucket(
		bucket,
		storage.MatchPathEqual(licenseFilePath),
	)
}

// *** PRIVATE ***

// moduleReadBucket

type moduleReadBucket struct {
	logger    *zap.Logger
	getBucket func() (storage.ReadBucket, error)
	module    Module
	// We have to store a deterministic ordering of targetPaths so that Walk
	// has the same iteration order every time. We could have a different iteration order,
	// as storage.ReadBucket.Walk doesn't guarantee any iteration order, but that seems wonky.
	targetPaths          []string
	targetPathMap        map[string]struct{}
	targetExcludePathMap map[string]struct{}
	protoFileTargetPath  string
	includePackageFiles  bool

	pathToFileInfoCache       cache.Cache[string, FileInfo]
	pathToFastscanResultCache cache.Cache[string, fastscan.Result]
}

// module cannot be assumed to be functional yet.
// Do not call any functions on module.
func newModuleReadBucketForModule(
	ctx context.Context,
	logger *zap.Logger,
	// This function must already be filtered to include only module files and must be sync.OnceValues wrapped!
	syncOnceValuesGetBucketWithStorageMatcherApplied func() (storage.ReadBucket, error),
	module Module,
	targetPaths []string,
	targetExcludePaths []string,
	protoFileTargetPath string,
	includePackageFiles bool,
) (*moduleReadBucket, error) {
	// TODO: get these validations into a common place
	if protoFileTargetPath != "" && (len(targetPaths) > 0 || len(targetExcludePaths) > 0) {
		return nil, syserror.Newf("cannot set both protoFileTargetPath %q and either targetPaths %v or targetExcludePaths %v", protoFileTargetPath, targetPaths, targetExcludePaths)
	}
	if protoFileTargetPath != "" && normalpath.Ext(protoFileTargetPath) != ".proto" {
		return nil, syserror.Newf("protoFileTargetPath %q is not a .proto file", protoFileTargetPath)
	}
	return &moduleReadBucket{
		logger:               logger,
		getBucket:            syncOnceValuesGetBucketWithStorageMatcherApplied,
		module:               module,
		targetPaths:          targetPaths,
		targetPathMap:        slicesext.ToStructMap(targetPaths),
		targetExcludePathMap: slicesext.ToStructMap(targetExcludePaths),
		protoFileTargetPath:  protoFileTargetPath,
		includePackageFiles:  includePackageFiles,
	}, nil
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
	return b.getFileInfo(ctx, objectInfo)
}

func (b *moduleReadBucket) WalkFileInfos(
	ctx context.Context,
	fn func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	// Note that we must verify that at least one file in this ModuleReadBucket is
	// a .proto file, per the documentation on Module.
	protoFileTracker := newProtoFileTracker(b.module)

	walkFileInfosOptions := newWalkFileInfosOptions()
	for _, option := range options {
		option(walkFileInfosOptions)
	}
	bucket, err := b.getBucket()
	if err != nil {
		return err
	}

	if !walkFileInfosOptions.onlyTargetFiles {
		if err := bucket.Walk(
			ctx,
			"",
			func(objectInfo storage.ObjectInfo) error {
				fileInfo, err := b.getFileInfo(ctx, objectInfo)
				if err != nil {
					return err
				}
				protoFileTracker.track(fileInfo)
				return fn(fileInfo)
			},
		); err != nil {
			return err
		}
		return protoFileTracker.validate()
	}

	targetFileWalkFunc := func(objectInfo storage.ObjectInfo) error {
		fileInfo, err := b.getFileInfo(ctx, objectInfo)
		if err != nil {
			return err
		}
		protoFileTracker.track(fileInfo)
		if !fileInfo.IsTargetFile() {
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
			// Still need to determine IsTargetFile as a file could be excluded with excludeTargetPaths.
			if err := bucket.Walk(ctx, targetPath, targetFileWalkFunc); err != nil {
				return err
			}
		}
		// We can't determine if the Module had any .proto file paths, as we only walked
		// the target paths. We don't return any value from protoFileTracker.validate().
		return nil
	}
	if err := bucket.Walk(ctx, "", targetFileWalkFunc); err != nil {
		return err
	}
	return protoFileTracker.validate()
}

func (b *moduleReadBucket) withModule(module Module) *moduleReadBucket {
	// We want to avoid sync.OnceValueing getBucket Twice, so we have a special copy function here
	// instead of calling newModuleReadBucket.
	//
	// This technically doesn't matter anymore since we don't sync.OnceValue getBucket inside newModuleReadBucket
	// anymore, but we keep this around in case we change that back.
	return &moduleReadBucket{
		logger:               b.logger,
		getBucket:            b.getBucket,
		module:               module,
		targetPaths:          b.targetPaths,
		targetPathMap:        b.targetPathMap,
		targetExcludePathMap: b.targetExcludePathMap,
		protoFileTargetPath:  b.protoFileTargetPath,
		includePackageFiles:  b.includePackageFiles,
	}
}

func (*moduleReadBucket) isModuleReadBucket() {}

func (b *moduleReadBucket) getFileInfo(ctx context.Context, objectInfo storage.ObjectInfo) (FileInfo, error) {
	return b.pathToFileInfoCache.GetOrAdd(
		// We know that storage.ObjectInfo will always have the same values for the same
		// ObjectInfo returned from a common bucket, this is documented. Therefore, we
		// can cache based on just the path.
		objectInfo.Path(),
		func() (FileInfo, error) {
			return b.getFileInfoUncached(ctx, objectInfo)
		},
	)
}

func (b *moduleReadBucket) getFileInfoUncached(ctx context.Context, objectInfo storage.ObjectInfo) (FileInfo, error) {
	fileType, err := classifyPathFileType(objectInfo.Path())
	if err != nil {
		// Given our matching in the constructor, all file paths should be classified.
		// A lack of classification is a system error.
		return nil, syserror.Wrap(err)
	}
	isTargetFile, err := b.getIsTargetFileForPathUncached(ctx, objectInfo.Path())
	if err != nil {
		return nil, err
	}
	return newFileInfo(
		objectInfo,
		b.module,
		fileType,
		isTargetFile,
		func() ([]string, error) {
			if fileType != FileTypeProto {
				return nil, nil
			}
			fastscanResult, err := b.getFastscanResultForPath(ctx, objectInfo.Path())
			if err != nil {
				return nil, err
			}
			// This also has the effect of copying the slice.
			return slicesext.ToUniqueSorted(slicesext.Map(fastscanResult.Imports, func(imp fastscan.Import) string { return imp.Path })), nil
		},
		func() (string, error) {
			if fileType != FileTypeProto {
				return "", nil
			}
			fastscanResult, err := b.getFastscanResultForPath(ctx, objectInfo.Path())
			if err != nil {
				return "", err
			}
			return fastscanResult.PackageName, nil
		},
	), nil
}

func (b *moduleReadBucket) getIsTargetFileForPathUncached(ctx context.Context, path string) (bool, error) {
	if !b.module.IsTarget() {
		// If the Module is not targeted, the file is automatically not targeted.
		//
		// Note we can change IsTarget via setIsTarget during ModuleSetBuilder building,
		// so we do not want to cache this value.
		return false, nil
	}
	// We already validate that we don't set this alongside targetPaths and targetExcludePaths
	if b.protoFileTargetPath != "" {
		fileType, err := classifyPathFileType(path)
		if err != nil {
			return false, err
		}
		if fileType != FileTypeProto {
			// We are targeting a .proto file and this file is not a .proto file, therefore
			// this file is not targeted.
			return false, nil
		}
		isProtoFileTargetPath := path == b.protoFileTargetPath
		if isProtoFileTargetPath {
			// Regardless of includePackageFiles, we always return true.
			return true, nil
		}
		if !b.includePackageFiles {
			// If we don't include package files, then we don't have a match, return false.
			return false, nil
		}
		// We now need to see if we have the same package as the protoFileTargetPath file.
		//
		// We've now deferred having to get fastscan.Results as much as we can.
		protoFileTargetFastscanResult, err := b.getFastscanResultForPath(ctx, b.protoFileTargetPath)
		if err != nil {
			return false, err
		}
		if protoFileTargetFastscanResult.PackageName == "" {
			// Don't do anything if the target file does not have a package.
			return false, nil
		}
		fastscanResult, err := b.getFastscanResultForPath(ctx, path)
		if err != nil {
			return false, err
		}
		// If the package is the same, this is a target.
		return protoFileTargetFastscanResult.PackageName == fastscanResult.PackageName, nil
	}
	switch {
	case len(b.targetPathMap) == 0 && len(b.targetExcludePathMap) == 0:
		// If we did not target specific Files, all Files in a targeted Module are targeted.
		return true, nil
	case len(b.targetPathMap) == 0 && len(b.targetExcludePathMap) != 0:
		// We only have exclude paths, no paths.
		return !normalpath.MapHasEqualOrContainingPath(b.targetExcludePathMap, path, normalpath.Relative), nil
	case len(b.targetPathMap) != 0 && len(b.targetExcludePathMap) == 0:
		// We only have paths, no exclude paths.
		return normalpath.MapHasEqualOrContainingPath(b.targetPathMap, path, normalpath.Relative), nil
	default:
		// We have both paths and exclude paths.
		return normalpath.MapHasEqualOrContainingPath(b.targetPathMap, path, normalpath.Relative) &&
			!normalpath.MapHasEqualOrContainingPath(b.targetExcludePathMap, path, normalpath.Relative), nil
	}
}

// Only will work for .proto files.
func (b *moduleReadBucket) getFastscanResultForPath(ctx context.Context, path string) (fastscan.Result, error) {
	return b.pathToFastscanResultCache.GetOrAdd(
		path,
		func() (fastscan.Result, error) {
			return b.getFastscanResultForPathUncached(ctx, path)
		},
	)
}

func (b *moduleReadBucket) getFastscanResultForPathUncached(
	ctx context.Context,
	path string,
) (fastscanResult fastscan.Result, retErr error) {
	fileType, err := classifyPathFileType(path)
	if err != nil {
		return fastscan.Result{}, err
	}
	if fileType != FileTypeProto {
		// We should have validated this WAY before.
		return fastscan.Result{}, syserror.Newf("cannot get fastscan.Result for non-proto file %q", path)
	}
	// We *cannot* use GetFile here, because getFileInfo -> getFastscanResultForPath -> getFileInfo,
	// and this causes a circular wait with the cache locks.
	bucket, err := b.getBucket()
	if err != nil {
		return fastscan.Result{}, err
	}
	readObjectCloser, err := bucket.Get(ctx, path)
	if err != nil {
		return fastscan.Result{}, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	fastscanResult, err = fastscan.Scan(path, readObjectCloser)
	if err != nil {
		var syntaxError fastscan.SyntaxError
		if errors.As(err, &syntaxError) {
			fileAnnotationSet, err := bufprotocompile.FileAnnotationSetForErrorsWithPos(
				syntaxError,
				bufprotocompile.WithExternalPathResolver(
					func(path string) string {
						fileInfo, err := bucket.Stat(ctx, path)
						if err != nil {
							return path
						}
						return fileInfo.ExternalPath()
					},
				),
			)
			if err != nil {
				return fastscan.Result{}, err
			}
			return fastscan.Result{}, fileAnnotationSet
		}
		return fastscan.Result{}, err
	}
	return fastscanResult, nil
}

// targetedModuleReadBucket

type targetedModuleReadBucket struct {
	delegate ModuleReadBucket
}

func newTargetedModuleReadBucket(delegate ModuleReadBucket) *targetedModuleReadBucket {
	return &targetedModuleReadBucket{
		delegate: delegate,
	}
}

func (t *targetedModuleReadBucket) GetFile(ctx context.Context, path string) (File, error) {
	// Stat'ing the targeted bucket, not the delegate.
	if _, err := t.StatFileInfo(ctx, path); err != nil {
		return nil, err
	}
	return t.delegate.GetFile(ctx, path)
}

func (t *targetedModuleReadBucket) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	fileInfo, err := t.delegate.StatFileInfo(ctx, path)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsTargetFile() {
		return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	return fileInfo, nil
}

func (t *targetedModuleReadBucket) WalkFileInfos(
	ctx context.Context,
	fn func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	return t.delegate.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			return fn(fileInfo)
		},
		append(
			options,
			WalkFileInfosWithOnlyTargetFiles(),
		)...,
	)
}

func (*targetedModuleReadBucket) ShouldBeSelfContained() bool {
	// We've filtered out non-target files, this should not be considered self-contained.
	return false
}

func (t *targetedModuleReadBucket) getFastscanResultForPath(ctx context.Context, path string) (fastscan.Result, error) {
	if _, err := t.StatFileInfo(ctx, path); err != nil {
		return fastscan.Result{}, err
	}
	return t.delegate.getFastscanResultForPath(ctx, path)
}

func (*targetedModuleReadBucket) isModuleReadBucket() {}

// filteredModuleReadBucket

type filteredModuleReadBucket struct {
	delegate              ModuleReadBucket
	fileTypeMap           map[FileType]struct{}
	shouldBeSelfContained bool
}

func newFilteredModuleReadBucket(
	delegate ModuleReadBucket,
	fileTypes []FileType,
) *filteredModuleReadBucket {
	fileTypeMap := slicesext.ToStructMap(fileTypes)
	_, containsFileTypeProto := fileTypeMap[FileTypeProto]
	return &filteredModuleReadBucket{
		delegate:              delegate,
		fileTypeMap:           fileTypeMap,
		shouldBeSelfContained: delegate.ShouldBeSelfContained() && containsFileTypeProto,
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

func (f *filteredModuleReadBucket) ShouldBeSelfContained() bool {
	return f.shouldBeSelfContained
}

func (f *filteredModuleReadBucket) getFastscanResultForPath(ctx context.Context, path string) (fastscan.Result, error) {
	if _, err := f.StatFileInfo(ctx, path); err != nil {
		return fastscan.Result{}, err
	}
	return f.delegate.getFastscanResultForPath(ctx, path)
}

func (*filteredModuleReadBucket) isModuleReadBucket() {}

// multiModuleReadBucket

type multiModuleReadBucket struct {
	delegates             []ModuleReadBucket
	shouldBeSelfContained bool
}

func newMultiModuleReadBucket(
	delegates []ModuleReadBucket,
	shouldBeSelfContained bool,
) *multiModuleReadBucket {
	return &multiModuleReadBucket{
		delegates:             delegates,
		shouldBeSelfContained: shouldBeSelfContained,
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

func (m *multiModuleReadBucket) ShouldBeSelfContained() bool {
	return m.shouldBeSelfContained
}

func (m *multiModuleReadBucket) getFastscanResultForPath(ctx context.Context, path string) (fastscan.Result, error) {
	_, delegateIndex, err := m.getFileInfoAndDelegateIndex(ctx, "stat", path)
	if err != nil {
		return fastscan.Result{}, err
	}
	return m.delegates[delegateIndex].getFastscanResultForPath(ctx, path)
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
	prefix, err := normalpath.NormalizeAndValidate(prefix)
	if err != nil {
		return err
	}
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
		"%s exists in multiple locations: %v",
		path,
		strings.Join(
			slicesext.Map(
				fileInfos,
				func(fileInfo FileInfo) string {
					return fileInfo.ExternalPath()
				},
			),
			" ",
		),
	)
}

// protoFileTracker tracks if we found a .proto file for WalkFileInfos.
//
// This allows us to fulfill the documentation for ModuleReadBucket on Module where at least
// one .proto file will exist in a ModuleReadBucket.
type protoFileTracker struct {
	module Module
	found  bool
}

func newProtoFileTracker(module Module) *protoFileTracker {
	return &protoFileTracker{
		module: module,
		found:  false,
	}
}

func (t *protoFileTracker) track(fileInfo FileInfo) {
	if fileInfo.FileType() == FileTypeProto {
		t.found = true
	}
}

func (t *protoFileTracker) validate() error {
	if !t.found {
		// Prefer BucketID over OpaqueID as the user will understand this better.
		id := t.module.BucketID()
		if id == "" {
			id = t.module.OpaqueID()
		}
		return newErrNoProtoFiles(id)
	}
	return nil
}

func (b *moduleReadBucket) ShouldBeSelfContained() bool {
	return false
}

type walkFileInfosOptions struct {
	onlyTargetFiles bool
}

func newWalkFileInfosOptions() *walkFileInfosOptions {
	return &walkFileInfosOptions{}
}
