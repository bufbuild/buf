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

var (
	// PrefixContainsWorkspaceFile returns true if the bucket contains a "workspace file"
	// at the prefix.
	//
	// A workspace file roots a Workspace. It is either a buf.work.yaml or buf.work file,
	// or a v2 buf.yaml file.
	//
	// This is used by buffetch when searching for the root of the workspace.
	PrefixContainsWorkspaceFile = prefixContainsWorkspaceFile
	// PrefixContainsModuleFile returns true if the bucket contains a "module file"
	// at the prefix.
	//
	// A module file roots a Module. It is either a v1 or v1beta1 buf.yaml or buf.mod file,
	// or a v2 buf.yaml file that has a module with directory ".".
	//
	// This is used by buffetch when searching for the root of the module when dealing with ProtoFileRefs.
	PrefixContainsModuleFile = prefixContainsModuleFile
)

func prefixContainsWorkspaceFile(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalSubDirPath string,
) (bool, error) {
	bufWorkYAMLFile, err := GetBufWorkYAMLFileForPrefix(ctx, bucket, prefix)
	bufWorkYAMLFileExists := err == nil
	if bufWorkYAMLFileExists {
		relSubDirPath, err := normalpath.Rel(prefix, originalSubDirPath)
		if err != nil {
			return false, err
		}
		// If the buf.work.yaml contains the subDirPath, then we have found a workspace file for this subdirectory.
		if _, ok := slicesext.ToStructMap(bufWorkYAMLFile.DirPaths())[relSubDirPath]; ok {
			return true, nil
		}
	}
	fileVersion, err := GetBufYAMLFileVersionForPrefix(ctx, bucket, prefix)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if bufWorkYAMLFileExists {
		// Not a great way to surface this to the user.
		return false, fmt.Errorf("cannot have buf.work.yaml and buf.yaml in the same directory %q", prefix)
	}
	return fileVersion == FileVersionV2, nil
}

func prefixContainsModuleFile(
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
