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

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

// BucketTargeting provides targeting information for the bucket based on any controlling
// workspaces that have been found.
type BucketTargeting interface {
	// ControllingWorkpsace returns the information for the controlling workspace, if one was
	// found. If not found, then this will be nil.
	ControllingWorkspace() ControllingWorkspace
	// SubDirPath returns the input directory relative to the controlling workspace, if one
	// was found, otherwise it is relative to the root of the bucket
	SubDirPath() string
	// TargetPaths returns the target paths relative to the controlling workpsace, if one was
	// found, otherwise it is relative to the root of the bucket.
	TargetPaths() []string
	// TargetExcludePaths returns the target exclude paths relative to the controlling
	// workspace, if one is found, otherwise it is relative to the root of the bucket.
	TargetExcludePaths() []string

	isBucketTargeting()
}

// NewBucketTargeting returns new targeting information for the given bucket, input dir,
// target paths, and target exclude paths.
//
// The subDirPath, targetPaths, and targetExcludePaths are expected to be relative to the
// root of the bucket.
//
// If a controlling workspace is found, the input dir, target paths, and target exclude
// paths are mapped as relative paths to the controlling workspace path.
// Otherwise, the input dir, target paths, and target exclude paths are returned as-is.
func NewBucketTargeting(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	subDirPath string,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc TerminateFunc,
) (BucketTargeting, error) {
	return newBucketTargeting(
		ctx,
		logger,
		bucket,
		subDirPath,
		targetPaths,
		targetExcludePaths,
		terminateFunc,
	)
}

// *** PRIVATE ***

var _ BucketTargeting = &bucketTargeting{}

type bucketTargeting struct {
	controllingWorkspace ControllingWorkspace
	subDirPath           string
	targetPaths          []string
	targetExcludePaths   []string
}

func newBucketTargeting(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	subDirPath string,
	targetPaths []string,
	targetExcludePaths []string,
	terminateFunc TerminateFunc,
) (*bucketTargeting, error) {
	// First we map the controlling workspace for the subDirpath.
	controllingWorkspace, mappedInputDir, err := mapControllingWorkspaceAndPath(
		ctx,
		logger,
		bucket,
		subDirPath,
		terminateFunc,
	)
	if err != nil {
		return nil, err
	}
	// If no controlling workspace was found, we map the target paths and target exclude
	// paths to the input dir.
	mappedTargetPaths := targetPaths
	mappedTargetExcludePaths := targetExcludePaths
	if controllingWorkspace != nil && controllingWorkspace.Path() != "." {
		// If a controlling workspace was found, we map the paths to the controlling workspace
		// because we'll be working with a workspace bucket.
		for i, targetPath := range targetPaths {
			mappedTargetPath, err := normalpath.Rel(controllingWorkspace.Path(), targetPath)
			if err != nil {
				return nil, err
			}
			mappedTargetPath, err = normalpath.NormalizeAndValidate(mappedTargetPath)
			if err != nil {
				return nil, err
			}
			mappedTargetPaths[i] = mappedTargetPath
		}
		for i, targetExcludePath := range targetExcludePaths {
			mappedTargetExcludePath, err := normalpath.Rel(controllingWorkspace.Path(), targetExcludePath)
			if err != nil {
				return nil, err
			}
			mappedTargetExcludePath, err = normalpath.NormalizeAndValidate(mappedTargetExcludePath)
			if err != nil {
				return nil, err
			}
			mappedTargetExcludePaths[i] = mappedTargetExcludePath
		}
	}
	return &bucketTargeting{
		controllingWorkspace: controllingWorkspace,
		subDirPath:           mappedInputDir,
		targetPaths:          mappedTargetPaths,
		targetExcludePaths:   mappedTargetExcludePaths,
	}, nil
}

func (b *bucketTargeting) ControllingWorkspace() ControllingWorkspace {
	return b.controllingWorkspace
}

func (b *bucketTargeting) SubDirPath() string {
	return b.subDirPath
}

func (b *bucketTargeting) TargetPaths() []string {
	return b.targetPaths
}

func (b *bucketTargeting) TargetExcludePaths() []string {
	return b.targetExcludePaths
}

func (*bucketTargeting) isBucketTargeting() {}

// mapControllingWorkspaceAndPath takes a bucket, path, and terminate func and returns the
// controlling workspace and mapped path.
func mapControllingWorkspaceAndPath(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	path string,
	terminateFunc TerminateFunc,
) (ControllingWorkspace, string, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, "", err
	}
	// If no terminateFunc is passed, we can simply assume that we are mapping the bucket at
	// the path.
	if terminateFunc == nil {
		return nil, path, nil
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
		controllingWorkspace, err := terminateFunc(ctx, bucket, curDirPath, path)
		if err != nil {
			return nil, "", err
		}
		if controllingWorkspace != nil {
			logger.Debug(
				"buffetch termination found",
				zap.String("curDirPath", curDirPath),
				zap.String("path", path),
			)
			subDirPath, err := normalpath.Rel(curDirPath, path)
			if err != nil {
				return nil, "", err
			}
			return controllingWorkspace, subDirPath, nil
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
	// No controlling workspace is found, we simply return the input dir
	return nil, path, nil
}
