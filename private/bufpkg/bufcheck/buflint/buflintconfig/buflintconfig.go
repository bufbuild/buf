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

package buflintconfig

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	lintv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/lint/v1"
)

const (
	// These versions match the versions in bufconfig. We cannot take an explicit dependency
	// on bufconfig without creating a circular dependency.
	v1Beta1Version = "v1beta1"
	v1Version      = "v1"
)

// Config is the lint check config.
type Config struct {
	// Use is a list of rule and/or category IDs that are included in the lint check.
	Use []string
	// Except is a list of the rule and/or category IDs that are excluded from the lint check.
	Except []string
	// IgnoreRootPaths is a list of paths of directories and/or files that should be ignored by the lint check.
	// All paths are relative to the root of the module.
	IgnoreRootPaths []string
	// IgnoreIDOrCategoryToRootPaths is a map of rule and/or category IDs to directory and/or file paths to exclude from the
	// lint check.
	IgnoreIDOrCategoryToRootPaths map[string][]string
	// EnumZeroValueSuffix controls the behavior of the ENUM_ZERO_VALUE lint rule ID. By default, this rule
	// verifies that the zero value of all enums ends in _UNSPECIFIED. This config allows the user to override
	// this value with the given string.
	EnumZeroValueSuffix string
	// RPCAllowSameRequestResponse allows the same message type for both the request and response of an RPC.
	RPCAllowSameRequestResponse bool
	// RPCAllowGoogleProtobufEmptyRequests allows the RPC requests to use the google.protobuf.Empty message.
	RPCAllowGoogleProtobufEmptyRequests bool
	// RPCAllowGoogleProtobufEmptyResponse allows the RPC responses to use the google.protobuf.Empty message.
	RPCAllowGoogleProtobufEmptyResponses bool
	// ServiceSuffix applies to the SERVICE_SUFFIX rule ID. By default, the rule verifies that all service names
	// end with the suffix Service. This allows users to override the value with the given string.
	ServiceSuffix string
	// AllowCommentIgnores turns on comment-driven ignores.
	AllowCommentIgnores bool
	// Version represents the version of the lint rule and category IDs that should be used with this config.
	Version string
}

// NewConfigV1Beta1 returns a new Config.
func NewConfigV1Beta1(externalConfig ExternalConfigV1Beta1) *Config {
	return &Config{
		Use:                                  externalConfig.Use,
		Except:                               externalConfig.Except,
		IgnoreRootPaths:                      externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths:        externalConfig.IgnoreOnly,
		EnumZeroValueSuffix:                  externalConfig.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          externalConfig.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  externalConfig.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: externalConfig.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        externalConfig.ServiceSuffix,
		AllowCommentIgnores:                  externalConfig.AllowCommentIgnores,
		Version:                              v1Beta1Version,
	}
}

// NewConfigV1 returns a new Config.
func NewConfigV1(externalConfig ExternalConfigV1) *Config {
	return &Config{
		Use:                                  externalConfig.Use,
		Except:                               externalConfig.Except,
		IgnoreRootPaths:                      externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths:        externalConfig.IgnoreOnly,
		EnumZeroValueSuffix:                  externalConfig.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          externalConfig.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  externalConfig.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: externalConfig.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        externalConfig.ServiceSuffix,
		AllowCommentIgnores:                  externalConfig.AllowCommentIgnores,
		Version:                              v1Version,
	}
}

// ConfigForProto returns the Config given the proto.
func ConfigForProto(protoConfig *lintv1.Config) *Config {
	return &Config{
		Use:                                  protoConfig.GetUseIds(),
		Except:                               protoConfig.GetExceptIds(),
		IgnoreRootPaths:                      protoConfig.GetIgnorePaths(),
		IgnoreIDOrCategoryToRootPaths:        ignoreIDOrCategoryToRootPathsForProto(protoConfig.GetIgnoreIdPaths()),
		EnumZeroValueSuffix:                  protoConfig.GetEnumZeroValueSuffix(),
		RPCAllowSameRequestResponse:          protoConfig.GetRpcAllowSameRequestResponse(),
		RPCAllowGoogleProtobufEmptyRequests:  protoConfig.GetRpcAllowGoogleProtobufEmptyRequests(),
		RPCAllowGoogleProtobufEmptyResponses: protoConfig.GetRpcAllowGoogleProtobufEmptyResponses(),
		ServiceSuffix:                        protoConfig.GetServiceSuffix(),
		AllowCommentIgnores:                  protoConfig.GetAllowCommentIgnores(),
		Version:                              protoConfig.GetVersion(),
	}
}

