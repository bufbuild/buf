// Copyright 2020-2022 Buf Technologies, Inc.
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

package push

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/github"
	"github.com/bufbuild/buf/private/pkg/rpc"
)

type checkType int

const (
	checkTypeNone checkType = iota + 1
	checkTypeGithub
)

var checkTypeStrings = map[checkType]string{
	checkTypeNone:   "none",
	checkTypeGithub: "github",
}

var stringToCheckType = map[string]checkType{
	"none":   checkTypeNone,
	"github": checkTypeGithub,
}

var allCheckTypeStrings = []string{
	"none",
	"github",
}

const (
	githubTokenEnvKey      = "GITHUB_TOKEN"
	githubRepositoryEnvKey = "GITHUB_REPOSITORY"
	githubAPIURLEnvKey     = "GITHUB_API_URL"
)

func (g checkType) String() string {
	got, ok := checkTypeStrings[g]
	if !ok {
		return fmt.Sprintf("unknown(%d)", g)
	}
	return got
}

// githubCheck is a vcs check that uses the github api to prevent pushing outdated commits to BSR.
//
// It requires:
//  - environment variable GITHUB_REPOSITORY is set with the github repository name in the format "owner/repository"
// 	- environment variable GITHUB_TOKEN is set with a github token with access to the repository
// 	- exactly one element of tags is a git commit hash (called taggedGitCommit below)
//
// It errors if the head of any track
//   1. has a different digest from module
//     and
//   2. is not already tagged with taggedGitCommit
//     and
//   3. has no tags that represent a git commit that is behind taggedGitCommit in the github repository
//     and
//	 4. has one or more tags that represent git commits that are ahead of taggedGitCommit in the github repository
func githubCheck(
	ctx context.Context,
	envContainer app.EnvContainer,
	apiProvider registryv1alpha1apiclient.Provider,
	module bufmodule.Module,
	moduleIdentity bufmoduleref.ModuleIdentity,
	tags []string,
	tracks []string,
) error {
	digest, err := bufmodule.ModuleDigestB1(ctx, module)
	if err != nil {
		return err
	}
	var taggedGitCommit string
	for _, tag := range tags {
		if !maybeGitCommit(tag) {
			continue
		}
		if taggedGitCommit != "" {
			return fmt.Errorf("exactly one tag must be a 40 character git commit hash when --vcs-check=github")
		}
		taggedGitCommit = tag
	}
	if taggedGitCommit == "" {
		return fmt.Errorf("exactly one tag must be a 40 character git commit hash when --vcs-check=github")
	}
	repositoryCommitService, err := apiProvider.NewRepositoryCommitService(ctx, moduleIdentity.Remote())
	if err != nil {
		return err
	}
	token := envContainer.Env(githubTokenEnvKey)
	if token == "" {
		return fmt.Errorf("environment variable %s must be set when --vcs-check=github", githubTokenEnvKey)
	}
	githubRepository := envContainer.Env(githubRepositoryEnvKey)
	if githubRepository == "" {
		return fmt.Errorf("environment variable %s must be set when --vcs-check=github", githubRepositoryEnvKey)
	}
	// baseURL is optional
	baseURL := envContainer.Env(githubAPIURLEnvKey)
	userAgent := fmt.Sprintf("buf/%s", bufcli.Version)
	githubClient, err := github.NewClient(ctx, token, userAgent, baseURL, githubRepository)
	if err != nil {
		return err
	}
	for _, track := range tracks {
		trackHead, err := repositoryCommitService.GetRepositoryCommitByReference(
			ctx,
			moduleIdentity.Owner(),
			moduleIdentity.Repository(),
			track,
		)
		if err != nil {
			if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
				// It's always ok to push to a new track
				continue
			}
			return err
		}
		if trackHead.Digest == digest {
			// It's always ok to push to a track with the same digest
			continue
		}
		diverged := false
		behind := false
		ahead := false
		identical := false
		for _, tag := range trackHead.Tags {
			if !maybeGitCommit(tag.Name) {
				continue
			}
			status, err := githubClient.CompareCommits(ctx, tag.Name, taggedGitCommit)
			if err != nil {
				if github.IsNotFoundError(err) {
					// It can't be an ancestor of a commit that doesn't exist.
					continue
				}
				return err
			}
			switch status {
			case github.CompareCommitsStatusDiverged:
				diverged = true
			case github.CompareCommitsStatusBehind:
				behind = true
			case github.CompareCommitsStatusAhead:
				ahead = true
			case github.CompareCommitsStatusIdentical:
				identical = true
			}
		}
		if ahead || identical {
			// It's ok to push if the new commit is ahead of or the same as any tagged git commit...even if it is behind
			// other tagged git commits.
			continue
		}
		if diverged || behind {
			return fmt.Errorf("not pushing because %s is behind the head of %s", taggedGitCommit, track)
		}
		// Tracks with no tagged git commits are ok to push.
	}
	return nil
}

// maybeGitCommit returns true if the string is 40 characters long and only contains hexadecimal characters.
func maybeGitCommit(s string) bool {
	if len(s) != 40 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
