// Package encodingutil provides encoding utilities.
package encodingutil

import (
	"bytes"
	"encoding/json"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"gopkg.in/yaml.v3"
)

// UnmarshalJSONStrict unmarshals the data as JSON, returning a user error on failure.
//
// If the data length is 0, this is a no-op.
// If an error is returned, it will be with CodeInvalidArgument.
func UnmarshalJSONStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(data))
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(v); err != nil {
		return errs.NewInvalidArgumentf("could not unmarshal as JSON: %v", err)
	}
	return nil
}

// UnmarshalYAMLStrict unmarshals the data as YAML, returning a user error on failure.
//
// If the data length is 0, this is a no-op.
// If an error is returned, it will be with CodeInvalidArgument.
func UnmarshalYAMLStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	yamlDecoder := yaml.NewDecoder(bytes.NewReader(data))
	yamlDecoder.KnownFields(true)
	if err := yamlDecoder.Decode(v); err != nil {
		return errs.NewInvalidArgumentf("could not unmarshal as YAML: %v", err)
	}
	return nil
}

// UnmarshalJSONOrYAMLStrict unmarshals the data as JSON or YAML in order, returning
// a user error with both errors on failure.
//
// If the data length is 0, this is a no-op.
// If an error is returned, it will be with CodeInvalidArgument.
func UnmarshalJSONOrYAMLStrict(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if jsonErr := UnmarshalJSONStrict(data, v); jsonErr != nil {
		if yamlErr := UnmarshalYAMLStrict(data, v); yamlErr != nil {
			return errs.NewInvalidArgument(jsonErr.Error() + "\n" + yamlErr.Error())
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
