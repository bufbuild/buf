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

package internal

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"go.uber.org/zap"
)

// Prune prunes the buf.lock.
//
// Used by both mod prune and mod update.
func Prune(
	ctx context.Context,
	logger *zap.Logger,
	controller bufctl.Controller,
	workspaceDepManager bufworkspace.WorkspaceDepManager,
	dirPath string,
) error {
	workspace, err := controller.GetWorkspace(ctx, dirPath, bufctl.WithIgnoreAndDisallowV1BufWorkYAMLs())
	if err != nil {
		return err
	}
	// Make sure the workspace builds.
	if _, err := controller.GetImageForWorkspace(
		ctx,
		workspace,
		bufctl.WithImageExcludeSourceInfo(true),
	); err != nil {
		return err
	}
	// Compute those dependencies that are in buf.yaml that are not used at all, and warn
	// about them.
	malformedDeps, err := bufworkspace.MalformedDepsForWorkspace(workspace)
	if err != nil {
		return err
	}
	for _, malformedDep := range malformedDeps {
		switch t := malformedDep.Type(); t {
		case bufworkspace.MalformedDepTypeUnused:
			logger.Sugar().Warnf(
				`Module %s is declared in your buf.yaml deps but is unused. This command only modifies buf.locks, not buf.yamls, please %s from your buf.yaml deps if it is not needed.`,
				malformedDep.ModuleFullName(),
				malformedDep.ModuleFullName(),
			)
		default:
			return fmt.Errorf("unknown MalformedDepType: %v", t)
		}
	}
	// Sep that actual computed remote dependencies based on imports. These are all
	// that is needed for buf.lock.
	depModules, err := bufmodule.RemoteDepsForModuleSet(workspace)
	if err != nil {
		return err
	}
	depModuleKeys, err := slicesext.MapError(
		depModules,
		func(remoteDep bufmodule.RemoteDep) (bufmodule.ModuleKey, error) {
			return bufmodule.ModuleToModuleKey(remoteDep, workspaceDepManager.BufLockFileDigestType())
		},
	)
	if err != nil {
		return err
	}
	return workspaceDepManager.UpdateBufLockFile(ctx, depModuleKeys)
}
