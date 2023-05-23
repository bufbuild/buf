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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIdent(t *testing.T) {
	ident, err := parseIdent([]byte("Foo <bar@baz> 1680571785 +0445"))

	require.NoError(t, err)
	location := time.FixedZone("UTC+0445", 4*60*60+45*60)
	assert.Equal(t, ident.Name(), "Foo")
	assert.Equal(t, ident.Email(), "bar@baz")
	assert.Equal(t, ident.Timestamp(), time.Unix(1680571785, 0).In(location))
	assert.Equal(t, ident.Timestamp().Unix(), int64(1680571785))
}
