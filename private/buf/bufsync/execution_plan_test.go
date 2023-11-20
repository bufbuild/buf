package bufsync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionPlanSortsModuleBranchesCorrectly(t *testing.T) {
	t.Run("UnknownModuleSortIndex", func(t *testing.T) {
		_, err := newExecutionPlan(nil, []ModuleBranch{newModuleBranch("", "module1", nil, nil)}, nil)
		require.Error(t, err)
	})
	t.Run("KnownModuleSortIndex", func(t *testing.T) {
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
