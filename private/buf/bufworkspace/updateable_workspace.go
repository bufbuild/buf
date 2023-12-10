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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

type updateableWorkspace struct {
	*workspace

	bucket storage.WriteBucket
}

func newUpdateableWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (*updateableWorkspace, error) {
	workspace, err := newWorkspaceForBucket(ctx, logger, bucket, moduleDataProvider, options...)
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
