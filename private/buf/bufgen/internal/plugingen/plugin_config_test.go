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

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugingen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginConfig_ParseRemoteHostName(t *testing.T) {
	host, err := parseCuratedRemoteHostName("buf.build/protocolbuffers/go:v1.28.1")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseCuratedRemoteHostName("buf.build/protocolbuffers/go")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseLegacyRemoteHostName("buf.build/protocolbuffers/plugins/go:v1.28.1-1")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
	host, err = parseLegacyRemoteHostName("buf.build/protocolbuffers/plugins/go")
	require.NoError(t, err)
	require.Equal(
		t,
		"buf.build",
		host,
	)
}
