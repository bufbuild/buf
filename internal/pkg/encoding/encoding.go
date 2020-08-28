// Copyright 2020 Buf Technologies, Inc.
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

// Package encoding provides encoding utilities.
package encoding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalJSONStrict unmarshals the data as JSON, returning a user error on failure.
//
// If the data length is 0, this is a no-op.
func UnmarshalJSONStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(data))
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(v); err != nil {
		return fmt.Errorf("could not unmarshal as JSON: %v", err)
	}
	return nil
}

// UnmarshalYAMLStrict unmarshals the data as YAML, returning a user error on failure.
//
// If the data length is 0, this is a no-op.
func UnmarshalYAMLStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	yamlDecoder := yaml.NewDecoder(bytes.NewReader(data))
	yamlDecoder.KnownFields(true)
	if err := yamlDecoder.Decode(v); err != nil {
		return fmt.Errorf("could not unmarshal as YAML: %v", err)
	}
	return nil
}

// UnmarshalJSONOrYAMLStrict unmarshals the data as JSON or YAML in order, returning
// a user error with both errors on failure.
//
// If the data length is 0, this is a no-op.
func UnmarshalJSONOrYAMLStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if jsonErr := UnmarshalJSONStrict(data, v); jsonErr != nil {
		if yamlErr := UnmarshalYAMLStrict(data, v); yamlErr != nil {
			return errors.New(jsonErr.Error() + "\n" + yamlErr.Error())
		}
	}
	return nil
}

// GetJSONStringOrStringValue returns the JSON string for the RawMessage if the
// RawMessage is a string, and the raw value as a string otherwise.
//
// If the RawMessage is empty, this returns "".
func GetJSONStringOrStringValue(rawMessage json.RawMessage) string {
	if len(rawMessage) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(rawMessage, &s); err == nil {
		return s
	}
	return string(rawMessage)
}

// MarshalYAML marshals the given value into YAML.
func MarshalYAML(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}
