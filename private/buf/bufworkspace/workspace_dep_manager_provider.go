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

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

// WorkspaceDepManagerProvider provides WorkspaceDepManagers.
type WorkspaceDepManagerProvider interface {
	// GetWorkspaceDepManagerForBucket returns a new WorkspaceDepManager for the given Bucket.
	//
	// If the workspace is not updateable, an error is returned.
	//
	// If the underlying bucket has a v2 buf.yaml at the root, this builds a WorkspaceDepManager for this buf.yaml,
	// using TargetSubDirPath for targeting.
	//
	// Otherwise, this builds a Workspace with a single module at the TargetSubDirPath (which may be "."), igoring buf.work.yamls.
	// Directories with buf.work.yamls cannot be directly targeted.

	// Note this is the same logic as if WithIgnoreAndDisallowV1BufWorkYAMLs is applied with WorkspaceProvider!! If you want an equivalent
	// Workspace, you need to use this option!
	//
	// All parsing of configuration files is done behind the scenes here.
	GetWorkspaceDepManager(
		ctx context.Context,
		bucket storage.ReadWriteBucket,
		bucketTargeting buftarget.BucketTargeting,
	) (WorkspaceDepManager, error)
}

// NewWorkspaceDepManagerProvider returns a new WorkspaceDepManagerProvider.
func NewWorkspaceDepManagerProvider(
	logger *zap.Logger,
	tracer tracing.Tracer,
) WorkspaceDepManagerProvider {
	return newWorkspaceDepManagerProvider(
		logger,
		tracer,
	)
}

// *** PRIVATE ***

type workspaceDepManagerProvider struct {
	logger *zap.Logger
	tracer tracing.Tracer
}

func newWorkspaceDepManagerProvider(
	logger *zap.Logger,
	tracer tracing.Tracer,
) *workspaceDepManagerProvider {
	return &workspaceDepManagerProvider{
		logger: logger,
		tracer: tracer,
	}
}

func (w *workspaceDepManagerProvider) GetWorkspaceDepManager(
	ctx context.Context,
	bucket storage.ReadWriteBucket,
	bucketTargeting buftarget.BucketTargeting,
) (_ WorkspaceDepManager, retErr error) {
	controllingWorkspace := bucketTargeting.ControllingWorkspace()
	if controllingWorkspace != nil && controllingWorkspace.BufYAMLFile() != nil {
		// A v2 workspace was found, but we make sure
		bufYAMLFile := controllingWorkspace.BufYAMLFile()
		if bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
			return newWorkspaceDepManager(bucket, controllingWorkspace.Path(), true), nil
		}
	}
	// Otherwise we simply ignore any buf.work.yaml that was found and attempt to build
	// a v1 module at the SubDirPath
	return newWorkspaceDepManager(bucket, bucketTargeting.SubDirPath(), false), nil
}
