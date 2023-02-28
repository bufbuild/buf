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

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const semverRegex = `((0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)`

func main() {
	versionPtr := flag.String("version", "", "the version number (required)")
	datePtr := flag.String("date", "", "the release date (optional, defaults to today)")
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Fprintln(os.Stderr, "usage: updatechangelog <release|unrelease> <filename.md>")
		os.Exit(1)
	}
	operation, filename := flag.Arg(0), flag.Arg(1)
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Could not read CHANGELOG.md")
		os.Exit(1)
	}
	switch operation {
	case "release":
		if *versionPtr == "" {
			fmt.Fprintln(os.Stderr, "Error: Please provide a version argument")
			os.Exit(1)
		}
		version := *versionPtr
		data = release(data, version, *datePtr)
	case "unrelease":
		data = unrelease(data)
	default:
		fmt.Fprintln(os.Stderr, "Error: usage: updatechangelog <release|unrelease> <filename.md>")
	}
	err = os.WriteFile("CHANGELOG.md", data, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Could not write to CHANGELOG.md")
		os.Exit(1)
	}
}

func release(data []byte, version string, date string) []byte {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	re := regexp.MustCompile(`## \[Unreleased\]`)
	newData := re.ReplaceAll(data, []byte(fmt.Sprintf("## [%s] - %s", version, date)))
	re = regexp.MustCompile(fmt.Sprintf(`\[Unreleased\].*?%s\.\.\.HEAD`, semverRegex))
	lastVersionFoo := re.FindStringSubmatch(string(newData))
	if len(lastVersionFoo) != 0 {
		lastVersion := lastVersionFoo[1]
		if lastVersion != "" {
			newData = re.ReplaceAll(newData, []byte(fmt.Sprintf("[%s]: https://github.com/bufbuild/buf/compare/v%s...%s", version, lastVersion, version)))
		}
	}
	return newData
}

func unrelease(data []byte) []byte {
	re := regexp.MustCompile(`# Changelog`)
	data = re.ReplaceAll(data, []byte(`# Changelog

## [Unreleased]

- No changes yet.`))
	lastLinkRe := regexp.MustCompile(fmt.Sprintf(`\[v%s\].*?v%s\.\.\.v%s`, semverRegex, semverRegex, semverRegex))
	lastVersions := lastLinkRe.FindStringSubmatch(string(data))
	data = []byte(
		strings.Replace(string(data),
			lastVersions[0],
			fmt.Sprintf(`[Unreleased]: https://github.com/bufbuild/buf/compare/v%s...HEAD
%s`, lastVersions[1], lastVersions[0]), 1))
	return data
}
