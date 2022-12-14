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

package cache

import (
	"testing"

	auditv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/audit/v1alpha1"
	"github.com/bufbuild/buf/pkg/transform/internal/protodescriptor"
	"github.com/bufbuild/buf/pkg/transform/internal/protoencoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
)

type builder func(*Cache)

func newClient(_ *testing.T, opts ...builder) *Cache {
	out := &Cache{
		data: make(map[string]any),
	}
	for _, apply := range opts {
		apply(out)
	}
	return out
}

func getPack(t *testing.T) protoencoding.Resolver {
	proto := auditv1alpha1.File_buf_alpha_audit_v1alpha1_event_proto
	descriptorProto := protodesc.ToFileDescriptorProto(proto)
	resolver, err := protoencoding.NewResolver(
		protodescriptor.FileDescriptorsForFileDescriptorProtos(descriptorProto)...)
	require.NoError(t, err)
	return resolver
}

func TestNewCache(t *testing.T) {
	t.Run("singleton returned on independent calls", func(t *testing.T) {
		firstCache := NewCache()
		secondCache := NewCache()
		assert.Equal(t, &firstCache, &secondCache)
	})
}

func TestCache_Load(t *testing.T) {
	tests := []struct {
		name string
		resp protoencoding.Resolver
		want bool
	}{
		{
			name: "nil response",
			want: false,
		},
		{
			name: "not nil response",
			resp: getPack(t),
			want: true,
		},
	}
	for _, tt := range tests {
		test := tt
		c := newClient(t)
		c.Save("foo", test.resp)
		got, ok := c.Load("foo")
		if test.want {
			assert.True(t, ok)
			assert.NotNil(t, got)
			assert.Equal(t, test.resp, got)
		} else {
			assert.False(t, ok)
			assert.Nil(t, got)
		}
	}
}

func TestCache_SaveLoad(t *testing.T) {
	cache := NewCache()
	t.Run("save and load information into cache", func(t *testing.T) {
		in := getPack(t)
		key := "foo"
		cache.Save(key, in)
		got, ok := cache.Load(key)
		assert.True(t, ok)
		assert.Equal(t, in, got)
	})
}
