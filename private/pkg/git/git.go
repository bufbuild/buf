// Copyright 2020-2025 Buf Technologies, Inc.
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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"buf.build/go/app"
	"buf.build/go/standard/xos/xexec"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

const (
	gitCommand      = "git"
	tagsPrefix      = "refs/tags/"
	headsPrefix     = "refs/heads/"
	pseudoRefSuffix = "^{}"
)

var (
	// ErrRemoteNotFound is returned from GetRemote when the specified remote name is not
	// found in the current git checkout.
	ErrRemoteNotFound = errors.New("git remote not found")

	// ErrInvalidGitCheckout is returned from CheckDirectoryIsValidGitCheckout when the
	// specified directory is not a valid git checkout.
	ErrInvalidGitCheckout = errors.New("invalid git checkout")

	// ErrInvalidRef is returned from IsValidRef if the ref does not exist in the
	// repository containing the given directory (or, if the directory is not
	// a valid git checkout).
	ErrInvalidRef = errors.New("invalid git ref")
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
	Matcher           storage.Matcher
	Name              Name
	RecurseSubmodules bool
	SubDir            string
	Filter            string
}

// NewCloner returns a new Cloner.
func NewCloner(
	logger *slog.Logger,
	storageosProvider storageos.Provider,
	options ClonerOptions,
) Cloner {
	return newCloner(logger, storageosProvider, options)
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
func NewLister() Lister {
	return newLister()
}

// ListFilesAndUnstagedFilesOptions are options for ListFilesAndUnstagedFiles.
type ListFilesAndUnstagedFilesOptions struct {
	// IgnorePathRegexps are regexes of paths to ignore.
	//
	// These must be unnormalized in the manner of the local OS that the Lister
	// is being applied to.
	IgnorePathRegexps []*regexp.Regexp
}

// Remote represents a Git remote and provides associated metadata.
type Remote interface {
	// Name of the remote (e.g. "origin")
	Name() string
	// HEADBranch is the name of the HEAD branch of the remote.
	HEADBranch() string
	// Hostname is the host name parsed from the remote URL. If the remote is an unknown
	// kind, then this may be an empty string.
	Hostname() string
	// RepositoryPath is the path to the repository based on the remote URL. If the remote
	// is an unknown kind, then this may be an empty string.
	RepositoryPath() string
	// SourceControlURL makes the best effort to construct a user-facing source control url
	// given a commit sha string based on the remote source, and available hostname and
	// repository path information.
	//
	// If the remote hostname contains bitbucket (e.g. bitbucket.mycompany.com or bitbucket.org),
	// we construct the source control URL as:
	//
	//   https://<hostname>/<repository-path>/commits/<git-commit-sha>
	//
	// If the remote hostname contains github (e.g. github.mycompany.com or github.com), we
	// construct the source control URL as:
	//   https://<hostname>/repository-path>/commit/git-commit-sha>
	//
	// If the remote hostname contains gitlab (e.g. gitlab.mycompany.com or gitlab.com), we
	// construct the source control URL as:
	//   https://<hostname>/repository-path>/commit/git-commit-sha>
	//
	// If the remote is unknown and/or no hostname/repository path information is available,
	// this will return an empty string.
	//
	// This does not do any validation against the gitCommitSha provided.
	SourceControlURL(gitCommitSha string) string

	isRemote()
}

// GetRemote gets the Git remote based on the given remote name.
// In order to query the remote information, we need to pass in the env with appropriate
// permissions.
func GetRemote(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir string,
	name string,
) (Remote, error) {
	return getRemote(ctx, envContainer, dir, name)
}

// CheckDirectoryIsValidGitCheckout runs a simple git rev-parse. In the case where the
// directory is not a valid git checkout (e.g. the directory is not a git repository or
// the directory does not exist), this will return a 128. We handle that and return an
// ErrInvalidGitCheckout to the user.
func CheckDirectoryIsValidGitCheckout(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir string,
) error {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("rev-parse"),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(app.Environ(envContainer)),
	); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 128 {
				return fmt.Errorf("dir %s: %w", dir, ErrInvalidGitCheckout)
			}
		}
		return err
	}
	return nil
}

