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

package bufworkspace

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type workspace struct {
	bufmodule.ModuleSet

	opaqueIDToLintConfig     map[string]bufconfig.LintConfig
	opaqueIDToBreakingConfig map[string]bufconfig.BreakingConfig
	generateConfigs          []bufconfig.GenerateConfig
	configuredDepModuleRefs  []bufmodule.ModuleRef
	lockedDepModuleKeys      []bufmodule.ModuleKey
}

func newWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	options ...WorkspaceOption,
) (*workspace, error) {
	workspaceOptions := newWorkspaceOptions()
	for _, option := range options {
		option(workspaceOptions)
	}
	// This also automatically makes "" into "."
	var err error
	workspaceOptions.subDirPath, err = normalpath.NormalizeAndValidate(workspaceOptions.subDirPath)
	if err != nil {
		return nil, err
	}
	// TODO
	return nil, errors.New("TODO newWorkspaceForBucket")
}

func (w *workspace) GetLintConfigForOpaqueID(opaqueID string) bufconfig.LintConfig {
	return w.opaqueIDToLintConfig[opaqueID]
}

func (w *workspace) GetBreakingConfigForOpaqueID(opaqueID string) bufconfig.BreakingConfig {
	return w.opaqueIDToBreakingConfig[opaqueID]
}

func (w *workspace) GenerateConfigs() []bufconfig.GenerateConfig {
	return slicesextended.Copy(w.generateConfigs)
}

func (w *workspace) ConfiguredDepModuleRefs() []bufmodule.ModuleRef {
	return slicesextended.Copy(w.configuredDepModuleRefs)
}

func (w *workspace) LockedDepModuleKeys() []bufmodule.ModuleKey {
	return slicesextended.Copy(w.lockedDepModuleKeys)
}

func (*workspace) isWorkspace() {}

// *** PRIVATE ***

const (
	// searchResultTypeBufYAMLV1OrV1Beta1NoBufWorkYAMLV1 indicates that a v1 or v1beta1 buf.yaml
	// was found at the subDirPath, but no v1 buf.work.yaml was found anywhere in the bucket
	// going up from the subDirPath to the root of the bucket.
	searchResultTypeBufYAMLV1OrV1Beta1NoBufWorkYAMLV1 searchResultType = iota + 1
	// searchResultTypeBufYAMLV1OrV1Beta1WithBufWorkYAMLV1 indicates that a v1 or v1beta1 buf.yaml
	// was found at the subDirPath, and a v1 buf.work.yaml was found in the bucket going up
	// from the subDirPath to the root of the bucket.
	searchResultTypeBufYAMLV1OrV1Beta1WithBufWorkYAMLV1
	// searchResultTypeBufYAMLV1OrV1Beta1NoBufWorkYAMLV1 indicates that no v1 or v1beta1 buf.yaml
	// was found, and no buf.work.yaml was found. In this situation, we act as if there was a
	// default v1 buf.yaml at the subDirPath with no enclosing workspace.
	searchResultTypeNoBufYAMLV1OrV1Beta1NoBufWorkYAMLV1
	// searchResultTypeBufYAMLV1OrV1Beta1WithBufWorkYAMLV1 indicates that no v1 or v1beta1 buf.work.yaml
	// was found, but a buf.work.yaml was found. In this situation, we act as if there was a default
	// v1 buf.yaml at the subDirPath, using the buf.work.yaml as the workspace if the buf.work.yaml
	// contains the subDirPath as a directory.
	searchResultTypeNoBufYAMLV1OrV1Beta1WithBufWorkYAMLV1
)

type searchResultType int

type searchResult struct {
	// ctx is the input Context.
	ctx context.Context
	// bucket is the input storage.ReadBucket.
	bucket storage.ReadBucket
	// subDirPath is the input subDirPath.
	subDirPath string

	// searchResultType indicates how we should handle this search result.
	searchResultType searchResultType
	// bufYAMLV1OrV1Beta1Path indicates the path within the bucket that contains the
	// v1 or v1beta1 buf.yaml
	bufYAMLV1OrV1Beta1Path string
	// bufWorkYAMLPath indicates the path within the bucket that contains the buf.work.yaml.
	bufWorkYAMLPath string
}

func newSearchResult(
	ctx context.Context,
	bucket storage.ReadBucket,
	subDirPath string,
) (*searchResult, error) {
	curPath := subDirPath
	for {
		bufWorkYAMLVersion, err := bufconfig.GetBufWorkYAMLFileVersionForPrefix(ctx, bucket, curPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		bufYAMLVersion, err := bufconfig.GetBufYAMLFileVersionForPrefix(ctx, bucket, curPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		_ = bufWorkYAMLVersion
		_ = bufYAMLVersion
		if curPath == "." {
			break
		}
		curPath = normalpath.Dir(curPath)
	}
	return nil, errors.New("TODO")
}
