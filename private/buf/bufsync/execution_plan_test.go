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

package bufsync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionPlanSortsModuleBranchesCorrectly(t *testing.T) {
	t.Parallel()
	t.Run("UnknownModuleSortIndex", func(t *testing.T) {
		t.Parallel()
		_, err := newExecutionPlan(nil, []ModuleBranch{newModuleBranch("", "module1", nil, nil)}, nil)
		require.Error(t, err)
	})
	t.Run("KnownModuleSortIndex", func(t *testing.T) {
		t.Parallel()
		expectedOrder := []string{
			"module1",
			"module-9",
			"A_module",
			"someOtherModule",
			"0module",
		}
		unsorted := []ModuleBranch{
			newModuleBranch("", "module1", nil, nil),
			newModuleBranch("", "0module", nil, nil),
			newModuleBranch("", "someOtherModule", nil, nil),
			newModuleBranch("", "module-9", nil, nil),
			newModuleBranch("", "A_module", nil, nil),
		}

		plan, err := newExecutionPlan(expectedOrder, unsorted, nil)
		require.NoError(t, err)
		var actualOrder []string
		for _, moduleBranch := range plan.ModuleBranchesToSync() {
			actualOrder = append(actualOrder, moduleBranch.Directory())
		}
		require.Equal(t, expectedOrder, actualOrder)
	})
}
