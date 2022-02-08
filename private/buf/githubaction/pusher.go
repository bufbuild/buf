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

package githubaction

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/rpc"
)

type pusher struct {
	githubSHA     string
	track         string
	githubRefName string

	moduleIdentity bufmoduleref.ModuleIdentity
	module         bufmodule.Module
	protoModule    *modulev1alpha1.Module

	registryProvider registryv1alpha1apiclient.Provider
	githubClient     *githubClient
}

func newPusher(
	ctx context.Context,
	container app.EnvContainer,
	registryProvider registryv1alpha1apiclient.Provider,
	moduleIdentity bufmoduleref.ModuleIdentity,
	module bufmodule.Module,
	protoModule *modulev1alpha1.Module,
	bufVersion string,
) (*pusher, error) {
	_, err := GetBufTokenValue(container)
	if err != nil {
		return nil, err
	}
	githubSHA, err := GetGithubShaValue(container)
	if err != nil {
		return nil, err
	}
	track, err := GetTrackValue(container)
	if err != nil {
		return nil, err
	}
	track = sanitizeTrackName(track)
	githubToken, err := GetGithubTokenValue(container)
	if err != nil {
		return nil, err
	}
	githubRepository, err := GetGithubRepositoryValue(container)
	if err != nil {
		return nil, err
	}
	githubRefName, err := GetGithubRefNameValue(container)
	if err != nil {
		return nil, err
	}
	githubAPIURL, err := GetGithubAPIURLValue(container)
	if err != nil {
		return nil, err
	}

	userAgent := fmt.Sprintf("buf/%s", bufVersion)
	ghClient, err := newGithubClient(ctx, githubToken, userAgent, githubAPIURL, githubRepository)
	if err != nil {
		return nil, err
	}

	return &pusher{
		githubSHA:     githubSHA,
		track:         track,
		githubRefName: githubRefName,

		moduleIdentity: moduleIdentity,
		module:         module,
		protoModule:    protoModule,

		registryProvider: registryProvider,
		githubClient:     ghClient,
	}, nil
}

