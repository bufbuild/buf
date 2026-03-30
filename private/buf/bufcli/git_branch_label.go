// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcli

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"buf.build/go/app"
	"github.com/bufbuild/buf/private/pkg/git"
)

// GetGitBranchLabelForModule returns the current git branch name if auto-label
// behavior is enabled for the given module name.
//
// Returns ("", false, nil) if auto-label is not enabled for this module, if the
// environment variable BUF_USE_GIT_BRANCH_AS_LABEL is set to "OFF", if the directory
// is not a git repository, or if the current branch is in the disable list.
func GetGitBranchLabelForModule(
	ctx context.Context,
	logger *slog.Logger,
	envContainer app.EnvContainer,
	dir string,
	moduleName string,
	useGitBranchAsLabel []string,
	disableLabelForBranch []string,
) (string, bool, error) {
	if len(useGitBranchAsLabel) == 0 {
		return "", false, nil
	}
	if !slices.Contains(useGitBranchAsLabel, moduleName) {
		return "", false, nil
	}
	if strings.EqualFold(envContainer.Env(useGitBranchAsLabelEnvKey), "off") {
		return "", false, nil
	}
	branch, err := git.GetCurrentBranch(ctx, envContainer, dir)
	if err != nil {
		logger.WarnContext(ctx, "not in a git repository, skipping auto-label", slog.String("error", err.Error()))
		return "", false, nil
	}
	if slices.Contains(disableLabelForBranch, branch) {
		return "", false, nil
	}
	// BSR labels do not work with go get or npm SDKs if they contain "/" characters.
	// Convert to "_".
	label := strings.ReplaceAll(branch, "/", "_")
	return label, true, nil
}
