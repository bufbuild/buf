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

package bufworkspace

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// UpdateableWorkspace is a workspace that can be updated.
type UpdateableWorkspace interface {
	Workspace

	// PutBufLockFile updates the lock file that backs this Workspace.
	//
	// If a buf.lock does not exist, one will be created.
	//
	// This will fail for UpdateableWorkspaces not created from v2 buf.yamls.
	PutBufLockFile(ctx context.Context, bufLockFile bufconfig.BufLockFile) error

	isUpdateableWorkspace()
}

// NewUpdateableWorkspaceForBucket returns a new Workspace for the given Bucket.
//
// All parsing of configuration files is done behind the scenes here.
// This function can only read v2 buf.yamls.
func NewUpdateableWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadWriteBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (UpdateableWorkspace, error) {
	return newUpdateableWorkspaceForBucket(ctx, bucket, moduleDataProvider, options...)
}

// *** PRIVATE ***

type updateableWorkspace struct {
	*workspace

	bucket storage.WriteBucket
}

func newUpdateableWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadWriteBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (*updateableWorkspace, error) {
	workspace, err := newWorkspaceForBucket(ctx, bucket, moduleDataProvider, options...)
	if err != nil {
		return nil, err
	}
	return &updateableWorkspace{
		workspace: workspace,
		bucket:    bucket,
	}, nil
}

func (w *updateableWorkspace) PutBufLockFile(ctx context.Context, bufLockFile bufconfig.BufLockFile) error {
	if !w.isV2BufYAMLWorkspace {
		// TODO: better error message
		return errors.New(`migrate to v2 buf.yaml via "buf migrate" to update your buf.lock file`)
	}
	if bufLockFile.FileVersion() != bufconfig.FileVersionV2 {
		// TODO: better error message
		// This is kind of a system error.
		return errors.New(`can only update to v2 buf.locks`)
	}
	return bufconfig.PutBufLockFileForPrefix(ctx, w.bucket, w.bufLockDirPath, bufLockFile)
}

func (*updateableWorkspace) isUpdateableWorkspace() {}
