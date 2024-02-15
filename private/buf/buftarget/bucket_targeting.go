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

package buftarget

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// TODO(doria): we need to add the following info:
// BufWorkYAMLDirPaths (for v1 vs. v2 workspaces)
// Document that everything here is relative paths

// BucketTargeting provides the path to the controllng workspace, target paths, and target
// exclude paths mapped to the bucket.
type BucketTargeting interface {
	// ControllingWorkspace returns the path for the controlling workspace relative to the root of the bucket.
	ControllingWorkspacePath() string
	// InputPath returns the input path relative to the root fo the bucket
	// TODO(doria): should the input path be treated differently from all other target paths?
	InputPath() string
	// TargetPaths returns the target paths relative to the root of the bucket.
	TargetPaths() []string
	// TargetExcludePaths returns the target exclude paths relative to the root of the bucket.
	TargetExcludePaths() []string
	// Terminated returns whether the controlling workspace was found through the terminateFunc.
	// TODO(doria): should be be able to kill this.
	Terminated() bool

	isBucketTargeting()
}

func NewBucketTargeting(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	inputPath string,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc TerminateFunc, // TODO(doria): move that out of buffetch
) (BucketTargeting, error) {
	return newBucketTargeting(ctx, logger, bucket, inputPath, targetPaths, targetExcludePaths, terminateFunc)
}

// *** PRIVATE ***

var (
	_ BucketTargeting = &bucketTargeting{}
)

type bucketTargeting struct {
	controllingWorkspacePath string
	inputPath                string
	targetPaths              []string
	targetExcludePaths       []string
	terminated               bool
}

func (b *bucketTargeting) ControllingWorkspacePath() string {
	return b.controllingWorkspacePath
}

func (b *bucketTargeting) InputPath() string {
	return b.inputPath
}

func (b *bucketTargeting) TargetPaths() []string {
	return b.targetPaths
}

func (b *bucketTargeting) TargetExcludePaths() []string {
	return b.targetExcludePaths
}

func (b *bucketTargeting) Terminated() bool {
	return b.terminated
}

func (b *bucketTargeting) isBucketTargeting() {
}

func newBucketTargeting(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	inputPath string,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc TerminateFunc,
) (*bucketTargeting, error) {
	// First we map the controlling workspace for the inputPath.
	controllingWorkspacePath, mappedInputPath, terminated, err := mapControllingWorkspaceAndPath(
		ctx,
		logger,
		bucket,
		inputPath,
		terminateFunc,
	)
	if err != nil {
		return nil, err
	}
	mappedTargetPaths := make([]string, len(targetPaths))
	// Then we do the same for each target path. If the target paths resolve to different
	// controlling workspaces, then we return an error.
	// We currently do not compile nested workspaces, but this algorithm lets us potentially
	// handle nested workspaces in the future.
	// TODO(doria): this shouldn't have a big impact on performance, right?
	// TODO(doria): do we need to handle that there was a termination through the paths? maybe.
	for i, targetPath := range targetPaths {
		controllingWorkspacePathForTargetPath, mappedPath, _, err := mapControllingWorkspaceAndPath(
			ctx,
			logger,
			bucket,
			targetPath,
			terminateFunc,
		)
		if err != nil {
			return nil, err
		}
		if controllingWorkspacePathForTargetPath != controllingWorkspacePath {
			return nil, fmt.Errorf("more than one workspace resolved for given paths: %q, %q", controllingWorkspacePathForTargetPath, controllingWorkspacePath)
		}
		mappedTargetPaths[i] = mappedPath
	}
	// NOTE: we do not map excluded paths to their own workspaces -- we use the controlling
	// workspace we resolved through our input path and target paths. If an excluded path does
	// not exist, we do not validate this.
	mappedTargetExcludePaths := make([]string, len(targetExcludePaths))
	for i, targetExcludePath := range targetExcludePaths {
		mappedTargetExcludePath, err := normalpath.Rel(controllingWorkspacePath, targetExcludePath)
		if err != nil {
			return nil, err
		}
		mappedTargetExcludePaths[i] = mappedTargetExcludePath
	}
	return &bucketTargeting{
		controllingWorkspacePath: controllingWorkspacePath,
		inputPath:                mappedInputPath,
		targetPaths:              mappedTargetPaths,
		targetExcludePaths:       mappedTargetExcludePaths,
		terminated:               terminated,
	}, nil
}

// mapControllingWorkspaceAndPath takes a bucket, path, and terminate func and returns the
// controlling workspace path and mapped path.
func mapControllingWorkspaceAndPath(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	path string,
	terminateFunc TerminateFunc,
) (string, string, bool, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return "", "", false, err
	}
	// If no terminateFunc is passed, we can simply assume that we are mapping the bucket at
	// the path.
	if terminateFunc == nil {
		return path, ".", false, nil
	}
	// We can't do this in a traditional loop like this:
	//
	// for curDirPath := path; curDirPath != "."; curDirPath = normalpath.Dir(curDirPath) {
	//
	// If we do that, then we don't run terminateFunc for ".", which we want to so that we get
	// the correct value for the terminate bool.
	//
	// Instead, we effectively do a do-while loop.
	curDirPath := path
	for {
		terminate, err := terminateFunc(ctx, bucket, curDirPath, path)
		if err != nil {
			return "", "", false, err
		}
		if terminate {
			logger.Debug(
				"buffetch termination found",
				zap.String("curDirPath", curDirPath),
				zap.String("path", path),
			)
			subDirPath, err := normalpath.Rel(curDirPath, path)
			if err != nil {
				return "", "", false, err
			}
			return curDirPath, subDirPath, true, nil
		}
		if curDirPath == "." {
			break
		}
		curDirPath = normalpath.Dir(curDirPath)
	}
	logger.Debug(
		"buffetch no termination found",
		zap.String("path", path),
	)
	return curDirPath, ".", false, nil
}
