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

package bufconfigtest

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/stretchr/testify/require"
)

func NewTestManagedDisableRule(
	t *testing.T,
	path string,
	moduleFullName string,
	fieldName string,
	fileOption bufconfig.FileOption,
	fieldOption bufconfig.FieldOption,
) bufconfig.ManagedDisableRule {
	disable, err := bufconfig.NewDisableRule(
		path,
		moduleFullName,
		fieldName,
		fileOption,
		fieldOption,
	)
	require.NoError(t, err)
	return disable
}

func NewTestFileOptionOverrideRule(
	t *testing.T,
	path string,
	moduleFullName string,
	fileOption bufconfig.FileOption,
	value interface{},
) bufconfig.ManagedOverrideRule {
	fileOptionOverride, err := bufconfig.NewFileOptionOverrideRule(
		path,
		moduleFullName,
		fileOption,
		value,
	)
	require.NoError(t, err)
	return fileOptionOverride
}

func NewTestFieldOptionOverrideRule(
	t *testing.T,
	path string,
	moduleFullName string,
	fieldName string,
	fieldOption bufconfig.FieldOption,
	value interface{},
) bufconfig.ManagedOverrideRule {
	fieldOptionOverrid, err := bufconfig.NewFieldOptionOverrideRule(
		path,
		moduleFullName,
		bufconfig.FileOptionPhpMetadataNamespace.String(),
		fieldOption,
		value,
	)
	require.NoError(t, err)
	return fieldOptionOverrid
}
