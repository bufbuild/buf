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

package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaceSliceOrStringToCommaSepString(t *testing.T) {
	t.Parallel()
	testInterfaceSliceOrStringToCommaSepString(t, "mystring", "mystring")
	testInterfaceSliceOrStringToCommaSepString(
		t,
		[]interface{}{
			interface{}("mystring"),
			interface{}("mystring2"),
		},
		"mystring,mystring2",
	)
	testInterfaceSliceOrStringToCommaSepString(t, nil, "")

	_, err := InterfaceSliceOrStringToCommaSepString(1)
	require.Error(t, err)
}

func testInterfaceSliceOrStringToCommaSepString(t *testing.T, in interface{}, expected string) {
	v, err := InterfaceSliceOrStringToCommaSepString(in)
	require.NoError(t, err)
	require.Equal(t, expected, v)
}
