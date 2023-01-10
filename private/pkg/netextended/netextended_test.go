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

package netextended

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		description string
		hostname    string
		isValid     bool
	}{
		{
			description: "localhost is valid",
			hostname:    "localhost",
			isValid:     true,
		},
		{
			description: "foo.com is valid",
			hostname:    "foo.com",
			isValid:     true,
		},
		{
			description: "foo.bar.com is valid",
			hostname:    "foo.bar.com",
			isValid:     true,
		},
		{
			description: "domain name with port is valid",
			hostname:    "localhost:3000",
			isValid:     true,
		},
		{
			description: "IPV4 is valid",
			hostname:    "10.40.210.253",
			isValid:     true,
		},
		{
			description: "IPV4 with port is valid",
			hostname:    "0.0.0.0:64514",
			isValid:     true,
		},
		{
			description: "IPV6 is valid",
			hostname:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			isValid:     true,
		},
		{
			description: "IPV6 with port is valid",
			hostname:    "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:64514",
			isValid:     true,
		},
		{
			description: "0.0.0.0 is valid",
			hostname:    "0.0.0.0",
			isValid:     true,
		},
		{
			description: "127.0.0.1 is valid",
			hostname:    "127.0.0.1",
			isValid:     true,
		},
		{
			description: "malformed IP is invalid",
			hostname:    "127.0.0.256",
			isValid:     false,
		},
		{
			description: "malformed domain name is invalid",
			hostname:    "this-domain-name-has-too-many-characters-in-a-segment-to-be-considered-valid.com",
			isValid:     false,
		},
		{
			description: "does not allow invalid characters",
			hostname:    "is.this.a.valid.domain?",
			isValid:     false,
		},
		{
			description: "hostname must be set",
			hostname:    "",
			isValid:     false,
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.description, func(t *testing.T) {
			hostname, err := ValidateHostname(tt.hostname)
			if tt.isValid {
				assert.Equal(t, tt.hostname, hostname)
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
