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

// TODO: This will probably go into some common bufcheck package that includes
// both the client and server. We will need to set options on the client side.
package bufcheckserveropt

import (
	"fmt"

	"github.com/bufbuild/bufplugin-go/check"
)

const (
	enumZeroValueSuffixKey                  = "enum_zero_value_suffix"
	rpcAllowSameRequestResponseKey          = "rpc_allow_same_request_response"
	rpcAllowGoogleProtobufEmptyRequestsKey  = "rpc_allow_google_protobuf_empty_requests"
	rpcAllowGoogleProtobufEmptyResponsesKey = "rpc_allow_google_protobuf_empty_responses"
	serviceSuffixKey                        = "service_suffix"

	defaultEnumZeroValueSuffix = "_UNSPECIFIED"
	defaultServiceSuffix       = "Service"
)

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
