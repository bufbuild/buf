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

package bufworkspace

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// WorkspaceDepManager is a workspace that can be updated.
//
// A workspace can be updated if it was backed by a v2 buf.yaml, or a single, targeted, local
// Module from defaults or a v1beta1/v1 buf.yaml exists in the Workspace. Config overrides
// can also not be used, but this is enforced via the WorkspaceBucketOption/WorkspaceDepManagerBucketOption
// difference.
//
// buf.work.yamls are ignored when constructing an WorkspaceDepManager.
type WorkspaceDepManager interface {
	// BufLockFileDigestType returns the DigestType that the buf.lock file expects.
	BufLockFileDigestType() bufmodule.DigestType
	// ExisingBufLockFileDepModuleKeys returns the ModuleKeys from the buf.lock file.
	ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error)
	// UpdateBufLockFile updates the lock file that backs the Workspace to contain exactly
	// the given ModuleKeys.
	//
	// If a buf.lock does not exist, one will be created.
	UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey) error
	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// The ModuleRefs in this list will be unique by ModuleFullName. If there are two ModuleRefs
	// in the buf.yaml with the same ModuleFullName but different Refs, an error will be given
	// at workspace constructions. For example, with v1 buf.yaml, this is a union of the deps in
	// the buf.yaml files in the workspace. If different buf.yamls had different refs, an error
	// will be returned - we have no way to resolve what the user intended.
	//
	// Sorted.
	ConfiguredDepModuleRefs(ctx context.Context) ([]bufmodule.ModuleRef, error)

	isWorkspaceDepManager()
}

// *** PRIVATE ***

type workspaceDepManager struct {
	bucket storage.ReadWriteBucket
	// targetSubDirPath is the relative path within the bucket where a buf.yaml file should be and where a
	// buf.lock can be written.
	//
	// If isV2 is true, this will be "." - buf.yamls and buf.locks live at the root of the workspace.
	//
	// If isV2 is false, this will be the path to the single, local, targeted Module within the workspace
	// This is the only situation where we can do an update for a v1 buf.lock.
	targetSubDirPath string
	// If true, the workspace was created from v2 buf.yamls
	//
	// If false, the workspace was created from defaults, or v1beta1/v1 buf.yamls.
	//
	// This is used to determine what DigestType to use, and what version
	// of buf.lock to write.
	isV2 bool
}

func newWorkspaceDepManager(
	bucket storage.ReadWriteBucket,
	targetSubDirPath string,
	isV2 bool,
) *workspaceDepManager {
	return &workspaceDepManager{
		bucket:           bucket,
		targetSubDirPath: targetSubDirPath,
		isV2:             isV2,
	}
}

func (w *workspaceDepManager) ConfiguredDepModuleRefs(ctx context.Context) ([]bufmodule.ModuleRef, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if bufYAMLFile == nil {
		return nil, nil
	}
	switch fileVersion := bufYAMLFile.FileVersion(); fileVersion {
	case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
		if w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v1beta1, v1", w.targetSubDirPath, fileVersion)
		}
	case bufconfig.FileVersionV2:
		if !w.isV2 {
			return nil, syserror.Newf("buf.yaml at %q did had version %v but expected v12", w.targetSubDirPath, fileVersion)
		}
	default:
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
	return bufYAMLFile.ConfiguredDepModuleRefs(), nil
}

func (w *workspaceDepManager) BufLockFileDigestType() bufmodule.DigestType {
	if w.isV2 {
		return bufmodule.DigestTypeB5
	}
	return bufmodule.DigestTypeB4
}

func (w *workspaceDepManager) ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.DepModuleKeys(), nil
}

func (w *workspaceDepManager) UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey) error {
	var bufLockFile bufconfig.BufLockFile
	var err error
	if w.isV2 {
		bufLockFile, err = bufconfig.NewBufLockFile(bufconfig.FileVersionV2, depModuleKeys)
		if err != nil {
			return err
		}
	} else {
		fileVersion := bufconfig.FileVersionV1
		existingBufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, w.bucket, w.targetSubDirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		} else {
			fileVersion = existingBufYAMLFile.FileVersion()
		}
		bufLockFile, err = bufconfig.NewBufLockFile(fileVersion, depModuleKeys)
		if err != nil {
			return err
		}
	}
	return bufconfig.PutBufLockFileForPrefix(ctx, w.bucket, w.targetSubDirPath, bufLockFile)
}

func (*workspaceDepManager) isWorkspaceDepManager() {}
