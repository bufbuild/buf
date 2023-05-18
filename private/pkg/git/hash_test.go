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

package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHashFromHex(t *testing.T) {
	hex := "5edab9f970913225f985d9673ac19d61d36f0942"

	id, err := parseHashFromHex(hex)

	require.NoError(t, err)
	require.Equal(t, id.Hex(), hex)
	require.Equal(t, id.Raw(), []byte{0x5e, 0xda, 0xb9, 0xf9, 0x70, 0x91, 0x32, 0x25, 0xf9, 0x85, 0xd9, 0x67, 0x3a, 0xc1, 0x9d, 0x61, 0xd3, 0x6f, 0x9, 0x42})
}

func TestNewHashFromBytes(t *testing.T) {
	bytes := []byte{0x5e, 0xda, 0xb9, 0xf9, 0x70, 0x91, 0x32, 0x25, 0xf9, 0x85, 0xd9, 0x67, 0x3a, 0xc1, 0x9d, 0x61, 0xd3, 0x6f, 0x9, 0x42}

	id, err := newHashFromBytes(bytes)

	require.NoError(t, err)
	require.Equal(t, id.Hex(), "5edab9f970913225f985d9673ac19d61d36f0942")
	require.Equal(t, id.Raw(), bytes)
}
