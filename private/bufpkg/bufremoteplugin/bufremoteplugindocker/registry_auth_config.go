// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufremoteplugindocker

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// RegistryAuthConfig represents the fields required to authenticate with the Docker Engine API.
// Ref: https://docs.docker.com/engine/api/v1.41/#section/Authentication
type RegistryAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"` // domain/ip without a protocol
}

// ToHeader marshals the auth information as a base64 encoded JSON object.
// This is suitable for passing to the Docker API as the X-Registry-Auth header.
func (r *RegistryAuthConfig) ToHeader() (string, error) {
	var buffer strings.Builder
	writer := base64.NewEncoder(base64.URLEncoding, &buffer)
	err := json.NewEncoder(writer).Encode(r)
	if err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// fromHeader decodes auth information from a base64 encoded JSON object (see ToHeader).
func (r *RegistryAuthConfig) fromHeader(encoded string) error {
	base64Reader := base64.NewDecoder(base64.URLEncoding, strings.NewReader(encoded))
	if err := json.NewDecoder(base64Reader).Decode(r); err != nil {
		return err
	}
	return nil
}
