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

// TODO: this doesn't work for ProtoFileRefs. You don't want to require the originalSubDirPath,
// which is just the directory of the .proto file, to be pointed to by the workspace. You want
// to bypass this requirement. Solution is to completely separate terminateFunc and protoFileTerminateFunc,
// and be more lenient on the controlling workspace for ProtoFileRefs to not require the directory to
// be pointed to...probably?

// FindControllingWorkspace searches for a workspace file at prefix that controls originalSubDirPath.
// A workspace file is either a buf.work.yaml file or a v2 buf.yaml file, and the file controls
// originalSubDirPath if either (1) we are directly targeting the workspace file, i.e prefix == originalSubDirPath,
// or (2) the workspace file refers to the config.subDirPath. If we find a controlling workspace
// file, we use this to build our workspace. If we don't, return nil.
//
// This is used by both buffetch/internal.Reader via PrefixContainsWorkspaceFile and NewWorkspaceForBucket,
// which do their own independent searches and do not depend on each other.
func FindControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
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
		if prefix == originalSubDirPath {
			// We've referred to our workspace file directy, we're good to go.
			return newFindControllingWorkspaceResult(true, nil), nil
		}
		dirPathMap := make(map[string]struct{})
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			dirPathMap[moduleConfig.DirPath()] = struct{}{}
		}
		if _, ok := dirPathMap[relDirPath]; ok {
			// This workspace file refers to curDurPath, we're good to go.
			return newFindControllingWorkspaceResult(true, nil), nil
		}
	}
	if bufWorkYAMLExists {
		_, refersToCurDirPath := slicesext.ToStructMap(bufWorkYAMLFile.DirPaths())[relDirPath]
		if prefix == originalSubDirPath || refersToCurDirPath {
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

// PrefixContainsWorkspaceFile returns true if the bucket contains a "workspace file"
// that controls originalSubDirPath at the prefix.
//
// A workspace file roots a Workspace. It is either a buf.work.yaml or buf.work file,
// or a v2 buf.yaml file.
//
// This is used by buffetch when searching for the root of the workspace.
func PrefixContainsWorkspaceFile(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (bool, error) {
	findControllingWorkspaceResult, err := FindControllingWorkspace(ctx, bucket, prefix, originalSubDirPath)
	if err != nil {
		return false, err
	}
	return findControllingWorkspaceResult.Found(), nil
}

// PrefixContainsModuleFile returns true if the bucket contains a "module file"
// at the prefix.
//
// A module file roots a Module. It is either a v1 or v1beta1 buf.yaml or buf.mod file,
// or a v2 buf.yaml file that has a module with directory ".".
//
// This is used by buffetch when searching for the root of the module when dealing with ProtoFileRefs.
func PrefixContainsModuleFile(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (bool, error) {
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
