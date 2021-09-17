// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmodule

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

var _ ModuleFileSet = &moduleFileSet{}

type moduleFileSet struct {
	Module

	allModuleReadBucket moduleReadBucket

	// Optional; configured when there is at least one
	// target path (i.e. --path) specified by the user.
	targetPaths              [][]string
	targetPathsAllowNotExist bool
}

func newModuleFileSet(
	module Module,
	dependencies []Module,
	options ...ModuleFileSetOption,
) *moduleFileSet {
	// TODO: We can remove the getModuleRef method on the
	// Module type if we fetch FileInfos from the Module
	// and plumb in the ModuleRef here.
	//
	// This approach assumes that all of the FileInfos returned
	// from SourceFileInfos will have their ModuleRef
	// set to the same value. That can be enforced here.
	moduleReadBuckets := []moduleReadBucket{
		newSingleModuleReadBucket(
			module.getSourceReadBucket(),
			module.getModuleIdentity(),
			module.getCommit(),
		),
	}
	for _, dependency := range dependencies {
		moduleReadBuckets = append(
			moduleReadBuckets,
			newSingleModuleReadBucket(
				dependency.getSourceReadBucket(),
				dependency.getModuleIdentity(),
				dependency.getCommit(),
			),
		)
	}
	moduleFileSet := &moduleFileSet{
		Module:              module,
		allModuleReadBucket: newMultiModuleReadBucket(moduleReadBuckets...),
	}
	for _, option := range options {
		option(moduleFileSet)
	}
	return moduleFileSet
}

func (m *moduleFileSet) AllFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error) {
	var fileInfos []bufmoduleref.FileInfo
	if walkErr := m.allModuleReadBucket.WalkModuleFiles(ctx, "", func(moduleObjectInfo *moduleObjectInfo) error {
		if err := bufmoduleref.ValidateModuleFilePath(moduleObjectInfo.Path()); err != nil {
			return err
		}
		isNotImport, err := storage.Exists(ctx, m.Module.getSourceReadBucket(), moduleObjectInfo.Path())
		if err != nil {
			return err
		}
		fileInfo, err := bufmoduleref.NewFileInfo(
			moduleObjectInfo.Path(),
			moduleObjectInfo.ExternalPath(),
			!isNotImport,
			moduleObjectInfo.ModuleIdentity(),
			moduleObjectInfo.Commit(),
		)
		if err != nil {
			return err
		}
		fileInfos = append(fileInfos, fileInfo)
		return nil
	}); walkErr != nil {
		return nil, walkErr
	}
	bufmoduleref.SortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *moduleFileSet) GetModuleFile(ctx context.Context, path string) (ModuleFile, error) {
	if err := bufmoduleref.ValidateModuleFilePath(path); err != nil {
		return nil, err
	}
	readObjectCloser, err := m.allModuleReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	isNotImport, err := storage.Exists(ctx, m.Module.getSourceReadBucket(), path)
	if err != nil {
		return nil, err
	}
	moduleObjectInfo, err := m.allModuleReadBucket.StatModuleFile(ctx, path)
	if err != nil {
		return nil, err
	}
	fileInfo, err := bufmoduleref.NewFileInfo(
		readObjectCloser.Path(),
		readObjectCloser.ExternalPath(),
		!isNotImport,
		moduleObjectInfo.ModuleIdentity(),
		moduleObjectInfo.Commit(),
	)
	if err != nil {
		return nil, err
	}
	return newModuleFile(fileInfo, readObjectCloser), nil
}

func (m *moduleFileSet) TargetFileInfos(ctx context.Context) ([]bufmoduleref.FileInfo, error) {
	if len(m.targetPaths) == 0 {
		// If we haven't configured any target targetPaths on the MpduleFileSet,
		// we can defer to the target module's implementation.
		return m.Module.TargetFileInfos(ctx)
	}
	allFileInfos, err := m.AllFileInfos(ctx)
	if err != nil {
		return nil, err
	}
	// At this point, we have a list of sets, where at least one
	// of the paths in each set needs to be matched.
	//
	// Some of these paths might actually be directories, so we need
	// to consider those paths matched if we find at least one file
	// that matches it.
	//
	// For example,
	//
	//  [][]string{
	//    {"foo/bar.proto", "acme/foo/bar.proto"}
	//    {"bar", "acme/bar"}
	//  }
	//
	// If we encounter "bar/baz.proto", then we will have satisfied
	// the second set (because that path satisfies the "bar" directory).
	pathTracker := newPathTracker(m.targetPaths)
	targetFileInfos := make([]bufmoduleref.FileInfo, 0, len(m.targetPaths))
	for _, fileInfo := range allFileInfos {
		if pathTracker.contains(fileInfo.Path()) {
			targetFileInfos = append(targetFileInfos, fileInfo)
			pathTracker.mark(fileInfo.Path())
		}
	}
	if !m.targetPathsAllowNotExist {
		for path := range pathTracker.paths {
			if _, ok := pathTracker.seen[path]; !ok {
				// No match, this is an error given that allowNotExist is false.
				return nil, fmt.Errorf(`path "%s" has no matching file in the module`, pathTracker.format(path))
			}
		}
	}
	return targetFileInfos, nil
}

// pathTracker tracks all of the paths used for a ModuleFileSet.
// If one of the paths in a set is used, all of its associated links
// are marked.
type pathTracker struct {
	// Includes all of the paths in a single set.
	paths map[string]struct{}

	// Associates all of the paths in the same set.
	links map[string][]string

	// Tracks which paths have been seen.
	seen map[string]struct{}

	// Maps each path back to its original form,
	// which is the first path in the set.
	original map[string]string
}

// newPathTracker returns a new pathTracker suitable for determining
// if the target paths configured for a ModuleFileSet are matched.
func newPathTracker(targetPaths [][]string) *pathTracker {
	paths := make(map[string]struct{})
	links := make(map[string][]string)
	original := make(map[string]string)
	for _, set := range targetPaths {
		for i := 0; i < len(set); i++ {
			for j := 0; j < len(set); j++ {
				if i != j {
					links[set[i]] = append(links[set[i]], set[j])
				}
				original[set[j]] = set[0]
			}
			paths[set[i]] = struct{}{}
		}
	}
	return &pathTracker{
		paths:    paths,
		links:    links,
		original: original,
		seen:     make(map[string]struct{}),
	}
}

// contains returns true if this path has at least one matching
// path in the union.
func (d *pathTracker) contains(path string) bool {
	return len(normalpath.MapAllEqualOrContainingPathMap(d.paths, path, normalpath.Relative)) > 0
}

// format returns first path in its set. This is primarily
// used to generate deterministic error messages as we range
// over the paths in each set.
func (d *pathTracker) format(path string) string {
	value, ok := d.original[path]
	if !ok {
		return path
	}
	return value
}

// mark marks all of the paths associated with the given path as seen.
func (d *pathTracker) mark(path string) {
	// This path could represent either a file or a directory,
	// so we need to mark each of the potential files and directories
	// as seen.
	matchingPaths := normalpath.MapAllEqualOrContainingPathMap(d.paths, path, normalpath.Relative)
	for matchingPath := range matchingPaths {
		d.seen[matchingPath] = struct{}{}
		for _, matchingLink := range d.links[matchingPath] {
			d.seen[matchingLink] = struct{}{}
		}
	}
}

func (*moduleFileSet) isModuleFileSet() {}
