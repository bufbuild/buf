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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
)

const (
	bitBucketHostname        = "bitbucket"
	githubHostname           = "github"
	gitlabHostname           = "gitlab"
	gitSuffix                = ".git"
	githubRemoteURLFormat    = "https://%s%s/commit/%s"
	gitlabRemoteURLFormat    = "https://%s%s/commit/%s"
	bitBucketRemoteURLFormat = "https://%s%s/commits/%s"

	// remoteKindUnknown is a remote to a unknown Git source.
	remoteKindUnknown remoteKind = iota + 1
	// RemoteKindGitHub is a remote to a GitHub Git source.
	remoteKindGitHub
	// RemoteKindGitLab is a remote to a GitLab Git source.
	remoteKindGitLab
	// RemoteKindBitBucket is a remote to a BitBucket Git source.
	remoteKindBitBucket
)

// remoteKind is the kind of remote based on Git source (e.g. GitHub, GitLab, BitBucket, etc.)
type remoteKind int

type remote struct {
	name           string
	kind           remoteKind
	hostname       string
	repositoryPath string
	headBranch     string
}

func (r *remote) Name() string {
	return r.name
}

func (r *remote) Hostname() string {
	return r.hostname
}

func (r *remote) RepositoryPath() string {
	return r.repositoryPath
}

func (r *remote) HEADBranch() string {
	return r.headBranch
}

func (r *remote) isRemote() {}

func (r *remote) SourceControlURL(gitCommitSha string) string {
	switch r.kind {
	case remoteKindBitBucket:
		return fmt.Sprintf(
			bitBucketRemoteURLFormat,
			r.Hostname(),
			r.RepositoryPath(),
			gitCommitSha,
		)
	case remoteKindGitHub:
		return fmt.Sprintf(
			githubRemoteURLFormat,
			r.Hostname(),
			r.RepositoryPath(),
			gitCommitSha,
		)
	case remoteKindGitLab:
		return fmt.Sprintf(
			gitlabRemoteURLFormat,
			r.Hostname(),
			r.RepositoryPath(),
			gitCommitSha,
		)
	}
	// Unknown remote kind, we return an empty URL.
	return ""
}

func newRemote(
	name string,
	kind remoteKind,
	hostname string,
	repositoryPath string,
	headBranch string,
) *remote {
	return &remote{
		name:           name,
		kind:           kind,
		hostname:       hostname,
		repositoryPath: repositoryPath,
		headBranch:     headBranch,
	}
}

func getRemote(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	dir string,
	name string,
) (*remote, error) {
	if err := validateRemoteExists(ctx, runner, envContainer, dir, name); err != nil {
		return nil, err
	}
	hostname, repositoryPath, err := getRemoteURLMetadata(
		ctx,
		runner,
		envContainer,
		dir,
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL metadata: %w", err)
	}
	headBranch, err := getRemoteHEADBranch(ctx, runner, envContainer, dir, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote HEAD branch: %w", err)
	}
	return newRemote(
		name,
		getRemoteKindFromHostname(hostname),
		hostname,
		repositoryPath,
		headBranch,
	), nil
}

func validateRemoteExists(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	dir string,
	name string,
) error {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("ls-remote", "--exit-code", name),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
		command.RunWithEnv(app.EnvironMap(envContainer)),
	); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ProcessState.ExitCode() == 128 {
				return fmt.Errorf("remote %s: %w", name, ErrRemoteNotFound)
			}
		}
		return err
	}
	return nil
}

func getRemoteURLMetadata(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	dir string,
	remote string,
) (string, string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		// We use `git config --get remote.<remote>.url` instead of `git remote get-url
		// since it is more specific to the checkout.
		command.RunWithArgs("config", "--get", fmt.Sprintf("remote.%s.url", remote)),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
		command.RunWithEnv(app.EnvironMap(envContainer)),
	); err != nil {
		return "", "", err
	}
	hostname, repositoryPath := parseRawRemoteURL(strings.TrimSpace(stdout.String()))
	return hostname, repositoryPath, nil
}

func parseRawRemoteURL(rawURL string) (string, string) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// Attempt to check if this is a scp-style URL.
		parsed = parseSCPLikeURL(rawURL)
	}
	// If we are unable to parse host and path information from the URL, then we return
	// no information.
	if parsed == nil {
		return "", ""
	}
	return parsed.Hostname(), strings.TrimSuffix(parsed.Path, gitSuffix)
}

func parseSCPLikeURL(rawURL string) *url.URL {
	// https://git-scm.com/docs/git-clone#_git_urls
	// An alternative scp-like syntax may also be used with the ssh protocol:
	//   [<user>@]<host>:/<path-to-git-repo>
	//
	// To parse this, we use a regexp from cmd/go/internal/vcs/vcs.go
	// https://cs.opensource.google/go/go/+/refs/tags/go1.21.10:src/cmd/go/internal/vcs/vcs.go;l=281-283
	scpSyntaxRegexp := regexp.MustCompile(`^(\w+)@([\w.-]+):(.*)$`)
	if match := scpSyntaxRegexp.FindStringSubmatch(rawURL); match != nil {
		// The host and path are the only relevant URL components to our use case.
		return &url.URL{
			Host: match[2],
			Path: "/" + match[3], // scp-like URLs do not have the leading slash
		}
	}
	return nil
}

// getRemoteHEADBranch returns the HEAD branch based on the given remote and given
// directory. Querying the remote for the HEAD branch requires passing the
// environment for permissions.
func getRemoteHEADBranch(
	ctx context.Context,
	runner command.Runner,
	envContainer app.EnvContainer,
	dir string,
	remote string,
) (string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("remote", "show", remote),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
		command.RunWithEnv(app.EnvironMap(envContainer)),
	); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if branch, isHEADBranch := strings.CutPrefix(line, "HEAD branch:"); isHEADBranch {
			return strings.TrimSpace(branch), nil
		}
	}
	return "", errors.New("no HEAD branch information found")
}

func getRemoteKindFromHostname(hostname string) remoteKind {
	if strings.Contains(hostname, bitBucketHostname) {
		return remoteKindBitBucket
	}
	if strings.Contains(hostname, githubHostname) {
		return remoteKindGitHub
	}
	if strings.Contains(hostname, gitlabHostname) {
		return remoteKindGitLab
	}
	return remoteKindUnknown
}
