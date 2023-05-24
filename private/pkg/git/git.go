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

package git

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

const (
	// DotGitDir is a relative path to the `.git` directory.
	DotGitDir = ".git"

	// ModeUnknown is a mode's zero value.
	ModeUnknown FileMode = 0
	// ModeFile is a blob that should be written as a plain file.
	ModeFile FileMode = 010_0644
	// ModeExec is a blob that should be written with the executable bit set.
	ModeExe FileMode = 010_0755
	// ModeDir is a tree to be unpacked as a subdirectory in the current
	// directory.
	ModeDir FileMode = 004_0000
	// ModeSymlink is a blob with its content being the path linked to.
	ModeSymlink FileMode = 012_0000
	// ModeSubmodule is a commit that the submodule is checked out at.
	ModeSubmodule FileMode = 016_0000
)

var ErrSubTreeNotFound = errors.New("subtree not found")

// FileMode is how to interpret a tree node's object. See the Mode* constants
// for how to interpret each mode value.
type FileMode uint32

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
	storageosProvider storageos.Provider,
	runner command.Runner,
	options ClonerOptions,
) Cloner {
	return newCloner(logger, storageosProvider, runner, options)
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

// BranchIterator ranges over branches for a git repository.
//
// Only branches from the remote named `origin` are ranged.
type BranchIterator interface {
	// ForEachBranch ranges over branches in the repository in an undefined order.
	ForEachBranch(func(branch string) error) error
}

// CommitIterator ranges over commits for a git repository.
//
// Only commits from the remote named `origin` are ranged.
type CommitIterator interface {
	// BaseBranch is the base branch of the repository. This is either
	// configured via the `WithBaseBranch` option, or discovered via the
	// remote named `origin`. Therefore, discovery requires that the repository
	// is pushed to the remote.
	BaseBranch() string
	// ForEachCommit ranges over commits for the target branch in reverse topological order.
	//
	// Parents are visited before children, and only left parents are visited (i.e.,
	// commits from branches merged into the target branch are not visited).
	ForEachCommit(branch string, f func(commit Commit) error) error
}

type CommitIteratorOption func(*commitIteratorOpts) error

// CommitIteratorWithBaseBranch configures the base branch for this iterator.
func CommitIteratorWithBaseBranch(name string) CommitIteratorOption {
	return func(r *commitIteratorOpts) error {
		if name == "" {
			return errors.New("base branch cannot be empty")
		}
		r.baseBranch = name
		return nil
	}
}

// NewBranchIterator creates a new BranchIterator that can range over branches.
func NewBranchIterator(
	gitDirPath string,
	objectReader ObjectReader,
) (BranchIterator, error) {
	return newBranchIterator(gitDirPath, objectReader)
}

// NewCommitIterator creates a new CommitIterator that can range over commits.
//
// By default, NewCommitIterator will attempt to detect the base branch if the repository
// has been pushed. This may fail is the repository is not pushed. In this case, use the
// `CommitIteratorWithBaseBranch` option.
func NewCommitIterator(
	gitDirPath string,
	objectReader ObjectReader,
	options ...CommitIteratorOption,
) (CommitIterator, error) {
	var opts commitIteratorOpts
	for _, option := range options {
		if err := option(&opts); err != nil {
			return nil, err
		}
	}
	return newCommitIterator(gitDirPath, objectReader, opts)
}

// Hash represents the hash of a Git object (tree, blob, or commit).
type Hash interface {
	// Hex is the hexadecimal representation of this ID.
	Hex() string
	// String returns the hexadecimal representation of this ID.
	String() string
}

// NewHashFromHex creates a new hash that is validated.
func NewHashFromHex(value string) (Hash, error) {
	return parseHashFromHex(value)
}

// Ident is a git user identifier. These typically represent authors and committers.
type Ident interface {
	// Name is the name of the user.
	Name() string
	// Email is the email of the user.
	Email() string
	// Timestamp is the time at which this identity was created. For authors it's the
	// commit's author time, and for committers it's the commit's commit time.
	Timestamp() time.Time
}

// Commit represents a commit object.
//
// All commits will have a non-nil Tree. All but the root commit will contain >0 parents.
type Commit interface {
	// Hash is the Hash for this commit.
	Hash() Hash
	// Tree is the ID to the git tree for this commit.
	Tree() Hash
	// Parents is the set of parents for this commit. It may be empty.
	//
	// By convention, the first parent in a multi-parent commit is the merge target.
	Parents() []Hash
	// Author is the user who authored the commit.
	Author() Ident
	// Committer is the user who created the commit.
	Committer() Ident
	// Message is the commit message.
	Message() string
}

// ObjectReader reads objects (commits, trees, blobs) from a `.git` directory.
type ObjectReader interface {
	// Blob reads the blob identified by the hash.
	Blob(id Hash) ([]byte, error)
	// Commit reads the commit identified by the hash.
	Commit(id Hash) (Commit, error)
	// Tree reads the tree identified by the hash.
	Tree(id Hash) (Tree, error)
	// Close closes the reader.
	Close() error
}

// OpenObjectReader opens a new Reader that can read objects from a `.git` directory.
//
// The provided path to the `.git` dir need not be normalized or cleaned.
//
// Internally, OpenObjectReader will spawns a new process to communicate with `git-cat-file`,
// so the caller must close the reader function to clean up resources.
func OpenObjectReader(
	gitDirPath string,
	runner command.Runner,
) (ObjectReader, error) {
	return newObjectReader(gitDirPath, runner)
}

// Tree is a git tree, which are a manifest of other git objects, including other trees.
type Tree interface {
	// Hash is the Hash for this Tree.
	Hash() Hash
	// Nodes is the set of nodes in this Tree.
	Nodes() []Node
	// Traverse walks down a tree, following the name-path specified,
	// and returns the terminal Node. If no node is found, it returns
	// ErrSubTreeNotFound.
	Traverse(objectReader ObjectReader, names ...string) (Node, error)
}

// Node is a reference to an object contained in a tree. These objects have
// a file mode associated with them, which hints at the type of object located
// at ID (tree or blob).
type Node interface {
	// Hash is the Hash of the object referenced by this Node.
	Hash() Hash
	// Name is the name of the object referenced by this Node.
	Name() string
	// Mode is the file mode of the object referenced by this Node.
	Mode() FileMode
}
