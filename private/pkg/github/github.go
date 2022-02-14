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

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v42/github"
)

type CompareCommitsStatus int

// The possible values for returned from githubClient.CompareCommits.
// see https://stackoverflow.com/a/23969867
const (
	CompareCommitsStatusDiverged CompareCommitsStatus = iota + 1
	CompareCommitsStatusIdentical
	CompareCommitsStatusAhead
	CompareCommitsStatusBehind
)

var compareCommitStatusStrings = map[CompareCommitsStatus]string{
	CompareCommitsStatusDiverged:  "diverged",
	CompareCommitsStatusIdentical: "identical",
	CompareCommitsStatusAhead:     "ahead",
	CompareCommitsStatusBehind:    "behind",
}

var stringsToCompareCommitStatus = map[string]CompareCommitsStatus{
	"diverged":  CompareCommitsStatusDiverged,
	"identical": CompareCommitsStatusIdentical,
	"ahead":     CompareCommitsStatusAhead,
	"behind":    CompareCommitsStatusBehind,
}

func (s CompareCommitsStatus) String() string {
	got, ok := compareCommitStatusStrings[s]
	if !ok {
		return fmt.Sprintf("unknown(%d)", s)
	}
	return got
}

type Client interface {
	// CompareCommits compares two commits and returns the status of head relative to base.
	CompareCommits(ctx context.Context, base string, head string) (CompareCommitsStatus, error)
}

// NewClient returns a new github client.
// baseURL is optional and defaults to https://api.github.com/.
func NewClient(
	ctx context.Context,
	githubToken string,
	userAgent string,
	baseURL string,
	repository string,
) (Client, error) {
	return newGithubClient(ctx, githubToken, userAgent, baseURL, repository)
}

// IsNotFoundError returns true if the error is a *github.ErrorResponse with a 404 status code.
func IsNotFoundError(err error) bool {
	var errorResponse *github.ErrorResponse
	if !errors.As(err, &errorResponse) {
		return false
	}
	return errorResponse.Response.StatusCode == http.StatusNotFound
}
