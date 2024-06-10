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

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/diff"
)

// DiffOption is an option for Diff.
type DiffOption func(*diffOptions)

// DiffWithSuppressCommands returns a new DiffOption that suppresses printing of commands.
func DiffWithSuppressCommands() DiffOption {
	return func(diffOptions *diffOptions) {
		diffOptions.suppressCommands = true
	}
}

// DiffWithSuppressCommands returns a new DiffOption that suppresses printing of timestamps.
func DiffWithSuppressTimestamps() DiffOption {
	return func(diffOptions *diffOptions) {
		diffOptions.suppressTimestamps = true
	}
}

// DiffWithExternalPaths returns a new DiffOption that prints diffs with external paths
// instead of paths.
func DiffWithExternalPaths() DiffOption {
	return func(diffOptions *diffOptions) {
		diffOptions.externalPaths = true
	}
}

// DiffWithExternalPathPrefixes returns a new DiffOption that sets the external path prefixes for the buckets.
//
// If a file is in one bucket but not the other, it will be assumed that the file begins
// with the given prefix, and this prefix should be substituted for the other prefix.
//
// For example, if diffing the directories "test/a" and "test/b", use "test/a/" and "test/b/",
// and a file that is in one with path "test/a/foo.txt" will be shown as not
// existing as "test/b/foo.txt" in two.
//
// Note that the prefixes are directly concatenated, so "/" should be included generally.
//
// This option has no effect if DiffWithExternalPaths is not set.
// This option is not required if the prefixes are equal.
func DiffWithExternalPathPrefixes(
	oneExternalPathPrefix string,
	twoExternalPathPrefix string,
) DiffOption {
	return func(diffOptions *diffOptions) {
		if oneExternalPathPrefix != twoExternalPathPrefix {
			// we don't know if external paths are file paths or not
			// so we just operate on pure string-prefix paths
			// this comes up with for example s3://
			diffOptions.oneExternalPathPrefix = oneExternalPathPrefix
			diffOptions.twoExternalPathPrefix = twoExternalPathPrefix
		}
	}
}

// DiffWithTransform returns a DiffOption that adds a transform function. The transform function will be run on each
// file being compared before it is diffed. transform takes the arguments:
//
//	side: one or two whether it is the first or second item in the diff
//	filename: the filename including path
//	content: the file content.
//
// transform returns a string that is the transformed content of filename.
//
// TODO: this needs to be refactored or removed, especially the implicit side enum.
// Perhaps provide a transform function for a given bucket and apply it there.
func DiffWithTransform(
	transform func(side string, filename string, content []byte) []byte,
) DiffOption {
	return func(diffOptions *diffOptions) {
		diffOptions.transforms = append(diffOptions.transforms, transform)
	}
}

