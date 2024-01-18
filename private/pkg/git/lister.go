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
	"os"
	"regexp"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

type lister struct {
	runner command.Runner
}

func newLister(runner command.Runner) *lister {
	return &lister{
		runner: runner,
	}
}

func (l *lister) ListFilesAndUnstagedFiles(
	ctx context.Context,
	container app.EnvStdioContainer,
	options ListFilesAndUnstagedFilesOptions,
) ([]string, error) {
	allFilesOutput, err := command.RunStdout(
		ctx,
		container,
		l.runner,
		"git",
		"ls-files",
		"--cached",
		"--modified",
		"--others",
		"--exclude-standard",
	)
	if err != nil {
		return nil, err
	}
	deletedFilesOutput, err := command.RunStdout(
		ctx,
		container,
		l.runner,
		"git",
		"ls-files",
		"--deleted",
	)
	if err != nil {
		return nil, err
	}
	return slicesext.ToUniqueSorted(
		filterNonRegularFiles(
			stringSliceExceptMatches(
				stringSliceExcept(
					// This may not work in all Windows scenarios as we only split on "\n" but
					// this is no worse than we previously had.
					stringutil.SplitTrimLinesNoEmpty(string(allFilesOutput)),
					stringutil.SplitTrimLinesNoEmpty(string(deletedFilesOutput)),
				),
				options.IgnorePathRegexps,
			),
		),
	), nil
}

// stringSliceExcept returns all elements in source that are not in except.
func stringSliceExcept(source []string, except []string) []string {
	exceptMap := slicesext.ToStructMap(except)
	result := make([]string, 0, len(source))
	for _, s := range source {
		if _, ok := exceptMap[s]; !ok {
			result = append(result, s)
		}
	}
	return result
}

// stringSliceExceptMatches returns all elements in source that do not match
// any of the regexps.
func stringSliceExceptMatches(source []string, regexps []*regexp.Regexp) []string {
	if len(regexps) == 0 {
		return source
	}
	result := make([]string, 0, len(source))
	for _, s := range source {
		if !matchesAny(s, regexps) {
			result = append(result, s)
		}
	}
	return result
}

// matchesAny returns true if any of regexps match.
func matchesAny(s string, regexps []*regexp.Regexp) bool {
	for _, regexp := range regexps {
		if regexp.MatchString(s) {
			return true
		}
	}
	return false
}

// filterNonRegularFiles returns all regular files.
//
// This does an os.Stat call, so the files must exist for this to work.
// Given our usage here, this is true by the time this function is called.
func filterNonRegularFiles(files []string) []string {
	filteredFiles := make([]string, 0, len(files))
	for _, file := range files {
		if fileInfo, err := os.Stat(file); err == nil && fileInfo.Mode().IsRegular() {
			filteredFiles = append(filteredFiles, file)
		}
	}
	return filteredFiles
}
