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

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
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
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadWriteBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...WorkspaceBucketOption,
) (UpdateableWorkspace, error) {
	return newUpdateableWorkspaceForBucket(ctx, logger, tracer, bucket, clientProvider, moduleDataProvider, commitProvider, options...)
}

// *** PRIVATE ***

type updateableWorkspace struct {
	*workspace

	bucket storage.WriteBucket
}

func newUpdateableWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadWriteBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...WorkspaceBucketOption,
) (*updateableWorkspace, error) {
	workspace, err := newWorkspaceForBucket(ctx, logger, tracer, bucket, clientProvider, moduleDataProvider, commitProvider, options...)
	if err != nil {
		return nil, err
	}
	return &updateableWorkspace{
		workspace: workspace,
		bucket:    bucket,
	}, nil
}

func (w *updateableWorkspace) PutBufLockFile(ctx context.Context, bufLockFile bufconfig.BufLockFile) error {
	if !w.isV2 {
		// TODO: enable for v1beta1/v1
		return errors.New(`migrate to v2 buf.yaml via "buf migrate" to update your buf.lock file`)
	}
	if bufLockFile.FileVersion() != bufconfig.FileVersionV2 {
		// TODO: enable for v1beta1/v1
		return errors.New(`can only update to v2 buf.locks`)
	}
	// TODO: make it so that v2 files only do b5 digests
	return bufconfig.PutBufLockFileForPrefix(ctx, w.bucket, ".", bufLockFile)
}

func (*updateableWorkspace) isUpdateableWorkspace() {}