// DiffBytes does a diff of the ReadBuckets.
func DiffBytes(
	ctx context.Context,
	runner command.Runner,
	one ReadBucket,
	two ReadBucket,
	options ...DiffOption,
) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := Diff(ctx, runner, buffer, one, two, options...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Diff writes a diff of the ReadBuckets to the Writer.
func Diff(
	ctx context.Context,
	runner command.Runner,
	writer io.Writer,
	one ReadBucket,
	two ReadBucket,
	options ...DiffOption,
) error {
	_, err := DiffWithFilenames(ctx, runner, writer, one, two, options...)
	return err
}

// DiffWithFilenames writes a diff of the ReadBuckets to the Writer and returns
// the names of any file paths that contained differences. The returned paths
// are in sorted (ascending) order.
//
// Note that the returned paths are determined by comparing the before and after
// bytes, not just based on whether the configured diff tool reports something.
// This can be used to avoid re-writing files whose contents don't actually need
// to change.
func DiffWithFilenames(
	ctx context.Context,
	runner command.Runner,
	writer io.Writer,
	one ReadBucket,
	two ReadBucket,
	options ...DiffOption,
) ([]string, error) {
	diffOptions := newDiffOptions()
	for _, option := range options {
		option(diffOptions)
	}
	externalPaths := diffOptions.externalPaths
	oneExternalPathPrefix := diffOptions.oneExternalPathPrefix
	twoExternalPathPrefix := diffOptions.twoExternalPathPrefix

	oneObjectInfos, err := allObjectInfos(ctx, one, "")
	if err != nil {
		return nil, err
	}
	twoObjectInfos, err := allObjectInfos(ctx, two, "")
	if err != nil {
		return nil, err
	}
	sortObjectInfos(oneObjectInfos)
	sortObjectInfos(twoObjectInfos)
	onePathToObjectInfo := pathToObjectInfo(oneObjectInfos)
	twoPathToObjectInfo := pathToObjectInfo(twoObjectInfos)
	var changedPaths []string

	for _, oneObjectInfo := range oneObjectInfos {
		path := oneObjectInfo.Path()
		oneDiffPath, err := getDiffPathForObjectInfo(
			oneObjectInfo,
			externalPaths,
			oneExternalPathPrefix,
		)
		if err != nil {
			return nil, err
		}
		oneData, err := ReadPath(ctx, one, path)
		if err != nil {
			return nil, err
		}
		var twoData []byte
		var twoDiffPath string
		if twoObjectInfo, ok := twoPathToObjectInfo[path]; ok {
			twoData, err = ReadPath(ctx, two, path)
			if err != nil {
				return nil, err
			}
			twoDiffPath, err = getDiffPathForObjectInfo(
				twoObjectInfo,
				externalPaths,
				twoExternalPathPrefix,
			)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(oneData, twoData) {
				changedPaths = append(changedPaths, path)
			}
		} else {
			changedPaths = append(changedPaths, path)
			twoDiffPath, err = getDiffPathForNotFound(
				oneObjectInfo,
				externalPaths,
				oneExternalPathPrefix,
				twoExternalPathPrefix,
			)
			if err != nil {
				return nil, err
			}
		}
		for _, transform := range diffOptions.transforms {
			oneData = transform("one", oneDiffPath, oneData)
			twoData = transform("two", twoDiffPath, twoData)
		}
		diffData, err := diff.Diff(
			ctx,
			runner,
			oneData,
			twoData,
			oneDiffPath,
			twoDiffPath,
			diffOptions.toDiffPackageOptions()...,
		)
		if err != nil {
			return nil, err
		}
		if len(diffData) > 0 {
			if _, err := writer.Write(diffData); err != nil {
				return nil, err
			}
		}
	}
	for _, twoObjectInfo := range twoObjectInfos {
		path := twoObjectInfo.Path()
		if _, ok := onePathToObjectInfo[path]; !ok {
			changedPaths = append(changedPaths, path)
			twoData, err := ReadPath(ctx, two, path)
			if err != nil {
				return nil, err
			}
			oneDiffPath, err := getDiffPathForNotFound(
				twoObjectInfo,
				externalPaths,
				twoExternalPathPrefix,
				oneExternalPathPrefix,
			)
			if err != nil {
				return nil, err
			}
			twoDiffPath, err := getDiffPathForObjectInfo(
				twoObjectInfo,
				externalPaths,
				twoExternalPathPrefix,
			)
			if err != nil {
				return nil, err
			}
			diffData, err := diff.Diff(
				ctx,
				runner,
				nil,
				twoData,
				oneDiffPath,
				twoDiffPath,
				diffOptions.toDiffPackageOptions()...,
			)
			if err != nil {
				return nil, err
			}
			if len(diffData) > 0 {
				if _, err := writer.Write(diffData); err != nil {
					return nil, err
				}
			}
		}
	}
	// changedPaths will be *mostly* sorted. But paths in "two" that were not present
	// in "one" will appear last, even if sort order would have them interleaved.
	// So we must sort explicitly.
	sort.Strings(changedPaths)
	return changedPaths, nil
}

func getDiffPathForObjectInfo(
	objectInfo ObjectInfo,
	externalPaths bool,
	externalPathPrefix string,
) (string, error) {
	if !externalPaths {
		return objectInfo.Path(), nil
	}
	externalPath := objectInfo.ExternalPath()
	if externalPathPrefix == "" {
		return externalPath, nil
	}
	if !strings.HasPrefix(externalPath, externalPathPrefix) {
		return "", fmt.Errorf("diff: expected %s to have prefix %s", externalPath, externalPathPrefix)
	}
	return externalPath, nil
}

func getDiffPathForNotFound(
	foundObjectInfo ObjectInfo,
	externalPaths bool,
	foundExternalPathPrefix string,
	notFoundExternalPathPrefix string,
) (string, error) {
	if !externalPaths {
		return foundObjectInfo.Path(), nil
	}
	externalPath := foundObjectInfo.ExternalPath()
	switch {
	case foundExternalPathPrefix == "" && notFoundExternalPathPrefix == "":
		// no prefix, just return external path
		return externalPath, nil
	case foundExternalPathPrefix == "" && notFoundExternalPathPrefix != "":
		// the not-found side has a prefix, append the external path to this prefix, and we're done
		return notFoundExternalPathPrefix + externalPath, nil
	default:
		//foundExternalPathPrefix != "" && notFoundExternalPathPrefix == ""
		//foundExternalPathPrefix != "" && notFoundExternalPathPrefix != ""
		if !strings.HasPrefix(externalPath, foundExternalPathPrefix) {
			return "", fmt.Errorf("diff: expected %s to have prefix %s", externalPath, foundExternalPathPrefix)
		}
		return notFoundExternalPathPrefix + strings.TrimPrefix(externalPath, foundExternalPathPrefix), nil
	}
}

type diffOptions struct {
	suppressCommands      bool
	suppressTimestamps    bool
	externalPaths         bool
	oneExternalPathPrefix string
	twoExternalPathPrefix string
	transforms            []func(side string, filename string, content []byte) []byte
}

func newDiffOptions() *diffOptions {
	return &diffOptions{}
}

func (d *diffOptions) toDiffPackageOptions() []diff.DiffOption {
	var diffPackageOptions []diff.DiffOption
	if d.suppressCommands {
		diffPackageOptions = append(diffPackageOptions, diff.DiffWithSuppressCommands())
	}
	if d.suppressTimestamps {
		diffPackageOptions = append(diffPackageOptions, diff.DiffWithSuppressTimestamps())
	}
	return diffPackageOptions
}
