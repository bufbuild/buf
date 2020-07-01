// Copyright 2020 Buf Technologies, Inc.
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

package bufmod

import (
	"context"
	"sort"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

// Builder builds modules.
type Builder interface {
	// BuildForBucket builds a module for the given bucket.
	//
	// If bucketRelPaths is empty, all files are built.
	// bucketRelPaths should be relative to the bucket, not the roots.
	BuildForBucket(
		ctx context.Context,
		readBucket storage.ReadBucket,
		config *Config,
		options ...BuildForBucketOption,
	) (bufcore.Module, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(logger *zap.Logger) Builder {
	return newBuilder(logger)
}

// BuildForBucketOption is an option for BuildForBucket.
type BuildForBucketOption func(*buildForBucketOptions)

// WithBucketRelPaths returns a new BuildForBucketOption that specifies specific file paths to build.
//
// These paths must exist.
// These paths must be relative to the bucket.
// These paths will be normalized and validated.
// Multiple calls to this option will override previous calls.
func WithBucketRelPaths(bucketRelPaths ...string) BuildForBucketOption {
	return func(buildForBucketOptions *buildForBucketOptions) {
		buildForBucketOptions.bucketRelPaths = bucketRelPaths
	}
}

// WithBucketRelPathsAllowNotExistOnWalk returns a BuildForBucketOption that says that the
// bucket relative paths specified with WithBucketRelPaths may not exist on module TargetFileInfos
// calls.
//
// GetFileInfo and GetFile will still operate as normal.
func WithBucketRelPathsAllowNotExistOnWalk() BuildForBucketOption {
	return func(buildForBucketOptions *buildForBucketOptions) {
		buildForBucketOptions.bucketRelPathsAllowNotExistOnWalk = true
	}
}

// Config is a configuration for build.
type Config struct {
	// RootToExcludes contains a map from root to the excludes for that root.
	//
	// Roots are the root directories within a bucket to search for Protobuf files.
	//
	// There will be no between the roots, ie foo/bar and foo are not allowed.
	// All Protobuf files must be unique relative to the roots, ie if foo and bar
	// are roots, then foo/baz.proto and bar/baz.proto are not allowed.
	//
	// All roots will be normalized and validated.
	//
	// Excludes are the directories within a bucket to exclude.
	//
	// There should be no overlap between the excludes, ie foo/bar and foo are not allowed.
	//
	// All excludes must reside within a root, but none will be equal to a root.
	// All excludes will be normalized and validated.
	// The excludes in this map will be relative to the root they map to!
	//
	// If RootToExcludes is empty, the default is "." with no excludes.
	RootToExcludes map[string][]string
}

// NewConfig returns a new, validated Config for the ExternalConfig.
func NewConfig(externalConfig ExternalConfig) (*Config, error) {
	return newConfig(externalConfig)
}

// Roots returns the Roots.
func (c *Config) Roots() []string {
	roots := make([]string, 0, len(c.RootToExcludes))
	for root := range c.RootToExcludes {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots
}

// ExternalConfig is an external config.
type ExternalConfig struct {
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}

// ResolveExternalFilePaths resolves the filePaths.
//func ResolveExternalFilePaths(roots []string, filePaths ...string) ([]string, error) {
//// TODO: likely move this to our own handling
//return protoparse.ResolveFilenames(roots, filePaths...)
//}
