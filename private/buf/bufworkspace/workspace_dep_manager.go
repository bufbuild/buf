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
	HasConfiguredDepModuleRefs

	// BufLockFileDigestType returns the DigestType that the buf.lock file expects.
	BufLockFileDigestType() bufmodule.DigestType
	// ExisingBufLockFileDepModuleKeys returns the ModuleKeys from the buf.lock file.
	ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error)
	// UpdateBufLockFile updates the lock file that backs the Workspace to contain exactly
	// the given ModuleKeys.
	//
	// If a buf.lock does not exist, one will be created.
	UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey) error

	isWorkspaceDepManager()
}

// *** PRIVATE ***

type workspaceDepManager struct {
	bucket                  storage.ReadWriteBucket
	configuredDepModuleRefs []bufmodule.ModuleRef
	// If true, the workspace was created from v2 buf.yamls
	//
	// If false, the workspace was created from defaults, or v1beta1/v1 buf.yamls.
	//
	// This is used to determine what DigestType to use, and what version
	// of buf.lock to write.
	isV2 bool
	// updateableBufLockDirPath is the relative path within the bucket where a buf.lock can be written.
	//
	// If isV2 is true, this will be "." if no config overrides were used - buf.locks live at the root of the workspace.
	// If isV2 is false, this will be the path to the single, local, targeted Module within the workspace if no config
	// overrides were used. This is the only situation where we can do an update for a v1 buf.lock.
	// If isV2 is false and there is not a single, local, targeted Module, or a config override was used, this will be empty.
	//
	// The option WithIgnoreAndDisallowV1BufWorkYAMLs is used by updateabeWorkspace to try
	// to satisfy the v1 condition.
	updateableBufLockDirPath string
}

func newWorkspaceDepManager(
	bucket storage.ReadWriteBucket,
	configuredDepModuleRefs []bufmodule.ModuleRef,
	isV2 bool,
	updateableBufLockDirPath string,
) *workspaceDepManager {
	return &workspaceDepManager{
		bucket:                   bucket,
		configuredDepModuleRefs:  configuredDepModuleRefs,
		isV2:                     isV2,
		updateableBufLockDirPath: updateableBufLockDirPath,
	}
}

func (w *workspaceDepManager) ConfiguredDepModuleRefs() []bufmodule.ModuleRef {
	return w.configuredDepModuleRefs
}

func (w *workspaceDepManager) BufLockFileDigestType() bufmodule.DigestType {
	if w.isV2 {
		return bufmodule.DigestTypeB5
	}
	return bufmodule.DigestTypeB4
}

func (w *workspaceDepManager) ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.updateableBufLockDirPath)
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
		// This means that v1beta1 buf.yamls may be paired with v1 buf.locks, but that's probably OK?
		// TODO: Verify
		bufLockFile, err = bufconfig.NewBufLockFile(bufconfig.FileVersionV1, depModuleKeys)
		if err != nil {
			return err
		}
	}
	return bufconfig.PutBufLockFileForPrefix(ctx, w.bucket, w.updateableBufLockDirPath, bufLockFile)
}

func (*workspaceDepManager) isWorkspaceDepManager() {}

func (*workspaceDepManager) isHasConfiguredDepModuleRefs() {}
