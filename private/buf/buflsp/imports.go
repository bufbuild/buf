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

// This file defines wrappers over Buf CLI constructs for searching for imports.

package buflsp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// configs is the result of findConfigs(). See that function for more information.
type configs struct {
	yaml         bufconfig.BufYAMLFile
	dir          protocol.URI
	workspace    bufconfig.BufWorkYAMLFile
	workspaceDir protocol.URI
}

// findConfigs finds the Buf configuration files in-scope for the given URI.
func findConfigs(
	ctx context.Context,
	bucket storage.ReadBucket,
	root string,
	logger *zap.Logger,
	uri protocol.URI,
) (*configs, error) {
	// Dismantle the input URI into a path that we can search in the bucket for.
	searchIn, err := normalpath.Rel(root, uri.Filename())
	if err != nil {
		return nil, err
	}
	searchIn = normalpath.Dir(searchIn)

	configs := new(configs)
	var dir, workspaceDir string
	for {
		if configs.yaml == nil {
			yaml, err := bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, searchIn)
			if !errors.Is(err, fs.ErrNotExist) {
				if err != nil {
					return nil, err
				}

				configs.yaml = yaml
				dir = searchIn
				if configs.yaml.FileVersion() == bufconfig.FileVersionV2 {
					break
				}
			}
		}

		yaml, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, bucket, searchIn)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return nil, err
			}

			configs.workspace = yaml
			workspaceDir = searchIn
			break
		}

		searchIn = normalpath.Dir(searchIn)
		if searchIn == "." {
			break
		}
	}

	// Make the config directories into absolute URIs.
	if dir != "" {
		configs.dir = protocol.URI("file://" + normalpath.Join(root, dir))
		logger.Sugar().Debugf("found buf.yaml for %q in %q", uri, configs.dir)
	}
	if workspaceDir != "" {
		configs.workspaceDir = protocol.URI("file://" + normalpath.Join(root, workspaceDir))
		logger.Sugar().Debugf("found buf.work.yaml for %q in %q", uri, configs.workspaceDir)
	}

	return configs, nil
}

// findImportable finds all files that can potentially be imported by the proto file at
// uri. This returns a map from potential Protobuf import path to the URI of the file it would import.
//
// Note that this performs no validation on these files, because those files might be open in the
// editor and might contain invalid syntax at the moment. We only want to get their paths and nothing
// more.
func findImportable(
	ctx context.Context,
	bucket storage.ReadBucket,
	root string,
	logger *zap.Logger,
	uri protocol.URI,
) (map[string]protocol.URI, error) {
	configs, err := findConfigs(ctx, bucket, root, logger, uri)
	if err != nil {
		return nil, fmt.Errorf("could not find buf workspace over %v", uri)
	}

	imports := make(map[string]protocol.URI)
	if configs.yaml != nil && configs.yaml.FileVersion() == bufconfig.FileVersionV2 {
		for _, moduleConf := range configs.yaml.ModuleConfigs() {
			// All of the .proto files in the module live under moduleConf.DirPath(), so
			// we join with dir to make it absolute.
			moduleRoot := normalpath.Join(configs.dir.Filename(), moduleConf.DirPath())

			var walker fs.WalkDirFunc = func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// Silently discard any permission errors and resume walking.
					if errors.Is(err, fs.ErrPermission) {
						return nil
					}

					return err
				}

				// path is absolute, but we want the path relative to the module root above.
				modulePath, err := normalpath.Rel(moduleRoot, path)
				if err != nil {
					// This should never happen. If it does, we should just skip this directory.
					logger.Sugar().Warnf("encountered path %q not relative to %q while building module", path, moduleRoot)
					return fs.SkipDir
				}

				if !d.IsDir() && strings.HasSuffix(d.Name(), ".proto") {
					imports[modulePath] = protocol.URI(fmt.Sprintf("file://%s", path))
					return nil
				}

				// Everything in RootToExcludes is relative to the module root, so we
				// search for rel therein.
				if slices.Contains(moduleConf.RootToExcludes()["."], modulePath) {
					// If we find an excluded directory, we skip the whole thing.
					return fs.SkipDir
				}

				return nil
			}

			includes := moduleConf.RootToIncludes()["."]
			if len(includes) > 0 {
				for _, include := range includes {
					// Like RootToExcludes, everything here is relative to the module root,
					// so to walk it we need to make it absolute.
					path := normalpath.Join(moduleRoot, include)
					if err := filepath.WalkDir(path, walker); err != nil {
						return nil, err
					}
				}
			} else {
				if err := filepath.WalkDir(moduleRoot, walker); err != nil {
					return nil, err
				}
			}
		}
	}

	return imports, nil
}
