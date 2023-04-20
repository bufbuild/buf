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

package object

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	var hash ID
	err = hash.UnmarshalBinary(bytes)
	require.NoError(t, err)
	txt, err := hash.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, hash.String(), string(txt))
	bin, err := hash.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, bytes, bin)
}