// ProtoForConfig takes a *Config and returns the proto representation.
func ProtoForConfig(config *Config) *lintv1.Config {
	return &lintv1.Config{
		UseIds:                               config.Use,
		ExceptIds:                            config.Except,
		IgnorePaths:                          config.IgnoreRootPaths,
		IgnoreIdPaths:                        protoForIgnoreIDOrCategoryToRootPaths(config.IgnoreIDOrCategoryToRootPaths),
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix,
		RpcAllowSameRequestResponse:          config.RPCAllowSameRequestResponse,
		RpcAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests,
		RpcAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        config.ServiceSuffix,
		AllowCommentIgnores:                  config.AllowCommentIgnores,
		Version:                              config.Version,
	}
}

// ExternalConfigV1Beta1 is an external config.
type ExternalConfigV1Beta1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly                           map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	EnumZeroValueSuffix                  string              `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool                `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool                `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool                `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string              `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
	AllowCommentIgnores                  bool                `json:"allow_comment_ignores,omitempty" yaml:"allow_comment_ignores,omitempty"`
}

// ExternalConfigV1 is an external config.
type ExternalConfigV1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly                           map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	EnumZeroValueSuffix                  string              `json:"enum_zero_value_suffix,omitempty" yaml:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool                `json:"rpc_allow_same_request_response,omitempty" yaml:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool                `json:"rpc_allow_google_protobuf_empty_requests,omitempty" yaml:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool                `json:"rpc_allow_google_protobuf_empty_responses,omitempty" yaml:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	ServiceSuffix                        string              `json:"service_suffix,omitempty" yaml:"service_suffix,omitempty"`
	AllowCommentIgnores                  bool                `json:"allow_comment_ignores,omitempty" yaml:"allow_comment_ignores,omitempty"`
}

// ExternalConfigV1Beta1ForConfig takes a *Config and returns the v1beta1 externalconfig representation.
func ExternalConfigV1Beta1ForConfig(config *Config) ExternalConfigV1Beta1 {
	return ExternalConfigV1Beta1{
		Use:                                  config.Use,
		Except:                               config.Except,
		Ignore:                               config.IgnoreRootPaths,
		IgnoreOnly:                           config.IgnoreIDOrCategoryToRootPaths,
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          config.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        config.ServiceSuffix,
		AllowCommentIgnores:                  config.AllowCommentIgnores,
	}
}

// ExternalConfigV1ForConfig takes a *Config and returns the v1 externalconfig representation.
func ExternalConfigV1ForConfig(config *Config) ExternalConfigV1 {
	return ExternalConfigV1{
		Use:                                  config.Use,
		Except:                               config.Except,
		Ignore:                               config.IgnoreRootPaths,
		IgnoreOnly:                           config.IgnoreIDOrCategoryToRootPaths,
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          config.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        config.ServiceSuffix,
		AllowCommentIgnores:                  config.AllowCommentIgnores,
	}
}

// BytesForConfig takes a *Config and returns the deterministic []byte representation.
// We use an unexported intermediary JSON form and sort all fields to ensure that the bytes
// associated with the *Config are deterministic.
func BytesForConfig(config *Config) ([]byte, error) {
	if config == nil {
		return nil, nil
	}
	return json.Marshal(configToJSON(config))
}

type configJSON struct {
	Use                                  []string      `json:"use,omitempty"`
	Except                               []string      `json:"except,omitempty"`
	IgnoreRootPaths                      []string      `json:"ignore_root_paths,omitempty"`
	IgnoreIDOrCategoryToRootPaths        []idPathsJSON `json:"ignore_id_to_root_paths,omitempty"`
	EnumZeroValueSuffix                  string        `json:"enum_zero_value_suffix,omitempty"`
	RPCAllowSameRequestResponse          bool          `json:"rpc_allow_same_request_response,omitempty"`
	RPCAllowGoogleProtobufEmptyRequests  bool          `json:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	RPCAllowGoogleProtobufEmptyResponses bool          `json:"rpc_allow_google_protobuf_empty_response,omitempty"`
	ServiceSuffix                        string        `json:"service_suffix,omitempty"`
	AllowCommentIgnores                  bool          `json:"allow_comment_ignores,omitempty"`
	Version                              string        `json:"version,omitempty"`
}

