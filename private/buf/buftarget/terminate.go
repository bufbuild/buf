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

package buftarget

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// TerminateFunc is a termination function.
type TerminateFunc func(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error)

// TerminateAtControllingWorkspace implements a TerminateFunc and returns the workspace controlling
// the input, if one is found at the given prefix.
func TerminateAtControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	return terminateAtControllingWorkspace(ctx, bucket, prefix, originalInputPath)
}

// TerminateAtV1Module is a special terminate func that returns a controlling workspace with
// a v1 module confiugration if found at the given prefix.
func TerminateAtV1Module(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	return terminateAtV1Module(ctx, bucket, prefix, originalInputPath)
}

// *** PRIVATE ***

func terminateAtControllingWorkspace(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	bufWorkYAMLFile, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufWorkYAMLExists := err == nil
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	bufYAMLExists := err == nil
	if bufWorkYAMLExists && bufYAMLExists {
		// This isn't actually the external directory path, but we do the best we can here for now.
		return nil, fmt.Errorf("cannot have a buf.work.yaml and buf.yaml in the same directory %q", prefix)
	}
	relInputPath, err := normalpath.Rel(prefix, originalInputPath)
	if err != nil {
		return nil, err
	}
	if bufYAMLExists && bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
		// A input directory with a v2 buf.yaml is the controlling workspace for itself.
		if prefix == originalInputPath {
			return newControllingWorkspace(prefix, nil, bufYAMLFile), nil
		}
		for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
			// For a prefix/buf.yaml with:
			//  version: v2
			//  modules:
			//    - path: foo
			//    - path: dir/bar
			//    - path: dir/baz
			//  ...
			// - If the input is a module path (one of prefix/foo, prefix/dir/bar or prefix/dir/baz),
			//   then the input is a module controlled by the workspace at prefix.
			// - If the input is inside one of the module DirPaths (e.g. prefix/foo/suffix or prefix/dir/bar/suffix)
			//   we still consider prefix to be the workspace that controls the input. It is then up
			//   to the caller to decide what to do with this information. For example, the caller could
			//   say this is equivalent to input being prefix/foo with --path=prefix/foo/suffix specified,
			//   or it could say this is invalid, or the caller is not be concerned with validity.
			if normalpath.EqualsOrContainsPath(moduleConfig.DirPath(), relInputPath, normalpath.Relative) {
				return newControllingWorkspace(prefix, nil, bufYAMLFile), nil
			}
			// Only in v2: if the input is not any of the module paths but contains a module path,
			// e.g. prefix/dir, we also consider prefix to be the controlling workspace, because in v2
			// an input is allowed to be a subset of a workspace's modules. In this example, input prefix/dir
			// is two modules, one at prefix/dir/bar and the other at prefix/dir/baz.
			if normalpath.EqualsOrContainsPath(relInputPath, moduleConfig.DirPath(), normalpath.Relative) {
				return newControllingWorkspace(prefix, nil, bufYAMLFile), nil
			}
		}
	}
	if bufWorkYAMLExists {
		// A input directory with a buf.work.yaml is the controlling workspace for itself.
		if prefix == originalInputPath {
			return newControllingWorkspace(prefix, bufWorkYAMLFile, nil), nil
		}
		for _, dirPath := range bufWorkYAMLFile.DirPaths() {
			// Unlike v2 workspaces, we only check whether the input is a module path or is contained
			// in a module path.
			if normalpath.EqualsOrContainsPath(dirPath, relInputPath, normalpath.Relative) {
				return newControllingWorkspace(prefix, bufWorkYAMLFile, nil), nil
			}
		}
	}
	return nil, nil
}

func terminateAtV1Module(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
	originalInputPath string,
) (ControllingWorkspace, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, prefix)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && bufYAMLFile.FileVersion() == bufconfig.FileVersionV1 {
		return newControllingWorkspace(prefix, nil, bufYAMLFile), nil
	}
	return nil, nil
}
