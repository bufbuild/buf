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

package fetch

import (
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
)

var (
	_ ParsedGitRef = &gitRef{}

	gitSchemePrefixToGitScheme = map[string]GitScheme{
		"http://":  GitSchemeHTTP,
		"https://": GitSchemeHTTPS,
		"file://":  GitSchemeLocal,
		"ssh://":   GitSchemeSSH,
	}
)

type gitRef struct {
	format            string
	path              string
	gitScheme         GitScheme
	gitRefName        git.RefName
	recurseSubmodules bool
}

func newGitRef(
	format string,
	path string,
	gitRefName git.RefName,
	recurseSubmodules bool,
) (*gitRef, error) {
	gitScheme, path, err := getGitSchemeAndPath(path)
	if err != nil {
		return nil, err
	}
	return buildGitRef(
		format,
		path,
		gitScheme,
		gitRefName,
		recurseSubmodules,
	), nil
}

func buildGitRef(
	format string,
	path string,
	gitScheme GitScheme,
	gitRefName git.RefName,
	recurseSubmodules bool,
) *gitRef {
	return &gitRef{
		format:            format,
		path:              path,
		gitScheme:         gitScheme,
		gitRefName:        gitRefName,
		recurseSubmodules: recurseSubmodules,
	}
}

func (r *gitRef) Format() string {
	return r.format
}

func (r *gitRef) Path() string {
	return r.path
}

func (r *gitRef) GitScheme() GitScheme {
	return r.gitScheme
}

func (r *gitRef) GitRefName() git.RefName {
	return r.gitRefName
}

func (r *gitRef) RecurseSubmodules() bool {
	return r.recurseSubmodules
}

func (*gitRef) ref()       {}
func (*gitRef) bucketRef() {}
func (*gitRef) gitRef()    {}

func getGitSchemeAndPath(path string) (GitScheme, string, error) {
	if path == "" {
		return 0, "", newNoPathError()
	}
	if path == "-" {
		return 0, "", newInvalidGitPathError(path)
	}
	if path == app.DevNullFilePath {
		return 0, "", newInvalidGitPathError(path)
	}
	for prefix, gitScheme := range gitSchemePrefixToGitScheme {
		if strings.HasPrefix(path, prefix) {
			path := strings.TrimPrefix(path, prefix)
			if gitScheme == GitSchemeLocal {
				path = normalpath.Normalize(path)
			}
			if path == "" {
				return 0, "", newNoPathError()
			}
			return gitScheme, path, nil
		}
	}
	if strings.Contains(path, "://") {
		return 0, "", newInvalidGitPathError(path)
	}
	return GitSchemeLocal, normalpath.Normalize(path), nil
}
