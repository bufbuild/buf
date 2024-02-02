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

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
		options ...WorkspaceDepManagerOption,
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
	options ...WorkspaceDepManagerOption,
) (_ WorkspaceDepManager, retErr error) {
	config, err := newWorkspaceDepManagerConfig(options)
	if err != nil {
		return nil, err
	}
	workspaceTargeting, err := newWorkspaceTargeting(
		ctx,
		w.logger,
		bucket,
		config.targetSubDirPath,
		nil,
		true, // Disallow buf.work.yamls when doing management of deps for v1
	)
	if err != nil {
		return nil, err
	}
	if workspaceTargeting.isV2() {
		return newWorkspaceDepManager(bucket, workspaceTargeting.v2DirPath, true), nil
	}
	if len(workspaceTargeting.v1DirPaths) != 1 {
		// This is because of disallowing buf.work.yamls
		return nil, syserror.Newf("expected a single v1 dir path from workspace targeting but got %v", workspaceTargeting.v1DirPaths)
	}
	return newWorkspaceDepManager(bucket, workspaceTargeting.v1DirPaths[0], false), nil
}
