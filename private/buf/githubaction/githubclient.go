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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

// The possible values for returned from githubClient.compareCommits.
// see https://stackoverflow.com/a/23969867
const (
	// commits were introduced on both the head and base branch since the common ancestor
	compareCommitsStatusDiverged = "diverged"
	// commits were introduced on head after the common ancestor with base
	compareCommitsStatusAhead = "ahead"
	// commits were introduced on base after the common ancestor with head
	compareCommitsStatusBehind = "behind"
	// base and head point to same commit
	compareCommitsStatusIdentical = "identical"
)

func validateCompareCommitStatus(status string) bool {
	return status == compareCommitsStatusDiverged ||
		status == compareCommitsStatusAhead ||
		status == compareCommitsStatusBehind ||
		status == compareCommitsStatusIdentical
}

type githubClient struct {
	client *github.Client
	owner  string
	repo   string
}

func newGithubClient(
	ctx context.Context,
	githubToken string,
	userAgent string,
	baseURL string,
	repository string,
) (*githubClient, error) {
	owner, repo, err := parseGithubRepository(repository)
	if err != nil {
		return nil, err
	}
	goGithubClient, err := newGoGithubClient(ctx, githubToken, userAgent, baseURL)
	if err != nil {
		return nil, err
	}
	return &githubClient{
		client: goGithubClient,
		owner:  owner,
		repo:   repo,
	}, nil
}

func (c *githubClient) compareCommits(ctx context.Context, base string, head string) (string, error) {
	comp, _, err := c.client.Repositories.CompareCommits(ctx, c.owner, c.repo, base, head, nil)
	if err != nil {
		return "", err
	}
	status := comp.GetStatus()
	if !validateCompareCommitStatus(comp.GetStatus()) {
		return "", fmt.Errorf("unexpected status: %s", status)
	}
	return status, nil
}

// maybePostStatus posts a status if the token has permission.
func (c *githubClient) maybePostStatus(
	ctx context.Context,
	githubCommit string,
	state string,
	context string,
	description string,
	targetURL string,
) error {
	_, _, err := c.client.Repositories.CreateStatus(ctx, c.owner, c.repo, githubCommit, &github.RepoStatus{
		State:       &state,
		Context:     &context,
		Description: &description,
		TargetURL:   &targetURL,
	})
	if err != nil {
		var errorResponse *github.ErrorResponse
		if errors.As(err, &errorResponse) && errorResponse.Response.StatusCode == http.StatusForbidden {
			return nil
		}
		return fmt.Errorf("failed posting status to github: %w", err)
	}
	return nil
}

func newGoGithubClient(
	ctx context.Context,
	token string,
	userAgent string,
	baseURL string,
) (*github.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("github token is empty")
	}
	client := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	var err error
	if baseURL != "" {
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		client.BaseURL, err = url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse base url: %w", err)
		}
	}
	client.UserAgent = userAgent
	return client, nil
}

func parseGithubRepository(repository string) (owner string, name string, _ error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository: %s", repository)
	}
	return parts[0], parts[1], nil
}
