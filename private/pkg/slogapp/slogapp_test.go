// Copyright 2020-2025 Buf Technologies, Inc.
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

package slogapp

import (
	"encoding/json"
	"strings"
	"testing"

	"buf.build/go/app/appext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoStack(t *testing.T) {
	t.Parallel()
	var sb strings.Builder
	logger, err := NewLogger(&sb, appext.LogLevelInfo, appext.LogFormatJSON)
	require.NoError(t, err)
	logger.Error("boom")
	var logFields map[string]any
	err = json.Unmarshal([]byte(sb.String()), &logFields)
	require.NoError(t, err)
	assert.NotContains(t, logFields, "stacktrace")
}
