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

package bufwire

import (
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
)

func newJSONMarshaler(
	resolver protoencoding.Resolver,
	messageRef buffetch.MessageRef,
) protoencoding.Marshaler {
	jsonMarshalerOptions := []protoencoding.JSONMarshalerOption{
		//protoencoding.JSONMarshalerWithIndent(),
	}
	if messageRef.UseProtoNames() {
		jsonMarshalerOptions = append(
			jsonMarshalerOptions,
			protoencoding.JSONMarshalerWithUseProtoNames(),
		)
	}
	if messageRef.UseEnumNumbers() {
		jsonMarshalerOptions = append(
			jsonMarshalerOptions,
			protoencoding.JSONMarshalerWithUseEnumNumbers(),
		)
	}
	return protoencoding.NewJSONMarshaler(resolver, jsonMarshalerOptions...)
}

func newYAMLMarshaler(
	resolver protoencoding.Resolver,
	messageRef buffetch.MessageRef,
) protoencoding.Marshaler {
	yamlMarshalerOptions := []protoencoding.YAMLMarshalerOption{
		protoencoding.YAMLMarshalerWithIndent(),
	}
	if messageRef.UseProtoNames() {
		yamlMarshalerOptions = append(
			yamlMarshalerOptions,
			protoencoding.YAMLMarshalerWithUseProtoNames(),
		)
	}
	if messageRef.UseEnumNumbers() {
		yamlMarshalerOptions = append(
			yamlMarshalerOptions,
			protoencoding.YAMLMarshalerWithUseEnumNumbers(),
		)
	}
	return protoencoding.NewYAMLMarshaler(resolver, yamlMarshalerOptions...)
}
