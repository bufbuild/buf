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
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/spf13/pflag"
)

// Prune prunes the buf.lock.
//
// Used by both mod prune and mod update.
func Prune(
	ctx context.Context,
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

// BindLSRulesAll binds the all flag for an ls rules command.
func BindLSRulesAll(flagSet *pflag.FlagSet, addr *bool, flagName string) {
	flagSet.BoolVar(
		addr,
		flagName,
		false,
		"List all rules and not just those currently configured",
	)
}

// BindLSRulesConfig binds the config flag for an ls rules command.
func BindLSRulesConfig(flagSet *pflag.FlagSet, addr *string, flagName string, allFlagName string, versionFlagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"",
		fmt.Sprintf(
			`The buf.yaml file or data to use for configuration. Ignored if --%s or --%s is specified`,
			allFlagName,
			versionFlagName,
		),
	)
}

// BindLSRulesFormat binds the format flag for an ls rules command.
func BindLSRulesFormat(flagSet *pflag.FlagSet, addr *string, flagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"text",
		fmt.Sprintf(
			"The format to print rules as. Must be one of %s",
			stringutil.SliceToString(bufcheck.AllRuleFormatStrings),
		),
	)
}

// BindLSRulesVersion binds the version flag for an ls rules command.
func BindLSRulesVersion(flagSet *pflag.FlagSet, addr *string, flagName string, allFlagName string) {
	flagSet.StringVar(
		addr,
		flagName,
		"", // do not set a default as we need to know if this is unset
		fmt.Sprintf(
			"List all the rules for the given configuration version. Implies --%s. Must be one of %s",
			allFlagName,
			slicesext.Map(
				bufconfig.AllFileVersions,
				func(fileVersion bufconfig.FileVersion) string {
					return fileVersion.String()
				},
			),
		),
	)
}
