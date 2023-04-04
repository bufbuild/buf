// Copyright 2020-2023 Buf Technologies, Inc.
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

// Package main supplies the update-changelog command that updates the CHANGELOG.md. The tool accepts two operations: "release" and "unrelease".
// "update-changelog release" requires a filename argument (default CHANGELOG.md), a --version flag in the form vx.y.z, an optional --date flag.
// If no date is supplied the current date will be used.
// "update-changelog unrelease" does not require any flags or arguments except for the optional filename and will add `Unreleased` sections to the changelog.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	semverRegex         = `((0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)`
	updateChangelogName = "update-changelog"
)

var (
	lastLinkRegexp         = regexp.MustCompile(fmt.Sprintf(`\[v%s\].*?v%s\.\.\.v%s`, semverRegex, semverRegex, semverRegex))
	headerRegexp           = regexp.MustCompile(`# Changelog`)
	unreleasedHeaderRegexp = regexp.MustCompile(fmt.Sprintf(`\[Unreleased\]: (.*?)v%s\.\.\.HEAD`, semverRegex))
	unreleasedLinkRegexp   = regexp.MustCompile(`## \[Unreleased\]`)
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appflag.NewBuilder(updateChangelogName)
	flags := newFlags()
	return &appcmd.Command{
		Use: updateChangelogName,
		SubCommands: []*appcmd.Command{
			{
				Use:   "unrelease <changelog>",
				Short: "Adds an Unreleased section to the changelog",
				Run: builder.NewRunFunc(func(ctx context.Context, container appflag.Container) error {
					return unrelease(container)
				}),
				Args: cobra.ExactArgs(1),
			},
			{
				Use:   "release <changelog> --version=<version>",
				Short: "Adds a new release section to the changelog",
				Run: builder.NewRunFunc(func(_ context.Context, container appflag.Container) error {
					return release(container, flags)
				}),
				BindFlags: flags.Bind,
				Args:      cobra.ExactArgs(1),
			},
		},
	}
}

type updateChangelogReleaseFlags struct {
	version string
	date    string
}

func newFlags() *updateChangelogReleaseFlags {
	return &updateChangelogReleaseFlags{}
}

func (f *updateChangelogReleaseFlags) Bind(flagSet *pflag.FlagSet) {
	today := time.Now().Format("2006-01-02")
	flagSet.StringVarP(&f.version, "version", "", "", "The release version (required for release operation)")
	flagSet.StringVarP(&f.date, "date", "", today, "The release date in YYYY-MM-DD (optional, defaults to today if not supplied)")
}

// unrelease adds an Unreleased section to the changelog file.
// It also updates the Unreleased link at the bottom of the file.
// $ update-changelog unrelease
// +## [Unreleased]
// +
// +- No changes yet.
// +
// ...
// +[Unreleased]: https://github.com/bufbuild/buf/compare/v1.17.0...HEAD
func unrelease(container appflag.Container) error {
	filename, contents, err := changelogFile(container)
	if err != nil {
		return err
	}
	repoURL := getRepoURL(contents)
	contents = headerRegexp.ReplaceAll(contents, []byte(`# Changelog

## [Unreleased]

- No changes yet.`))
	lastVersions := lastLinkRegexp.FindStringSubmatch(string(contents))
	if len(lastVersions) < 2 {
		return errors.New("error: Could not find last release version")
	}
	contents = []byte(
		strings.Replace(string(contents),
			lastVersions[0],
			fmt.Sprintf(`[Unreleased]: %s/compare/v%s...HEAD
%s`, repoURL, lastVersions[1], lastVersions[0]), 1))
	err = os.WriteFile(filename, contents, 0o600)
	if err != nil {
		return errors.New("error: Could not write to file")
	}
	return nil
}

// release adds a new release section to the changelog.
// It updates the Unreleased section to the new version and adds a new Unreleased section.
// It also updates the Unreleased link at the bottom of the file.
// $ update-changelog release --version=1.17.0
// -## [Unreleased]
// +## [v1.17.0] - 2023-04-04
// ...
// -[Unreleased]: https://github.com/bufbuild/buf/compare/v1.16.0...HEAD
// +[v1.17.0]: https://github.com/bufbuild/buf/compare/v1.16.0...v1.17.0
func release(container appflag.Container, flags *updateChangelogReleaseFlags) error {
	if flags.version == "" {
		return errors.New("error: Please provide a version flag")
	}
	filename, oldContents, err := changelogFile(container)
	if err != nil {
		return err
	}
	repoURL := getRepoURL(oldContents)
	newContents := unreleasedLinkRegexp.ReplaceAll(oldContents, []byte(fmt.Sprintf("## [%s] - %s", flags.version, flags.date)))
	if lastVersionMatches := unreleasedHeaderRegexp.FindStringSubmatch(string(newContents)); len(lastVersionMatches) != 0 {
		lastVersion := lastVersionMatches[2]
		if lastVersion != "" {
			newContents = unreleasedHeaderRegexp.ReplaceAll(newContents, []byte(fmt.Sprintf("[%s]: %s/compare/v%s...%s", flags.version, repoURL, lastVersion, flags.version)))
		}
	}
	err = os.WriteFile(filename, newContents, 0o600)
	if err != nil {
		return errors.New("error: Could not write to file")
	}
	return nil
}

// getRepoURL returns the repo URL from the changelog file
// for example, if the changelog file has the following line:
// [v1.99.0]: https://github.com/bufbuild/buf/compare/v1.16.0...v1.99.0
// then this function will return "https://github.com/bufbuild/buf"
func getRepoURL(data []byte) string {
	repoRegexp := regexp.MustCompile(`\[.*?]: (.*?)\/compare`)
	newData := repoRegexp.FindStringSubmatch(string(data))
	if len(newData) == 0 {
		return ""
	}
	return newData[1]
}

// changelogFile returns the contents of the changelog file from a appflag.Container.
func changelogFile(container appflag.Container) (string, []byte, error) {
	filename := container.Arg(0)
	if filename == "" {
		filename = "CHANGELOG.md"
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", nil, errors.New("error: Could not read file")
	}
	return filename, data, err
}
