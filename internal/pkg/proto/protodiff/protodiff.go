// Copyright 2020 Buf Technologies Inc.
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

package protodiff

import (
	"github.com/bufbuild/buf/internal/pkg/diff"
	"github.com/bufbuild/buf/internal/pkg/proto/protoencoding"
	"google.golang.org/protobuf/types/descriptorpb"
)

// DiffFileDescriptorSetsWire diffs the two FileDescriptorSets using proto.MarshalWire.
func DiffFileDescriptorSetsWire(one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet, name string) (string, error) {
	oneData, err := protoencoding.NewWireMarshaler().Marshal(one)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewWireMarshaler().Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := diff.Diff(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffFileDescriptorSetsJSON diffs the two FileDescriptorSets using JSON.
//
// TODO: this does NOT use any resolver, so extensions will be dropped. This needs to be updated.
func DiffFileDescriptorSetsJSON(one *descriptorpb.FileDescriptorSet, two *descriptorpb.FileDescriptorSet, name string) (string, error) {
	oneData, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(one)
	if err != nil {
		return "", err
	}
	twoData, err := protoencoding.NewJSONMarshalerIndent(nil).Marshal(two)
	if err != nil {
		return "", err
	}
	output, err := diff.Diff(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