type idPathsJSON struct {
	ID    string   `json:"id,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

func configToJSON(config *Config) *configJSON {
	ignoreIDPathsJSON := make([]idPathsJSON, 0, len(config.IgnoreIDOrCategoryToRootPaths))
	for ignoreID, rootPaths := range config.IgnoreIDOrCategoryToRootPaths {
		rootPathsCopy := make([]string, len(rootPaths))
		copy(rootPathsCopy, rootPaths)
		sort.Strings(rootPathsCopy)
		ignoreIDPathsJSON = append(ignoreIDPathsJSON, idPathsJSON{
			ID:    ignoreID,
			Paths: rootPathsCopy,
		})
	}
	sort.Slice(ignoreIDPathsJSON, func(i, j int) bool { return ignoreIDPathsJSON[i].ID < ignoreIDPathsJSON[j].ID })
	// We should not be sorting in place for the config structure, since it will mutate the
	// underlying config ordering.
	use := make([]string, len(config.Use))
	copy(use, config.Use)
	except := make([]string, len(config.Except))
	copy(except, config.Except)
	ignoreRootPaths := make([]string, len(config.IgnoreRootPaths))
	copy(ignoreRootPaths, config.IgnoreRootPaths)
	sort.Strings(use)
	sort.Strings(except)
	sort.Strings(ignoreRootPaths)
	return &configJSON{
		Use:                                  use,
		Except:                               except,
		IgnoreRootPaths:                      ignoreRootPaths,
		IgnoreIDOrCategoryToRootPaths:        ignoreIDPathsJSON,
		EnumZeroValueSuffix:                  config.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          config.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  config.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: config.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        config.ServiceSuffix,
		AllowCommentIgnores:                  config.AllowCommentIgnores,
		Version:                              config.Version,
	}
}

// PrintFileAnnotations prints the FileAnnotations to the Writer.
//
// Also accepts config-ignore-yaml.
func PrintFileAnnotations(
	writer io.Writer,
	fileAnnotations []bufanalysis.FileAnnotation,
	formatString string,
) error {
	switch s := strings.ToLower(strings.TrimSpace(formatString)); s {
	case "config-ignore-yaml":
		return printFileAnnotationsConfigIgnoreYAML(writer, fileAnnotations)
	default:
		return bufanalysis.PrintFileAnnotations(writer, fileAnnotations, s)
	}
}

func printFileAnnotationsConfigIgnoreYAML(
	writer io.Writer,
	fileAnnotations []bufanalysis.FileAnnotation,
) error {
	if len(fileAnnotations) == 0 {
		return nil
	}
	ignoreIDToRootPathMap := make(map[string]map[string]struct{})
	for _, fileAnnotation := range fileAnnotations {
		fileInfo := fileAnnotation.FileInfo()
		if fileInfo == nil || fileAnnotation.Type() == "" {
			continue
		}
		rootPathMap, ok := ignoreIDToRootPathMap[fileAnnotation.Type()]
		if !ok {
			rootPathMap = make(map[string]struct{})
			ignoreIDToRootPathMap[fileAnnotation.Type()] = rootPathMap
		}
		rootPathMap[fileInfo.Path()] = struct{}{}
	}
	if len(ignoreIDToRootPathMap) == 0 {
		return nil
	}

	sortedIgnoreIDs := make([]string, 0, len(ignoreIDToRootPathMap))
	ignoreIDToSortedRootPaths := make(map[string][]string, len(ignoreIDToRootPathMap))
	for id, rootPathMap := range ignoreIDToRootPathMap {
		sortedIgnoreIDs = append(sortedIgnoreIDs, id)
		rootPaths := make([]string, 0, len(rootPathMap))
		for rootPath := range rootPathMap {
			rootPaths = append(rootPaths, rootPath)
		}
		sort.Strings(rootPaths)
		ignoreIDToSortedRootPaths[id] = rootPaths
	}
	sort.Strings(sortedIgnoreIDs)

	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`version: v1
lint:
  ignore_only:
`)
	for _, id := range sortedIgnoreIDs {
		_, _ = buffer.WriteString("    ")
		_, _ = buffer.WriteString(id)
		_, _ = buffer.WriteString(":\n")
		for _, rootPath := range ignoreIDToSortedRootPaths[id] {
			_, _ = buffer.WriteString("      - ")
			_, _ = buffer.WriteString(rootPath)
			_, _ = buffer.WriteString("\n")
		}
	}
	_, err := writer.Write(buffer.Bytes())
	return err
}

func ignoreIDOrCategoryToRootPathsForProto(protoIgnoreIDPaths []*lintv1.IDPaths) map[string][]string {
	if protoIgnoreIDPaths == nil {
		return nil
	}
	ignoreIDOrCategoryToRootPaths := make(map[string][]string)
	for _, protoIgnoreIDPath := range protoIgnoreIDPaths {
		ignoreIDOrCategoryToRootPaths[protoIgnoreIDPath.GetId()] = protoIgnoreIDPath.GetPaths()
	}
	return ignoreIDOrCategoryToRootPaths
}

func protoForIgnoreIDOrCategoryToRootPaths(ignoreIDOrCategoryToRootPaths map[string][]string) []*lintv1.IDPaths {
	if ignoreIDOrCategoryToRootPaths == nil {
		return nil
	}
	idPathsProto := make([]*lintv1.IDPaths, 0, len(ignoreIDOrCategoryToRootPaths))
	for id, paths := range ignoreIDOrCategoryToRootPaths {
		idPathsProto = append(idPathsProto, &lintv1.IDPaths{
			Id:    id,
			Paths: paths,
		})
	}
	return idPathsProto
}
