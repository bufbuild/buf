// Copyright 2020-2022 Buf Technologies, Inc.
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

package appcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRemoteNotEmpty(t *testing.T) {
	require.Error(t, ValidateRemoteNotEmpty(""))
	require.NoError(t, ValidateRemoteNotEmpty("buf.build"))
}

func TestValidateRemoteHasNoPaths(t *testing.T) {
	require.NoError(t, ValidateRemoteHasNoPaths(""))
	require.NoError(t, ValidateRemoteHasNoPaths("buf.build"))
	require.NoError(t, ValidateRemoteHasNoPaths("buf.build/"))

	err := ValidateRemoteHasNoPaths("buf.build//")
	assert.Equal(t, err.Error(), `invalid remote address, must not contain any paths. Try removing "//" from the address.`)

	err = ValidateRemoteHasNoPaths("buf.build/test1")
	assert.Equal(t, err.Error(), `invalid remote address, must not contain any paths. Try removing "/test1" from the address.`)

	err = ValidateRemoteHasNoPaths("buf.build/test1/")
	assert.Equal(t, err.Error(), `invalid remote address, must not contain any paths. Try removing "/test1/" from the address.`)

	err = ValidateRemoteHasNoPaths("buf.build/test1/test2")
	assert.Equal(t, err.Error(), `invalid remote address, must not contain any paths. Try removing "/test1/test2" from the address.`)

}
