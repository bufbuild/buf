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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
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
