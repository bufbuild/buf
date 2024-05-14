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
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bufbuild/buf/private/pkg/command"
)

const (
	githubGitlabRemoteURLFormat = "https://%s%s/commit/%s"
	bitBucketRemoteURLFormat    = "https://%s%s/commits/%s"
	bitBucketHostname           = "bitbucket"
	githubHostname              = "github"
	gitlabHostname              = "gitlab"
)

// GetSourceControlURL gets the parses a user-facing source control URL based on the URL
// from git config based on the given remote, git commit sha, and directory.
//
// It checks the hostname of repository remote URL -- if it contains a known hostname,
// github, gitlab, and bitbucket, we construct a URL. Otherwise, we return an empty source
// control URL.
func GetSourceControlURL(
	ctx context.Context,
	runner command.Runner,
	dir string,
	remote string,
	gitCommitSha string,
) (string, error) {
	repositoryURL, err := getRepositoryRemoteURL(ctx, runner, dir, remote)
	if err != nil {
		return "", err
	}
	return parseSourceControlURL(repositoryURL, gitCommitSha)
}

func getRepositoryRemoteURL(
	ctx context.Context,
	runner command.Runner,
	dir string,
	remote string,
) (string, error) {
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
	); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// parseSourceControlURL takes a repository URL and git commit sha and makes the best
// effort to construct a user-facing URL based on the hostname.
//
// If the hostname contains bitbucket (e.g. bitbucket.mycompany.com or bitbucket.org),
// then it uses the route /commits for the git commit sha.
// If the hostname contains github (e.g. github.mycompany.com or github.com) or gitlab
// (e.g. gitlab.mycompany.com or gitlab.com) then it uses the route /commit for the git
// commit sha.
func parseSourceControlURL(
	rawRepositoryURL string,
	gitCommitSha string,
) (string, error) {
	repositoryURL, err := url.Parse(rawRepositoryURL)
	if err != nil {
		return "", err
	}
	if strings.Contains(repositoryURL.Hostname(), bitBucketHostname) {
		return fmt.Sprintf(
			bitBucketRemoteURLFormat,
			repositoryURL.Hostname(),
			strings.TrimSuffix(repositoryURL.Path, ".git"),
			gitCommitSha,
		), nil
	}
	if strings.Contains(repositoryURL.Hostname(), githubHostname) || strings.Contains(repositoryURL.Hostname(), gitlabHostname) {
		return fmt.Sprintf(
			githubGitlabRemoteURLFormat,
			repositoryURL.Hostname(),
			strings.TrimSuffix(repositoryURL.Path, ".git"),
			gitCommitSha,
		), nil
	}
	return "", nil
}
