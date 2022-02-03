// Copyright 2020-2022 Buf Technologies, Inc.
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

// Code generated by protoc-gen-go-api. DO NOT EDIT.

package registryv1alpha1api

import (
	context "context"
)

// JSONSchemaService serves JSONSchemas describing protobuf types in buf
// modules.
type JSONSchemaService interface {
	// GetJSONSchema allows users to get an (approximate) json schema for a
	// protobuf type.
	GetJSONSchema(
		ctx context.Context,
		owner string,
		repository string,
		reference string,
		typeName string,
	) (jsonSchema []byte, err error)
}