// CheckForUncommittedGitChanges checks if there are any uncommitted and/or unchecked
// changes from git based on the given directory.
func CheckForUncommittedGitChanges(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir string,
) ([]string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	var modifiedFiles []string
	environ := app.Environ(envContainer)
	// Unstaged changes
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("diff", "--name-only"),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(environ),
	); err != nil {
		return nil, fmt.Errorf("failed to get unstaged changes: %w: %s", err, stderr.String())
	}
	modifiedFiles = append(modifiedFiles, getAllTrimmedLinesFromBuffer(stdout)...)

	stdout = bytes.NewBuffer(nil)
	stderr = bytes.NewBuffer(nil)
	// Staged changes
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("diff", "--name-only", "--cached"),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(environ),
	); err != nil {
		return nil, fmt.Errorf("failed to get staged changes: %w: %s", err, stderr.String())
	}

	modifiedFiles = append(modifiedFiles, getAllTrimmedLinesFromBuffer(stdout)...)
	return modifiedFiles, nil
}

// GetCurrentHEADGitCommit returns the current HEAD commit based on the given directory.
func GetCurrentHEADGitCommit(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir string,
) (string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("rev-parse", "HEAD"),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(app.Environ(envContainer)),
	); err != nil {
		return "", fmt.Errorf("failed to get current HEAD commit: %w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetRefsForGitCommitAndRemote returns all refs pointing to a given commit based on the
// given remote for the given directory. Querying the remote for refs information requires
// passing the environment for permissions.
func GetRefsForGitCommitAndRemote(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir string,
	remote string,
	gitCommitSha string,
) ([]string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("ls-remote", "--heads", "--tags", remote),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(app.Environ(envContainer)),
	); err != nil {
		return nil, fmt.Errorf("failed to get refs for remote %s: %w: %s", remote, err, stderr.String())
	}
	scanner := bufio.NewScanner(stdout)
	var refs []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if ref, found := strings.CutPrefix(line, gitCommitSha); found {
			ref = strings.TrimSpace(ref)
			if tag, isTag := strings.CutPrefix(ref, tagsPrefix); isTag {
				// Remove the ^{} suffix for pseudo-ref tags
				tag, _ = strings.CutSuffix(tag, pseudoRefSuffix)
				refs = append(refs, tag)
				continue
			}
			if branch, isBranchHead := strings.CutPrefix(ref, headsPrefix); isBranchHead {
				refs = append(refs, branch)
			}
		}
	}
	return refs, nil
}

// IsValidRef returns whether or not ref is a valid git ref for the git
// repository that contains dir. Returns nil if the ref is valid.
func IsValidRef(
	ctx context.Context,
	envContainer app.EnvContainer,
	dir, ref string,
) error {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(
		ctx,
		gitCommand,
		xexec.WithArgs("rev-parse", "--verify", ref),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(app.Environ(envContainer)),
	); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 128 {
				return fmt.Errorf("could not find ref %s in %s: %w", ref, dir, ErrInvalidRef)
			}
		}
		return err
	}
	return nil
}

// ReadFileAtRef will read the file at path rolled back to the given ref, if
// it exists at that ref.
//
// Specifically, this will take path, and find the associated .git directory
// (and thus, repository root) associated with that path, if any. It will then
// look up the commit corresponding to ref in that git directory, and within
// that commit, the blob corresponding to that path (relative to the repository
// root).
func ReadFileAtRef(
	ctx context.Context,
	envContainer app.EnvContainer,
	path, ref string,
) ([]byte, error) {
	orig := path
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Find a directory with a .git directory.
	dir := filepath.Dir(path)
	for {
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			break
		} else if errors.Is(err, os.ErrNotExist) {
			parent := filepath.Dir(dir)
			if parent == dir {
				return nil, fmt.Errorf("could not find .git directory for %s: %w", orig, ErrInvalidGitCheckout)
			}
			dir = parent
		} else {
			return nil, err
		}
	}

	// This gives us the name of the blob we're looking for.
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return nil, err
	}

	// Call git show to show us the file we want.
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := xexec.Run(ctx, gitCommand,
		xexec.WithArgs("--no-pager", "show", ref+":"+rel),
		xexec.WithStdout(stdout),
		xexec.WithStderr(stderr),
		xexec.WithDir(dir),
		xexec.WithEnv(app.Environ(envContainer)),
	); err != nil {
		return nil, fmt.Errorf("failed to get text of file %s at ref %s: %w: %s", orig, ref, err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func getAllTrimmedLinesFromBuffer(buffer *bytes.Buffer) []string {
	scanner := bufio.NewScanner(buffer)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	return lines
}
