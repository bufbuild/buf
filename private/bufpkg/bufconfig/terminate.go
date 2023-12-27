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

package bufconfig

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// FindControllingWorkspaceResult is a result from FindControllingWorkspace.
type FindControllingWorkspaceResult interface {
	// If true, a controlling workspace was found.
	Found() bool
	// If non-empty, this indicates that the found workspace was a buf.work.yaml,
	// and the directory paths were these paths.
	//
	// These paths include the input prefix, and are the full paths to the directories
	// within the input bucket.
	//
	// If empty, this indicates that the found workspace was a buf.yaml.
	BufWorkYAMLDirPaths() []string

	isFindControllingWorkspaceResult()
}

// FindControllingWorkspace searches for a workspace file at prefix that controls originalSubDirPath.
//
// # A workspace file is either a buf.work.yaml file or a v2 buf.yaml file
//
// The workspace file controls originalSubDirPath if either:
//  1. prefix == originalSubDirPath, that is we're just directly targeting originalSubDirPath.
//  3. The workspace file refers to the originalSubDirPath via "directories" in buf.work.yaml or "directory" in buf.yaml.
//
// This is used by both buffetch/internal.Reader via the Prefix functions, and NewWorkspaceForBucket,
// which do their own independent searches and do not depend on each other.
func FindControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (FindControllingWorkspaceResult, error) {
	return findControllingWorkspace(ctx, bucket, prefix, originalSubDirPath, true)
}

// TerminateForNonProtoFileRef returns true if the bucket contains a workspace file
// that controls originalSubDirPath at the prefix.
//
// See the commentary on FindControllingWorkspace.
//
// This is used by buffetch when searching for the root of the workspace.
// See buffetch/internal.WithGetBucketTerminateFunc for more information.
func TerminateAtControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (bool, error) {
	findControllingWorkspaceResult, err := findControllingWorkspace(ctx, bucket, prefix, originalSubDirPath, true)
	if err != nil {
		return false, err
	}
	return findControllingWorkspaceResult.Found(), nil
}

// TerminateAtEnclosingModuleOrWorkspaceForProtoFileRef returns true if the bucket contains
// either a module file or workspace file at the prefix.
//
// A module file is either a v1 or v1beta1 buf.yaml or buf.mod file, or a v2 buf.yaml file that
// has a module with directory ".". This is configuration for a module rooted at this directory.
//
// A workspace file is either a buf.work.yaml file or a v2 buf.yaml file.
//
// As opposed to TerminateForNonProtoFileRef, this does not require the prefix to point to the
// originalSubDirPath - ProtoFileRefs assume that if you have a workspace file, it controls the ProtoFileRef.
//
// This is used by buffetch when searching for the root of the module when dealing with ProtoFileRefs.
// See buffetch/internal.WithGetBucketProtoFileTerminateFunc for more information.
func TerminateAtEnclosingModuleOrWorkspaceForProtoFileRef(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (bool, error) {
	findControllingWorkspaceResult, err := findControllingWorkspace(ctx, bucket, prefix, originalSubDirPath, false)
	if err != nil {
		return false, err
	}
	if findControllingWorkspaceResult.Found() {
		return true, nil
	}

	bufYAMLFile, err := GetBufYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1, FileVersionV1Beta1:
		// If we have a v1 or v1beta1 buf.yaml, we automatically know this is a module file.
		return true, nil
	case FileVersionV2:
		// If we have a v2, this is only a "module file" if it contains a module with directory ".".
		// Otherwise, we don't want to stop here. Remember, this only wants to stop at module files,
		// not workspace files.
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			if moduleConfig.DirPath() == "." {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

// *** PRIVATE ***

type findControllingWorkspaceResult struct {
	found               bool
	bufWorkYAMLDirPaths []string
}

func newFindControllingWorkspaceResult(
	found bool,
	bufWorkYAMLDirPaths []string,
) *findControllingWorkspaceResult {
	return &findControllingWorkspaceResult{
		found:               found,
		bufWorkYAMLDirPaths: bufWorkYAMLDirPaths,
	}
}

func (f *findControllingWorkspaceResult) Found() bool {
	return f.found
}

func (f *findControllingWorkspaceResult) BufWorkYAMLDirPaths() []string {
	return f.bufWorkYAMLDirPaths
}

func (*findControllingWorkspaceResult) isFindControllingWorkspaceResult() {}

// findControllingWorkspace adds the property that the prefix may not be required to point to originalSubDirPath.
//
// We don't require the workspace file to point to originalSubDirPath when finding the enclosing module or
// workspace for a ProtoFileRef.
func findControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
	requirePrefixWorkspaceToPointToOriginalSubDirPath bool,
) (FindControllingWorkspaceResult, error) {
	bufWorkYAMLFile, err := GetBufWorkYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufWorkYAMLExists := err == nil
	bufYAMLFile, err := GetBufYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufYAMLExists := err == nil
	if bufWorkYAMLExists && bufYAMLExists {
		// This isn't actually the external directory path, but we do the best we can here for now.
		return nil, fmt.Errorf("cannot have a buf.work.yaml and buf.yaml in the same directory %q", prefix)
	}

	// Find the relative path of our original target subDirPath vs where we currently are.
	// We only stop the loop if a v2 buf.yaml or a buf.work.yaml lists this directory,
	// or if the original target subDirPath points ot the workspace file itself.
	//
	// Example: we inputted foo/bar/baz, we're currently at foo. We want to make sure
	// that buf.work.yaml lists bar/baz as a directory. If so, this buf.work.yaml
	// relates to our current directory.
	//
	// Example: we inputted foo/bar/baz, we're at foo/bar/baz. Great.
	relDirPath, err := normalpath.Rel(prefix, originalSubDirPath)
	if err != nil {
		return nil, err
	}
	if bufYAMLExists && bufYAMLFile.FileVersion() == FileVersionV2 {
		if !requirePrefixWorkspaceToPointToOriginalSubDirPath {
			// We don't require the workspace to point to the prefix (likely because we're
			// finding the controlling workspace for a ProtoFileRef), we're good to go.
			return newFindControllingWorkspaceResult(true, nil), nil
		}
		if prefix == originalSubDirPath {
			// We've referred to our workspace file directly, we're good to go.
			return newFindControllingWorkspaceResult(true, nil), nil
		}
		dirPathMap := make(map[string]struct{})
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			dirPathMap[moduleConfig.DirPath()] = struct{}{}
		}
		if _, ok := dirPathMap[relDirPath]; ok {
			// This workspace file refers to curDirPath, we're good to go.
			return newFindControllingWorkspaceResult(true, nil), nil
		}
	}
	if bufWorkYAMLExists {
		_, refersToCurDirPath := slicesext.ToStructMap(bufWorkYAMLFile.DirPaths())[relDirPath]
		if prefix == originalSubDirPath || refersToCurDirPath || !requirePrefixWorkspaceToPointToOriginalSubDirPath {
			// We don't actually need to parse the buf.work.yaml again - we have all the information
			// we need. Just figure out the actual paths within the bucket of the modules, and go
			// right to newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1.
			moduleDirPaths := make([]string, len(bufWorkYAMLFile.DirPaths()))
			for i, dirPath := range bufWorkYAMLFile.DirPaths() {
				moduleDirPaths[i] = normalpath.Join(prefix, dirPath)
			}
			return newFindControllingWorkspaceResult(true, moduleDirPaths), nil
		}
	}
	return newFindControllingWorkspaceResult(false, nil), nil
}
