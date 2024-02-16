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

package buftarget

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// TerminateFunc is a termination function.
type TerminateFunc func(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error)

// TerminateAtControllingWorkspace implements a TerminateFunc and returns controlling workspace
// if one is found at the given prefix.
//
// All paths must be relative.
func TerminateAtControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	return terminateAtControllingWorkspace(ctx, bucket, prefix, originalInputPath)
}

// *** PRIVATE ***

func terminateAtControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	bufWorkYAMLFile, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufWorkYAMLExists := err == nil
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufYAMLExists := err == nil
	if bufWorkYAMLExists && bufYAMLExists {
		// This isn't actually the external directory path, but we do the best we can here for now.
		return nil, fmt.Errorf("cannot have a buf.work.yaml and buf.yaml in the same directory %q", prefix)
	}
	if bufYAMLExists && bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
		// We don't require the workspace to point to the prefix (likely because we're
		// finding the controlling workspace for a ProtoFileRef), we're good to go.
		return newControllingWorkspace(prefix, nil, bufYAMLFile), nil
	}
	if bufWorkYAMLExists {
		// For v1 workspaces, we ensure that the module paths lists actually contain the the
		// original input paths. If not, then we do not use this workspace.
		relDirPath, err := normalpath.Rel(prefix, originalInputPath)
		if err != nil {
			return nil, err
		}
		_, refersToCurDirPath := slicesext.ToStructMap(bufWorkYAMLFile.DirPaths())[relDirPath]
		if prefix == originalInputPath || refersToCurDirPath {
			return newControllingWorkspace(prefix, bufWorkYAMLFile, nil), nil
		}
	}
	return nil, nil
}
