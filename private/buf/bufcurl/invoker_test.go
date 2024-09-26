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

package bufcurl

import (
	"context"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCountUnrecognized(t *testing.T) {
	t.Parallel()
	descriptors, err := (&protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: []string{"./testdata"},
		},
	}).Compile(context.Background(), "test.proto")
	require.NoError(t, err)
	msgType, err := descriptors.AsResolver().FindMessageByName("foo.bar.Message")
	require.NoError(t, err)
	msg := msgType.New()
	msgData, err := os.ReadFile("./testdata/testdata.txt")
	require.NoError(t, err)
	err = protoencoding.NewTxtpbUnmarshaler(nil).Unmarshal(msgData, msg.Interface())
	require.NoError(t, err)
	// Add some unrecognized bytes
	unknownBytes := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}

	msg.SetUnknown(unknownBytes)
	expectedUnrecognized := len(unknownBytes)

	msg.Get(msgType.Descriptor().Fields().ByName("msg")).Message().SetUnknown(unknownBytes[:10])
	expectedUnrecognized += 10
	msg.Get(msgType.Descriptor().Fields().ByName("grp")).Message().SetUnknown(unknownBytes[:6])
	expectedUnrecognized += 6

	slice := msg.Get(msgType.Descriptor().Fields().ByName("rmsg")).List()
	slice.Get(0).Message().SetUnknown(unknownBytes[:10])
	slice.Get(1).Message().SetUnknown(unknownBytes[:5])
	expectedUnrecognized += 15
	slice = msg.Get(msgType.Descriptor().Fields().ByName("rgrp")).List()
	slice.Get(0).Message().SetUnknown(unknownBytes[:3])
	slice.Get(1).Message().SetUnknown(unknownBytes[:8])
	expectedUnrecognized += 11

	mapVal := msg.Get(msgType.Descriptor().Fields().ByName("mvmsg")).Map()
	mapVal.Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
		v.Message().SetUnknown(unknownBytes[:6])
		expectedUnrecognized += 6
		return true
	})

	unrecognized := countUnrecognized(msg)
	assert.Equal(t, expectedUnrecognized, unrecognized)
}