func (p *pusher) push(ctx context.Context) error {
	// Don't proceed unless we are working with the head of the git branch
	ghStatus, err := p.githubClient.compareCommits(ctx, p.githubRefName, p.githubSHA)
	if err != nil {
		return err
	}

	// TODO: consider how we actually want to communicate this. Returning an error for now, although we should probably just log it and exit 0.
	if ghStatus == compareCommitsStatusDiverged || ghStatus == compareCommitsStatusBehind {
		return fmt.Errorf("github ref %s is behind %s", p.githubRefName, p.githubSHA)
	}

	// Get the track if it exists

	referenceService, err := p.registryProvider.NewReferenceService(ctx, p.moduleIdentity.Remote())
	if err != nil {
		return err
	}

	trackReference, err := referenceService.GetReferenceByName(
		ctx,
		p.track,
		p.moduleIdentity.Owner(),
		p.moduleIdentity.Repository(),
	)
	if err != nil {
		if rpc.GetErrorCode(err) != rpc.ErrorCodeNotFound {
			return err
		}

		// The track does not exist, so there is no need for any other checks. Just push it.
		return p.pushAndStatus(ctx)
	}
	repositoryTrack := trackReference.GetTrack()
	if repositoryTrack == nil {
		return fmt.Errorf("reference %q exists but is not a track", p.track)
	}

	repositoryCommitService, err := p.registryProvider.NewRepositoryCommitService(ctx, p.moduleIdentity.Remote())
	if err != nil {
		return err
	}

	trackHeadCommit, err := repositoryCommitService.GetRepositoryCommitByReference(
		ctx,
		p.moduleIdentity.Owner(),
		p.moduleIdentity.Repository(),
		p.githubRefName,
	)
	if err != nil {
		if rpc.GetErrorCode(err) != rpc.ErrorCodeNotFound {
			return err
		}
		// It's an empty track. We can just push to it.
		return p.pushAndStatus(ctx)
	}

	// If any tag is an exact match for p.githubSHA, then the track is already up to date.
	for _, tag := range trackHeadCommit.Tags {
		if tag.Name == p.githubSHA {
			return p.postPushSuccessStatus(ctx, trackHeadCommit.Name)
		}
	}

	// If the track head has the same digest, we can just push to it to get it tagged with the new SHA.
	digestMatch, err := bufmodule.ModuleMatchesDigest(ctx, p.module, trackHeadCommit.Digest)
	if err != nil {
		return err
	}
	if digestMatch {
		return p.pushAndStatus(ctx)
	}

	// Check whether the sha is ahead of, behind or diverged from the track.

	diverged := false
	behind := false
	ahead := false
	taggedWithGithubSHA := false
	for _, tag := range trackHeadCommit.Tags {
		if !isPossibleGitCommitHash(tag.Name) {
			continue
		}
		var compareResult string
		compareResult, err = p.githubClient.compareCommits(ctx, tag.Name, p.githubSHA)
		if err != nil {
			if rpc.GetErrorCode(err) == rpc.ErrorCodeNotFound {
				// The tag is not a commit hash that GitHub knows about. This is curious, but we can only ignore it.
				continue
			}
			return err
		}
		taggedWithGithubSHA = true
		switch compareResult {
		case compareCommitsStatusDiverged:
			diverged = true
		case compareCommitsStatusBehind:
			behind = true
		case compareCommitsStatusAhead:
			ahead = true
		case compareCommitsStatusIdentical:
			// We must be racing with another workflow run that pushed the commit already. We can exit with no error.
			return nil
		}
	}
	if !taggedWithGithubSHA {
		// Something else pushed a commit to this track. We can't be sure that our commit is up to date, so we need to
		// error.
		return fmt.Errorf("track %q is in an unknown state", p.track)
	}

	switch {
	case ahead:
		// We proceed if we are ahead of any tag.
	case behind, diverged:
		// Don't push if we are behind the head of the track.
		// TODO: maybe post a status or emit a warning
		return nil
	}

	// Check for a bsr commit tagged with the git commit sha. If it exists, we just need to add it to the track.

	repositoryCommit, err := repositoryCommitService.GetRepositoryCommitByReference(
		ctx,
		p.moduleIdentity.Owner(),
		p.moduleIdentity.Repository(),
		p.githubSHA,
	)
	if err != nil {
		if rpc.GetErrorCode(err) != rpc.ErrorCodeNotFound {
			return err
		}
		// No commit found, proceed with push.
	} else {
		repositoryTrackCommitService, err := p.registryProvider.NewRepositoryTrackCommitService(ctx, p.moduleIdentity.Remote())
		if err != nil {
			return err
		}

		// The commit already exists. No need to push. Just add it to the track.
		_, err = repositoryTrackCommitService.GetRepositoryTrackCommitByRepositoryCommit(
			ctx,
			repositoryTrack.Id,
			repositoryCommit.Id,
		)
		if err != nil {
			if rpc.GetErrorCode(err) != rpc.ErrorCodeNotFound {
				return err
			}
			// No track commit found, proceed with creating one.
		} else {
			// The commit is already on the track. Nothing to do.
			return nil
		}

		_, err = repositoryTrackCommitService.CreateRepositoryTrackCommit(
			ctx,
			repositoryTrack.Id,
			repositoryCommit.Name,
		)
		if err != nil {
			return err
		}
		return p.postPushSuccessStatus(ctx, repositoryCommit.Name)
	}

	return p.pushAndStatus(ctx)
}

func (p *pusher) pushAndStatus(ctx context.Context) error {
	pushService, err := p.registryProvider.NewPushService(ctx, p.moduleIdentity.Remote())
	if err != nil {
		return err
	}
	pin, err := pushService.Push(
		ctx,
		p.moduleIdentity.Owner(),
		p.moduleIdentity.Repository(),
		"",
		p.protoModule,
		nil,
		[]string{p.track},
	)
	if err != nil {
		return err
	}
	return p.postPushSuccessStatus(ctx, pin.Commit)
}

func (p *pusher) postPushSuccessStatus(ctx context.Context, bsrCommit string) error {
	targetURL := fmt.Sprintf("https://%s/%s/%s/tree/%s",
		p.moduleIdentity.Remote(),
		p.moduleIdentity.Owner(),
		p.moduleIdentity.Repository(),
		bsrCommit,
	)
	description := fmt.Sprintf("pushed to %s track", p.track)
	return p.githubClient.maybePostStatus(ctx, p.githubSHA, "success", "buf-push", description, targetURL)
}

func isPossibleGitCommitHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	if _, err := hex.DecodeString(s); err != nil {
		return false
	}
	return true
}

// sanitizeTrackName down cases track and replaces all non-alphanumeric characters with dashes.
func sanitizeTrackName(track string) string {
	var sanitized strings.Builder
	for _, r := range strings.ToLower(track) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sanitized.WriteRune(r)
			continue
		}
		sanitized.WriteRune('-')
	}
	return sanitized.String()
}
