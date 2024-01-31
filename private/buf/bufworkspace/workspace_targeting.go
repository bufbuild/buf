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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// workspaceTargeting figures out what directories to target and at what version for both
// the WorkpaceProvider and WorkspaceDepManagerProvider.
type workspaceTargeting struct {
	// v1DirPaths are the target v1 directory paths that should contain Modules.
	//
	// This is set if we are within a buf.work.yaml, if we directly targeted a v1 buf.yaml, or
	// if no configuration was found (in which case we default to v1).
	v1DirPaths []string
	// v2DirPath is the path to the v2 buf.yaml.
	v2DirPath string
}

func newWorkspaceTargeting(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	targetSubDirPath string,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
	ignoreAndDisallowV1BufWorkYAMLs bool,
) (*workspaceTargeting, error) {
	targetSubDirPath, err := normalpath.NormalizeAndValidate(targetSubDirPath)
	if err != nil {
		return nil, err
	}
	if overrideBufYAMLFile != nil {
		logger.Debug(
			"targeting workspace with config override",
			zap.String("targetSubDirPath", targetSubDirPath),
		)
		switch fileVersion := overrideBufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			// Operate as if there was no buf.work.yaml, only a v1 buf.yaml at the specified
			// targetSubDirPath, specifying a single module.
			return &workspaceTargeting{
				v1DirPaths: []string{targetSubDirPath},
			}, nil
		case bufconfig.FileVersionV2:
			// Operate as if there was a v2 buf.yaml at the target sub directory path.
			return &workspaceTargeting{
				v2DirPath: targetSubDirPath,
			}, nil
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}

	findControllingWorkspaceResult, err := bufconfig.FindControllingWorkspace(ctx, bucket, ".", targetSubDirPath)
	if err != nil {
		return nil, err
	}
	if findControllingWorkspaceResult.Found() {
		// We have a v1 buf.work.yaml, per the documentation on bufconfig.FindControllingWorkspace.
		if bufWorkYAMLDirPaths := findControllingWorkspaceResult.BufWorkYAMLDirPaths(); len(bufWorkYAMLDirPaths) > 0 {
			if ignoreAndDisallowV1BufWorkYAMLs {
				// targetSubDirPath is normalized, so if it was empty, it will be ".".
				if targetSubDirPath == "." {
					// If config.targetSubDirPath is ".", this means we targeted a buf.work.yaml, not an individual module within the buf.work.yaml
					// This is disallowed.
					return nil, errors.New(`Workspaces defined with buf.work.yaml cannot be updated or pushed, only
the individual modules within a workspace can be updated or pushed. Workspaces
defined with a v2 buf.yaml can be updated, see the migration documentation for more details.`)
				}
				// We targeted a specific module within the workspace. Based on the option we provided, we're going to ignore
				// the workspace entirely, and just act as if the buf.work.yaml did not exist.
				logger.Debug(
					"targeting workspace, ignoring v1 buf.work.yaml, just building on module at target",
					zap.String("targetSubDirPath", targetSubDirPath),
				)
				return &workspaceTargeting{
					v1DirPaths: []string{targetSubDirPath},
				}, nil
			}
			logger.Debug(
				"targeting workspace based on v1 buf.work.yaml",
				zap.String("targetSubDirPath", targetSubDirPath),
				zap.Strings("bufWorkYAMLDirPaths", bufWorkYAMLDirPaths),
			)
			return &workspaceTargeting{
				v1DirPaths: bufWorkYAMLDirPaths,
			}, nil
		}
		logger.Debug(
			"targeting workspace based on v2 buf.yaml",
			zap.String("targetSubDirPath", targetSubDirPath),
		)
		// We have a v2 buf.yaml.
		return &workspaceTargeting{
			v2DirPath: ".",
		}, nil
	}

	logger.Debug(
		"targeting workspace with no found buf.work.yaml or buf.yaml",
		zap.String("targetSubDirPath", targetSubDirPath),
	)
	// We did not find any buf.work.yaml or buf.yaml, operate as if a
	// default v1 buf.yaml was at config.targetSubDirPath.
	return &workspaceTargeting{
		v1DirPaths: []string{targetSubDirPath},
	}, nil
}

func (w *workspaceTargeting) isV2() bool {
	return w.v2DirPath != ""
}
