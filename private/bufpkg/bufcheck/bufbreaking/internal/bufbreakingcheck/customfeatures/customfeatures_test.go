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

package customfeatures

import (
	"testing"

	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestResolveCppFeatures(t *testing.T) {
	t.Parallel()
	field := (*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().Fields().ByName("package")
	val, err := ResolveCppFeature(field, "string_type", protoreflect.EnumKind)
	require.NoError(t, err)
	// This will use the default value for proto2
	require.Equal(t, protobuf.CppFeatures_STRING.Number(), val.Enum())
}

func TestResolveJavaFeatures(t *testing.T) {
	t.Parallel()
	field := (*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().Fields().ByName("package")
	val, err := ResolveJavaFeature(field, "utf8_validation", protoreflect.EnumKind)
	require.NoError(t, err)
	// This will use the default value for proto2
	require.Equal(t, protobuf.JavaFeatures_DEFAULT.Number(), val.Enum())
}
