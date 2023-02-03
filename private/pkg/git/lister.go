package git

import (
	"context"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
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
	return stringutil.SliceToUniqueSortedSlice(
		filterIgnorePaths(
			stringSliceExcept(
				// This may not work in all Windows scenarios as we only split on "\n" but
				// this is no worse than we previously had.
				stringutil.SplitTrimLinesNoEmpty(string(allFilesOutput)),
				stringutil.SplitTrimLinesNoEmpty(string(deletedFilesOutput)),
			),
			options.IgnorePaths,
		),
	), nil
}

// stringSliceExcept returns all elements in source that are not in except.
func stringSliceExcept(source []string, except []string) []string {
	sourceMap := stringutil.SliceToMap(source)
	exceptMap := stringutil.SliceToMap(except)
	result := make([]string, 0, len(source))
	for s := range sourceMap {
		if _, ok := exceptMap[s]; !ok {
			result = append(result, s)
		}
	}
	return result
}

// filterIgnorePaths filters the files that contain any of the ignorePaths
// as a substring.
func filterIgnorePaths(files []string, ignorePaths []string) []string {
	if len(ignorePaths) == 0 {
		return files
	}
	unnormalizedIgnorePathMap := unnormalizedPathMap(ignorePaths)
	filteredFiles := make([]string, 0, len(files))
	for _, file := range files {
		if !fileMatches(file, unnormalizedIgnorePathMap) {
			filteredFiles = append(filteredFiles, file)
		}
	}
	return filteredFiles
}

// unnormalizedPathMap returns a map of the paths, but unnormalised.
//
// We return a map to remove duplicates easily.
func unnormalizedPathMap(paths []string) map[string]struct{} {
	unnormalizedPaths := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		unnormalizedPaths[normalpath.Unnormalize(path)] = struct{}{}
	}
	return unnormalizedPaths
}

// fileMatches returns true if any of the unnormalizedMatchPaths are
// a substring of the file.
func fileMatches(file string, unnormalizedMatchPaths map[string]struct{}) bool {
	for unnormalizedMatchPath := range unnormalizedMatchPaths {
		if strings.Contains(file, unnormalizedMatchPath) {
			return true
		}
	}
	return false
}
