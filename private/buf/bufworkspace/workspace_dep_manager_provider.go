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
	ctx, span := w.tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	config, err := newWorkspaceDepManagerConfig(options)
	if err != nil {
		return nil, err
	}

	findControllingWorkspaceResult, err := bufconfig.FindControllingWorkspace(ctx, bucket, ".", config.targetSubDirPath)
	if err != nil {
		return nil, err
	}
	if findControllingWorkspaceResult.Found() {
		// We have a v1 buf.work.yaml, per the documentation on bufconfig.FindControllingWorkspace.
		if bufWorkYAMLDirPaths := findControllingWorkspaceResult.BufWorkYAMLDirPaths(); len(bufWorkYAMLDirPaths) > 0 {
			// config.targetSubDirPath is normalized, so if it was empty, it will be ".".
			if config.targetSubDirPath == "." {
				// If config.targetSubDirPath is ".", this means we targeted a buf.work.yaml, not an individual module within the buf.work.yaml
				// This is disallowed.
				return nil, errors.New(`Workspaces defined with buf.work.yaml cannot be updated or pushed, only
the individual modules within a workspace can be updated or pushed. Workspaces
defined with a v2 buf.yaml can be updated, see the migration documentation for more details.`)
			}
			// We targeted a specific module within the workspace. Based on the option we provided, we're going to ignore
			// the workspace entirely, and just act as if the buf.work.yaml did not exist.
			w.logger.Debug(
				"creating new workspace dep manager, ignoring v1 buf.work.yaml, just building on module at target",
				zap.String("targetSubDirPath", config.targetSubDirPath),
			)
			return newWorkspaceDepManager(bucket, config.targetSubDirPath, false), nil
		}
		w.logger.Debug(
			"creating new workspace dep manager based on v2 buf.yaml",
			zap.String("targetSubDirPath", config.targetSubDirPath),
		)
		// We have a v2 buf.yaml.
		return newWorkspaceDepManager(bucket, ".", true), nil
	}

	w.logger.Debug(
		"creating new workspace dep manager with no found buf.work.yaml or buf.yaml",
		zap.String("targetSubDirPath", config.targetSubDirPath),
	)
	// We did not find any buf.work.yaml or buf.yaml, operate as if a
	// default v1 buf.yaml was at config.targetSubDirPath.
	return newWorkspaceDepManager(bucket, config.targetSubDirPath, false), nil
}
