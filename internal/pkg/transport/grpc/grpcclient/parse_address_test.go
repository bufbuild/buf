// Copyright 2020-2021 Buf Technologies, Inc.
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

package grpcclient

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{
			name:     "Just host adds dns scheme",
			input:    "buf.build",
			expected: "dns:///buf.build",
		},
		{
			name:     "Host and port adds dns scheme",
			input:    "buf.build:443",
			expected: "dns:///buf.build:443",
		},
		{
			name:     "Just IPv4 adds passthrough scheme",
			input:    "1.2.3.4",
			expected: "passthrough:///1.2.3.4",
		},
		{
			name:     "IPv4 and port adds dns scheme",
			input:    "1.2.3.4:443",
			expected: "passthrough:///1.2.3.4:443",
		},
		{
			name:     "Just IPv6 adds passthrough scheme",
			input:    "::",
			expected: "passthrough:///::",
		},
		{
			name:     "IPv6 and port adds dns scheme",
			input:    "[::]:443",
			expected: "passthrough:///[::]:443",
		},
		{
			name:     "Explicit scheme doesn't change anything",
			input:    "dns:///buf.build",
			expected: "dns:///buf.build",
		},
		{
			name:     "Explicit scheme and authority doesn't change anything",
			input:    "dns://someauthority/buf.build",
			expected: "dns://someauthority/buf.build",
		},
		{
			name:     "Explicit unix socket",
			input:    "unix:///path/to/some/file",
			expected: "unix:///path/to/some/file",
		},
		{
			name:        "HTTTPS scheme doesn't work",
			input:       "https:///buf.build",
			expectedErr: errors.New(`unexpected gRPC scheme "https", only "dns" and "unix" are supported`),
		},
		{
			name:        "Missing authority",
			input:       "dns://buf.build",
			expectedErr: errors.New(`malformed address "dns://buf.build", expected authority and host`),
		},
		{
			name:        "Empty address",
			input:       "",
			expectedErr: errors.New("address is required"),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := parseAddress(tc.input)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, output)
			}
		})
	}
}
