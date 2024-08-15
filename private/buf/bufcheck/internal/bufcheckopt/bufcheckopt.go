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

package bufcheckopt

import (
	"encoding/json"
	"fmt"

	"github.com/bufbuild/bufplugin-go/check"
)

const (
	enumZeroValueSuffixKey                  = "enum_zero_value_suffix"
	rpcAllowSameRequestResponseKey          = "rpc_allow_same_request_response"
	rpcAllowGoogleProtobufEmptyRequestsKey  = "rpc_allow_google_protobuf_empty_requests"
	rpcAllowGoogleProtobufEmptyResponsesKey = "rpc_allow_google_protobuf_empty_responses"
	serviceSuffixKey                        = "service_suffix"
	commentExcludesKey                      = "comment_excludes"

	defaultEnumZeroValueSuffix = "_UNSPECIFIED"
	defaultServiceSuffix       = "Service"
)

// OptionsSpec builds check.Options for clients.
//
// These can then be sent over the wire to servers.
//
// Note that we don't expose OptionsSpec for the server-side rules, instead we rely
// on the static functions, as we want to move our rules to be as native to bufplugin-go
// as possible. Instead of i.e. attaching an Options struct to bufcheckserverutil.Requests,
// we have individual rules go through the direct reading of check.Options using
// the static functions below.
//
// Only use this on the client side.
type OptionsSpec struct {
	EnumZeroValueSuffix                  string
	RPCAllowSameRequestResponse          bool
	RPCAllowGoogleProtobufEmptyRequests  bool
	RPCAllowGoogleProtobufEmptyResponses bool
	ServiceSuffix                        string
	// CommentExcludes are lines of comments that should be excluded for the COMMENT.* Rules.
	//
	// If a comment line starts with one of these excludes, it is not considered an actual comment.
	//
	// Right now, this should just be []string{"buf:lint:ignore"}, however we do this as a proper option
	// to maintain the client/server split we want; the server (ie the Rules) should not have the lint comment
	// ignore strings as part of their logic, all lint comment ignore logic is a client-side concern. However,
	// it is concievable that a COMMENT.* Rule might want to say "I don't want to consider this generic
	// line to be a comment", which is exclusive of the lint comment ignore logic. We could even potentially
	// give users the ability to configure things to ignore as part of their buf.yaml configuration. So,
	// this feels OK to expose here.
	//
	// In practice, right now, the client-side should just set this to be []string{"buf:lint:ignore"}.
	//
	// All elements must be non-empty.
	CommentExcludes []string
}

// ToOptions builds a check.Options.
func (o *OptionsSpec) ToOptions() (check.Options, error) {
	keyToValue := make(map[string][]byte, 5)
	if value := o.EnumZeroValueSuffix; len(value) > 0 {
		keyToValue[enumZeroValueSuffixKey] = []byte(value)
	}
	if o.RPCAllowSameRequestResponse {
		keyToValue[rpcAllowSameRequestResponseKey] = []byte("true")
	}
	if o.RPCAllowGoogleProtobufEmptyRequests {
		keyToValue[rpcAllowGoogleProtobufEmptyRequestsKey] = []byte("true")
	}
	if o.RPCAllowGoogleProtobufEmptyResponses {
		keyToValue[rpcAllowGoogleProtobufEmptyResponsesKey] = []byte("true")
	}
	if value := o.ServiceSuffix; len(value) > 0 {
		keyToValue[serviceSuffixKey] = []byte(value)
	}
	if value := o.CommentExcludes; len(value) > 0 {
		optionValue, err := toStringSliceValue(value)
		if err != nil {
			return nil, err
		}
		keyToValue[commentExcludesKey] = optionValue
	}
	return check.NewOptions(keyToValue)
}

// GetEnumZeroValueSuffix gets the enum zero-value suffix.
//
// Returns the default suffix if the option is not set.
func GetEnumZeroValueSuffix(options check.Options) string {
	if value := options.Get(enumZeroValueSuffixKey); len(value) > 0 {
		return string(value)
	}
	return defaultEnumZeroValueSuffix
}

// GetRPCAllowSameRequestResponse returns true if the rpc_allow_same_request_response option is set to true.
//
// Returns error if the value was unrecognized.
func GetRPCAllowSameRequestResponse(options check.Options) (bool, error) {
	return getBoolValue(options, rpcAllowSameRequestResponseKey)
}

// GetRPCAllowGoogleProtobufEmptyRequests returns true if the rpc_allow_google_protobuf_empty_requests
// option is set to true.
//
// Returns error if the value was unrecognized.
func GetRPCAllowGoogleProtobufEmptyRequests(options check.Options) (bool, error) {
	return getBoolValue(options, rpcAllowGoogleProtobufEmptyRequestsKey)
}

// GetRPCAllowGoogleProtobufEmptyResponses returns true if the rpc_allow_google_protobuf_empty_responses
// option is set to true.
//
// Returns error if the value was unrecognized.
func GetRPCAllowGoogleProtobufEmptyResponses(options check.Options) (bool, error) {
	return getBoolValue(options, rpcAllowGoogleProtobufEmptyResponsesKey)
}

// GetServiceSuffix gets the service suffix.
//
// Returns the default suffix if the option is not set.
func GetServiceSuffix(options check.Options) string {
	if value := options.Get(serviceSuffixKey); len(value) > 0 {
		return string(value)
	}
	return defaultServiceSuffix
}

// CommentExcludes are lines of comments that should be excluded for the COMMENT.* Rules.
//
// If a comment line starts with one of these excludes, it is not considered an actual comment.
//
// The returned slice is guaranteed to have only non-empty elements.
func GetCommentExcludes(options check.Options) ([]string, error) {
	return getStringSliceValue(options, commentExcludesKey)
}

// *** PRIVATE ***

func getBoolValue(options check.Options, key string) (bool, error) {
	switch value := string(options.Get(key)); value {
	case "true":
		return true, nil
	case "false", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value for option %s: %q", key, value)
	}
}

func getStringSliceValue(options check.Options, key string) ([]string, error) {
	value := options.Get(key)
	if len(value) == 0 {
		return nil, nil
	}
	var s []string
	if err := json.Unmarshal(value, &s); err != nil {
		return nil, fmt.Errorf("invalid value for option %s: %q: %w", key, string(value), err)
	}
	for _, e := range s {
		if len(e) == 0 {
			return nil, fmt.Errorf("invalid value for option %s: %q: all elements must be non-empty", key, string(value))
		}
	}
	return s, nil
}

func toStringSliceValue(s []string) ([]byte, error) {
	if len(s) == 0 {
		return nil, nil
	}
	for _, e := range s {
		if len(e) == 0 {
			return nil, fmt.Errorf("could not marshal %v into a value to be sent as a check.Options value: all elements must be non-empty", s)
		}
	}
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("could not marshal %v into a value to be sent as a check.Options value: %w", s, err)
	}
	return data, nil
}
