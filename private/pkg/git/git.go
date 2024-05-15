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

package git

import (
	"context"
	"regexp"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

// Name is a name identifiable by git.
type Name interface {
	// If cloneBranch returns a non-empty string, any clones will be performed with --branch set to the value.
	cloneBranch() string
	// If checkout returns a non-empty string, a checkout of the value will be performed after cloning.
	checkout() string
}

// NewBranchName returns a new Name for the branch.
func NewBranchName(branch string) Name {
	return newBranch(branch)
}

// NewTagName returns a new Name for the tag.
func NewTagName(tag string) Name {
	return newBranch(tag)
}

// NewRefName returns a new Name for the ref.
func NewRefName(ref string) Name {
	return newRef(ref)
}

// NewRefNameWithBranch returns a new Name for the ref while setting branch as the clone target.
func NewRefNameWithBranch(ref string, branch string) Name {
	return newRefWithBranch(ref, branch)
}

// Cloner clones git repositories to buckets.
type Cloner interface {
	// CloneToBucket clones the repository to the bucket.
	//
	// The url must contain the scheme, including file:// if necessary.
	// depth must be > 0.
	CloneToBucket(
		ctx context.Context,
		envContainer app.EnvContainer,
		url string,
		depth uint32,
		writeBucket storage.WriteBucket,
		options CloneToBucketOptions,
	) error
}

// CloneToBucketOptions are options for Clone.
type CloneToBucketOptions struct {
	Mapper            storage.Mapper
	Name              Name
	RecurseSubmodules bool
}

// NewCloner returns a new Cloner.
func NewCloner(
	logger *zap.Logger,
	tracer tracing.Tracer,
	storageosProvider storageos.Provider,
	runner command.Runner,
	options ClonerOptions,
) Cloner {
	return newCloner(logger, tracer, storageosProvider, runner, options)
}

// ClonerOptions are options for a new Cloner.
type ClonerOptions struct {
	HTTPSUsernameEnvKey      string
	HTTPSPasswordEnvKey      string
	SSHKeyFileEnvKey         string
	SSHKnownHostsFilesEnvKey string
}

// Lister lists files in git repositories.
type Lister interface {
	// ListFilesAndUnstagedFiles lists all files checked into git except those that
	// were deleted, and also lists unstaged files.
	//
	// This does not list unstaged deleted files
	// This does not list unignored files that were not added.
	// This ignores regular files.
	//
	// This is used for situations like license headers where we want all the
	// potential git files during development.
	//
	// The returned paths will be unnormalized.
	//
	// This is the equivalent of doing:
	//
	//	comm -23 \
	//		<(git ls-files --cached --modified --others --no-empty-directory --exclude-standard | sort -u | grep -v -e IGNORE_PATH1 -e IGNORE_PATH2) \
	//		<(git ls-files --deleted | sort -u)
	ListFilesAndUnstagedFiles(
		ctx context.Context,
		envContainer app.EnvStdioContainer,
		options ListFilesAndUnstagedFilesOptions,
	) ([]string, error)
}

// NewLister returns a new Lister.
func NewLister(runner command.Runner) Lister {
	return newLister(runner)
}

// ListFilesAndUnstagedFilesOptions are options for ListFilesAndUnstagedFiles.
type ListFilesAndUnstagedFilesOptions struct {
	// IgnorePathRegexps are regexes of paths to ignore.
	//
	// These must be unnormalized in the manner of the local OS that the Lister
	// is being applied to.
	IgnorePathRegexps []*regexp.Regexp
}

// RemoteKind is the kind of remote based on Git source (e.g. GitHub, GitLab, BitBucket, etc.)
type RemoteKind int

const (
	// RemoteKindUnknown is a remote to a unknown Git source.
	RemoteKindUnknown RemoteKind = iota + 1
	// RemoteKindGitHub is a remote to a GitHub Git source.
	RemoteKindGitHub
	// RemoteKindGitLab is a remote to a GitLab Git source.
	RemoteKindGitLab
	// RemoteKindBitBucket is a remote to a BitBucket Git source.
	RemoteKindBitBucket
)

// Remote represents a Git remote and provides associated metadata.
type Remote interface {
	// Name of the remote (e.g. "origin")
	Name() string
	// HEADBranch is the name of the HEAD branch of the remote.
	HEADBranch() string
	// Kind of remote (e.g. GitHub, GitLab, BitBucket, etc.).
	Kind() RemoteKind
	// Hostname is the host name parsed from the remote URL. If the remote is an unknown
	// kind, then this may be an empty string.
	Hostname() string
	// RepositoryPath is the path to the repository based on the remote URL. If the remote
	// is an unknown kind, then this may be an empty string.
	RepositoryPath() string
}

// GetRemote gets the Git remote based on the given remote name.
// In order to query the remote information, we need to pass in the env with appropriate
// permissions.
func GetRemote(
	ctx context.Context,
	runner command.Runner,
	env map[string]string,
	dir string,
	name string,
) (Remote, error) {
	return getRemote(ctx, runner, env, dir, name)
}
