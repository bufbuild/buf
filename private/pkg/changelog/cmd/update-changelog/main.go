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
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/spf13/pflag"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	semverRegex = `((0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)`
	name        = "update-changelog"
	timeout     = 120 * time.Second
)

func main() {
	appcmd.Main(context.Background(), newCommand())
}

func newCommand() *appcmd.Command {
	builder := appflag.NewBuilder(
		name,
		appflag.BuilderWithTimeout(timeout),
		appflag.BuilderWithTracing(),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use: name,
		SubCommands: []*appcmd.Command{
			{Use: "unrelease", Run: builder.NewRunFunc(func(ctx context.Context, container appflag.Container) error {
				return unrelease(ctx, container)
			})},
			{Use: "release", Run: builder.NewRunFunc(func(ctx context.Context, container appflag.Container) error {
				return release(ctx, container, flags)
			}),
			},
		},
		BindPersistentFlags: builder.BindRoot,
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
	var today = time.Now().Format("2006-01-02")
	flagSet.StringVarP(&f.version, "version", "v", "", "the release version (required for release operation)")
	flagSet.StringVarP(&f.date, "date", "d", today, "the release date in YYYY-MM-DD (optional, defaults to today if not supplied)")
}

func unrelease(_ context.Context, container appflag.Container) error {
	filename := container.Arg(1)
	if filename == "" {
		filename = "CHANGELOG.md"
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return errors.New("error: Could not read file")
	}
	re := regexp.MustCompile(`# Changelog`)
	repoURL := getRepoURL(data)
	data = re.ReplaceAll(data, []byte(`# Changelog

## [Unreleased]

- No changes yet.`))
	lastLinkRegexp := regexp.MustCompile(fmt.Sprintf(`\[v%s\].*?v%s\.\.\.v%s`, semverRegex, semverRegex, semverRegex))
	lastVersions := lastLinkRegexp.FindStringSubmatch(string(data))
	data = []byte(
		strings.Replace(string(data),
			lastVersions[0],
			fmt.Sprintf(`[Unreleased]: %s/compare/v%s...HEAD
%s`, repoURL, lastVersions[1], lastVersions[0]), 1))
	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return errors.New("error: Could not write to file")
	}
	return nil
}

func release(_ context.Context, container appflag.Container, flags *updateChangelogReleaseFlags) error {
	filename := container.Arg(1)
	if filename == "" {
		filename = "CHANGELOG.md"
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return errors.New("error: Could not read file")
	}
	if flags.version == "" {
		return errors.New("error: Please provide a version flag")
	}
	repoURL := getRepoURL(data)
	re := regexp.MustCompile(`## \[Unreleased\]`)
	newData := re.ReplaceAll(data, []byte(fmt.Sprintf("## [%s] - %s", flags.version, flags.date)))
	re = regexp.MustCompile(fmt.Sprintf(`\[Unreleased\]: (.*?)v%s\.\.\.HEAD`, semverRegex))
	lastVersionFoo := re.FindStringSubmatch(string(newData))
	if len(lastVersionFoo) != 0 {
		lastVersion := lastVersionFoo[2]
		if lastVersion != "" {
			newData = re.ReplaceAll(newData, []byte(fmt.Sprintf("[%s]: %s/compare/v%s...%s", flags.version, repoURL, lastVersion, flags.version)))
		}
	}
	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return errors.New("error: Could not write to file")
	}
	return nil
}

func getRepoURL(data []byte) string {
	re := regexp.MustCompile(`\[.*?]: (.*?)\/compare`)
	newData := re.FindStringSubmatch(string(data))
	if len(newData) == 0 {
		return ""
	}
	return newData[1]
}
