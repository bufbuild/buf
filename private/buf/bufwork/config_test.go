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

package bufwork

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfigV1Basic(t *testing.T) {
	t.Parallel()
	config, err := newConfigV1(
		ExternalConfigV1{
			Version:     "v1",
			Directories: []string{"./foo", "./bar/../bar"},
		},
		"buf.work.yaml",
	)
	require.NoError(t, err)
	// sorted
	require.Equal(t, []string{"bar", "foo"}, config.Directories)
}

func TestNewConfigV1RootDirectoryError(t *testing.T) {
	t.Parallel()
	_, err := newConfigV1(
		ExternalConfigV1{
			Version:     "v1",
			Directories: []string{"."},
		},
		"buf.work.yaml",
	)
	require.Error(t, err)
}
