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

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

// UpdateableWorkspace is a workspace that can be updated.
//
// A workspace can be updated if it was backed by a v2 buf.yaml, or a single, targeted, local
// Module from defaults or a v1beta1/v1 buf.yaml exists in the Workspace. Config overrides
// can also not be used, but this is enforced via the WorkspaceBucketOption/UpdateableWorkspaceBucketOption
// difference.
//
// buf.work.yamls are ignored when constructing an UpdateableWorkspace.
type UpdateableWorkspace interface {
	Workspace

	// BufLockFileDigestType returns the DigestType that the buf.lock file expects.
	BufLockFileDigestType() bufmodule.DigestType
	// ExisingBufLockFileDepModuleKeys returns the ModuleKeys from the updateable buf.lock file.
	//
	// We use this in a convoluted way - once we do the update, we attempt to rebuild the Workspace. If the build
	// fails, we try to revert the buf.lock file.
	//
	// This could be refactored to be much better - in a perfect world, we'd update the buf.lock file virtually,
	// do a rebuild, and only actually write to disk if we succeeded. See buf mod prune for more details.
	ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error)
	// UpdateBufLockFile updates the lock file that backs this Workspace to contain exactly
	// the given ModuleKeys.
	//
	// If a buf.lock does not exist, one will be created.
	UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey) error

	isUpdateableWorkspace()
}

// NewUpdateableWorkspaceForBucket returns a new Workspace for the given Bucket.
//
// If the workspace is not updateable, an error is returned.
func NewUpdateableWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadWriteBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...UpdateableWorkspaceBucketOption,
) (UpdateableWorkspace, error) {
	return newUpdateableWorkspaceForBucket(ctx, logger, tracer, bucket, clientProvider, moduleDataProvider, commitProvider, options...)
}

// *** PRIVATE ***

type updateableWorkspace struct {
	*workspace

	bucket storage.ReadWriteBucket
}

func newUpdateableWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadWriteBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...UpdateableWorkspaceBucketOption,
) (*updateableWorkspace, error) {
	workspaceBucketOptions := make([]WorkspaceBucketOption, 0, len(options)+1)
	for _, option := range options {
		workspaceBucketOptions = append(workspaceBucketOptions, option)
	}
	workspaceBucketOptions = append(workspaceBucketOptions, withIgnoreAndDisallowV1BufWorkYAMLs())
	workspace, err := newWorkspaceForBucket(ctx, logger, tracer, bucket, clientProvider, moduleDataProvider, commitProvider, workspaceBucketOptions...)
	if err != nil {
		return nil, err
	}
	if !workspace.createdFromBucket {
		// Something really bad would have to happen for this to happen.
		return nil, syserror.New("workspace.createdFromBucket not set for a workspace created from newUpdateableWorkspaceForBucket")
	}
	if workspace.updateableBufLockDirPath == "" {
		// This means we messed up in our building of the Workspace.
		return nil, syserror.New("workspace.updateableBufLockDirPath not set for a workspace created from newUpdateableWorkspaceForBucket")
	}
	return &updateableWorkspace{
		workspace: workspace,
		bucket:    bucket,
	}, nil
}

func (w *updateableWorkspace) BufLockFileDigestType() bufmodule.DigestType {
	if w.isV2 {
		return bufmodule.DigestTypeB5
	}
	return bufmodule.DigestTypeB4
}

func (w *updateableWorkspace) ExistingBufLockFileDepModuleKeys(ctx context.Context) ([]bufmodule.ModuleKey, error) {
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, w.bucket, w.updateableBufLockDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return bufLockFile.DepModuleKeys(), nil
}

func (w *updateableWorkspace) UpdateBufLockFile(ctx context.Context, depModuleKeys []bufmodule.ModuleKey) error {
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

func (*updateableWorkspace) isUpdateableWorkspace() {}
