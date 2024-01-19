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

package bufmodule

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// TODO: Remove when we use this
var _ = resolveModuleKeys

// resolveModuleKeys gets the ModuleKey with the latest create time.
//
// All ModuleKeys expected to have the same ModuleFullName.
func resolveModuleKeys(
	ctx context.Context,
	commitProvider CommitProvider,
	moduleKeys []ModuleKey,
) (ModuleKey, error) {
	if len(moduleKeys) == 0 {
		return nil, syserror.New("expected at least one ModuleKey")
	}
	if len(moduleKeys) == 1 {
		return moduleKeys[0], nil
	}
	// Validate we're all within one registry for now.
	if moduleFullNameStrings := slicesext.ToUniqueSorted(
		slicesext.Map(
			moduleKeys,
			func(moduleKey ModuleKey) string { return moduleKey.ModuleFullName().String() },
		),
	); len(moduleFullNameStrings) > 1 {
		return nil, fmt.Errorf("multiple ModuleFullNames detected: %s", strings.Join(moduleFullNameStrings, ", "))
	}
	// Returned commits are in same order as input ModuleKeys
	commits, err := commitProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	createTime, err := commits[0].CreateTime()
	if err != nil {
		return nil, err
	}
	moduleKey := moduleKeys[0]
	// i+1 is index inside moduleKeys.
	//
	// Find the commit with the latest CreateTime, this is the ModuleKey you want to return.
	for i, commit := range commits[1:] {
		iCreateTime, err := commit.CreateTime()
		if err != nil {
			return nil, err
		}
		if iCreateTime.After(createTime) {
			moduleKey = moduleKeys[i+1]
			createTime = iCreateTime
		}
	}
	return moduleKey, nil
}
