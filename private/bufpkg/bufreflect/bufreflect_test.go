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

package bufreflect

import (
	"testing"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestUnmarshalDescriptorInfoCompatibility verifies that the generated MarshalWithDescriptorInfo
// method is compatible with both UnmarshalDescriptorInfo and proto.Unmarshal.
func TestUnmarshalDescriptorInfoCompatibility(t *testing.T) {
	module := modulev1alpha1.Module{
		Documentation: "# Buf Docs",
	}

	bytes, err := module.MarshalWithDescriptorInfo()
	require.NoError(t, err)

	descriptorInfo, err := UnmarshalDescriptorInfo(bytes)
	require.NoError(t, err)
	assert.Equal(t, "buf.alpha.module.v1alpha1.Module", descriptorInfo.GetTypeName())

	serializedModule := new(modulev1alpha1.Module)
	require.NoError(t, proto.Unmarshal(bytes, serializedModule))
	assert.Equal(t, "# Buf Docs", serializedModule.GetDocumentation())
}
