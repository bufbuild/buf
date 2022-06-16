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

package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	assert.Equal(t, "", GetIncomingContextHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingContextHeader(ctx, "foo"))
	assert.NotNil(t, GetIncomingContextHeaders(ctx))
	assert.Empty(t, GetIncomingContextHeaders(ctx))
	assert.NotNil(t, GetOutgoingContextHeaders(ctx))
	assert.Empty(t, GetOutgoingContextHeaders(ctx))

	ctx = WithIncomingContextHeader(ctx, "foo", "foo1")
	assert.Equal(t, "foo1", GetIncomingContextHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingContextHeader(ctx, "foo"))
	assert.Equal(t, map[string]string{"foo": "foo1"}, GetIncomingContextHeaders(ctx))
	assert.Empty(t, GetOutgoingContextHeaders(ctx))

	ctx = WithIncomingContextHeader(ctx, "Foo", "foo2")
	assert.Equal(t, "foo2", GetIncomingContextHeader(ctx, "Foo"))
	assert.Equal(t, "foo2", GetIncomingContextHeader(ctx, "foo"))
	assert.Equal(t, "", GetOutgoingContextHeader(ctx, "Foo"))
	assert.Equal(t, "", GetOutgoingContextHeader(ctx, "foo"))
	assert.Equal(t, map[string]string{"foo": "foo2"}, GetIncomingContextHeaders(ctx))
	assert.Empty(t, GetOutgoingContextHeaders(ctx))

	ctx = WithOutgoingContextHeader(ctx, "bar", "bar1")
	assert.Equal(t, "bar1", GetOutgoingContextHeader(ctx, "bar"))
	assert.Equal(t, map[string]string{"bar": "bar1"}, GetOutgoingContextHeaders(ctx))

	ctx = WithOutgoingContextHeader(ctx, "Bar", "bar2")
	assert.Equal(t, "bar2", GetOutgoingContextHeader(ctx, "Bar"))
	assert.Equal(t, "bar2", GetOutgoingContextHeader(ctx, "bar"))
	assert.Equal(t, map[string]string{"bar": "bar2"}, GetOutgoingContextHeaders(ctx))

	headers := map[string]string{
		"foo": "foo3",
		"bar": "bar3",
	}
	headers2 := map[string]string{
		"foo": "foo4",
		"bar": "bar4",
		"baz": "baz4",
	}
	ctx = WithIncomingContextHeaders(ctx, headers)
	assert.Equal(t, headers, GetIncomingContextHeaders(ctx))
	ctx = WithIncomingContextHeaders(ctx, headers2)
	assert.Equal(t, headers2, GetIncomingContextHeaders(ctx))
	// make sure that the headers map is not modified
	assert.Equal(
		t,
		map[string]string{
			"foo": "foo3",
			"bar": "bar3",
		},
		headers,
	)
	ctx = WithOutgoingContextHeaders(ctx, headers)
	assert.Equal(t, headers, GetOutgoingContextHeaders(ctx))
	ctx = WithOutgoingContextHeaders(ctx, headers2)
	assert.Equal(t, headers2, GetOutgoingContextHeaders(ctx))
	// make sure that the headers map is not modified
	assert.Equal(
		t,
		map[string]string{
			"foo": "foo3",
			"bar": "bar3",
		},
		headers,
	)
}
